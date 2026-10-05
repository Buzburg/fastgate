package gate

import (
	"strings"
	"time"
)

type GateVerdict string

const (
	VerdictPass    GateVerdict = "PASS_CLEAN"
	VerdictFail    GateVerdict = "FAIL_KNOWN_ERROR"
	VerdictNeutral GateVerdict = "REQUIRE_REASONING"
)

type ExecutionAnalysis struct {
	Verdict        GateVerdict   `json:"verdict"`
	Summary        string        `json:"summary"`
	CanBypassLLM   bool          `json:"can_bypass_llm"`
	SuggestedAction string       `json:"suggested_action"`
	EvaluationTime time.Duration `json:"evaluation_time_ms"`
}

type GateKeeper struct{}

func NewGateKeeper() *GateKeeper {
	return &GateKeeper{}
}

// EvaluateOutput scans tool results (like test or build output) to determine if expensive LLM triage can be avoided
func (g *GateKeeper) EvaluateOutput(command string, output string, exitCode int) ExecutionAnalysis {
	start := time.Now()
	lowerOut := strings.ToLower(output)

	// Clean pass detection
	if exitCode == 0 {
		if strings.Contains(lowerOut, "test result: ok") ||
			strings.Contains(lowerOut, "passed") ||
			strings.Contains(lowerOut, "all tests pass") ||
			strings.Contains(lowerOut, "build completed successfully") ||
			strings.Contains(lowerOut, "success") {
			return ExecutionAnalysis{
				Verdict:         VerdictPass,
				Summary:         "Deterministic pass verified locally without upstream LLM tokens.",
				CanBypassLLM:    true,
				SuggestedAction: "Proceed to next task step immediately.",
				EvaluationTime:  time.Since(start),
			}
		}
	}

	// Syntax / Compilation error detection
	if exitCode != 0 {
		if strings.Contains(lowerOut, "syntaxerror") ||
			strings.Contains(lowerOut, "cannot find module") ||
			strings.Contains(lowerOut, "undefined reference") ||
			strings.Contains(lowerOut, "compilation error") {
			return ExecutionAnalysis{
				Verdict:         VerdictFail,
				Summary:         "Deterministic build/syntax failure detected.",
				CanBypassLLM:    false, // Agent needs to see the compiler error to fix it
				SuggestedAction: "Forward compiler trace to model with focused context.",
				EvaluationTime:  time.Since(start),
			}
		}
	}

	return ExecutionAnalysis{
		Verdict:         VerdictNeutral,
		Summary:         "Non-deterministic or multi-faceted output requiring full model evaluation.",
		CanBypassLLM:    false,
		SuggestedAction: "Forward to upstream LLM.",
		EvaluationTime:  time.Since(start),
	}
}
