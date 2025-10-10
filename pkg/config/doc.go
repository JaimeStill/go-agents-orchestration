// Package config provides configuration structures for orchestration components.
//
// This package defines configuration types for hub instances and other orchestration
// primitives, establishing sensible defaults while allowing customization for
// different deployment scenarios.
//
// # Hub Configuration
//
// HubConfig defines settings for hub instances:
//
//	cfg := config.HubConfig{
//	    Name:              "processing-hub",
//	    ChannelBufferSize: 100,
//	    DefaultTimeout:    30 * time.Second,
//	    Logger:            slog.New(slog.NewJSONHandler(os.Stdout, nil)),
//	}
//
//	hub := hub.New(ctx, cfg)
//
// # Default Configuration
//
// The package provides defaults for common scenarios:
//
//	cfg := config.DefaultHubConfig()
//	// Name: "default"
//	// ChannelBufferSize: 100
//	// DefaultTimeout: 30s
//	// Logger: slog.Default()
//
// # Configuration Fields
//
// Name: Identifies the hub instance for logging and metrics
//
// ChannelBufferSize: Controls message channel capacity, affecting:
//   - Message throughput under load
//   - Memory usage per agent
//   - Backpressure characteristics
//
// DefaultTimeout: Request-response timeout when not specified by context:
//   - Prevents indefinite blocking
//   - Can be overridden per-request via context.WithTimeout
//
// Logger: Structured logging for hub operations:
//   - Agent registration/unregistration
//   - Message routing
//   - Error conditions
//
// # Integration with go-agents
//
// This package integrates with go-agents configuration by using slog.Logger
// from the standard library, ensuring consistent logging across the ecosystem.
//
// # Design Principles
//
// Following go-agents-orchestration design principles:
//
//   - Configuration only exists during initialization
//   - Does not persist into runtime components
//   - Validation happens at point of use (hub/messaging packages)
//   - No circular dependencies with domain packages
package config
