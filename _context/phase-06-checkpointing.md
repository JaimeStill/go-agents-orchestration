# Phase 6: Checkpointing Infrastructure - Implementation Guide

## Problem Context

State graphs currently execute from start to finish without recovery capability. If execution fails midway through a long-running workflow, all progress is lost and the entire graph must be re-executed from the beginning. This is inefficient for workflows with expensive operations (LLM calls, data processing) and prevents reliable execution of long-running multi-agent workflows.

Phase 6 adds production-grade checkpointing to state graphs, enabling:
- Automatic state snapshots at configurable intervals during execution
- Recovery from the last successful checkpoint after failures
- Configurable checkpoint preservation for audit trails
- Observer integration for checkpoint lifecycle visibility

## Architecture Approach

### Core Insight: Checkpoint IS State

The key architectural insight is that **a checkpoint is simply State captured at a particular execution stage**. State already contains all necessary checkpoint metadata:
- `runID` - unique execution identifier
- `checkpointNode` - which node produced this State
- `timestamp` - when this State was created/checkpointed
- `data` - application state
- `observer` - execution context

Therefore, we don't need a separate Checkpoint wrapper type. State serves directly as the checkpoint.

### Design Decisions

Based on planning discussions, the following design decisions were confirmed:

1. **Checkpoint IS State**: No separate Checkpoint struct - State contains all checkpoint metadata
2. **CheckpointStore Works with State**: Store interface operates directly on State objects
3. **State-Encapsulated Behavior**: State provides checkpoint methods (e.g., `state.Checkpoint(store)`)
4. **Checkpoint Interval**: Configured at graph construction via GraphConfig.Checkpoint
5. **Interval=0 Semantics**: Means checkpointing disabled
6. **Resume Starting Point**: Skip to next node after checkpoint (checkpoint represents completed work)
7. **Save Error Handling**: Fail-fast - stop execution if checkpoint save fails
8. **Cleanup Behavior**: Configurable via `Preserve` flag in CheckpointConfig
9. **Execute Signature**: Remains `(State, error)` - no breaking changes needed

### Checkpoint Save Timing

Checkpoints are saved AFTER node execution completes successfully:

```
Execute Node → Success → Update State → Check Interval → state.Checkpoint(store)
```

This means:
- State represents: "Node X completed successfully, here's the resulting state"
- Resume starts from: Next node after the checkpointed node
- Re-execution avoided: Node already executed, don't repeat expensive operations

### Integration with GraphConfig

Checkpoint configuration follows the existing pattern established for Observer:
- CheckpointConfig defined in `pkg/config/state.go`
- Added to GraphConfig structure
- Resolved at graph construction time
- Configuration lifecycle: exists only during initialization

## Implementation Steps

### Step 1: Extend State with Checkpoint Metadata

**File**: `pkg/state/state.go`

Modify the State struct to include checkpoint metadata and add accessor methods.

**Add Fields to State Struct**:

Locate the State struct definition (around line 20) and add checkpoint metadata fields:

```go
type State struct {
	data           map[string]any
	observer       observability.Observer
	runID          string
	checkpointNode string
	timestamp      time.Time
}
```

**Add Imports**:

Add these imports at the top of the file:

```go
"github.com/google/uuid"
"time"
```

**Modify State Constructor**:

Update the `New` function to initialize checkpoint metadata (around line 38):

```go
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
```

**Update Clone Method**:

Modify Clone to preserve checkpoint metadata (around line 73):

```go
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
```

**Add Checkpoint Metadata Accessors**:

Add these methods at the end of state.go (after Merge method):

```go
func (s State) RunID() string {
	return s.runID
}

func (s State) CheckpointNode() string {
	return s.checkpointNode
}

func (s State) Timestamp() time.Time {
	return s.timestamp
}

func (s State) SetCheckpointNode(node string) State {
	newState := s.Clone()
	newState.checkpointNode = node
	newState.timestamp = time.Now()
	return newState
}
```

### Step 2: Create CheckpointStore Interface and Implementation

**File**: `pkg/state/checkpoint.go` (NEW)

Create new file with checkpoint store abstraction.

**Package Declaration and Imports**:

```go
package state

import (
	"fmt"
	"sync"
)
```

**CheckpointStore Interface**:

```go
type CheckpointStore interface {
	Save(state State) error
	Load(runID string) (State, error)
	Delete(runID string) error
	List() ([]string, error)
}
```

**MemoryCheckpointStore Implementation**:

```go
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
```

**CheckpointStore Registry**:

Add registry infrastructure following the Observer pattern:

```go
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
```

**State Checkpoint Method**:

Add this convenience method to provide checkpoint saving:

```go
func (s State) Checkpoint(store CheckpointStore) error {
	return store.Save(s)
}
```

