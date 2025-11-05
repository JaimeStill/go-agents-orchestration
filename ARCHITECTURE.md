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

## Phase 2-3: State Management (Completed)

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
}

// StateNode represents a computation step in the graph
type StateNode interface {
    Execute(ctx context.Context, state State) (State, error)
}

// State represents data flowing through the graph
type State struct {
    data     map[string]any
    observer observability.Observer
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

Phase 2-3 implementation is complete:
- State type with immutable operations
- StateNode interface and FunctionNode
- Edge with transition predicates (AlwaysTransition, KeyExists, KeyEquals, Not, And, Or)
- StateGraph interface and concrete implementation
- Graph execution engine with cycle detection
- ExecutionError with rich context
- Observer integration throughout
- Test coverage: 95.6% (exceeds 80% requirement)

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
}

// ChainConfig defines configuration for sequential chain execution
type ChainConfig struct {
    CaptureIntermediateStates bool
    Observer                  string
}
```

**Implementation Status:**

Configuration package complete for current phases:
- HubConfig (Phase 1)
- GraphConfig (Phase 2)
- ChainConfig (Phase 4)
- Default configuration functions for all types
- Test coverage: 100%

## Phase 4-7: Workflow Patterns (Phase 4 Complete)

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

**Planned Patterns:**

2. **Parallel Execution** (Phase 5): Fan-out/fan-in with result aggregation and order preservation
3. **Conditional Routing** (Phase 7): State-based routing decisions with predicate evaluation
4. **Stateful Workflows** (Phase 7): Complex compositions combining patterns with state graphs

**Implementation Status:**

- Sequential Chain: Complete with comprehensive tests
- Test coverage: 97.4% (exceeds 80% requirement)
- Documentation: Complete with examples

### Pattern Independence

All workflow patterns are agnostic about processing approach:
- ✅ Direct go-agents usage (primary pattern - like classify-docs)
- ✅ Hub orchestration (optional for multi-agent coordination)
- ✅ Pure data transformation (no agents required)
- ✅ Mixed approaches (some steps with agents, some without)

The processor function signatures intentionally don't constrain implementation, enabling maximum flexibility.

## Phase 8: Observability (Planned)

### observability Package

**Purpose**: Production-grade tracing, metrics, and decision logging.

**Planned Features:**

- Execution trace capture across workflow steps
- Decision point logging with reasoning
- Performance metrics (latency, token usage, retries)
- Minimal performance overhead
- Optional (can be disabled)

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
