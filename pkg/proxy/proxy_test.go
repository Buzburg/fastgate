package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPreservesRequestAndStreaming(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body map[string]interface{}
		json.NewDecoder(r.Body).Decode(&body)
		if body["stream"] != true || body["max_tokens"] != float64(42) || body["system"] != "Keep context" {
			t.Errorf("lost request fields: %v", body)
		}
		if r.Header.Get("Authorization") != "Bearer fixture" {
			t.Error("lost authorization")
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(201)
		w.Write([]byte("data: fixture\n\n"))
	}))
	defer upstream.Close()
	s, err := NewServer(upstream.URL, 8080)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/v1/messages", strings.NewReader(`{"model":"fixture","stream":true,"max_tokens":42,"system":"Keep context","messages":[{"role":"user","content":"read file"}],"tools":[{"name":"read_file"},{"name":"write_to_file"}]}`))
	req.Header.Set("Authorization", "Bearer fixture")
	w := httptest.NewRecorder()
	s.handleProxy(w, req)
	if w.Code != 201 || w.Body.String() != "data: fixture\n\n" {
		t.Fatal(w.Code, w.Body.String())
	}
}
func TestRejectsInvalidUpstream(t *testing.T) {
	for _, u := range []string{"", "file:///tmp/test", "localhost:9999"} {
		if _, err := NewServer(u, 8080); err == nil {
			t.Errorf("accepted %q", u)
		}
	}
}

func TestPassThroughKeepsExplicitChoiceAndStructuredContent(t *testing.T) {
	for _, body := range []string{
		`{"model":"fixture","tool_choice":{"type":"tool","name":"write_to_file"},"messages":[{"role":"user","content":"read file"}],"tools":[{"name":"read_file"},{"name":"write_to_file"}]}`,
		`{"messages":[{"role":"user","content":[{"type":"text","text":"read file"}]}],"tools":[{"name":"read_file"},{"name":"write_to_file"}]}`,
	} {
		upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			b, _ := io.ReadAll(r.Body)
			if string(b) != body {
				t.Errorf("passthrough changed request: %s", b)
			}
			w.WriteHeader(429)
			w.Write([]byte("rate limited"))
		}))
		s, _ := NewServer(upstream.URL, 8080)
		w := httptest.NewRecorder()
		s.handleProxy(w, httptest.NewRequest("POST", "/v1/messages", strings.NewReader(body)))
		if w.Code != 429 || w.Body.String() != "rate limited" {
			t.Fatal(w.Code, w.Body.String())
		}
		upstream.Close()
	}
}