### Step 3: Add CheckpointConfig to GraphConfig

**File**: `pkg/config/state.go`

Add checkpoint configuration structure.

**CheckpointConfig Structure**:

Add this struct before the GraphConfig definition:

```go
type CheckpointConfig struct {
	Store    string `json:"store"`
	Interval int    `json:"interval"`
	Preserve bool   `json:"preserve"`
}

func DefaultCheckpointConfig() CheckpointConfig {
	return CheckpointConfig{
		Store:    "memory",
		Interval: 0,
		Preserve: false,
	}
}
```

**Update GraphConfig**:

Add CheckpointConfig field to GraphConfig:

```go
type GraphConfig struct {
	Name          string           `json:"name"`
	Observer      string           `json:"observer"`
	MaxIterations int              `json:"max_iterations"`
	Checkpoint    CheckpointConfig `json:"checkpoint"`
}
```

**Update DefaultGraphConfig**:

Modify DefaultGraphConfig to include checkpoint defaults:

```go
func DefaultGraphConfig(name string) GraphConfig {
	return GraphConfig{
		Name:          name,
		Observer:      "slog",
		MaxIterations: 1000,
		Checkpoint:    DefaultCheckpointConfig(),
	}
}
```

### Step 4: Integrate Checkpointing into StateGraph

**File**: `pkg/state/graph.go`

Add checkpoint infrastructure to stateGraph.

**Add Fields to stateGraph**:

Locate the stateGraph struct (around line 52) and add checkpoint fields:

```go
type stateGraph struct {
	name                string
	nodes               map[string]StateNode
	edges               map[string][]Edge
	entryPoint          string
	exitPoints          map[string]bool
	maxIterations       int
	observer            observability.Observer
	checkpointStore     CheckpointStore
	checkpointInterval  int
	preserveCheckpoints bool
}
```

**Update NewGraph Constructor**:

Modify NewGraph to initialize checkpoint infrastructure using the registry (around line 83):

```go
func NewGraph(cfg config.GraphConfig) (StateGraph, error) {
	observer, err := observability.GetObserver(cfg.Observer)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve observer: %w", err)
	}

	var checkpointStore CheckpointStore
	if cfg.Checkpoint.Interval > 0 {
		checkpointStore, err = GetCheckpointStore(cfg.Checkpoint.Store)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve checkpoint store: %w", err)
		}
	}

	return &stateGraph{
		name:                cfg.Name,
		nodes:               make(map[string]StateNode),
		edges:               make(map[string][]Edge),
		exitPoints:          make(map[string]bool),
		maxIterations:       cfg.MaxIterations,
		observer:            observer,
		checkpointStore:     checkpointStore,
		checkpointInterval:  cfg.Checkpoint.Interval,
		preserveCheckpoints: cfg.Checkpoint.Preserve,
	}, nil
}
```

**Modify Execute Method**:

Add checkpoint integration to the existing Execute method. Make these three specific additions:

**Addition 1: Update state with checkpoint node (after successful node execution)**

Locate this line in the Execute method (around line 340):
```go
state = newState
```

Replace it with:
```go
state = newState.SetCheckpointNode(current)
```

**Addition 2: Add checkpoint saving logic (after state update)**

Immediately after the `state = newState.SetCheckpointNode(current)` line, add this checkpoint saving block:

```go
if g.checkpointInterval > 0 && iterations%g.checkpointInterval == 0 {
	if err := state.Checkpoint(g.checkpointStore); err != nil {
		return state, &ExecutionError{
			NodeName: current,
			State:    state,
			Path:     path,
			Err:      fmt.Errorf("checkpoint save failed: %w", err),
		}
	}

	g.observer.OnEvent(ctx, observability.Event{
		Type:      observability.EventCheckpointSave,
		Timestamp: time.Now(),
		Source:    g.name,
		Data: map[string]any{
			"node":   current,
			"run_id": state.RunID(),
		},
	})
}
```

**Addition 3: Add checkpoint cleanup on successful completion (before exit point return)**

Locate the exit point check (around line 350):
```go
if g.exitPoints[current] {
	g.observer.OnEvent(ctx, observability.Event{
		Type:      observability.EventGraphComplete,
		Timestamp: time.Now(),
		Source:    g.name,
		Data: map[string]any{
			"exit_point": current,
			"iterations": iterations,
		},
	})

	return state, nil
}
```

Add checkpoint cleanup before the return statement:
```go
if g.exitPoints[current] {
	g.observer.OnEvent(ctx, observability.Event{
		Type:      observability.EventGraphComplete,
		Timestamp: time.Now(),
		Source:    g.name,
		Data: map[string]any{
			"exit_point": current,
			"iterations": iterations,
		},
	})

	if !g.preserveCheckpoints && g.checkpointInterval > 0 {
		g.checkpointStore.Delete(state.RunID())
	}

	return state, nil
}
```

