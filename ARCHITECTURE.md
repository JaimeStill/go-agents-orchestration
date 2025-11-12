# go-agents-orchestration - Architecture

## Overview

This document specifies the technical architecture of go-agents-orchestration, a supplemental package providing multi-agent coordination primitives for the [go-agents](https://github.com/JaimeStill/go-agents) ecosystem.

## Package Structure

### Dependency Hierarchy

Packages are organized in a strict dependency hierarchy (bottom-up):

```
Level 0: observability/     # Observer pattern (no dependencies)
    ↓
Level 1: messaging/          # Message primitives (no dependencies)
    ↓
Level 2: hub/               # Agent coordination (depends on messaging)
    ↓
Level 3: state/             # State graph execution (depends on observability)
    ↓
Level 4: patterns/          # Workflow patterns (depends on hub + state)
```

**Rationale**: Lower layers cannot import higher layers. This prevents circular dependencies and ensures each layer can be validated independently. Observability at Level 0 enables all layers to integrate observer pattern.

### Package Organization

```
github.com/JaimeStill/go-agents-orchestration/
├── observability/          # Level 0: Observer pattern
│   ├── observer.go         # Observer interface and Event
│   ├── registry.go         # Observer registry
│   └── doc.go             # Package documentation
│
├── messaging/              # Level 1: Message primitives
│   ├── message.go          # Message structure and helpers
│   ├── builder.go          # Fluent message builders
│   └── types.go            # MessageType, Priority enums
│
├── hub/                    # Level 2: Agent coordination
│   ├── hub.go              # Hub interface and implementation
│   ├── agent.go            # Agent interface (minimal contract)
│   ├── handler.go          # MessageHandler type and MessageContext
│   ├── channel.go          # Message channel wrapper
│   ├── registry.go         # Agent registration logic
│   └── metrics.go          # Hub metrics
│
├── config/                 # Configuration structures
│   ├── hub.go              # HubConfig
│   └── state.go            # GraphConfig
│
├── state/                  # Level 3: State graph execution
│   ├── state.go            # State type with immutable operations
│   ├── node.go             # StateNode interface and FunctionNode
│   ├── edge.go             # Edge and transition predicates
│   ├── graph.go            # StateGraph interface and implementation
│   ├── error.go            # ExecutionError type
│   └── doc.go             # Package documentation
│
├── patterns/               # Level 4: Workflow patterns (Phase 4-5)
│   ├── chain.go            # Sequential chains
│   ├── parallel.go         # Parallel execution
│   ├── conditional.go      # Conditional routing
│   └── stateful.go         # Stateful workflows
│
└── examples/               # Integration examples
    └── phase-01-hubs/      # Hub coordination example
```

## Phase 1: Foundation (Completed)

### messaging Package

**Purpose**: Provide message structures and builders for inter-agent communication.

**Core Types:**

```go
// Message represents a message sent between agents
type Message struct {
    // Core fields
    ID   string      // Unique message identifier
    From string      // Sender agent ID
    To   string      // Recipient agent ID
    Type MessageType // Message type (request, response, notification, broadcast)
    Data any         // Message payload

    // Routing and correlation
    ReplyTo   string    // For response correlation
    Topic     string    // For pub/sub messaging
    Timestamp time.Time // Message creation time

    // Optional metadata
    Priority Priority           // Message priority
    Headers  map[string]string  // Custom headers
}

// MessageType defines inter-agent message types
type MessageType string

const (
    MessageTypeRequest      MessageType = "request"
    MessageTypeResponse     MessageType = "response"
    MessageTypeNotification MessageType = "notification"
    MessageTypeBroadcast    MessageType = "broadcast"
)

// Priority defines message priority levels
type Priority int

const (
    PriorityLow Priority = iota
    PriorityNormal
    PriorityHigh
    PriorityCritical
)
```

**Message Builders:**

Fluent API for constructing messages:

```go
// Create request
msg := messaging.NewRequest("agent-a", "agent-b", taskData).
    Priority(messaging.PriorityHigh).
    Headers(map[string]string{"source": "workflow"}).
    Build()

// Create response
resp := messaging.NewResponse("agent-b", "agent-a", msg.ID, resultData).Build()

// Create notification
notif := messaging.NewNotification("agent-a", "agent-b", statusUpdate).Build()

// Create broadcast
broadcast := messaging.NewBroadcast("agent-a", announcement).Build()
```

**Design Principles:**
- Minimal core fields (ID, From, To, Type, Data)
- Optional metadata fields for extensibility
- Immutable once built
- No behavior, just data structures

### hub Package

**Purpose**: Provide agent coordination and message routing.

**Core Interfaces:**

```go
// Agent represents the minimal interface for hub registration
type Agent interface {
    ID() string
}

// Hub provides agent coordination and message routing
type Hub interface {
    // Agent lifecycle
    RegisterAgent(agent Agent, handler MessageHandler) error
    UnregisterAgent(agentID string) error

    // Communication patterns
    Send(ctx context.Context, from, to string, data any) error
    Request(ctx context.Context, from, to string, data any) (*messaging.Message, error)
    Broadcast(ctx context.Context, from string, data any) error

    // Pub/Sub
    Subscribe(agentID, topic string) error
    Publish(ctx context.Context, from, topic string, data any) error

    // Management
    GetMetrics() Metrics
    Shutdown(timeout time.Duration) error
}

// MessageHandler processes incoming messages for an agent
type MessageHandler func(
    ctx context.Context,
    message *messaging.Message,
    context *MessageContext,
) (*messaging.Message, error)

// MessageContext provides context information to message handlers
type MessageContext struct {
    HubName string
    Agent   Agent
}
```

**Hub Implementation:**

```go
// hub implementation (internal)
type hub struct {
    name string

    // Agent registry
    agents      map[string]*registration
    agentsMutex sync.RWMutex

    // Request-response correlation
    responseChannels map[string]chan *messaging.Message
    responsesMutex   sync.RWMutex

    // Pub/Sub
    subscriptions map[string]map[string]*registration
    subsMutex     sync.RWMutex

    // Configuration
    channelBufferSize int
    defaultTimeout    time.Duration

    // Observability
    logger  *slog.Logger
    metrics *Metrics

    // Lifecycle
    ctx    context.Context
    cancel context.CancelFunc
    done   chan struct{}
}

type registration struct {
    Agent   Agent
    Handler MessageHandler
    Channel *MessageChannel[*messaging.Message]
}
```

**Communication Patterns:**

1. **Send (Fire-and-Forget)**:
   - Asynchronous message delivery
   - No response expected
   - Returns error if delivery fails

2. **Request/Response**:
   - Synchronous request with response correlation
   - Timeout support via context
   - Response delivered via correlation ID

3. **Broadcast**:
   - Send to all registered agents except sender
   - Best-effort delivery
   - Partial failures logged but don't abort

4. **Pub/Sub**:
   - Topic-based subscription model
   - Multiple subscribers per topic
   - Delivery to all subscribers

**Message Channel:**

Context-aware channel wrapper for type-safe messaging:

```go
type MessageChannel[T any] struct {
    channel    chan T
    context    context.Context
    bufferSize int
    closed     atomic.Int32
}

// Send with context cancellation support
func (mc *MessageChannel[T]) Send(ctx context.Context, message T) error

// Receive with context cancellation support
func (mc *MessageChannel[T]) Receive(ctx context.Context) (T, error)

// Non-blocking receive for polling
func (mc *MessageChannel[T]) TryReceive() (T, bool)
```

**Multi-Hub Coordination:**

Agents can register with multiple hubs simultaneously:

```go
// Create hubs
globalHub := hub.New(ctx, globalConfig)
taskHub := hub.New(ctx, taskConfig)

// Agent participates in both hubs
globalHub.RegisterAgent(agent, globalHandler)
taskHub.RegisterAgent(agent, taskHandler)

// Agent receives messages from both hubs via different handlers
```

**Fractal Growth Pattern:**

Hubs can network through shared agents:

```
Hub A ← Agent X → Hub B
          ↓
        Hub C
```

Agent X acts as a bridge, enabling cross-hub communication.

**Hub Metrics:**

Track communication statistics for observability:

```go
type Metrics struct {
    LocalAgents   int // Agents registered in this hub
    MessagesSent  int // Messages sent through hub
    MessagesRecv  int // Messages received by hub
}

// Get snapshot of current metrics
snapshot := hub.Metrics()
```

**Implementation Status:**

Phase 1 implementation is complete with:
- Full messaging package with builders and types
- Complete hub implementation with all communication patterns
- MessageChannel for context-aware message delivery
- Multi-hub coordination validated through ISS EVA example
- Test coverage: 86% (exceeds 80% requirement)
- Integration example: `examples/phase-01-hubs` demonstrates all patterns

### config Package

**Purpose**: Hub configuration structures.

**Configuration Structures:**

```go
// HubConfig defines configuration for a Hub instance
type HubConfig struct {
    // Hub identity
    Name string

    // Communication settings
    ChannelBufferSize int
    DefaultTimeout    time.Duration

    // Observability
    Logger *slog.Logger
}

// DefaultHubConfig provides sensible defaults
func DefaultHubConfig() HubConfig {
    return HubConfig{
        Name:              "hub",
        ChannelBufferSize: 100,
        DefaultTimeout:    30 * time.Second,
        Logger:            slog.Default(),
    }
}
```

**Configuration Lifecycle:**

1. Load configuration from code/files/environment
2. Transform to domain objects at package boundaries
3. Runtime behavior depends on initialized state
4. Configuration does not persist beyond initialization

## Phase 2-3, 6: State Management (Completed)

### state Package

**Purpose**: LangGraph-inspired state graph execution with Go-native patterns.

**Key Concepts:**

- **StateGraph**: Workflow definition with nodes and edges
- **StateNode**: Computation step that receives and returns state
- **Edge**: Transition between nodes with optional predicates
- **State**: Data flowing through the graph (map[string]any)
- **Executor**: Engine that executes state graph transitions
- **ExecutionError**: Rich error context for debugging

**Core Interfaces:**

```go
// StateGraph defines a workflow as a graph of nodes and edges
type StateGraph interface {
    Name() string
    AddNode(name string, node StateNode) error
    AddEdge(from, to string, predicate TransitionPredicate) error
    SetEntryPoint(node string) error
    SetExitPoint(node string) error
    Execute(ctx context.Context, initialState State) (State, error)
    Resume(ctx context.Context, runID string) (State, error)  // Phase 6
}

// StateNode represents a computation step in the graph
type StateNode interface {
    Execute(ctx context.Context, state State) (State, error)
}

// State represents data flowing through the graph
type State struct {
    data           map[string]any
    observer       observability.Observer
    runID          string      // Phase 6: Execution identity
    checkpointNode string      // Phase 6: Last checkpointed node
    timestamp      time.Time   // Phase 6: Creation/checkpoint time
}

// TransitionPredicate determines which edge to follow
type TransitionPredicate func(state State) bool

// ExecutionError captures rich context when graph execution fails
type ExecutionError struct {
    NodeName string
    State    State
    Path     []string
    Err      error
}
```

**State Operations:**

- `New(observer)` - Create new state with observer
- `Clone()` - Deep copy state
- `Get(key)` - Retrieve value with existence check
- `Set(key, value)` - Create new state with updated value
- `Merge(other)` - Combine states (immutable)
- `RunID()` - Get execution identifier (Phase 6)
- `CheckpointNode()` - Get last checkpointed node (Phase 6)
- `Timestamp()` - Get creation/checkpoint time (Phase 6)
- `SetCheckpointNode(node)` - Update checkpoint metadata (Phase 6)
- `Checkpoint(store)` - Save state to store (Phase 6)

**Graph Execution:**

Phase 3 implementation provides complete execution engine:

1. **Graph Construction**: Build workflow with nodes and edges
2. **Validation**: Ensure structure completeness before execution
3. **Execution**: Traverse graph from entry to exit point
4. **Cycle Detection**: Track visit counts, emit events on revisits
5. **Max Iterations**: Prevent infinite loops (configurable limit)
6. **Observer Integration**: Emit events at all execution milestones
7. **Error Context**: Capture full execution state on failure

**Execution Features:**

- Linear path execution (A → B → C)
- Conditional routing via predicates
- Multiple exit points (success/failure paths)
- Context cancellation propagation
- Full execution path tracking
- Rich error context for debugging

**Observer Events:**

- `EventGraphStart` / `EventGraphComplete`
- `EventNodeStart` / `EventNodeComplete`
- `EventEdgeEvaluate` / `EventEdgeTransition`
- `EventCycleDetected`
- `EventCheckpointSave` / `EventCheckpointLoad` / `EventCheckpointResume` (Phase 6)

**Checkpointing (Phase 6):**

Phase 6 adds workflow persistence and recovery through checkpoint save/resume:

**Architecture**: State-centric checkpointing where State serves as self-describing execution artifact with embedded provenance metadata (runID, checkpointNode, timestamp). No separate Checkpoint wrapper - checkpoint IS State captured at execution stage.

**CheckpointStore Interface:**
```go
type CheckpointStore interface {
    Save(state State) error
    Load(runID string) (State, error)
    Delete(runID string) error
    List() ([]string, error)
}
```

**Checkpoint Lifecycle:**
1. Graph execution saves State at configured intervals
2. On success, checkpoints auto-deleted (unless Preserve=true)
3. On failure, checkpoints persist for Resume
4. Resume loads checkpoint and continues from next node

**Configuration:**
- `Interval`: Checkpoint every N nodes (0 = disabled)
- `Store`: CheckpointStore implementation name (registry resolution)
- `Preserve`: Keep checkpoints after success (default false)

**Resume Semantics:**
- Checkpoints saved AFTER node execution (represents completed work)
- Resume skips to next node after checkpoint
- Resume validates checkpoint exists and finds valid transition
- Errors if checkpoint at exit point (execution already complete)

**Implementations:**
- `MemoryCheckpointStore`: Thread-safe in-memory storage (development/testing)
- Custom stores via registry pattern (e.g., disk, database)

**Observer Integration:**
- `EventCheckpointSave`: Emitted when checkpoint saved during execution
- `EventCheckpointLoad`: Emitted when checkpoint loaded for resume
- `EventCheckpointResume`: Emitted when execution resumes from checkpoint

**Error Handling:**
- Checkpoint save failures halt execution (fail-fast for production reliability)
- Load failures return clear errors with checkpoint context
- Resume validates checkpoint state before continuing

**Test Coverage**: 82.4% (20 comprehensive tests)

**Integration with Hub:**

State graph nodes can use hub for agent coordination:

```go
// Hub node that requests agent processing
type HubNode struct {
    hub       hub.Hub
    fromAgent string
    toAgent   string
}

func (n *HubNode) Execute(ctx context.Context, state State) (State, error) {
    data, _ := state.Get("data")
    response, err := n.hub.Request(ctx, n.fromAgent, n.toAgent, data)
    if err != nil {
        return state, err
    }

    return state.Set("result", response.Data), nil
}
```

**Implementation Status:**

Phase 2-3, 6 implementation is complete:
- State type with immutable operations and checkpoint metadata (Phase 2, 6)
- StateNode interface and FunctionNode (Phase 2)
- Edge with transition predicates (AlwaysTransition, KeyExists, KeyEquals, Not, And, Or) (Phase 3)
- StateGraph interface with Execute and Resume (Phase 3, 6)
- Graph execution engine with cycle detection (Phase 3)
- CheckpointStore interface and MemoryCheckpointStore implementation (Phase 6)
- Checkpoint save/resume with configurable intervals (Phase 6)
- ExecutionError with rich context (Phase 3)
- Observer integration throughout (Phase 2-6)
- Test coverage: 82.4% state package (exceeds 80% requirement)

### observability Package

**Purpose**: Minimal observer pattern for zero-overhead execution telemetry.

**Core Types:**

```go
// Observer receives execution events
type Observer interface {
    OnEvent(ctx context.Context, event Event)
}

// Event represents an observable occurrence
type Event struct {
    Type      EventType
    Timestamp time.Time
    Source    string
    Data      map[string]any
}

// NoOpObserver provides zero-cost implementation
type NoOpObserver struct{}
```

**Observer Registry:**

```go
func GetObserver(name string) (Observer, error)
func RegisterObserver(name string, observer Observer)
```

**Event Types Defined:**

- Phase 2-3: State operations and graph execution
- Phase 4-5: Workflow patterns
- Phase 6: Checkpointing
- Phase 7: Conditional routing
- Phase 8: Full observability implementation

**Implementation Status:**

Observability infrastructure is complete:
- Observer interface and Event structure
- Registry for configuration-driven selection
- NoOpObserver for zero overhead
- EventType constants for all phases
- Event.Data contains metadata (not application data)
- Test coverage: 100%

### config Package

**Purpose**: Configuration structures for orchestration primitives.

**Configuration Structures:**

```go
// HubConfig defines configuration for a Hub instance
type HubConfig struct {
    Name              string
    ChannelBufferSize int
    DefaultTimeout    time.Duration
    Logger            *slog.Logger
}

// GraphConfig defines configuration for state graphs
type GraphConfig struct {
    Name          string
    Observer      string
    MaxIterations int
    Checkpoint    CheckpointConfig  // Phase 6
}

// CheckpointConfig controls workflow state persistence
type CheckpointConfig struct {
    Store    string  // CheckpointStore name (registry resolution)
    Interval int     // Checkpoint every N nodes (0 = disabled)
    Preserve bool    // Keep checkpoints after success
}

// ChainConfig defines configuration for sequential chain execution
type ChainConfig struct {
    CaptureIntermediateStates bool
    Observer                  string
}

// ParallelConfig defines configuration for parallel execution
type ParallelConfig struct {
    MaxWorkers int    // Exact worker count (0 = auto-detect)
    WorkerCap  int    // Max workers when auto-detecting
    FailFast   bool   // Stop on first error vs collect all
    Observer   string // Observer name for resolution
}
```

**Implementation Status:**

Configuration package complete for current phases:
- HubConfig (Phase 1)
- GraphConfig with CheckpointConfig (Phase 2, 6)
- ChainConfig (Phase 4)
- ParallelConfig (Phase 5)
- Default configuration functions for all types
- Test coverage: 100%

## Phase 4-7: Workflow Patterns (Phase 4-5 Complete)

### workflows Package

**Purpose**: Composable workflow patterns for orchestrating multi-step processes.

**Location**: `pkg/workflows/`

**Design Philosophy**: Generic orchestration primitives that work with any item and context types, extracted from real-world usage (classify-docs). All patterns support direct go-agents usage as the primary pattern, with optional hub coordination for multi-agent orchestration.

**Implemented Patterns:**

1. **Sequential Chain** (Phase 4 - Complete): Linear workflow with state accumulation implementing fold/reduce pattern
   - Generic over TItem (items to process) and TContext (accumulated state)
   - Observer integration with chain and step-level events
   - Rich error context via ChainError[TItem, TContext]
   - Optional intermediate state capture
   - Progress callback support
   - Test coverage: 97.4%

2. **Parallel Execution** (Phase 5 - Complete): Concurrent processing with worker pool and result aggregation
   - Generic over TItem (items to process) and TResult (processing results)
   - Worker pool auto-detection (min(NumCPU*2, WorkerCap, len(items)))
   - Fail-fast and collect-all-errors modes
   - Order preservation despite concurrent execution
   - Three-channel coordination pattern for deadlock prevention
   - Background result collector for non-blocking operation
   - Atomic counter for thread-safe progress tracking
   - Observer integration with parallel and worker-level events
   - Rich error context via ParallelError[TItem] and TaskError[TItem]
   - Error categorization with frequency-based sorting
   - Progress callback support
   - Test coverage: 96.6%

**Planned Patterns:**

3. **Conditional Routing** (Phase 7): State-based routing decisions with predicate evaluation
4. **Stateful Workflows** (Phase 7): Complex compositions combining patterns with state graphs

**Implementation Status:**

- Sequential Chain: Complete with comprehensive tests (97.4% coverage)
- Parallel Execution: Complete with comprehensive tests (96.6% coverage)
- Overall workflows test coverage: 96.6% (exceeds 80% requirement)
- Documentation: Complete with examples for both patterns

### Pattern Independence

All workflow patterns are agnostic about processing approach:
- ✅ Direct go-agents usage (primary pattern - like classify-docs)
- ✅ Hub orchestration (optional for multi-agent coordination)
- ✅ Pure data transformation (no agents required)
- ✅ Mixed approaches (some steps with agents, some without)

The processor function signatures intentionally don't constrain implementation, enabling maximum flexibility.

## Phase 2-8: Observability (Basic Implementation Complete)

### observability Package

**Purpose**: Production-grade tracing, metrics, and decision logging.

**Location**: `pkg/observability/`

**Implemented Features (Phase 2-5):**

- **Observer Interface**: Minimal contract for event emission
  - `OnEvent(ctx context.Context, event Event)` method
  - Event structure with Type, Timestamp, Source, and Data
  - EventType constants for all workflow phases (2-8)

- **Observer Registry**: Configuration-driven observer selection
  - `GetObserver(name string)` for runtime resolution
  - `RegisterObserver(name, observer)` for extensibility
  - Enables JSON configuration with observer as string

- **NoOpObserver** (Phase 2): Zero-overhead implementation
  - Discards all events without processing
  - Used when observability not needed
  - Stateless and safe for concurrent use

- **SlogObserver** (Phase 5): Structured logging implementation
  - Integrates with Go's standard slog package
  - Logs all events at Info level with structured fields
  - Context-aware logging via InfoContext
  - Supports custom slog handlers (Text, JSON, custom)
  - Test coverage: 100%

**Default Observer:**

All configuration defaults now use "slog" observer for practical observability during development. Users can override to "noop" for zero overhead in production.

**Planned Features (Phase 8):**

- Advanced execution trace correlation across workflows
- Decision point logging with reasoning capture
- Performance metrics aggregation (latency, token usage, retries)
- Confidence scoring utilities
- OpenTelemetry integration

## Integration with go-agents

### Agent Wrapper Pattern

go-agents `Agent` provides LLM interaction. Hub expects minimal `ID()` interface. Composition pattern:

```go
// go-agents agent
llmAgent, _ := agent.New(agentConfig)

// Wrapper for hub participation
type MyAgent struct {
    id    string
    agent agent.Agent
}

func (a *MyAgent) ID() string { return a.id }

// Handler uses go-agents for LLM processing
handler := func(ctx context.Context, msg *messaging.Message, msgCtx *hub.MessageContext) (*messaging.Message, error) {
    // Use go-agents to process message
    response, err := a.agent.Chat(ctx, fmt.Sprintf("Process: %v", msg.Data))
    if err != nil {
        return nil, err
    }

    // Return response message
    return messaging.NewResponse(a.ID(), msg.From, msg.ID, response.Content()).Build(), nil
}

// Register with hub
hubInstance.RegisterAgent(myAgent, handler)
```

### Separation of Concerns

- **go-agents**: LLM protocol execution (Chat, Vision, Tools, Embeddings)
- **go-agents-orchestration**: Agent coordination and workflow orchestration
- **Integration**: Compose through wrapper pattern

## Design Patterns

### Contract Interface Pattern

Lower-level packages define minimal interfaces that higher-level packages implement:

- `hub.Agent` interface is minimal (`ID()` only)
- Users provide implementations that satisfy the contract
- Enables loose coupling and testability

### Configuration Lifecycle

1. Configuration exists only during initialization
2. Transform to domain objects at boundaries
3. Runtime behavior depends on initialized state
4. No configuration persistence beyond initialization

### Bottom-Up Development

1. Build and test lowest-level package (messaging)
2. Build next level using validated foundation (hub)
3. Each layer validates before adding the next
4. Prevents temporary broken states during development

### Go-Native Concurrency

- Use channels for message passing (not callbacks)
- Use contexts for cancellation and timeouts
- Use goroutines for concurrent message processing
- Embrace Go idioms rather than porting Python patterns
