# Phase 2: State Management Core

## Starting Point

Phase 2 implementation began with Phase 1 foundation:
- Hub and messaging packages complete (Level 1-2)
- Configuration package established
- Test infrastructure operational (86% coverage)
- go-agents v0.2.1 dependency integrated

Planning foundation from consolidated Phases 2-8 document:
- State management architecture defined
- Observer pattern designed for minimal-overhead observability
- Configuration lifecycle established
- Option A execution order chosen (State Infrastructure First)

Critical architectural insight: Observer integration must begin in Phase 2 to prevent retrofit friction.

## Implementation Overview

### observability Package (Level 0)

Implemented minimal observer pattern for zero-overhead execution telemetry.

**Files Created:**
- `pkg/observability/observer.go` - Observer interface, Event struct, EventType constants for all phases
- `pkg/observability/registry.go` - Observer registry for configuration-driven selection
- `pkg/observability/doc.go` - Package documentation with examples

**Observer Interface:**
```go
type Observer interface {
    OnEvent(ctx context.Context, event Event)
}

type Event struct {
    Type      EventType
    Timestamp time.Time
    Source    string
    Data      map[string]any  // Execution metadata, not application data
}
```

**Key EventTypes Defined:**
- Phase 2: `EventStateCreate`, `EventStateClone`, `EventStateMerge`, `EventStateRead`, `EventStateWrite`
- Phase 3: `EventGraphExecuteStart`, `EventGraphExecuteEnd`, `EventNodeExecuteStart`, `EventNodeExecuteEnd`
- Phase 4: `EventEdgeEvaluate`, `EventTransitionDecision`
- Phase 5-7: (Reserved for patterns)
- Phase 8: (Observer implementations)

**Observer Registry:**
```go
func GetObserver(name string) (Observer, error)
func RegisterObserver(name string, observer Observer)

// Built-in observers
var observers = map[string]Observer{
    "noop": NoOpObserver{},
}
```

**Key Design Decisions:**
- `Event.Data` contains execution metadata, not application data (privacy, performance, utility)
- NoOpObserver provides zero-overhead default
- Registry enables JSON config: `{"observer": "slog"}` → `GetObserver("slog")`
- All phase events defined upfront for API stability

### state Package (Level 3)

Implemented immutable state management with observer integration.

**Files Created:**
- `pkg/state/state.go` - Immutable State type with operations
- `pkg/state/node.go` - StateNode interface and FunctionNode implementation
- `pkg/state/edge.go` - Edge struct and transition predicates
- `pkg/state/graph.go` - StateGraph interface for workflow orchestration
- `pkg/state/doc.go` - Package documentation with examples

**State Implementation:**
```go
type State struct {
    data     map[string]any
    observer observability.Observer
}

func New(observer observability.Observer) State {
    return State{
        data:     make(map[string]any),
        observer: observer,
    }
}

func (s State) Clone() State {
    newState := State{
        data:     maps.Clone(s.data),  // Go 1.21+ improvement
        observer: s.observer,
    }
    s.observer.OnEvent(context.Background(), observability.Event{
        Type:      observability.EventStateClone,
        Timestamp: time.Now(),
        Source:    "state",
        Data:      map[string]any{},
    })
    return newState
}

func (s State) Merge(other State) State {
    newState := s.Clone()
    maps.Copy(newState.data, other.data)  // Go 1.21+ improvement
    // Emits EventStateMerge
    return newState
}
```

**User Improvements:**
- Used `maps.Clone()` instead of manual loop for better performance
- Used `maps.Copy()` for Merge operation (cleaner than manual iteration)

**StateNode Interface:**
```go
type StateNode interface {
    Execute(ctx context.Context, state State) (State, error)
}

// FunctionNode wraps functions as nodes
type FunctionNode struct {
    fn func(ctx context.Context, state State) (State, error)
}

func NewFunctionNode(fn func(context.Context, State) (State, error)) StateNode {
    return &FunctionNode{fn: fn}
}
```

**Edge and Predicates:**
```go
type Edge struct {
    From      string
    To        string
    Predicate TransitionPredicate
}

type TransitionPredicate func(state State) bool

// Predicate helpers
func AlwaysTransition() TransitionPredicate
func KeyExists(key string) TransitionPredicate
func KeyEquals(key string, value any) TransitionPredicate
func Not(predicate TransitionPredicate) TransitionPredicate
func And(predicates ...TransitionPredicate) TransitionPredicate
func Or(predicates ...TransitionPredicate) TransitionPredicate
```

**StateGraph Interface:**
```go
type StateGraph interface {
    AddNode(name string, node StateNode) error
    AddEdge(from, to string, predicate TransitionPredicate) error
    SetEntryPoint(node string) error
    SetExitPoint(node string) error
    Execute(ctx context.Context, initialState State) (State, error)  // Phase 3
}
```

