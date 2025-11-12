# Changelog

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
