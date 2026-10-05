package gate

import (
	"testing"
)

func TestGateKeeper_EvaluateOutput(t *testing.T) {
	gk := NewGateKeeper()

	// 1. Clean test pass
	analysis := gk.EvaluateOutput("cargo test", "test result: ok. 2 passed; 0 failed", 0)
	if analysis.Verdict != VerdictPass {
		t.Errorf("Expected VerdictPass, got %v", analysis.Verdict)
	}
	if !analysis.CanBypassLLM {
		t.Errorf("Expected CanBypassLLM to be true on clean test pass")
	}

	// 2. Syntax failure
	errAnalysis := gk.EvaluateOutput("node index.js", "SyntaxError: Unexpected token", 1)
	if errAnalysis.Verdict != VerdictFail {
		t.Errorf("Expected VerdictFail, got %v", errAnalysis.Verdict)
	}
}
