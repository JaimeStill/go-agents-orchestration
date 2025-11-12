# Phase 7: Conditional Routing + Integration - Development Session

## Session Overview

**Date**: November 12, 2025
**Phase**: Phase 7 - Conditional Routing + Integration
**Status**: Complete
**Coverage**: 95.3% workflows package (91.3% conditional, 80-100% integration helpers)

## Implementation Summary

Successfully implemented conditional routing pattern with predicate-based handler selection and state graph integration helpers. The implementation enables state-driven routing decisions within workflows and provides ChainNode, ParallelNode, and ConditionalNode helpers for composing workflow patterns within state graphs.

## Starting Point

Phase 6 (Checkpointing) complete with workflow persistence and recovery. Sequential chains (Phase 4) and parallel execution (Phase 5) patterns operational with 96.6% test coverage.

### Pre-Implementation State
- `pkg/workflows/chain.go`: Sequential chain execution with state accumulation
- `pkg/workflows/parallel.go`: Parallel execution with worker pool and result aggregation
- `pkg/workflows/error.go`: ChainError and ParallelError types
- `pkg/config/workflows.go`: ChainConfig and ParallelConfig
- No conditional routing pattern
- No state graph integration helpers

## Architecture Decisions

### No Hub Parameter in Integration Helpers

**Decision**: Integration helpers (ChainNode, ParallelNode, ConditionalNode) do not accept hub parameters.

**Rationale**: Processors can capture hub in closures if needed, eliminating unnecessary parameter clutter. Direct go-agents usage is the primary pattern, with hub coordination being optional.

**Example**:
```go
processor := func(ctx context.Context, item string, s state.State) (state.State, error) {
    response, _ := hub.Request(ctx, fromAgent, toAgent, item)
    return s.Set("result", response.Data), nil
}
```

### Aggregator Function for ParallelNode

**Decision**: ParallelNode requires aggregator function transforming results into state.

**Rationale**: Parallel execution produces `[]TResult`, but state graph expects `State`. Aggregator bridges this gap, enabling flexible result integration.

**Signature**:
```go
func(results []TResult, currentState state.State) state.State
```

### No Progress Callback for Conditional Routing

**Decision**: ProcessConditional does not accept progress callback parameter.

**Rationale**: Conditional routing is a single decision point, not an iterative process. Progress callbacks make sense for chains and parallel execution where multiple items are processed, but not for single predicate evaluation.

### Routes Structure with Default Handler

**Decision**: Routes struct contains Handlers map and optional Default handler.

**Initial Name**: RouteConfig
**Final Name**: Routes (Config suffix has specific meaning in codebase)

**Structure**:
```go
type Routes[TState any] struct {
    Handlers map[string]RouteHandler[TState]
    Default  RouteHandler[TState]
}
```

**Rationale**: Default handler provides fallback when predicate returns route not in Handlers map, enabling graceful handling of unexpected routes.

### ConditionalError Location

**Decision**: ConditionalError defined in `pkg/workflows/error.go` alongside ChainError and ParallelError.

**Rationale**: Aligns with existing pattern where all workflow error types are centralized in error.go.

## Implementation Details

### Conditional Routing Core

**File**: `pkg/workflows/conditional.go` (NEW - 224 lines)

**Core Types**:
```go
type RoutePredicate[TState any] func(state TState) (route string, err error)

type RouteHandler[TState any] func(
    ctx context.Context,
    state TState,
) (TState, error)

type Routes[TState any] struct {
    Handlers map[string]RouteHandler[TState]
    Default  RouteHandler[TState]
}
```

**ProcessConditional Function**:
- Evaluates predicate to select route
- Looks up handler in Routes.Handlers map
- Falls back to Default if route not found
- Executes selected handler with current state
- Returns updated state

**Observer Events**:
- `EventRouteEvaluate`: Before predicate evaluation
- `EventRouteSelect`: After route selection
- `EventRouteExecute`: After handler execution

**Context Cancellation**:
- Checks context before predicate evaluation
- Checks context before handler execution

### State Graph Integration Helpers

**File**: `pkg/workflows/integration.go` (NEW - 157 lines)