**Key Design Decisions:**
- Immutable state using `maps.Clone()` and `maps.Copy()` (Go 1.21+)
- Observer passed through all state operations
- State uses `map[string]any` (not generic) to prevent cascading generics and maintain pattern flexibility
- FunctionNode enables inline node definitions without custom types
- Predicates composable via And, Or, Not
- StateGraph execution deferred to Phase 3

### config Package Updates

Added state graph configuration structures.

**Files Created:**
- `pkg/config/state.go` - GraphConfig for state graph configuration

**GraphConfig Structure:**
```go
type GraphConfig struct {
    Name          string `json:"name"`
    Observer      string `json:"observer"`  // String for JSON, resolved via GetObserver()
    MaxIterations int    `json:"max_iterations"`
}

func DefaultGraphConfig(name string) GraphConfig {
    return GraphConfig{
        Name:          name,
        Observer:      "noop",
        MaxIterations: 1000,
    }
}
```

**Configuration Lifecycle:**
```go
// JSON → Config → Domain Objects
var cfg config.GraphConfig
json.Unmarshal(data, &cfg)
observer, err := observability.GetObserver(cfg.Observer)
graph := state.NewGraph(cfg, observer)  // Phase 3
```

**Key Design Decisions:**
- Observer as string enables JSON configuration without circular dependencies
- Configuration exists only during initialization
- GraphConfig in config package follows Phase 1 pattern (not in state package)
- No go-agents integration in Phase 2 (deferred to future phases)

### Testing

Comprehensive black-box test suite achieving 100% coverage.

**Test Structure:**
- `tests/observability/` - Observer interface, events, registry
- `tests/state/` - State operations, nodes, edges, predicates
- `tests/config/` - GraphConfig and HubConfig (Phase 1)

**Testing Approach:**
- Black-box testing with `package <name>_test`
- Table-driven tests for parameterized scenarios
- captureObserver helper for event validation
- Immutability validation in all state tests
- Context cancellation validation

**Coverage by Package:**
- `observability/`: 100% coverage
- `state/`: 100% coverage
- `config/`: 100% coverage

## Technical Decisions

### State Immutability via maps Package

Uses Go 1.21+ maps package for efficient cloning:

```go
func (s State) Clone() State {
    return State{
        data:     maps.Clone(s.data),  // Single function call
        observer: s.observer,
    }
}

func (s State) Merge(other State) State {
    newState := s.Clone()
    maps.Copy(newState.data, other.data)  // Efficient merge
    return newState
}
```

**Rationale:** Built-in maps functions provide better performance than manual loops, cleaner code, and compile-time validation. User identified this improvement during implementation.

### Observer Registry Pattern

Registry resolves string keys to Observer instances:

```go
var observers = map[string]Observer{
    "noop": NoOpObserver{},
}

func GetObserver(name string) (Observer, error) {
    observer, exists := observers[name]
    if !exists {
        return nil, fmt.Errorf("unknown observer: %s", name)
    }
    return observer, nil
}
```

