package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"net/url"
	"time"

	"github.com/Buzburg/fastgate/pkg/telemetry"
	"github.com/Buzburg/fastgate/pkg/triage"
)

type Server struct {
	UpstreamURL  *url.URL
	Port         int
	classifier   *triage.Classifier
	reverseProxy *httputil.ReverseProxy
}

type GenericChatRequest struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Tools    []map[string]interface{} `json:"tools,omitempty"`
}

func NewServer(upstream string, port int) (*Server, error) {
	u, err := url.Parse(upstream)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream url: %w", err)
	}
	if (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
		return nil, fmt.Errorf("upstream must be an http(s) URL without embedded credentials")
	}
	if port < 1 || port > 65535 {
		return nil, fmt.Errorf("invalid port")
	}

	s := &Server{
		UpstreamURL: u,
		Port:        port,
		classifier:  triage.NewClassifier(),
	}

	proxy := httputil.NewSingleHostReverseProxy(u)
	originalDirector := proxy.Director
	proxy.Director = func(r *http.Request) { originalDirector(r); r.Host = u.Host }
	proxy.FlushInterval = -1
	s.reverseProxy = proxy

	return s, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/metrics", s.handleMetrics)
	mux.HandleFunc("/healthz", s.handleHealth)
	mux.HandleFunc("/", s.handleProxy)

	addr := fmt.Sprintf("127.0.0.1:%d", s.Port)
	log.Printf("[fastgate] Proxy listening on %s (upstream: %s)", addr, s.UpstreamURL.String())
	return (&http.Server{Addr: addr, Handler: mux, ReadHeaderTimeout: 10 * time.Second}).ListenAndServe()
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok","engine":"fastgate-system1"}`))
}

func (s *Server) handleMetrics(w http.ResponseWriter, r *http.Request) {
	snap := telemetry.Global.Snapshot()
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(&snap)
}

func (s *Server) handleProxy(w http.ResponseWriter, r *http.Request) {
	// If not POST or not a completions/messages path, forward directly
	if r.Method != http.MethodPost || (r.URL.Path != "/v1/chat/completions" && r.URL.Path != "/v1/messages") {
		s.reverseProxy.ServeHTTP(w, r)
		return
	}

	bodyBytes, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 16<<20))
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	_ = r.Body.Close()

	var chatReq GenericChatRequest
	if err := json.Unmarshal(bodyBytes, &chatReq); err != nil || len(chatReq.Tools) == 0 {
		// Passthrough if not JSON or no tools defined
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		s.reverseProxy.ServeHTTP(w, r)
		return
	}

	// Preserve provider-specific parameters; only replace tools.
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(bodyBytes, &payload); err != nil {
		http.Error(w, "Invalid JSON", 400)
		return
	}
	// Explicit tool selection and non-text requests must pass through unchanged.
	if choice, ok := payload["tool_choice"]; ok && string(choice) != `"auto"` {
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		s.reverseProxy.ServeHTTP(w, r)
		return
	}
	// Extract prompt text from last user message
	var lastUserPrompt string
	for i := len(chatReq.Messages) - 1; i >= 0; i-- {
		msg := chatReq.Messages[i]
		if role, ok := msg["role"].(string); ok && role == "user" {
			if content, ok := msg["content"].(string); ok {
				lastUserPrompt = content
				break
			}
		}
	}

	if lastUserPrompt == "" {
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		s.reverseProxy.ServeHTTP(w, r)
		return
	}
	// Extract tool names
	var toolNames []string
	toolMap := make(map[string]map[string]interface{})
	for _, t := range chatReq.Tools {
		// Handle OpenAI format ("function": {"name": ...}) or Anthropic format ("name": ...)
		name := ""
		if fn, ok := t["function"].(map[string]interface{}); ok {
			if n, ok := fn["name"].(string); ok {
				name = n
			}
		} else if n, ok := t["name"].(string); ok {
			name = n
		}

		if name != "" {
			toolNames = append(toolNames, name)
			toolMap[name] = t
		}
	}

	// Run fast System 1 triage
	triageRes := s.classifier.Classify(lastUserPrompt, toolNames)

	// Filter down tool definitions
	var prunedTools []map[string]interface{}
	for _, keptName := range triageRes.RetainedTools {
		if toolDef, ok := toolMap[keptName]; ok {
			prunedTools = append(prunedTools, toolDef)
		}
	}

	// Rough token estimate: ~120 tokens per detailed tool definition
	origTokens := uint64(len(toolNames) * 120)
	prunedTokens := uint64(len(prunedTools) * 120)
	telemetry.Global.RecordOptimization(origTokens, prunedTokens)

	chatReq.Tools = prunedTools
	toolsJSON, _ := json.Marshal(chatReq.Tools)
	payload["tools"] = toolsJSON
	newBody, err := json.Marshal(payload)
	if err != nil {
		r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		s.reverseProxy.ServeHTTP(w, r)
		return
	}

	r.Body = io.NopCloser(bytes.NewBuffer(newBody))
	r.ContentLength = int64(len(newBody))
	r.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBody)))

	s.reverseProxy.ServeHTTP(w, r)
}