**ChainNode**:
- Wraps ProcessChain in StateNode
- Executes sequential processing within graphs
- Returns final state from chain

**ParallelNode**:
- Wraps ProcessParallel in StateNode
- Processes items concurrently via worker pool
- Aggregates results into state via aggregator function
- Returns aggregated state

**ConditionalNode**:
- Wraps ProcessConditional in StateNode
- Evaluates predicate and executes handler
- Returns updated state from handler

**Key Design**:
- All helpers return `state.StateNode` for graph integration
- Error wrapping preserves context: "chain node failed", "parallel node failed", "conditional node failed"
- Observer events propagate from wrapped patterns

### Error Type

**File**: `pkg/workflows/error.go` (UPDATED)

**ConditionalError**:
```go
type ConditionalError[TState any] struct {
    Route string
    State TState
    Err   error
}

func (e ConditionalError[TState]) Error() string
func (e ConditionalError[TState]) Unwrap() error
```

**Error Message Format**:
- With route: `"conditional routing failed on route '{route}': {error}"`
- Without route: `"conditional routing failed: {error}"`

### Configuration

**File**: `pkg/config/workflows.go` (UPDATED)

**ConditionalConfig**:
```go
type ConditionalConfig struct {
    Observer string
}

func DefaultConditionalConfig() ConditionalConfig {
    return ConditionalConfig{
        Observer: "slog",
    }
}
```

## Testing

**Files**:
- `tests/workflows/conditional_test.go` (NEW - 411 lines)
- `tests/workflows/integration_test.go` (NEW - 327 lines)

**Test Coverage**:
- Conditional routing: 91.3%
- Integration helpers: 80-100%
- Overall workflows package: 95.3%

**Conditional Routing Tests** (7 test functions):
1. Basic routing (3 scenarios: route1, route2, state-based)
2. Default handler fallback
3. Missing route without default (error)
4. Predicate evaluation error
5. Handler execution error
6. Context cancellation
7. Observer events emission
8. ConditionalError error messages
9. ConditionalError unwrap

**Integration Helper Tests** (5 test functions):
1. ChainNode in state graph
2. ParallelNode in state graph with aggregation
3. ConditionalNode in state graph
4. Combined integration (all three helpers)
5. Error propagation through wrappers

**All Tests Passing**: 12 new test functions + existing workflows tests = 100% pass rate

## Implementation Issues and Corrections

### Issue 1: ParallelNode Result Type Mismatch

**Problem**: Initial implementation passed `result` (ParallelResult struct) to aggregator instead of `result.Results` (the actual results array).

**Error**: `cannot use results (variable of struct type ParallelResult[TItem, TResult]) as []TResult`

**Resolution**: Changed `aggregator(result.Results, s)` to access the Results field.

### Issue 2: Graph Constructor Return Type

**Problem**: Example used `graph := state.NewGraph(graphCfg)` but NewGraph returns `(StateGraph, error)`.

**Resolution**: Updated example to handle error return: `graph, _ := state.NewGraph(graphCfg)`.

### Issue 3: DefaultGraphConfig Parameter

**Problem**: Called `DefaultGraphConfig()` without required graph name parameter.

**Resolution**: Updated to `DefaultGraphConfig("document-review")`.

### Issue 4: NoOpObserver Initialization

**Problem**: Tests used `observability.NewNoOpObserver()` but function doesn't exist.

**Resolution**: Changed to `&observability.NoOpObserver{}` for direct struct initialization.

### Issue 5: Test File Package Conflict

**Problem**: Created `testutil.go` with `package workflows_test` but `chain_test.go` already had `captureObserver` definition.

**Error**: `found packages workflows (chain_test.go) and workflows_test (testutil.go)`

**Resolution**: Removed `testutil.go`, used existing `newCaptureObserver()` from `chain_test.go`.

### Issue 6: Pre-existing Test Failure

**Problem**: `TestGraphConfig_DefaultGraphConfig` expected Observer="noop" but actual default is "slog".

**Context**: "This is from when we created the SlogObserver and made it the default. This test case must have been missed in that effort."

