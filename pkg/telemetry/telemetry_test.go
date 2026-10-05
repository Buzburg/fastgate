package telemetry

import (
	"testing"
)

func TestMetrics_RecordOptimization(t *testing.T) {
	m := &Metrics{}
	m.RecordOptimization(1000, 400) // Saved 600 tokens

	snap := m.Snapshot()
	if snap.TotalRequests != 1 {
		t.Errorf("Expected 1 request, got %d", snap.TotalRequests)
	}
	if snap.TokensSaved != 600 {
		t.Errorf("Expected 600 tokens saved, got %d", snap.TokensSaved)
	}
	if snap.EstimatedDollars <= 0 {
		t.Errorf("Expected positive estimated dollars saved, got %f", snap.EstimatedDollars)
	}
}
