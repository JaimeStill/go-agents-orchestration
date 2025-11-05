package state_test

import (
	"context"
	"testing"

	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
	"github.com/JaimeStill/go-agents-orchestration/pkg/state"
)

type captureObserver struct {
	events []observability.Event
}

func (c *captureObserver) OnEvent(ctx context.Context, event observability.Event) {
	c.events = append(c.events, event)
}

func TestState_New(t *testing.T) {
	tests := []struct {
		name     string
		observer observability.Observer
	}{
		{
			name:     "with NoOpObserver",
			observer: observability.NoOpObserver{},
		},
		{
			name:     "with nil observer",
			observer: nil,
		},
		{
			name:     "with capture observer",
			observer: &captureObserver{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := state.New(tt.observer)

			val, exists := s.Get("test")
			if exists {
				t.Error("New state should not have any keys")
			}
			if val != nil {
				t.Error("Get on non-existent key should return nil")
			}
		})
	}
}

func TestState_New_EmitsEvent(t *testing.T) {
	observer := &captureObserver{}
	state.New(observer)

	if len(observer.events) != 1 {
		t.Errorf("New() emitted %d events, want 1", len(observer.events))
	}
	if observer.events[0].Type != observability.EventStateCreate {
		t.Errorf("New() emitted event type %v, want %v",
			observer.events[0].Type, observability.EventStateCreate)
	}
}

func TestState_Clone(t *testing.T) {
	observer := &captureObserver{}
	original := state.New(observer)
	original = original.Set("key1", "value1")
	original = original.Set("key2", 42)

	observer.events = nil

	cloned := original.Clone()

	if len(observer.events) != 1 {
		t.Errorf("Clone() emitted %d events, want 1", len(observer.events))
	}
	if observer.events[0].Type != observability.EventStateClone {
		t.Errorf("Clone() emitted event type %v, want %v",
			observer.events[0].Type, observability.EventStateClone)
	}

	val1, exists1 := cloned.Get("key1")
	if !exists1 || val1 != "value1" {
		t.Error("Clone() should preserve key1")
	}

	val2, exists2 := cloned.Get("key2")
	if !exists2 || val2 != 42 {
		t.Error("Clone() should preserve key2")
	}
}

func TestState_Clone_IsIndependent(t *testing.T) {
	original := state.New(observability.NoOpObserver{})
	original = original.Set("shared", "original")

	cloned := original.Clone()
	cloned = cloned.Set("shared", "modified")

	origVal, _ := original.Get("shared")
	if origVal != "original" {
		t.Error("Modifying clone should not affect original")
	}
}

func TestState_Get(t *testing.T) {
	s := state.New(observability.NoOpObserver{})
	s = s.Set("string", "hello")
	s = s.Set("int", 42)
	s = s.Set("bool", true)

	tests := []struct {
		name       string
		key        string
		wantExists bool
		wantValue  any
	}{
		{
			name:       "existing string key",
			key:        "string",
			wantExists: true,
			wantValue:  "hello",
		},
		{
			name:       "existing int key",
			key:        "int",
			wantExists: true,
			wantValue:  42,
		},
		{
			name:       "existing bool key",
			key:        "bool",
			wantExists: true,
			wantValue:  true,
		},
		{
			name:       "non-existent key",
			key:        "missing",
			wantExists: false,
			wantValue:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, exists := s.Get(tt.key)
			if exists != tt.wantExists {
				t.Errorf("Get(%q) exists = %v, want %v", tt.key, exists, tt.wantExists)
			}
			if val != tt.wantValue {
				t.Errorf("Get(%q) value = %v, want %v", tt.key, val, tt.wantValue)
			}
		})
	}
}

func TestState_Set(t *testing.T) {
	observer := &captureObserver{}
	s := state.New(observer)
	observer.events = nil

	newState := s.Set("key", "value")

	if len(observer.events) != 2 {
		t.Errorf("Set() emitted %d events, want 2 (clone + set)", len(observer.events))
	}

	hasSet := false
	for _, event := range observer.events {
		if event.Type == observability.EventStateSet {
			hasSet = true
			if event.Data["key"] != "key" {
				t.Errorf("EventStateSet data[key] = %v, want %v",
					event.Data["key"], "key")
			}
		}
	}
	if !hasSet {
		t.Error("Set() did not emit EventStateSet")
	}

	val, exists := newState.Get("key")
	if !exists || val != "value" {
		t.Error("Set() did not set value correctly")
	}
}

func TestState_Set_IsImmutable(t *testing.T) {
	original := state.New(observability.NoOpObserver{})
	original = original.Set("key", "original")

	newState := original.Set("key", "modified")

	origVal, _ := original.Get("key")
	if origVal != "original" {
		t.Error("Set() should not modify original state")
	}

	newVal, _ := newState.Get("key")
	if newVal != "modified" {
		t.Error("Set() should create new state with modified value")
	}
}

func TestState_Merge(t *testing.T) {
	observer := &captureObserver{}
	s1 := state.New(observer)
	s1 = s1.Set("key1", "value1")
	s1 = s1.Set("shared", "original")

	s2 := state.New(observer)
	s2 = s2.Set("key2", "value2")
	s2 = s2.Set("shared", "overwrite")

	observer.events = nil

	merged := s1.Merge(s2)

	hasMerge := false
	for _, event := range observer.events {
		if event.Type == observability.EventStateMerge {
			hasMerge = true
			if event.Data["keys"] != 2 {
				t.Errorf("EventStateMerge data[keys] = %v, want %v",
					event.Data["keys"], 2)
			}
		}
	}
	if !hasMerge {
		t.Error("Merge() did not emit EventStateMerge")
	}

	val1, exists1 := merged.Get("key1")
	if !exists1 || val1 != "value1" {
		t.Error("Merge() should preserve original keys")
	}

	val2, exists2 := merged.Get("key2")
	if !exists2 || val2 != "value2" {
		t.Error("Merge() should add new keys from other")
	}

	valShared, existsShared := merged.Get("shared")
	if !existsShared || valShared != "overwrite" {
		t.Error("Merge() should overwrite keys from other")
	}
}

func TestState_Merge_IsImmutable(t *testing.T) {
	s1 := state.New(observability.NoOpObserver{})
	s1 = s1.Set("key", "original")

	s2 := state.New(observability.NoOpObserver{})
	s2 = s2.Set("key", "other")

	merged := s1.Merge(s2)

	val1, _ := s1.Get("key")
	if val1 != "original" {
		t.Error("Merge() should not modify first state")
	}

	val2, _ := s2.Get("key")
	if val2 != "other" {
		t.Error("Merge() should not modify second state")
	}

	valMerged, _ := merged.Get("key")
	if valMerged != "other" {
		t.Error("Merge() should create new state with merged values")
	}
}