**Resolution**: Updated test expectation from "noop" to "slog" in `tests/config/state_test.go`.

## Comprehensive Example

**File**: `examples/phase-07-conditional-routing/main.go` (NEW - 478 lines)

**Scenario**: Document review workflow with 6 agents (3 analysts + 3 reviewers)

**Architecture**:
1. **Analysis Node** (ChainNode): 3 analysts sequentially process document sections
2. **Review Node** (ParallelNode): 3 reviewers concurrently analyze document
3. **Consensus Node** (ConditionalNode): Routes to approve or revise based on review consensus
4. **Revision Loop**: Max 2 revisions with termination logic

**State Flow**:
- Initial: document text, sections, revision count
- Analysis: accumulated section summaries
- Review: aggregated review scores and comments
- Routing: approval status or revision instructions

**Features Demonstrated**:
- Sequential chain with state accumulation
- Parallel execution with result aggregation
- Conditional routing with default handler
- State graph composition
- Revision loops with KeyEquals predicate
- Real LLM agent integration

**Execution Output**: Successfully processes document through analysis → review → approval cycle with revision loop support.

## Documentation

**Godoc Comments Added**:

### pkg/workflows/conditional.go
- `RoutePredicate` type with example
- `RouteHandler` type with example
- `Routes` struct with field descriptions
- `ProcessConditional` function with comprehensive documentation

### pkg/workflows/integration.go
- `ChainNode` function with sequential processing description
- `ParallelNode` function with aggregation explanation
- `ConditionalNode` function with routing logic documentation

### pkg/workflows/error.go
- `ConditionalError` struct with field descriptions
- Error() and Unwrap() method documentation

### examples/phase-07-conditional-routing/README.md
- Workflow architecture explanation
- Execution flow diagrams
- Feature highlights
- Code structure walkthrough

### examples/README.md
- Phase 7 entry with demonstration details
- Performance characteristics
- Observer events
- Example dependencies

## Files Modified

1. **pkg/workflows/conditional.go** - NEW - Conditional routing pattern (224 lines)
2. **pkg/workflows/integration.go** - NEW - State graph helpers (157 lines)
3. **pkg/workflows/error.go** - UPDATED - Added ConditionalError type
4. **pkg/config/workflows.go** - UPDATED - Added ConditionalConfig
5. **tests/workflows/conditional_test.go** - NEW - Conditional routing tests (411 lines)
6. **tests/workflows/integration_test.go** - NEW - Integration helper tests (327 lines)
7. **tests/config/state_test.go** - UPDATED - Fixed DefaultGraphConfig test expectation
8. **examples/phase-07-conditional-routing/main.go** - NEW - Document review workflow (478 lines)
9. **examples/phase-07-conditional-routing/README.md** - NEW - Example documentation
10. **examples/README.md** - UPDATED - Added Phase 7 entry
11. **ARCHITECTURE.md** - UPDATED - Phase 7 implementation details
12. **README.md** - UPDATED - Phase 7 completion and examples

## Key Patterns Applied

### Generic Type Parameters
- RoutePredicate and RouteHandler generic over TState
- Integration helpers maintain generic signatures from wrapped patterns
- Type safety preserved through composition

### Registry Pattern
- ConditionalConfig.Observer resolves via observability registry
- Follows established pattern from other workflow configs
- Enables configuration-driven observer selection

### Error Context Preservation
- ConditionalError captures Route, State, and underlying error
- Integration helpers wrap errors with context
- Error chains preserved via Unwrap()

### Immutability Through Composition
- Handlers return new state (never mutate)
- Integration helpers pass state through transformations
- State flows immutably through graph execution

### Observer Integration
- Three event types for conditional routing lifecycle
- Events include metadata (route_count, route name, has_default)
- Integration with existing observer infrastructure

## Design Lessons

### Progress Callback Evaluation

**Initial Plan**: Include progress callback for conditional routing

**User Feedback**: "Conditional progress callbacks make no sense, so we should not implement them"

**Rationale**: Single decision point vs. iterative processing

**Application**: Progress callbacks only for patterns processing multiple items (chains, parallel)

**Result**: ConditionalNode has no progress parameter

