package observability

import (
	"fmt"
	"log/slog"
)

// observers registry maps observer names to implementations.
// Initialized with "noop" observer for zero-overhead observability.
var observers = map[string]Observer{
	"noop": NoOpObserver{},
	"slog": NewSlogObserver(slog.Default()),
}

// GetObserver retrieves a registered observer by name.
//
// This function enables configuration-driven observer selection, allowing JSON
// configurations to specify observers as strings that are resolved at runtime.
//
// Returns an error if the observer name is not registered.
//
// Example:
//
//	observer, err := observability.GetObserver("slog")
//	if err != nil {
//	    log.Fatal(err)
//	}
func GetObserver(name string) (Observer, error) {
	obs, exists := observers[name]
	if !exists {
		return nil, fmt.Errorf("unknown observer: %s", name)
	}
	return obs, nil
}

// RegisterObserver registers a custom observer implementation under the given name.
//
// This enables extensibility - users can provide custom Observer implementations
// and register them for use via configuration.
//
// Phase 8 will use this to register production observers:
//   - "slog" - Structured logging via Go's slog package
//   - "otel" - OpenTelemetry integration
//   - Custom implementations provided by users
//
// Example:
//
//	type MyObserver struct{ logger *slog.Logger }
//	func (o *MyObserver) OnEvent(ctx context.Context, event Event) {
//	    o.logger.Info("event", "type", event.Type, "source", event.Source)
//	}
//
//	observability.RegisterObserver("my-observer", &MyObserver{logger})
func RegisterObserver(name string, observer Observer) {
	observers[name] = observer
}