These are the ONLY changes needed to the Execute method. Leave all other code (error handling, edge evaluation, cycle detection, etc.) exactly as it currently is.

### Step 5: Add Resume Method

**File**: `pkg/state/graph.go`

Add resume capability to stateGraph.

**Resume Method**:

Add this method to the stateGraph type (after the Execute method):

```go
func (g *stateGraph) Resume(ctx context.Context, runID string) (State, error) {
	if g.checkpointStore == nil {
		return State{}, fmt.Errorf("checkpointing not enabled for this graph")
	}

	state, err := g.checkpointStore.Load(runID)
	if err != nil {
		return State{}, fmt.Errorf("failed to load checkpoint: %w", err)
	}

	g.observer.OnEvent(ctx, observability.Event{
		Type:      observability.EventCheckpointLoad,
		Timestamp: time.Now(),
		Source:    g.name,
		Data: map[string]any{
			"node":   state.CheckpointNode(),
			"run_id": runID,
		},
	})

	nextNode, err := g.findNextNode(state.CheckpointNode(), state)
	if err != nil {
		return State{}, fmt.Errorf("failed to find next node after checkpoint: %w", err)
	}

	g.observer.OnEvent(ctx, observability.Event{
		Type:      observability.EventCheckpointResume,
		Timestamp: time.Now(),
		Source:    g.name,
		Data: map[string]any{
			"checkpoint_node": state.CheckpointNode(),
			"resume_node":     nextNode,
			"run_id":          runID,
		},
	})

	return g.executeFrom(ctx, nextNode, state)
}
```

**Helper Method - findNextNode**:

Add this helper method (after Resume):

```go
func (g *stateGraph) findNextNode(fromNode string, state State) (string, error) {
	edges, hasEdges := g.edges[fromNode]
	if !hasEdges {
		if g.exitPoints[fromNode] {
			return "", fmt.Errorf("checkpoint was at exit point, execution already complete")
		}
		return "", fmt.Errorf("no outgoing edges from checkpoint node: %s", fromNode)
	}

	for i := range edges {
		edge := &edges[i]
		if edge.Predicate == nil || edge.Predicate(state) {
			return edge.To, nil
		}
	}

	return "", fmt.Errorf("no valid edge transition from checkpoint node: %s", fromNode)
}
```

**Helper Method - executeFrom**:

Add this helper method to resume execution from a specific node:

```go
func (g *stateGraph) executeFrom(ctx context.Context, startNode string, initialState State) (State, error) {
	if err := g.Validate(); err != nil {
		return State{}, err
	}

	current := startNode
	state := initialState
	iterations := 0
	visited := make(map[string]int)
	path := []string{}

	for {
		if err := ctx.Err(); err != nil {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("context cancelled: %w", err),
			}
		}

		iterations++
		if iterations > g.maxIterations {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("max iterations (%d) exceeded", g.maxIterations),
			}
		}

		visited[current]++
		path = append(path, current)

		if visited[current] > 1 {
			g.observer.OnEvent(ctx, observability.Event{
				Type:      observability.EventCycleDetected,
				Timestamp: time.Now(),
				Source:    g.name,
				Data: map[string]any{
					"node":        current,
					"visit_count": visited[current],
					"iteration":   iterations,
					"path_length": len(path),
				},
			})
		}

		node, exists := g.nodes[current]
		if !exists {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("node not found: %s", current),
			}
		}

		g.observer.OnEvent(ctx, observability.Event{
			Type:      observability.EventNodeStart,
			Timestamp: time.Now(),
			Source:    g.name,
			Data: map[string]any{
				"node":      current,
				"iteration": iterations,
			},
		})

		newState, err := node.Execute(ctx, state)

		g.observer.OnEvent(ctx, observability.Event{
			Type:      observability.EventNodeComplete,
			Timestamp: time.Now(),
			Source:    g.name,
			Data: map[string]any{
				"node":      current,
				"iteration": iterations,
				"error":     err != nil,
			},
		})

		if err != nil {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      err,
			}
		}

		state = newState.SetCheckpointNode(current)

		if g.checkpointInterval > 0 && iterations%g.checkpointInterval == 0 {
			if err := state.Checkpoint(g.checkpointStore); err != nil {
				return state, &ExecutionError{
					NodeName: current,
					State:    state,
					Path:     path,
					Err:      fmt.Errorf("checkpoint save failed: %w", err),
				}
			}

			g.observer.OnEvent(ctx, observability.Event{
				Type:      observability.EventCheckpointSave,
				Timestamp: time.Now(),
				Source:    g.name,
				Data: map[string]any{
					"node":   current,
					"run_id": state.RunID(),
				},
			})
		}

		if g.exitPoints[current] {
			g.observer.OnEvent(ctx, observability.Event{
				Type:      observability.EventGraphComplete,
				Timestamp: time.Now(),
				Source:    g.name,
				Data: map[string]any{
					"exit_point": current,
					"iterations": iterations,
				},
			})

			if !g.preserveCheckpoints && g.checkpointInterval > 0 {
				g.checkpointStore.Delete(state.RunID())
			}

			return state, nil
		}

		edges, hasEdges := g.edges[current]
		if !hasEdges {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("no outgoing edges from node: %s", current),
			}
		}

		nextNode := ""
		for i := range edges {
			edge := &edges[i]

			predicateResult := edge.Predicate == nil || edge.Predicate(state)

			g.observer.OnEvent(ctx, observability.Event{
				Type:      observability.EventEdgeEvaluate,
				Timestamp: time.Now(),
				Source:    g.name,
				Data: map[string]any{
					"from":   edge.From,
					"to":     edge.To,
					"result": predicateResult,
				},
			})

			if predicateResult && nextNode == "" {
				nextNode = edge.To
			}
		}

		if nextNode == "" {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("no valid edge transition from node: %s", current),
			}
		}

		g.observer.OnEvent(ctx, observability.Event{
			Type:      observability.EventEdgeTransition,
			Timestamp: time.Now(),
			Source:    g.name,
			Data: map[string]any{
				"from": current,
				"to":   nextNode,
			},
		})

		current = nextNode
	}
}
```