**Rationale:** Enables configuration-driven observer selection (`{"observer": "slog"}`), prevents circular dependencies (config doesn't import concrete observers), allows runtime extensibility via RegisterObserver.

### Event.Data as Metadata

Event.Data contains execution metadata, not application data:

```go
// Example: EventStateClone with empty Data
s.observer.OnEvent(ctx, observability.Event{
    Type:      observability.EventStateClone,
    Timestamp: time.Now(),
    Source:    "state",
    Data:      map[string]any{},  // No application data
})

// Example: EventStateMerge with keys changed
s.observer.OnEvent(ctx, observability.Event{
    Type:      observability.EventStateMerge,
    Timestamp: time.Now(),
    Source:    "state",
    Data:      map[string]any{
        "keys_added": len(other.data),
    },
})
```

**Rationale:**
- **Privacy:** Prevents exposing sensitive application data in telemetry
- **Performance:** Avoids deep cloning overhead for large state objects
- **Utility:** Metadata (keys changed, duration) more useful than values for observability

User question: "I get it, so Data in the Event is not necessarily the data encapsulated in the state, it's metadata about the state itself?" - **Confirmed**

### State Not Generic

State uses `map[string]any` instead of generic `State[T any]`:

```go
type State struct {
    data     map[string]any  // Not generic
    observer observability.Observer
}
```

**Rationale:**
- Avoids cascading generics through entire type system (StateNode, StateGraph, etc.)
- Enables dynamic workflows compatible with LangGraph patterns
- Maintains pattern flexibility (TContext can be State)
- Prevents composition friction in workflow patterns

User question: "Should State be generic?" - **No, map[string]any is correct**

### GraphConfig in config Package

GraphConfig placed in `pkg/config/state.go` not `pkg/state/graph.go`:

```
pkg/
├── observability/    # Level 0 (no dependencies)
├── config/          # Configuration structures
│   ├── hub.go
│   └── state.go     # GraphConfig here
└── state/           # Level 3 (depends on observability)
```

**Rationale:**
- Maintains proper package dependency hierarchy (config doesn't depend on state)
- Follows Phase 1 pattern (HubConfig in config package)
- Follows configuration lifecycle principle (config → domain objects at boundaries)
- Prevents circular dependencies

User question: "Why GraphConfig in graph.go not config/state.go?" - **Confirmed config package correct**

### Observer Integration from Phase 2

Observer hooks integrated in Phase 2 despite full implementations in Phase 8:

```go
// State operations emit events immediately
func (s State) Clone() State {
    newState := State{...}
    s.observer.OnEvent(ctx, Event{Type: EventStateClone, ...})
    return newState
}
```

**Rationale:** User's critical insight - "if observability isn't considered from the start, we'll have retrofit friction." Minimal observer interface allows integration now, implementations later. NoOpObserver provides zero-overhead default.

## Final Architecture State

### Package Structure

```
pkg/
├── observability/     # Level 0: Observer pattern
│   ├── observer.go
│   ├── registry.go
│   └── doc.go
├── messaging/         # Level 1: Message primitives (Phase 1)
│   ├── message.go
│   ├── types.go
│   └── builder.go
├── hub/              # Level 2: Agent coordination (Phase 1)
│   ├── hub.go
│   ├── handler.go
│   ├── channel.go
│   └── metrics.go
├── config/           # Configuration
│   ├── hub.go       # Phase 1
│   └── state.go     # Phase 2
└── state/            # Level 3: State management (Phase 2)
    ├── state.go
    ├── node.go
    ├── edge.go
    ├── graph.go
    └── doc.go
```

### Dependency Hierarchy

```
observability/ (Level 0 - no dependencies)
    ↓
messaging/ (Level 1 - no dependencies)
    ↓
hub/ (Level 2 - depends on messaging)
    ↓
state/ (Level 3 - depends on observability)
    ↓
Future: patterns/ (Level 4)
```

### Configuration Lifecycle

```
JSON Config → GraphConfig → GetObserver() → State → StateGraph

Example:
{
  "name": "document-workflow",
  "observer": "slog",
  "max_iterations": 500
}
    ↓
var cfg config.GraphConfig
json.Unmarshal(data, &cfg)
    ↓
observer, err := observability.GetObserver(cfg.Observer)
    ↓
state := state.New(observer)
    ↓
graph := state.NewGraph(cfg, observer)  // Phase 3
```

## Documentation Updates

Added comprehensive package documentation:

**pkg/observability/doc.go:**
- Observer pattern explanation
- Usage examples for NoOpObserver and registry
- Phase 8 integration notes

**pkg/state/doc.go:**
- State management overview
- Immutability explanation
- Usage examples with patterns
- Observer integration explanation

**pkg/config/state.go:**
- GraphConfig documentation
- JSON configuration examples
- Observer resolution pattern

All files include godoc comments with examples following Go documentation conventions.

## Test Results

Final test coverage: 100% (exceeds 80% requirement)

All tests passing with black-box testing approach:
- observability package: Observer interface, events, registry (100%)
- state package: State operations, nodes, edges, predicates (100%)
- config package: GraphConfig, HubConfig (100%)

**Test Compilation Fix:**
- Initial error: testObserver type defined inside test function
- Fix: Moved testObserver to package level
- Result: All tests compile and run successfully

## Phase 2 Completion Status

**Completed Objectives:**
- ✅ Implemented observability package (observer pattern and registry)
- ✅ Implemented state package (immutable state, nodes, edges, graph interface)
- ✅ Updated config package (GraphConfig)
- ✅ Created comprehensive tests (100% coverage)
- ✅ Validated observer integration pattern
- ✅ Documented all Phase 2 code

**Phase 2 Deliverables:**
- ✅ Working observer pattern with registry
- ✅ Immutable state operations
- ✅ StateNode and FunctionNode implementations
- ✅ Edge and transition predicates
- ✅ StateGraph interface (executor in Phase 3)
- ✅ Configuration structures for state graphs
- ✅ 100% test coverage
- ✅ Complete documentation

## Next Steps: Phase 3 Planning

Phase 3 will implement state graph execution:

**Planned Features:**
- StateGraph implementation (concrete type implementing interface)
- Graph executor with entry/exit point handling
- Transition evaluation using predicates
- Cycle detection and MaxIterations enforcement
- Validation of graph structure (entry/exit points exist, edges reference valid nodes)

**Integration:**
- Nodes can use hub for coordination
- State flows immutably through execution
- Observer captures execution telemetry (EventGraphExecuteStart, EventNodeExecuteStart, etc.)
- Context cancellation propagates through execution

**Configuration:**
- GraphConfig provides name, observer, max iterations
- Observer resolved via registry
- Graph validates configuration on creation

Phase 3 planning should focus on graph executor implementation details and validation requirements.
