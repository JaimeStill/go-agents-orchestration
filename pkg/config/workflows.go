package config

// ChainConfig defines configuration for sequential chain execution.
//
// This configuration follows the go-agents pattern: used only during initialization,
// then transformed into domain objects. The Observer field is a string to enable
// JSON configuration with observer resolution at runtime.
//
// Example JSON:
//
//	{
//	  "capture_intermediate_states": true,
//	  "observer": "slog"
//	}
//
// Example usage:
//
//	var cfg config.ChainConfig
//	json.Unmarshal(data, &cfg)
//	result, err := workflows.ProcessChain(ctx, cfg, items, initial, processor, progress)
type ChainConfig struct {
	// CaptureIntermediateStates determines whether to capture state after each step.
	// When true, ChainResult.Intermediate contains all intermediate states including initial.
	// When false, only final state is returned.
	CaptureIntermediateStates bool `json:"capture_intermediate_states"`

	// Observer specifies which observer implementation to use ("noop", "slog", etc.)
	Observer string `json:"observer"`
}

// DefaultChainConfig returns sensible defaults for chain execution.
//
// Uses "noop" observer for zero-overhead execution when observability not needed.
// Intermediate state capture is disabled by default to minimize memory usage.
func DefaultChainConfig() ChainConfig {
	return ChainConfig{
		CaptureIntermediateStates: false,
		Observer:                  "noop",
	}
}
