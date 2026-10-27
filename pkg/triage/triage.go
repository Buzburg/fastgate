package triage

import (
	"strings"
	"time"
)

type ToolIntent struct {
	Category       string
	RelevanceScore float64
	MatchedTools   []string
}

type TriageResult struct {
	DecisionDuration  time.Duration `json:"decision_duration_ms"`
	PrunedToolCount   int           `json:"pruned_tool_count"`
	RetainedTools     []string      `json:"retained_tools"`
	OriginalToolCount int           `json:"original_tool_count"`
	TokenSavingsPct   float64       `json:"token_savings_pct"`
	System1Verdict    string        `json:"system1_verdict"`
}

type Classifier struct {
	// Intent keyword mappings simulating System 1 decision vectors
	intentMap map[string][]string
}

func NewClassifier() *Classifier {
	return &Classifier{
		intentMap: map[string][]string{
			"filesystem_read":  {"read_file", "view_file", "list_dir", "cat", "find", "search_files"},
			"filesystem_write": {"write_to_file", "replace_file_content", "edit_file", "create_file", "delete_file"},
			"shell_execution":  {"run_command", "bash", "execute_command", "terminal", "powershell"},
			"web_search":       {"search_web", "web_search", "fetch_url", "browser", "read_url_content"},
			"subagent_control": {"invoke_subagent", "delegate", "send_message", "manage_task"},
		},
	}
}

// Classify analyzes the input prompt and tool catalog, returning only the relevant tools in <15ms
func (c *Classifier) Classify(prompt string, availableTools []string) TriageResult {
	start := time.Now()
	lowerPrompt := strings.ToLower(prompt)

	neededCategories := make(map[string]bool)

	// Sub-15ms fast intent pattern matching
	if strings.Contains(lowerPrompt, "read") || strings.Contains(lowerPrompt, "look") ||
		strings.Contains(lowerPrompt, "find") || strings.Contains(lowerPrompt, "where") ||
		strings.Contains(lowerPrompt, "search") || strings.Contains(lowerPrompt, "inspect") ||
		strings.Contains(lowerPrompt, "show") || strings.Contains(lowerPrompt, "cat") {
		neededCategories["filesystem_read"] = true
	}

	if strings.Contains(lowerPrompt, "write") || strings.Contains(lowerPrompt, "edit") ||
		strings.Contains(lowerPrompt, "fix") || strings.Contains(lowerPrompt, "modify") ||
		strings.Contains(lowerPrompt, "add") || strings.Contains(lowerPrompt, "update") ||
		strings.Contains(lowerPrompt, "change") || strings.Contains(lowerPrompt, "create") {
		neededCategories["filesystem_write"] = true
		neededCategories["filesystem_read"] = true
	}

	if strings.Contains(lowerPrompt, "run") || strings.Contains(lowerPrompt, "test") ||
		strings.Contains(lowerPrompt, "build") || strings.Contains(lowerPrompt, "exec") ||
		strings.Contains(lowerPrompt, "compile") || strings.Contains(lowerPrompt, "install") ||
		strings.Contains(lowerPrompt, "git") || strings.Contains(lowerPrompt, "cargo") ||
		strings.Contains(lowerPrompt, "npm") {
		neededCategories["shell_execution"] = true
	}

	if strings.Contains(lowerPrompt, "google") || strings.Contains(lowerPrompt, "search web") ||
		strings.Contains(lowerPrompt, "url") || strings.Contains(lowerPrompt, "http") ||
		strings.Contains(lowerPrompt, "online") || strings.Contains(lowerPrompt, "latest news") {
		neededCategories["web_search"] = true
	}

	if strings.Contains(lowerPrompt, "delegate") || strings.Contains(lowerPrompt, "agent") {
		neededCategories["subagent_control"] = true
	}
	// Unknown intent: preserve the complete catalog.
	if len(neededCategories) == 0 {
		return TriageResult{DecisionDuration: time.Since(start), RetainedTools: availableTools, OriginalToolCount: len(availableTools), System1Verdict: "PASSTHROUGH"}
	}
	// Always default to filesystem_read if nothing matched
	if len(neededCategories) == 0 {
		neededCategories["filesystem_read"] = true
		neededCategories["shell_execution"] = true
	}

	// Filter available tools against needed categories
	var retained []string
	retainedSet := make(map[string]bool)

	for cat := range neededCategories {
		tools := c.intentMap[cat]
		for _, toolName := range tools {
			retainedSet[toolName] = true
		}
	}

	for _, tool := range availableTools {
		known := false
		for _, names := range c.intentMap {
			for _, name := range names {
				if name == tool {
					known = true
				}
			}
		}
		if retainedSet[tool] || !known {
			retained = append(retained, tool)
		}
	}

	// Fallback guarantee: if everything was pruned, retain top 3
	if len(retained) == 0 && len(availableTools) > 0 {
		limit := len(availableTools)
		if len(availableTools) < limit {
			limit = len(availableTools)
		}
		retained = availableTools[:limit]
	}

	prunedCount := len(availableTools) - len(retained)
	savingsPct := 0.0
	if len(availableTools) > 0 {
		savingsPct = (float64(prunedCount) / float64(len(availableTools))) * 100.0
	}

	duration := time.Since(start)

	verdict := "DIRECT_DISPATCH"
	if len(retained) <= 3 {
		verdict = "AGGRESSIVE_PRUNE"
	}

	return TriageResult{
		DecisionDuration:  duration,
		PrunedToolCount:   prunedCount,
		RetainedTools:     retained,
		OriginalToolCount: len(availableTools),
		TokenSavingsPct:   savingsPct,
		System1Verdict:    verdict,
	}
}