### Step 6: Update StateGraph Interface

**File**: `pkg/state/graph.go`

Add Resume method to the StateGraph interface.

Locate the StateGraph interface (around line 28) and add the Resume method signature:

```go
type StateGraph interface {
	Name() string
	AddNode(name string, node StateNode) error
	AddEdge(from, to string, predicate TransitionPredicate) error
	SetEntryPoint(node string) error
	SetExitPoint(node string) error
	Execute(ctx context.Context, initialState State) (State, error)
	Resume(ctx context.Context, runID string) (State, error)
}
```

## Implementation Complete

After completing these steps:

1. State contains checkpoint metadata (runID, checkpointNode, timestamp)
2. CheckpointStore interface works directly with State objects
3. MemoryCheckpointStore provides thread-safe in-memory checkpoint storage
4. GraphConfig includes CheckpointConfig for checkpoint behavior configuration
5. StateGraph.Execute integrates checkpointing seamlessly
6. StateGraph.Resume enables recovery from checkpoints
7. Observer events provide visibility into checkpoint lifecycle
8. No breaking changes to Execute signature - remains `(State, error)`

## Key Architectural Benefits

1. **Checkpoint IS State**: Eliminated unnecessary abstraction layer
2. **Clean Signatures**: Execute returns `(State, error)` - State contains runID
3. **Self-Describing State**: State knows its execution provenance
4. **Natural Integration**: Checkpointing integrated into execution flow, not bolted on
5. **Configuration Pattern**: Follows established GraphConfig pattern
6. **Thread Safety**: MemoryCheckpointStore uses sync.RWMutex
7. **Fail-Fast**: Checkpoint save failures stop execution immediately

## Testing Responsibilities

Testing will be implemented by Claude after you signal implementation completion. Tests will validate:

- State checkpoint metadata preservation through Clone, Set, Merge operations
- MemoryCheckpointStore thread safety and CRUD operations
- Execute method checkpoint saving at configured intervals
- Resume recovery from checkpoints
- Checkpoint cleanup on success when Preserve=false
- Observer event emission for checkpoint operations
- Error handling for checkpoint save failures

Target coverage: 80%+ (aiming for 90%+ to match Phase 3 standards)

## Documentation Responsibilities

Claude will add godoc comments after implementation completion, covering:

- State checkpoint fields and methods
- CheckpointStore interface and MemoryCheckpointStore
- CheckpointConfig structure
- Execute checkpointing behavior
- Resume recovery process
- Usage examples for checkpoint-enabled graphs

## Future Extensibility

Potential enhancements beyond Phase 6 scope:

- **Disk-Based CheckpointStore**: Persistent storage for long-running workflows
- **Database CheckpointStore**: Distributed checkpoint storage (Redis, PostgreSQL)
- **Checkpoint Compression**: Reduce storage overhead for large states
- **Checkpoint History**: Preserve multiple checkpoints per runID with rollback capability
- **Checkpoint Metadata Enrichment**: Add environment info, Git commit hash, etc.

These extensions can be added by implementing additional CheckpointStore backends without changing the core architecture.
