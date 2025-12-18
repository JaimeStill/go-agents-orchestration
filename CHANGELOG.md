# Changelog

## [v0.3.0] - 2025-12-18

**Breaking**:

- `pkg/config` - ParallelConfig.FailFast field renamed to FailFastNil with accessor method

  The `FailFast` field is now `FailFastNil *bool` with a `FailFast()` method that handles nil-checking and returns the effective boolean value (defaulting to true when nil). This enables distinguishing between "not set" (nil, uses default true) and "explicitly set to false", which is required for proper Merge behavior where unspecified fields preserve defaults.

**Added**:

- `pkg/config` - Merge methods for configuration composition

  Added `Merge(*Config)` methods to all configuration types following the go-agents pattern. Enables layered configuration where loaded configs merge over defaults, with zero-values preserved from defaults.

- `pkg/state` - NewGraphWithDeps for dependency injection

  Added `NewGraphWithDeps(cfg, observer, checkpointStore)` constructor that accepts Observer and CheckpointStore instances directly instead of resolving from global registries. Enables cleaner integration where callers manage their own instances (e.g., per-execution database-backed stores).

- `pkg/state` - Thread-safe checkpoint store registry

  Added `sync.RWMutex` protection to the checkpoint store registry. `GetCheckpointStore` and `RegisterCheckpointStore` are now safe for concurrent use.

- `pkg/observability` - Thread-safe observer registry

  Added `sync.RWMutex` protection to the observer registry. `GetObserver` and `RegisterObserver` are now safe for concurrent use.

## [v0.2.0] - 2025-12-17

Breaking API changes to support JSON serialization of State for checkpoint persistence.

**Breaking**:

- `pkg/state` - State struct fields are now public with JSON tags

  Changed from private fields with getter methods to public fields with JSON serialization support. The `Data`, `RunID`, `CheckpointNode`, and `Timestamp` fields are now exported. The `Observer` field is excluded from JSON serialization via `json:"-"` tag. This enables direct JSON marshaling/unmarshaling for checkpoint persistence without intermediate transformation structs.

- `pkg/state` - Removed redundant getter methods from State

  Removed `RunID()`, `CheckpointNode()`, and `Timestamp()` methods. Access these values directly via fields: `state.RunID`, `state.CheckpointNode`, `state.Timestamp`.

**Added**:

- `pkg/state` - Edge.Name field for predicate identification

  Added optional `Name` field to Edge struct for identifying predicates during routing decisions. Enables observers to record which predicate was evaluated when capturing workflow execution history.

- `pkg/state` - Enhanced observer events with state snapshots and predicate details

  EventNodeStart now includes `input_snapshot` containing state data before node execution. EventNodeComplete now includes `output_snapshot` containing state data after node execution. EventEdgeTransition now includes `predicate_name` and `predicate_result` fields for routing decision audit trails.

## [v0.1.0] - 2025-11-12

Initial pre-release.

**Added**:

- `pkg/observability` - Observer pattern for execution event emission and structured logging

  Provides Observer interface for capturing execution events across all orchestration primitives. Includes NoOpObserver for zero-overhead operation and SlogObserver for structured logging via Go's standard slog package. Registry pattern enables configuration-driven observer selection and custom observer implementations.

- `pkg/messaging` - Message structures and builders for inter-agent communication

  Defines Message type with routing, correlation, and metadata fields. Provides fluent builders for creating Request, Response, Notification, and Broadcast messages. Supports message types, priority levels, custom headers, and topic-based pub/sub messaging.

- `pkg/hub` - Multi-hub agent coordination with message routing

  Enables multi-hub agent networking with hierarchical organization. Agents register with hubs via minimal Agent interface and message handlers. Supports communication patterns: Send (fire-and-forget), Request/Response (with correlation), Broadcast (all agents except sender), and Pub/Sub (topic-based with sender filtering). MessageChannel provides context-aware message delivery with cancellation support. Agents can participate in multiple hubs simultaneously, enabling cross-hub coordination through shared agents.

- `pkg/state` - State graph execution with checkpointing and persistence

  Provides LangGraph-inspired state graph execution with Go-native patterns. State type offers immutable operations (Get, Set, Clone, Merge) for workflow data. StateGraph interface enables workflow definition with nodes, edges, and transition predicates. Built-in predicates include AlwaysTransition, KeyExists, KeyEquals, and logical operators (Not, And, Or). Checkpointing support via CheckpointStore interface enables workflow persistence and recovery. MemoryCheckpointStore provides thread-safe in-memory storage. StateGraph.Resume continues execution from saved checkpoints with automatic cleanup on success.

- `pkg/workflows` - Composable patterns for sequential, parallel, and conditional workflows

  Implements three core workflow patterns: ProcessChain for sequential processing with state accumulation, ProcessParallel for concurrent execution with worker pools and result aggregation, and ProcessConditional for predicate-based routing. All patterns are generic over item and state types, support progress callbacks (chain and parallel), and provide rich error context. Integration helpers (ChainNode, ParallelNode, ConditionalNode) wrap patterns as StateNodes for composition within state graphs. Parallel execution features auto-detected worker counts, fail-fast and collect-all-errors modes, and order preservation despite concurrent execution.

- `pkg/config` - Configuration structures for all orchestration primitives

  Provides configuration types for all packages: HubConfig (buffer size, timeout), GraphConfig (observer, max iterations, checkpointing), ChainConfig (intermediate state capture, observer), ParallelConfig (worker count, fail-fast mode, observer), and ConditionalConfig (observer). All configuration types include default factory functions with sensible values.
