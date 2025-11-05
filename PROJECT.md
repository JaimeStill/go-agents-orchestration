# go-agents-orchestration - Project Roadmap

## Project Scope

### Core Mission

Provide Go-native agent coordination primitives that extend [go-agents](https://github.com/JaimeStill/go-agents) with multi-agent orchestration, LangGraph-inspired state management, and composable workflow patterns.

### What This Package Provides

**Hub Coordination:**
- Multi-hub agent networking with hierarchical organization
- Agent registration with message handlers
- Cross-hub message routing and communication
- Hub lifecycle management (initialization, shutdown, cleanup)
- Fractal growth through shared agents across hubs

**Messaging Primitives:**
- Structured message types for inter-agent communication
- Message builders for fluent message construction
- Message routing and filtering logic
- Channel-based message delivery using Go concurrency

**State Management:**
- State graph execution with nodes, edges, and transitions
- State structure definitions and mutation patterns
- Transition predicates for conditional routing
- Checkpointing for recovery and rollback
- Cycle detection and loop handling

**Workflow Patterns:**
- Sequential chains with state accumulation
- Parallel execution (fan-out/fan-in) with result aggregation
- Conditional routing with dynamic handler selection
- Stateful workflows with cycles and retries

**Observability Infrastructure:**
- Execution trace capture across workflow steps
- Decision point logging with reasoning
- Performance metrics (token usage, timing, retries)
- Production debugging and optimization support

### What This Package Does NOT Provide

Capabilities intentionally left to the go-agents library:

- **LLM Protocol Execution**: Chat, Vision, Tools, Embeddings
- **Provider Integration**: OpenAI, Azure, Ollama, etc.
- **HTTP Transport**: Connection pooling, retries, streaming
- **Capability Format System**: Provider-specific API structures
- **Configuration Management**: Provider and model configuration

## Development Approach

### Bottom-Up Construction

Build from lowest-level primitives to highest-level features:

```
Level 1: messaging/     → Message primitives (no dependencies)
Level 2: hub/          → Agent coordination (depends on messaging)
Level 3: state/        → State graph execution (depends on messaging)
Level 4: patterns/     → Workflow patterns (depends on hub + state)
Level 5: observability/ → Cross-cutting metrics (depends on all)
```

**Rationale**: Each layer validates before adding the next. Prevents temporary broken states during development.

### Configuration-Driven Initialization

Following go-agents patterns:

- Core structures initialized through configuration
- Configuration transforms to domain objects at boundaries
- Runtime behavior depends on initialized state, not config values
- Configuration does not persist beyond initialization

### Testing Strategy

- **Black-box testing**: `package_test` suffix, public API only
- **Table-driven tests**: Multiple scenarios with clear documentation
- **Coverage requirements**: 80% minimum, 100% for critical paths
- **HTTP mocking**: Test integrations without live services

## Development Phases

### Phase 1: Foundation (Completed)

**Goal**: Establish messaging and hub coordination primitives.

**Packages:**
- `messaging/`: Message structures, builders, types ✅
- `hub/`: Hub interface, agent registration, message routing ✅
- `config/`: Hub configuration structures ✅

**Communication Patterns:**
- Send (fire-and-forget) ✅
- Request/Response (with correlation) ✅
- Broadcast (all agents except sender) ✅
- Pub/Sub (topic-based messaging) ✅

**Deliverables:**
- Working hub implementation ported from research ✅
- Multi-hub coordination support ✅
- Integration examples with go-agents ✅
- Comprehensive test coverage (86%) ✅
- Documentation (README, PROJECT, ARCHITECTURE) ✅

**Success Criteria:**
- Agents can register with hubs via message handlers ✅
- All communication patterns work correctly ✅
- Multi-hub coordination validated ✅
- Tests achieve 80%+ coverage ✅ (86% achieved)

**Example:**
- `examples/phase-01-hubs`: ISS Maintenance EVA scenario demonstrating all communication patterns with 4 agents across 2 hubs

### Phase 2: State Graph Core Infrastructure

**Goal**: Establish state management primitives and observability foundation.

**Estimated Time**: 8-10 hours

**Packages:**
- `observability/`: Observer interface, Event structures, EventType constants
- `state/`: State type, StateNode interface, Edge, StateGraph interface

**Features:**
- Minimal observability interfaces for event emission
- State type with immutable operations (Clone, Get, Set, Merge)
- StateNode interface defining computation steps
- Edge types with transition predicates
- StateGraph interface for workflow definition
- Observer integration in all state operations

**Integration:**
- Direct go-agents usage (primary pattern)
- Optional hub coordination
- Observer hooks for state changes
- NoOpObserver for optional observability

**Success Criteria:**
- State operations work with immutability
- Observer receives events for state mutations
- StateNode contract is clear and minimal
- Edge predicates evaluate correctly
- Foundation established for graph execution and patterns
- Tests achieve 80%+ coverage

### Phase 3: State Graph Execution Engine (Completed)

**Goal**: Implement graph executor with cycle detection and conditional routing.

**Packages:**
- `state/`: Executor, cycle detection, iteration limits ✅

**Features:**
- Graph execution algorithm with node traversal ✅
- Linear path execution (A → B → C) ✅
- Conditional routing via predicates ✅
- Cycle detection and max iterations protection ✅
- Context cancellation support ✅
- Observer hooks for node execution and edge transitions ✅
- Multiple exit points support ✅
- Rich error context (ExecutionError) ✅
- Full execution path tracking ✅

**Integration:**
- Executes state graphs from Phase 2 ✅
- Nodes can use direct go-agents calls or hub coordination ✅
- Full observability of execution flow ✅

**Deliverables:**
- Concrete stateGraph implementation ✅
- Graph building methods (AddNode, AddEdge, SetEntryPoint, SetExitPoint) ✅
- Explicit validation method ✅
- Complete execution engine ✅
- ExecutionError with rich context ✅
- Comprehensive tests (95.6% coverage) ✅
- Complete documentation ✅

**Success Criteria:**
- ✅ Linear and conditional graphs execute correctly
- ✅ Cycle detection prevents infinite loops
- ✅ Context cancellation stops execution immediately
- ✅ Error propagation works correctly
- ✅ Observer captures all execution events
- ✅ Tests achieve 80%+ coverage (95.6% achieved)

### Phase 4: Sequential Chains Pattern

**Goal**: Extract and generalize sequential chain pattern from classify-docs.

**Estimated Time**: 6-7 hours

**Packages:**
- `patterns/`: Chain pattern, ChainConfig, ChainResult

**Features:**
- Generic sequential chain with state accumulation
- Extract from `classify-docs/pkg/processing/sequential.go`
- ChainConfig for intermediate state capture and fail-fast
- Progress callbacks for monitoring
- Observer hooks for step completion

**Integration:**
- Works with any TContext type (including State from Phase 2)
- Direct go-agents usage (no hub required)
- Optional hub coordination for multi-agent steps
- Observer integration for chain events

**Success Criteria:**
- Pattern extracted and generalized successfully
- Works with multiple context types
- State type works naturally as TContext
- Hub integration is optional
- Tests demonstrate various usage patterns
- Tests achieve 80%+ coverage

### Phase 5: Parallel Execution Pattern

**Goal**: Implement concurrent processing with result aggregation.

**Estimated Time**: 7-9 hours

**Packages:**
- `patterns/`: Parallel pattern, ParallelConfig, worker pool

**Features:**
- Port architecture from classify-docs git history (commit d97ab1c^)
- Worker pool with auto-detection (NumCPU * 2, capped)
- Order preservation through indexed results
- Background result collector (deadlock prevention)
- Fail-fast error handling with context cancellation
- Observer hooks for worker events

**Integration:**
- Direct go-agents usage for concurrent agent calls
- Optional hub coordination for agent routing
- Composable with sequential chains
- Observer integration for parallel events

**Success Criteria:**
- Worker pool scales correctly
- Result order preserved despite concurrent execution
- No deadlocks under stress testing
- Context cancellation stops all workers
- Hub integration is optional
- Tests achieve 80%+ coverage

### Phase 6: Checkpointing Infrastructure

**Goal**: Add production-grade recovery capability to state graphs.

**Estimated Time**: 5-6 hours

**Packages:**
- `state/`: Checkpoint, CheckpointStore, MemoryCheckpointStore

**Features:**
- Checkpoint structures capturing graph position and state
- CheckpointStore interface for persistence
- Memory-based checkpoint store implementation
- ExecuteWithCheckpoints for automatic checkpointing
- Resume function for recovery from checkpoint
- Observer hooks for checkpoint events

**Integration:**
- Extends state graph execution from Phase 3
- Checkpoint intervals configurable
- Optional feature (doesn't block patterns)
- Observer integration for checkpoint lifecycle

**Success Criteria:**
- Checkpoints capture state correctly
- Resume continues from correct position
- Checkpoint intervals work as configured
- Memory store handles concurrent access
- Observer captures checkpoint events
- Tests achieve 80%+ coverage

### Phase 7: Conditional Routing + Integration

**Goal**: Complete pattern suite and validate composition.

**Estimated Time**: 7-10 hours

**Packages:**
- `patterns/`: Conditional routing pattern, integration helpers

**Features:**
- Conditional routing with predicate-based handler selection
- Integration helpers (ChainNode, ParallelNode, ConditionalNode)
- Pattern composition within state graphs
- State graphs using patterns as node implementations
- Stateful workflow examples
- Observer hooks for routing decisions

**Integration:**
- Patterns compose with state graphs
- State graphs use patterns as nodes
- Patterns can use state graphs internally
- Full observability of composed workflows

**Success Criteria:**
- Conditional routing pattern implemented
- All patterns compose correctly
- State graphs can use patterns as nodes
- Patterns can use state graphs for complex logic
- Integration helpers simplify composition
- Comprehensive stateful workflow examples
- Tests achieve 80%+ coverage

### Phase 8: Observability Implementation

**Goal**: Implement full observability infrastructure on integrated observer foundation.

**Estimated Time**: 6-8 hours

**Packages:**
- `observability/`: Structured logging, metrics aggregation, trace correlation

**Features:**
- Implement Observer interface with production features
- Structured logging adapter (slog integration)
- Metrics aggregation and reporting
- Execution trace correlation across workflows
- Decision point logging with reasoning capture
- Performance metrics (latency, token usage, retries)
- Confidence scoring utilities

**Integration:**
- Leverages observer hooks from Phases 2-7
- Minimal performance overhead
- Optional (can be disabled via NoOpObserver)
- Works across all orchestration primitives

**Success Criteria:**
- Comprehensive tracing without performance degradation
- Metrics provide production debugging insights
- Decision logging captures reasoning
- Confidence scoring is accurate and useful
- Observability can be toggled on/off
- Tests achieve 80%+ coverage

## Success Criteria

The go-agents-orchestration package is successful when:

1. **API Validation**: Successfully exercises go-agents public API, identifying friction points or missing capabilities
2. **Pattern Completeness**: Implements sequential, parallel, conditional, and stateful workflow patterns
3. **Hub Coordination**: Demonstrates multi-hub coordination with cross-hub messaging
4. **State Execution**: Provides LangGraph-style state graph execution with Go-native patterns
5. **Observability**: Production-grade observability without performance degradation
6. **Go-Native Design**: Leverages Go concurrency primitives naturally
7. **Documentation**: Comprehensive documentation with examples for all patterns
8. **Test Coverage**: Achieves 80%+ test coverage with black-box testing

## Package Lifecycle

### Pre-Release Phase (v0.x.x)

- Breaking changes allowed during validation
- Validate go-agents integration patterns
- Gather feedback on API design
- Iterate on core abstractions

### Release Candidate (v1.0.0-rc.x)

- API stabilized
- All eight phases complete
- Documentation complete
- Test coverage meets requirements

### Stable Release (v1.0.0+)

- API stability guarantees
- Production-ready
- Semantic versioning followed
- Backward compatibility maintained

## Design Philosophy

### From go-agents

1. **Minimal Abstractions**: Provide only essential primitives for agent coordination
2. **Format Extensibility**: Enable new patterns without modifying core code
3. **Configuration-Driven**: Compose workflows through declarative configuration where appropriate
4. **Type Safety**: Leverage Go's type system for compile-time safety
5. **Go-Native Patterns**: Embrace Go concurrency idioms rather than porting Python patterns

### Orchestration-Specific

1. **Hub as Networking Fabric**: Persistent coordination primitive for agent networking
2. **State Graphs as Workflows**: Transient execution contexts for stateful workflows
3. **Composition over Inheritance**: Build complex behaviors through composition
4. **Bottom-Up Development**: Validate each layer before adding the next
5. **Configuration Lifecycle**: Config exists only during initialization

## Template for Future Packages

As the first supplemental package in the go-agents ecosystem, go-agents-orchestration serves as a template:

- Validates go-agents public API through real-world usage
- Establishes patterns for supplemental package development
- Provides feedback to go-agents for API improvements
- Demonstrates integration patterns for future packages
- Documents development approach and testing strategy
