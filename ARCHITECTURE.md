# go-agents-orchestration - Architecture

## Overview

This document specifies the technical architecture of go-agents-orchestration, a supplemental package providing multi-agent coordination primitives for the [go-agents](https://github.com/JaimeStill/go-agents) ecosystem.

## Package Structure

### Dependency Hierarchy

Packages are organized in a strict dependency hierarchy (bottom-up):

```
Level 1: messaging/          # Message primitives (no dependencies)
    ↓
Level 2: hub/               # Agent coordination (depends on messaging)
    ↓
Level 3: state/             # State graph execution (depends on messaging)
    ↓
Level 4: patterns/          # Workflow patterns (depends on hub + state)
    ↓
Level 5: observability/     # Cross-cutting metrics (depends on all)
```

**Rationale**: Lower layers cannot import higher layers. This prevents circular dependencies and ensures each layer can be validated independently.

### Package Organization

```
github.com/JaimeStill/go-agents-orchestration/
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
│   └── hub.go              # HubConfig with go-agents integration
│
├── state/                  # Level 3: State graph execution (Phase 2)
│   ├── graph.go            # StateGraph interface
│   ├── node.go             # StateNode interface
│   ├── edge.go             # Edge and transition logic
│   ├── state.go            # State structure
│   ├── executor.go         # Graph execution engine
│   └── checkpoint.go       # State checkpointing
│
├── patterns/               # Level 4: Workflow patterns (Phase 3)
│   ├── chain.go            # Sequential chains
│   ├── parallel.go         # Parallel execution
│   ├── conditional.go      # Conditional routing
│   └── stateful.go         # Stateful workflows
│
└── observability/          # Level 5: Cross-cutting (Phase 4)
    ├── trace.go            # Execution tracing
    ├── metrics.go          # Metrics collection
    └── decision.go         # Decision logging
```

## Phase 1: Foundation (Current)

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

### config Package

**Purpose**: Configuration structures integrating with go-agents.

**Configuration Structures:**

```go
// HubConfig defines configuration for a Hub instance
type HubConfig struct {
    // Hub identity
    Name string

    // Orchestrator configuration
    OrchestratorID    string
    OrchestratorAgent *agentconfig.AgentConfig // go-agents config

    // Communication settings
    ChannelBufferSize int
    DefaultTimeout    time.Duration

    // Observability
    Logger *slog.Logger
}
```

**Configuration Lifecycle:**

1. Load configuration from code/files/environment
2. Transform to domain objects at package boundaries
3. Runtime behavior depends on initialized state
4. Configuration does not persist beyond initialization

**Integration with go-agents:**

```go
// Create go-agents configuration
agentConfig := &config.AgentConfig{
    Transport: &config.TransportConfig{
        Provider: "openai",
        Model:    "gpt-4",
        APIKey:   "key",
    },
    SystemPrompt: "You are an orchestrator",
}

// Use in hub configuration
hubConfig := &config.HubConfig{
    Name:              "main-hub",
    OrchestratorID:    "orchestrator",
    OrchestratorAgent: agentConfig, // Link to go-agents
    ChannelBufferSize: 100,
    DefaultTimeout:    30 * time.Second,
}
```

## Phase 2: State Management (Planned)

### state Package

**Purpose**: LangGraph-inspired state graph execution with Go-native patterns.

**Key Concepts:**

- **StateGraph**: Workflow definition with nodes and edges
- **StateNode**: Computation step that receives and returns state
- **Edge**: Transition between nodes with optional predicates
- **State**: Data flowing through the graph
- **Executor**: Engine that executes state graph transitions
- **Checkpoint**: State snapshot for recovery

**Planned Interfaces:**

```go
// StateGraph defines a workflow as a graph of nodes and edges
type StateGraph interface {
    AddNode(name string, node StateNode) error
    AddEdge(from, to string, predicate TransitionPredicate) error
    SetEntryPoint(node string) error
    Execute(ctx context.Context, initialState State) (State, error)
}

// StateNode represents a computation step in the graph
type StateNode interface {
    Execute(ctx context.Context, state State) (State, error)
}

// State represents data flowing through the graph
type State map[string]any

// TransitionPredicate determines which edge to follow
type TransitionPredicate func(state State) bool
```

**Integration with Hub:**

State graph nodes CAN use hub for agent coordination:

```go
// Hub node that requests agent processing
type HubNode struct {
    hub       hub.Hub
    fromAgent string
    toAgent   string
}

func (n *HubNode) Execute(ctx context.Context, state State) (State, error) {
    response, err := n.hub.Request(ctx, n.fromAgent, n.toAgent, state["data"])
    if err != nil {
        return nil, err
    }

    state["result"] = response.Data
    return state, nil
}
```

## Phase 3: Workflow Patterns (Planned)

### patterns Package

**Purpose**: High-level workflow compositions built on state graphs.

**Planned Patterns:**

1. **Sequential Chain**: Linear workflow with state accumulation
2. **Parallel Execution**: Fan-out/fan-in with result aggregation
3. **Conditional Routing**: State-based routing decisions
4. **Stateful Workflows**: Complex state machines with cycles

These patterns compose state graphs and hub messaging into reusable workflow building blocks.

## Phase 4: Observability (Planned)

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
