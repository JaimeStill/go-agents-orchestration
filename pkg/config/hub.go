package config

import (
	"log/slog"
	"time"
)

// HubConfig defines configuration for a Hub instance.
type HubConfig struct {
	// Hub identity
	Name string

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
		ChannelBufferSize: 100,
		DefaultTimeout:    30 * time.Second,
		Logger:            slog.Default(),
	}
}
