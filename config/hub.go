package config

import (
	"log/slog"
	"time"

	agentconfig "github.com/JaimeStill/go-agents/pkg/config"
)

// HubConfig defines configuration for a Hub instance.
type HubConfig struct {
	// Hub identity
	Name string

	// Orchestrator configuration
	OrchestratorID    string
	OrchestratorAgent *agentconfig.AgentConfig // Link to go-agents

	// Communication settings
	ChannelBufferSize int
	DefaultTimeout    time.Duration

	// Observability
	Logger *slog.Logger
}

// DefaultHubConfig returns a HubConfig with sensible defaults.
func DefaultHubConfig() HubConfig {
	return HubConfig{
		Name:              "default",
		OrchestratorID:    "orchestrator",
		OrchestratorAgent: nil, // Must be provided by user
		ChannelBufferSize: 100,
		DefaultTimeout:    30 * time.Second,
		Logger:            slog.Default(),
	}
}
