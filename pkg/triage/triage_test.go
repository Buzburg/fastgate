package triage

import (
	"testing"
	"time"
)

func TestClassifier_Classify(t *testing.T) {
	classifier := NewClassifier()
	tools := []string{
		"read_file", "view_file", "write_to_file", "replace_file_content",
		"run_command", "search_web", "invoke_subagent",
	}

	// 1. Read-focused prompt
	res := classifier.Classify("Show me the contents of main.go", tools)
	if res.DecisionDuration > 15*time.Millisecond {
		t.Errorf("Expected sub-15ms triage latency, got %v", res.DecisionDuration)
	}
	if len(res.RetainedTools) == 0 {
		t.Errorf("Expected retained tools, got 0")
	}
	// Should contain read_file and NOT invoke_subagent
	hasRead := false
	for _, tool := range res.RetainedTools {
		if tool == "read_file" || tool == "view_file" {
			hasRead = true
		}
	}
	if !hasRead {
		t.Errorf("Expected read_file in retained tools")
	}

	// 2. Shell execution prompt
	resShell := classifier.Classify("Run the test suite with cargo test", tools)
	hasRun := false
	for _, tool := range resShell.RetainedTools {
		if tool == "run_command" {
			hasRun = true
		}
	}
	if !hasRun {
		t.Errorf("Expected run_command in retained tools for shell prompt")
	}
}
