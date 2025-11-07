package config

// GraphConfig defines configuration for state graph execution.
//
// This configuration follows the go-agents pattern: used only during initialization,
// then transformed into domain objects. The Observer field is a string to enable
// JSON configuration with observer resolution at runtime.
//
// Example JSON:
//
//	{
//	  "name": "document-workflow",
//	  "observer": "slog",
//	  "max_iterations": 500
//	}
//
// Example resolution:
//
//	var cfg config.GraphConfig
//	json.Unmarshal(data, &cfg)
//	observer, err := observability.GetObserver(cfg.Observer)
//	graph := state.NewGraph(cfg, observer)
type GraphConfig struct {
	// Name identifies the graph for observability
	Name string `json:"name"`

	// Observer specifies which observer implementation to use ("noop", "slog", etc.)
	Observer string `json:"observer"`

	// MaxIterations limits graph execution to prevent infinite loops
	MaxIterations int `json:"max_iterations"`
}

// DefaultGraphConfig returns sensible defaults for graph execution.
//
// Uses "noop" observer for zero-overhead execution when observability not needed.
// Sets MaxIterations to 1000 to protect against infinite loops.
func DefaultGraphConfig(name string) GraphConfig {
	return GraphConfig{
		Name:          name,
		Observer:      "slog",
		MaxIterations: 1000,
	}
}