### Configuration Naming Conventions

**Issue**: RouteConfig name conflicted with *Config suffix semantics

**User Insight**: "The *Config suffix has specific meaning in this codebase"

**Resolution**: Renamed to Routes to reflect mapping nature

**Application**: Reserve *Config suffix for configuration structures that participate in configuration lifecycle

**Lesson**: Naming conventions carry semantic weight; honor established patterns

### Error Type Organization

**Decision**: Move ConditionalError to error.go

**User Direction**: "moving ConditionalError and its methods to pkg/workflows/error.go to align with the existing pattern"

**Pattern**: All workflow error types centralized in single file

**Application**: Follow existing organizational patterns even if initial approach differs

**Benefit**: Consistency aids navigation and understanding

### Example Simplification

**Initial Plan**: Simple integration example + stateful workflow example

**User Feedback**: "I think we should just stick to one example for this phase: one that demonstrates the stateful workflows from Phase 7.4, but with actual agents integrated"

**Resolution**: Single comprehensive example showing all features

**Rationale**: One rich example better than multiple simple ones

**Result**: Document review workflow demonstrates ChainNode, ParallelNode, ConditionalNode, revision loops, and real agents

### Test Infrastructure Reuse

**Problem**: Created duplicate captureObserver in new test file

**Resolution**: Reused existing infrastructure from chain_test.go

**Lesson**: Audit existing test utilities before creating new ones

**Application**: Check for shared test infrastructure (observers, helpers, fixtures)

**Benefit**: Avoid duplication and package conflicts

## Production Considerations

### Type Safety
- Generic type parameters ensure compile-time safety
- Route name typos caught at runtime (missing handler error)
- Default handler provides runtime safety net

### Error Handling
- Predicate errors captured with state context
- Handler errors include route information
- Missing route without default returns clear error

### Context Cancellation
- Checks context before predicate evaluation
- Checks context before handler execution
- Honors cancellation at all decision points

### Observability
- Three distinct events for routing lifecycle
- Metadata includes route selection details
- Integration with existing observer infrastructure

## Future Extensibility

### Advanced Routing Strategies

**Multi-Route Selection**:
```go
type MultiRoutePredicate[TState any] func(state TState) ([]string, error)
```

**Weighted Routing**:
```go
type WeightedRoutes[TState any] struct {
    Handlers map[string]RouteHandler[TState]
    Weights  map[string]float64
}
```

### Route Middleware

**Pre/Post Processing**:
```go
type RouteMiddleware[TState any] func(
    next RouteHandler[TState],
) RouteHandler[TState]
```

### Conditional Composition

**Nested Conditionals**:
```go
approveHandler := workflows.ConditionalNode(subConfig, subPredicate, subRoutes)
routes.Handlers["approve"] = approveHandler.Execute
```

### Dynamic Route Discovery

**Runtime Route Registration**:
```go
type DynamicRoutes[TState any] struct {
    Resolver func(route string) (RouteHandler[TState], error)
    Default  RouteHandler[TState]
}
```

## Success Metrics

**Test Coverage**: 95.3% workflows package (exceeds 80% target)
**Test Quality**: 12 comprehensive test functions across 2 files, all passing
**Implementation Quality**: 6 minor corrections during implementation (types, constructors, initialization)
**API Clarity**: Clean integration with state graphs through helper functions
**Example Richness**: Single comprehensive example demonstrates all Phase 7 features
**Documentation**: Complete godoc comments for all public API
**Production Ready**: Type-safe, context-aware, observable, error-handled

## Remaining Work

None. Phase 7 is complete with:
- ✅ Conditional routing pattern (91.3% coverage)
- ✅ State graph integration helpers (80-100% coverage)
- ✅ Comprehensive example with real agents
- ✅ Observer integration
- ✅ Complete documentation and godoc comments
- ✅ ARCHITECTURE.md updated
- ✅ README.md updated

## Next Phase

Phase 8: Observability Implementation
- Advanced execution trace correlation across workflows
- Decision point logging with reasoning capture
- Performance metrics aggregation (latency, token usage, retries)
- Confidence scoring utilities
- OpenTelemetry integration
