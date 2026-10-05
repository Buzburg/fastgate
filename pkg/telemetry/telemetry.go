package telemetry

import (
	"sync"
	"sync/atomic"
)

type Metrics struct {
	TotalRequests    uint64  `json:"total_requests"`
	FastRouted       uint64  `json:"fast_routed_requests"`
	OriginalTokens   uint64  `json:"original_tokens"`
	PrunedTokens     uint64  `json:"pruned_tokens"`
	TokensSaved      uint64  `json:"tokens_saved"`
	EstimatedDollars float64 `json:"estimated_dollars_saved"`
	mu               sync.RWMutex
}

var Global = &Metrics{}

func (m *Metrics) RecordOptimization(origTokens, prunedTokens uint64) {
	atomic.AddUint64(&m.TotalRequests, 1)
	atomic.AddUint64(&m.FastRouted, 1)
	atomic.AddUint64(&m.OriginalTokens, origTokens)
	atomic.AddUint64(&m.PrunedTokens, prunedTokens)

	if origTokens > prunedTokens {
		saved := origTokens - prunedTokens
		atomic.AddUint64(&m.TokensSaved, saved)

		m.mu.Lock()
		// Benchmark estimate: $3.00 per million prompt tokens (Sonnet 3.7 tier)
		m.EstimatedDollars += float64(saved) / 1_000_000.0 * 3.00
		m.mu.Unlock()
	}
}

func (m *Metrics) Snapshot() Metrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return Metrics{
		TotalRequests:    atomic.LoadUint64(&m.TotalRequests),
		FastRouted:       atomic.LoadUint64(&m.FastRouted),
		OriginalTokens:   atomic.LoadUint64(&m.OriginalTokens),
		PrunedTokens:     atomic.LoadUint64(&m.PrunedTokens),
		TokensSaved:      atomic.LoadUint64(&m.TokensSaved),
		EstimatedDollars: m.EstimatedDollars,
	}
}
