# Phase 4: Sequential Chains Pattern

## Starting Point

Phase 4 implementation began with completed state management foundation:
- State management primitives complete (State, StateNode, Edge, StateGraph) from Phase 2
- State graph execution engine complete with 95.6% coverage from Phase 3
- Observer pattern fully integrated throughout state operations
- PROJECT.md, ARCHITECTURE.md, and design documents established

Source material for extraction:
- `github.com/JaimeStill/go-agents/tools/classify-docs/pkg/processing/sequential.go`
- Real-world sequential processor implementing fold/reduce pattern
- Document-specific implementation needing generalization

Critical architectural decisions made collaboratively with user:
- Package naming: `workflows/` (not `patterns/`)
- Error handling: Rich ChainError[TItem, TContext] with complete context
- Observer integration: Comprehensive event emission at chain and step levels
- Configuration: ChainConfig with CaptureIntermediateStates and Observer string
- Error return: Standard error interface (idiomatic Go)
- Configuration location: ChainConfig in pkg/config/workflows.go (not in workflows package)

## Implementation Overview

### Configuration Package Extension

Extended configuration package with workflows configuration.

**File Created:**
- `pkg/config/workflows.go` - ChainConfig with JSON support

**ChainConfig Structure:**
```go
type ChainConfig struct {
    CaptureIntermediateStates bool   `json:"capture_intermediate_states"`
    Observer                  string `json:"observer"`
}
```

**Key Implementation Details:**
- Observer as string (not ObserverOption) for JSON serialization
- Default observer "noop" for zero-overhead execution
- Follows same pattern as GraphConfig
- DefaultChainConfig() provides sensible defaults

### Workflows Package Implementation

Implemented complete sequential chain pattern with full generic support.

**Files Created:**
- `pkg/workflows/error.go` - ChainError type
- `pkg/workflows/chain.go` - Core chain implementation
- `pkg/workflows/doc.go` - Package documentation

**Package Structure:**
- error.go defines ChainError[TItem, TContext] first (dependency for chain.go)
- chain.go defines all types and ProcessChain function
- doc.go provides comprehensive package-level documentation

### Error Handling

Implemented rich error type for complete debugging context.

**ChainError Structure:**
```go
type ChainError[TItem, TContext any] struct {
    StepIndex int
    Item      TItem
    State     TContext
    Err       error
}
```

**Key Features:**
- Generic over both TItem and TContext
- Implements standard error interface
- Unwrap() support for errors.Is and errors.As
- Captures complete failure context (step, item, state, error)

### Sequential Chain Pattern

Implemented fold/reduce pattern with state accumulation.

**Core Types:**
- `StepProcessor[TItem, TContext]` - Processes single item with state
- `ProgressFunc[TContext]` - Optional progress callback
- `ChainResult[TContext]` - Result with final state, optional intermediate states, and step count

**ProcessChain Implementation:**
- Fully generic over TItem and TContext
- Observer resolution from config at function entry
- Empty chain handling (returns initial state)
- Intermediate state capture (includes initial state at index 0)
- Progress callbacks after each successful step
- Context cancellation checked at start of each iteration
- All errors wrapped in ChainError
- Observer events emitted at all key execution points

**Event Emission Pattern:**
1. EventChainStart - Before processing begins
2. EventStepStart - Before each step processes
3. EventStepComplete - After each step (success or error)
4. EventChainComplete - When chain finishes

### Documentation

Added comprehensive godoc comments to all exported types and functions.

**Documentation Coverage:**
- Package-level doc in doc.go explaining sequential chain pattern
- ChainConfig and DefaultChainConfig in config package
- ChainError with error unwrapping example
- StepProcessor with direct agent usage example
- ProgressFunc with usage example
- ChainResult with intermediate state explanation
- ProcessChain with complete parameter documentation and examples

**Example Patterns Documented:**
- Direct agent usage (primary pattern)
- State package integration
- Hub coordination (optional)
- Generic type flexibility

### Testing

Comprehensive black-box test suite achieving 97.4% coverage.

**File Created:**
- `tests/workflows/chain_test.go` - Complete test suite

**Test Coverage:**
- Basic chain execution (3-5 steps with state accumulation)
- Empty chain (0 items returns initial state)
- Single item chain
- Error in middle step (ChainError with context)
- Context cancellation mid-chain
- Progress callback invocation (called after each step)
- Intermediate state capture (includes initial + all steps)
- Large chains (1000 items, performance validation)
- Observer integration (event emission verification)
- Observer on error (error flags in events)
- ChainError unwrapping (errors.Is, errors.As)
- Generic types (custom structs for TItem and TContext)
- Invalid observer (error handling)
- No progress callback (nil handling)

**Test Patterns:**
- captureObserver helper for event validation
- Table-driven tests where appropriate
- Black-box testing (package workflows_test)
- Observer registration without cleanup (persistent)

**Coverage Results:**
- 97.4% coverage (exceeds 80% requirement)
- ProcessChain: 100% coverage
- ChainError.Unwrap: 100% coverage
- ChainError.Error: 0% (called but not directly tested, acceptable)

## Technical Decisions

### Package Naming: workflows/ not patterns/

Chosen for domain clarity:

**Rationale:**
- More specific and discoverable
- Better communicates intent (multi-step process orchestration)
- Aligns with user expectations ("workflow" terminology)
- Updated PROJECT.md, ARCHITECTURE.md throughout

### Configuration Location: config Package

ChainConfig placed in pkg/config/workflows.go:

**Rationale:**
- Follows project configuration principles
- Consistent with GraphConfig and HubConfig patterns
- Enables JSON serialization
- Configuration transforms to domain objects at boundaries
- Observer as string (not ObserverOption) for serialization

### Rich Error Context: ChainError[TItem, TContext]

Opted for complete error context over minimal:

**Rationale:**
- Aligns with ExecutionError pattern from state graphs
- Provides all context needed for debugging
- Generic over both types preserves complete failure state
- Standard error interface with Unwrap support
- Memory cost acceptable for debugging value

### Observer Integration: Comprehensive Events

Emit events at chain and step levels:

**Rationale:**
- Consistent with state graph execution pattern
- Enables per-step progress tracking
- Supports Phase 8 observability goals
- Performance impact minimal with NoOpObserver
- Complete execution trace for debugging

### Intermediate State Capture: Includes Initial

When CaptureIntermediateStates=true, includes initial state at index 0:

**Rationale:**
- Complete state evolution (initial → step1 → step2 → final)
- Useful for debugging state transformations
- Index correspondence: intermediate[i] = state after step i
- Maintains classify-docs behavior

### Progress Callback Timing: After Each Step

Progress callback called after successful step completion, not before:

**Rationale:**
- Progress represents completed work
- 0 completed might be confusing (nothing done yet)
- Consistent with classify-docs implementation
- Not called on errors

### Empty Chain Handling: Return Initial State

Empty chain returns initial state as final with 0 steps:

**Rationale:**
- Graceful handling without error
- Maintains type consistency (always returns ChainResult)
- Emits start/complete events for observability consistency
- Natural fold/reduce behavior

## Final Architecture State

### Package Structure

```
pkg/config/
├── workflows.go      # ChainConfig (Phase 4)
└── ...

pkg/workflows/
├── error.go          # ChainError type
├── chain.go          # Sequential chain implementation
└── doc.go            # Package documentation

tests/workflows/
└── chain_test.go     # Black-box tests (97.4% coverage)
```

### Implementation Status

**Phase 4 Delivered:**
- ✅ ChainConfig in config package with JSON support
- ✅ ChainError[TItem, TContext] with unwrap support
- ✅ StepProcessor[TItem, TContext] function type
- ✅ ProgressFunc[TContext] function type
- ✅ ChainResult[TContext] result type
- ✅ ProcessChain[TItem, TContext] implementation
- ✅ Complete observer integration
- ✅ Comprehensive godoc comments
- ✅ Black-box tests with 97.4% coverage
- ✅ Documentation updates (PROJECT.md, ARCHITECTURE.md, state-management-and-workflow-patterns.md)

### Execution Flow

```
Configuration → ProcessChain → Resolve Observer → Emit Start Event
                     ↓
                Empty Check → Return Initial if Empty
                     ↓
           Initialize Intermediate (if capturing)
                     ↓
        For Each Item:
           → Check Cancellation
           → Emit StepStart
           → Process Item
           → Emit StepComplete
           → Capture Intermediate (if enabled)
           → Progress Callback (if provided)
                     ↓
              Return ChainResult
           → Final State
           → Intermediate States (if captured)
           → Step Count
                     ↓
           Emit Complete Event
```

## Documentation Updates

All core documentation updated to reflect workflows package:

**PROJECT.md:**
- Phase 4 section updated with workflows/ package
- Phase 5 and 7 sections updated
- Development approach section updated (Level 4: workflows/)

**ARCHITECTURE.md:**
- Added workflows package section with Phase 4 details
- Documented sequential chain pattern
- Documented pattern independence
- Updated config package section with ChainConfig

**state-management-and-workflow-patterns.md:**
- Replaced all patterns/ references with workflows/
- Updated package paths, function names, config types
- Updated test organization section
- Updated immediate actions section

## Phase 4 Completion Status

**Completed Objectives:**
- ✅ Extracted sequential chain pattern from classify-docs
- ✅ Created workflows package with full generic support
- ✅ Implemented ChainConfig in config package
- ✅ Implemented ChainError with rich context
- ✅ Implemented ProcessChain with all features
- ✅ Integrated observer hooks throughout
- ✅ Created comprehensive tests (97.4% coverage)
- ✅ Added complete documentation
- ✅ Updated all project documentation

**Phase 4 Deliverables:**
- ✅ Working sequential chain pattern
- ✅ Generic over TItem and TContext
- ✅ State type works naturally as TContext
- ✅ Rich error context for debugging
- ✅ Complete observer integration
- ✅ Progress callback support
- ✅ Intermediate state capture
- ✅ Empty chain handling
- ✅ Context cancellation support
- ✅ 97.4% test coverage
- ✅ Complete documentation

**Success Criteria Met:**
- ✅ Pattern extracted and generalized successfully
- ✅ Works with multiple context types
- ✅ State type works naturally as TContext
- ✅ Hub integration is optional (pattern agnostic)
- ✅ Tests demonstrate various usage patterns
- ✅ Tests achieve 80%+ coverage (97.4% achieved)

## Next Steps: Phase 5 Planning

Phase 5 will implement parallel execution pattern:

**Planned Features:**
- Extract pattern from classify-docs git history (commit d97ab1c^)
- Worker pool with auto-detection (NumCPU * 2, capped)
- Order preservation through indexed results
- Background result collector (deadlock prevention)
- Fail-fast error handling with context cancellation
- Observer hooks for worker events

**Integration:**
- Direct go-agents usage (primary pattern)
- Optional hub coordination
- Composable with sequential chains
- Use State as context naturally

Phase 5 planning should focus on worker pool architecture and result ordering strategy.
