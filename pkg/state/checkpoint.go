package state

import (
	"fmt"
	"sync"
)

type CheckpointStore interface {
	Save(state State) error
	Load(runID string) (State, error)
	Delete(runID string) error
	List() ([]string, error)
}

type memoryCheckpointStore struct {
	states map[string]State
	mu     sync.RWMutex
}

func NewMemoryCheckpointStore() CheckpointStore {
	return &memoryCheckpointStore{
		states: make(map[string]State),
	}
}

func (m *memoryCheckpointStore) Save(state State) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.states[state.RunID()] = state
	return nil
}

func (m *memoryCheckpointStore) Load(runID string) (State, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	state, exists := m.states[runID]
	if !exists {
		return State{}, fmt.Errorf("checkpoint not found: %s", runID)
	}
	return state, nil
}

func (m *memoryCheckpointStore) Delete(runID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.states, runID)
	return nil
}

func (m *memoryCheckpointStore) List() ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]string, 0, len(m.states))
	for id := range m.states {
		ids = append(ids, id)
	}
	return ids, nil
}

var checkpointStores = map[string]CheckpointStore{
	"memory": NewMemoryCheckpointStore(),
}

func GetCheckpointStore(name string) (CheckpointStore, error) {
	store, exists := checkpointStores[name]
	if !exists {
		return nil, fmt.Errorf("unknown checkpoint store: %s", name)
	}
	return store, nil
}

func RegisterCheckpointStore(name string, store CheckpointStore) {
	checkpointStores[name] = store
}
