package state

import (
	"context"
	"maps"
	"time"

	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
	"github.com/google/uuid"
)

// State represents immutable workflow state flowing through graph execution.
//
// State uses map[string]any for maximum flexibility, enabling dynamic workflows
// similar to LangGraph. All operations are immutable - modifications return new
// State instances with updated values.
//
// Observer integration is built-in from Phase 2, enabling production-grade
// observability without retrofit friction in later phases.
type State struct {
	data           map[string]any
	observer       observability.Observer
	runID          string
	checkpointNode string
	timestamp      time.Time
}

func (s State) RunID() string {
	return s.runID
}

func (s State) CheckpointNode() string {
	return s.checkpointNode
}

func (s State) Timestamp() time.Time {
	return s.timestamp
}

// New creates a new empty State with the given observer.
//
// If observer is nil, NoOpObserver is used automatically. This prevents nil
// pointer dereferences while enabling zero-overhead operation when observability
// is not needed.
//
// Example:
//
//	observer := observability.NoOpObserver{}
//	s := state.New(observer)
func New(observer observability.Observer) State {
	if observer == nil {
		observer = observability.NoOpObserver{}
	}

	s := State{
		data:      make(map[string]any),
		observer:  observer,
		runID:     uuid.New().String(),
		timestamp: time.Now(),
	}

	observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateCreate,
		Timestamp: s.timestamp,
		Source:    "state",
		Data:      map[string]any{},
	})

	return s
}

// Clone creates an independent copy of the State.
//
// The returned State has its own data map (shallow clone) but preserves the
// same observer reference. Modifications to the clone do not affect the original.
//
// Uses maps.Clone for efficient copying.
//
// Example:
//
//	original := state.New(observer).Set("key", "value")
//	cloned := original.Clone()
//	cloned = cloned.Set("key", "modified")
//	// original still has "value", cloned has "modified"
func (s State) Clone() State {
	newState := State{
		data:           maps.Clone(s.data),
		observer:       s.observer,
		runID:          s.runID,
		checkpointNode: s.checkpointNode,
		timestamp:      s.timestamp,
	}

	s.observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateClone,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"keys": len(newState.data)},
	})

	return newState
}

// Get retrieves a value from the State by key.
//
// Returns the value and true if the key exists, nil and false otherwise.
// Callers should check the exists flag before using the value to avoid nil panics.
//
// Example:
//
//	value, exists := state.Get("user")
//	if !exists {
//	    log.Fatal("user not found in state")
//	}
//	user := value.(string)  // Type assertion required due to any type
func (s State) Get(key string) (any, bool) {
	val, exists := s.data[key]
	return val, exists
}

// Set creates a new State with the key-value pair added or updated.
//
// The original State is not modified (immutability). The new State preserves
// all existing keys and adds/updates the specified key.
//
// Emits EventStateSet through the observer.
//
// Example:
//
//	s1 := state.New(observer)
//	s2 := s1.Set("user", "alice")
//	s3 := s2.Set("count", 42)
//	// s1 is empty, s2 has user, s3 has user+count
func (s State) Set(key string, value any) State {
	newState := s.Clone()
	newState.data[key] = value

	s.observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateSet,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"key": key},
	})

	return newState
}

func (s State) SetCheckpointNode(node string) State {
	newState := s.Clone()
	newState.checkpointNode = node
	newState.timestamp = time.Now()
	return newState
}

// Merge creates a new State combining this State with another State.
//
// Keys from the other State are copied into the new State, overwriting any
// existing keys with the same name. The original States are not modified.
//
// Uses maps.Copy for efficient merging.
//
// Emits EventStateMerge through the observer.
//
// Example:
//
//	s1 := state.New(observer).Set("user", "alice").Set("role", "admin")
//	s2 := state.New(observer).Set("count", 42).Set("role", "user")
//	merged := s1.Merge(s2)
//	// merged has: user=alice, role=user (overwritten), count=42
func (s State) Merge(other State) State {
	newState := s.Clone()
	maps.Copy(newState.data, other.data)

	s.observer.OnEvent(context.Background(), observability.Event{
		Type:      observability.EventStateMerge,
		Timestamp: time.Now(),
		Source:    "state",
		Data:      map[string]any{"keys": len(other.data)},
	})

	return newState
}

func (s State) Checkpoint(store CheckpointStore) error {
	return store.Save(s)
}
