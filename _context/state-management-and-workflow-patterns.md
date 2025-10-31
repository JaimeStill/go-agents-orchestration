# Phase 2 & 3: State Management and Workflow Patterns

## Overview

This document consolidates Phase 2 (State Management) and Phase 3 (Workflow Patterns) into a unified implementation plan with parallel execution tracks. This approach accelerates delivery while maintaining architectural integrity.

### Why Consolidate These Phases?

**Pattern Independence**: Sequential chains and parallel execution don't require state graphs to be valuable. They can be implemented immediately using Phase 1 foundations (hub + messaging).

**Faster Value Delivery**: Users gain workflow capabilities sooner while state graph infrastructure develops in parallel.

**Design Validation**: Real-world pattern usage informs state graph design through actual implementation feedback.

**Natural Integration**: Patterns and state graphs integrate organically without forced coupling:
- Patterns work standalone
- State graphs can use patterns as node implementations
- Patterns can optionally use state graphs for complex routing
- Composition happens naturally through interfaces

### Implementation Strategy

Two parallel tracks with clear integration points:

```
Track A: State Management (Sequential dependency chain)
├── 1. Core state structures (State, StateNode, Edge)
├── 2. Graph execution engine
├── 3. Checkpointing infrastructure
└── 4. Stateful workflows (requires tracks A + B)

Track B: Workflow Patterns (Independent implementations)
├── 1. Sequential chains (extract from classify-docs) ← START HERE
├── 2. Parallel execution (hub coordination)
└── 3. Conditional routing (optional state integration)

Integration: Track B patterns compose with Track A state graphs
```

## Track B: Workflow Patterns

Track B patterns provide immediate value and can be implemented using Phase 1 foundations.

### Pattern Independence

**Core Principle**: All workflow patterns are agnostic about processing approach:
- ✅ **Direct go-agents usage** (simpler default - like classify-docs)
- ✅ **Hub orchestration** (optional for multi-agent coordination)
- ✅ **Pure data transformation** (no agents required)
- ✅ **Mixed approaches** (some steps with agents, some without)

The processor function signature (`func(ctx, item, state) (state, error)`) intentionally doesn't constrain implementation. This enables:
- Single agent calling methods directly (Vision, Chat, Tools, Embeddings)
- Multiple agents without hub coordination
- Hub-based multi-agent orchestration (when needed)
- Non-agent processing (pure computation, data fetching, etc.)

**Example progression:**
```go
// Simple: Direct agent call
processor := func(ctx, item, state) {
    response, err := agent.Chat(ctx, item)
    return updateState(state, response), err
}

// Complex: Hub orchestration (only when needed)
processor := func(ctx, item, state) {
    hub.Broadcast(ctx, "coordinator", item)
    responses := collectFromAgents(ctx, hub, agentIDs)
    return aggregateIntoState(state, responses), nil
}
```

### Pattern: Sequential Chains

**Purpose**: Linear workflow with state accumulation across steps.

**Source**: Extract and generalize from `github.com/JaimeStill/go-agents/tools/classify-docs/pkg/processing/sequential.go`

**Primary Usage**: Direct go-agents calls without hub coordination. Hub integration is optional for advanced multi-agent orchestration.

#### Core Concept

```
Initial State → Item 1 → State v1 → Item 2 → State v2 → ... → Final State
```

Each step receives current state, processes an item, and returns updated state. The pattern handles orchestration, error propagation, and progress tracking.

#### API Design

**Primary Interface (Generic Over Item Type):**

```go
// pkg/patterns/chain.go

// StepProcessor processes a single item and updates the context state.
// 
// Parameters:
//   - ctx: Cancellation and timeout control
//   - item: The current item to process
//   - current: The accumulated state from previous steps
//
// Returns:
//   - Updated state after processing this item
//   - Error if processing fails (stops the chain)
type StepProcessor[TItem, TContext any] func(
    ctx context.Context,
    item TItem,
    current TContext,
) (TContext, error)

// ProgressFunc provides visibility into chain execution.
// Called after each successful step.
type ProgressFunc[TContext any] func(
    completed int,    // Steps completed so far
    total int,        // Total steps in chain
    current TContext, // Current state snapshot
)

// ProcessChain executes a sequential chain with state accumulation.
//
// The chain processes items in order, passing accumulated state between steps.
// Each step's processor receives the current item and state, returning updated state.
// Processing stops on first error (fail-fast).
//
// Example - Document processing:
//   result, err := ProcessChain(ctx, cfg, pages, initialPrompt, processPage, progress)
//
// Example - Agent conversation:
//   result, err := ProcessChain(ctx, cfg, questions, conversation, askAgent, progress)
//
// Example - Data transformation:
//   result, err := ProcessChain(ctx, cfg, records, summary, transformRecord, progress)
func ProcessChain[TItem, TContext any](
    ctx context.Context,
    cfg ChainConfig,
    items []TItem,
    initial TContext,
    processor StepProcessor[TItem, TContext],
    progress ProgressFunc[TContext],
) (ChainResult[TContext], error)
```

**Supporting Types:**

```go
// ChainConfig controls chain execution behavior.
type ChainConfig struct {
    // ExposeIntermediateStates captures state after each step.
    // Useful for debugging or auditing state evolution.
    // Default: false (only final state captured)
    ExposeIntermediateStates bool
    
    // FailFast stops processing on first error.
    // When false, continues processing and collects all errors.
    // Default: true
    FailFast bool
}

// ChainResult contains chain execution results.
type ChainResult[TContext any] struct {
    // Final state after all steps completed
    Final TContext
    
    // Intermediate states (if ExposeIntermediateStates = true)
    // Index 0 = initial state, Index N = state after step N
    Intermediate []TContext
    
    // Number of steps successfully completed
    Steps int
}

// DefaultChainConfig returns sensible defaults.
func DefaultChainConfig() ChainConfig {
    return ChainConfig{
        ExposeIntermediateStates: false,
        FailFast:                 true,
    }
}
```

#### Implementation Steps

**Step 1: Extract core logic from classify-docs**

File: `github.com/JaimeStill/go-agents/tools/classify-docs/pkg/processing/sequential.go`

Extract the generic pattern, removing document-specific dependencies:
- Replace `document.Page` with generic `TItem` type parameter
- Remove `config.SequentialConfig` dependency (use new `ChainConfig`)
- Keep the clean state accumulation logic intact
- Preserve error handling and progress reporting

**Step 2: Create patterns package**

```
pkg/patterns/
├── chain.go          # Sequential chain implementation
├── chain_test.go     # Comprehensive tests
└── doc.go            # Package documentation
```

**Step 3: Implement configuration**

Simplified configuration focused on chain behavior:
- `ExposeIntermediateStates`: Capture all intermediate states for debugging
- `FailFast`: Stop on first error vs. collect all errors

**Step 4: Add comprehensive tests**

Test scenarios:
- Basic state accumulation (3-5 steps)
- Empty chain (0 items)
- Single item chain
- Error handling (fail-fast mode)
- Context cancellation mid-chain
- Progress callback invocation
- Intermediate state capture
- Large chains (1000+ items for performance)

**Step 5: Create integration examples**

Examples demonstrating different use cases:
- Document processing (original classify-docs use case)
- Agent conversation chains
- Data transformation pipelines
- Hub messaging chains

#### Direct go-agents Usage (No Hub Required)

The sequential chain pattern works directly with go-agents without requiring hub coordination. This is the **simpler default approach** demonstrated in classify-docs.

**Pattern 1: Single Agent, Direct Calls**
```go
// Create agent using go-agents
agentConfig, _ := agentconfig.LoadAgentConfig("config.json")
agent, _ := agent.New(agentConfig)

// Sequential chain calling agent methods directly
questions := []string{
    "What is the main topic of this document?",
    "What are the key findings?",
    "What are the conclusions?",
}

processor := func(ctx context.Context, question string, conversation Conversation) (Conversation, error) {
    // Direct agent.Chat call (no hub)
    response, err := agent.Chat(ctx, question)
    if err != nil {
        return conversation, err
    }
    
    // Accumulate conversation state
    conversation.AddExchange(question, response.Content())
    return conversation, nil
}

result, err := patterns.ProcessChain(ctx, cfg, questions, initialConversation, processor, nil)
finalConversation := result.Final
```

**Pattern 2: Vision Agent Processing**
```go
// Process document pages sequentially with vision agent
visionAgent, _ := agent.New(visionConfig)
pages := []document.Page{page1, page2, page3}

processor := func(ctx context.Context, page document.Page, prompt string) (string, error) {
    // Render page to image
    imageData, err := page.ToImage(document.DefaultImageOptions())
    if err != nil {
        return prompt, err
    }
    
    // Encode for vision API
    encoded, err := encoding.EncodeImageDataURI(imageData, document.PNG)
    if err != nil {
        return prompt, err
    }
    
    // Direct agent.Vision call (no hub)
    response, err := visionAgent.Vision(ctx, "Analyze this page and update the prompt", []string{encoded})
    if err != nil {
        return prompt, err
    }
    
    // Return accumulated prompt
    return response.Content(), nil
}

result, err := patterns.ProcessChain(ctx, cfg, pages, initialPrompt, processor, nil)
finalPrompt := result.Final
```

**Pattern 3: Multiple Agents, Direct Calls**
```go
// Different agent per step, no hub required
type AnalysisStep struct {
    Name  string
    Agent agent.Agent
}

steps := []AnalysisStep{
    {Name: "technical", Agent: technicalAgent},
    {Name: "business", Agent: businessAgent},
    {Name: "legal", Agent: legalAgent},
}

processor := func(ctx context.Context, step AnalysisStep, report Report) (Report, error) {
    // Direct agent call for this analysis type
    response, err := step.Agent.Chat(ctx, fmt.Sprintf("Analyze: %v", report))
    if err != nil {
        return report, err
    }
    
    // Update report with this analysis
    report.AddAnalysis(step.Name, response.Content())
    return report, nil
}

result, err := patterns.ProcessChain(ctx, cfg, steps, initialReport, processor, nil)
```

**Pattern 4: Pure Data Transformation (No Agents)**
```go
// Sequential chain doesn't require agents at all
records := []Record{rec1, rec2, rec3, rec4}

processor := func(ctx context.Context, record Record, summary Summary) (Summary, error) {
    // Pure computation, no LLM calls
    summary.Count++
    summary.Total += record.Value
    summary.Items = append(summary.Items, record.ID)
    return summary, nil
}

result, err := patterns.ProcessChain(ctx, cfg, records, Summary{}, processor, nil)
```

#### Hub Integration Patterns (Optional Advanced Usage)

When you need **multi-agent coordination** or **cross-hub communication**, sequential chains can integrate with hub. This is optional - use only when orchestration complexity requires it.

**Pattern 1: Single Agent Per Step via Hub**
```go
// Each step sends work to one agent through hub
// Useful when agents are registered in hub and you need message routing
agentIDs := []string{"analyzer", "reviewer", "approver"}

processor := func(ctx context.Context, agentID string, report Report) (Report, error) {
    // Hub request instead of direct agent call
    response, err := hub.Request(ctx, "coordinator", agentID, report)
    if err != nil {
        return report, err
    }
    return response.Data.(Report), nil
}

result, err := patterns.ProcessChain(ctx, cfg, agentIDs, initialReport, processor, nil)
```

**Pattern 2: Multi-Agent Orchestration Per Step**
```go
// Each step orchestrates MULTIPLE agents through hub
// This is where hub provides real value - coordination within a step
steps := []string{"analysis", "review", "approval"}

processor := func(ctx context.Context, stepName string, report Report) (Report, error) {
    switch stepName {
    case "analysis":
        // Broadcast to analysis team (multiple agents)
        hub.Broadcast(ctx, "coordinator", report)
        
        // Collect and aggregate responses from multiple analysts
        responses := collectFromAnalysts(ctx, hub, []string{"analyst-1", "analyst-2", "analyst-3"})
        report.Analysis = aggregateAnalysis(responses)
        return report, nil
        
    case "review":
        // Parallel review by multiple reviewers through hub
        reviews := parallelReview(ctx, hub, report, []string{"reviewer-1", "reviewer-2"})
        report.Reviews = reviews
        return report, nil
        
    case "approval":
        // Single approver through hub
        response, err := hub.Request(ctx, "coordinator", "approver", report)
        if err != nil {
            return report, err
        }
        report.Approved = response.Data.(bool)
        return report, nil
    }
    return report, nil
}

result, err := patterns.ProcessChain(ctx, cfg, steps, initialReport, processor, nil)
```

**Pattern 3: Cross-Hub Coordination**
```go
// Coordinate agents across multiple hubs
type Step struct {
    Name string
    Hub  hub.Hub
    Agents []string
}

steps := []Step{
    {Name: "internal", Hub: internalHub, Agents: []string{"internal-1", "internal-2"}},
    {Name: "external", Hub: externalHub, Agents: []string{"external-1", "external-2"}},
}

processor := func(ctx context.Context, step Step, state State) (State, error) {
    // Coordinate agents in specific hub
    results := make([]Result, len(step.Agents))
    for i, agentID := range step.Agents {
        response, err := step.Hub.Request(ctx, "coordinator", agentID, state)
        if err != nil {
            return state, err
        }
        results[i] = response.Data.(Result)
    }
    
    // Aggregate results from this hub's agents
    state[step.Name] = aggregateResults(results)
    return state, nil
}

result, err := patterns.ProcessChain(ctx, cfg, steps, initialState, processor, nil)
```

**When to Use Hub Integration:**

Use direct go-agents calls when:
- ✅ Single agent processes each step
- ✅ No message routing complexity
- ✅ Simple sequential processing
- ✅ Classify-docs style workflows

Use hub integration when:
- ✅ Multiple agents coordinate within a step
- ✅ Cross-hub communication needed
- ✅ Dynamic agent selection/routing
- ✅ Pub/sub or broadcast patterns required

#### Success Criteria

**Implementation Complete When:**
- ✅ Core `ProcessChain` function implemented with generic types
- ✅ Configuration supports intermediate state capture and fail-fast modes
- ✅ Comprehensive tests achieve 80%+ coverage
- ✅ Integration examples demonstrate hub patterns
- ✅ Documentation covers all usage patterns
- ✅ Performance validated (1000+ item chains complete in reasonable time)

**Quality Validation:**
- ✅ Black-box tests only (no internal implementation access)
- ✅ Table-driven tests for multiple scenarios
- ✅ Error handling covers all failure modes
- ✅ Context cancellation works correctly
- ✅ Progress callbacks invoked at correct times

---

### Pattern: Parallel Execution

**Purpose**: Concurrent processing with result aggregation and order preservation.

**Inspiration**: Removed parallel processor from classify-docs (commit d97ab1c^)

**Primary Usage**: Direct go-agents calls for concurrent processing. Hub integration is optional for hub-based agent coordination.

#### Core Concept

```
        ┌─────────┐
Items → │ Worker  │ → Results
        │  Pool   │    (ordered)
        └─────────┘
```

Fan-out work to worker pool, collect results, preserve original order despite concurrent execution.

#### API Design

```go
// pkg/patterns/parallel.go

// ItemProcessor processes a single item concurrently.
// Multiple processors run in parallel worker pool.
type ItemProcessor[TItem, TResult any] func(
    ctx context.Context,
    item TItem,
) (TResult, error)

// ProcessParallel executes items concurrently with result aggregation.
//
// Items are distributed to worker pool and processed concurrently.
// Results are collected and returned in original item order.
// Processing stops on first error (fail-fast).
//
// Example - Parallel agent requests:
//   results, err := ProcessParallel(ctx, cfg, requests, processRequest, progress)
//
// Example - Concurrent data fetching:
//   data, err := ProcessParallel(ctx, cfg, urls, fetchURL, progress)
func ProcessParallel[TItem, TResult any](
    ctx context.Context,
    cfg ParallelConfig,
    items []TItem,
    processor ItemProcessor[TItem, TResult],
    progress ProgressFunc[int], // Reports completed count
) ([]TResult, error)
```

**Supporting Types:**

```go
// ParallelConfig controls parallel execution behavior.
type ParallelConfig struct {
    // MaxWorkers sets worker pool size.
    // 0 = auto-detect (min(runtime.NumCPU()*2, WorkerCap, len(items)))
    // Default: 0 (auto-detect)
    MaxWorkers int
    
    // WorkerCap limits auto-detected worker count.
    // Prevents excessive goroutines for large item sets.
    // Default: 16
    WorkerCap int
    
    // FailFast stops all workers on first error.
    // When false, all items processed and all errors collected.
    // Default: true
    FailFast bool
}

// DefaultParallelConfig returns sensible defaults.
func DefaultParallelConfig() ParallelConfig {
    return ParallelConfig{
        MaxWorkers: 0,  // Auto-detect
        WorkerCap:  16,
        FailFast:   true,
    }
}
```

#### Architecture

**Three-Channel Pattern:**

```go
// Work distribution
workQueue := make(chan indexedItem[TItem], len(items))

// Result collection
resultChannel := make(chan indexedResult[TResult], len(items))

// Completion signal
done := make(chan struct{})
```

**Goroutine Structure:**

```
Main Goroutine
├── Work Distributor (goroutine)
│   └── Sends indexed items to workQueue
├── Worker Pool (N goroutines)
│   ├── Worker 1: Reads workQueue → Processes → Sends resultChannel
│   ├── Worker 2: Reads workQueue → Processes → Sends resultChannel
│   └── Worker N: Reads workQueue → Processes → Sends resultChannel
└── Result Collector (background goroutine)
    └── Reads resultChannel → Builds ordered results → Signals done
```

**Key Patterns:**
- Background result collector prevents deadlocks
- Indexed results enable order preservation
- Context cancellation stops all workers immediately
- Fail-fast cancels context on first error

#### Implementation Steps

**Step 1: Port architecture from classify-docs**

Recover implementation from git history (commit d97ab1c^):
```bash
git show d97ab1c^:tools/classify-docs/pkg/processing/parallel.go
```

Key components to extract:
- Worker pool management
- Three-channel pattern (work, results, done)
- Background result collector
- Order preservation through indexed results
- Error handling with fail-fast

**Step 2: Remove document-specific dependencies**

Replace document.Page with generic TItem/TResult types while preserving:
- Worker pool auto-detection logic
- Deadlock prevention patterns
- Context coordination
- Order preservation

**Step 3: Simplify configuration**

Focus on parallel-specific concerns:
- Worker pool sizing (auto-detect with cap)
- Fail-fast behavior
- Remove retry logic (belongs in processor implementation)

**Step 4: Add comprehensive tests**

Test scenarios:
- Basic parallel processing (10-20 items, 4 workers)
- Worker count auto-detection
- Order preservation despite concurrent execution
- Error handling (fail-fast mode)
- Context cancellation (stops all workers)
- Empty input (0 items)
- Single item (degenerates gracefully)
- Large item sets (1000+ items)
- Progress callback invocation
- Deadlock prevention (verify background collector)

**Step 5: Create hub integration examples**

Demonstrate hub coordination patterns:
- Parallel agent requests
- Broadcast with parallel responses
- Concurrent pub/sub message distribution

#### Direct go-agents Usage (No Hub Required)

Parallel execution works directly with go-agents for concurrent agent processing without hub coordination.

**Pattern 1: Parallel Agent Calls (Single Agent)**
```go
// Process multiple requests concurrently with single agent
agent, _ := agent.New(agentConfig)
questions := []string{
    "What is machine learning?",
    "What is deep learning?",
    "What is reinforcement learning?",
    "What is supervised learning?",
}

processor := func(ctx context.Context, question string) (Answer, error) {
    // Direct agent.Chat call (no hub)
    response, err := agent.Chat(ctx, question)
    if err != nil {
        return Answer{}, err
    }
    
    return Answer{
        Question: question,
        Response: response.Content(),
    }, nil
}

// Process all questions concurrently
answers, err := patterns.ProcessParallel(ctx, cfg, questions, processor, nil)
```

**Pattern 2: Concurrent Data Fetching/Processing**
```go
// Parallel processing without agents at all
urls := []string{
    "https://api.example.com/data1",
    "https://api.example.com/data2",
    "https://api.example.com/data3",
}

processor := func(ctx context.Context, url string) (Data, error) {
    // Pure HTTP fetch, no LLM calls
    resp, err := http.Get(url)
    if err != nil {
        return Data{}, err
    }
    defer resp.Body.Close()
    
    var data Data
    json.NewDecoder(resp.Body).Decode(&data)
    return data, nil
}

// Fetch all URLs concurrently
results, err := patterns.ProcessParallel(ctx, cfg, urls, processor, nil)
```

**Pattern 3: Multiple Agents, Direct Calls**
```go
// Different agent per item, no hub coordination
type AnalysisTask struct {
    Document string
    Agent    agent.Agent
}

tasks := []AnalysisTask{
    {Document: doc1, Agent: technicalAgent},
    {Document: doc2, Agent: businessAgent},
    {Document: doc3, Agent: legalAgent},
    {Document: doc4, Agent: technicalAgent},
}

processor := func(ctx context.Context, task AnalysisTask) (Analysis, error) {
    // Direct agent call
    response, err := task.Agent.Chat(ctx, task.Document)
    if err != nil {
        return Analysis{}, err
    }
    
    return Analysis{
        Document: task.Document,
        Result:   response.Content(),
    }, nil
}

// Process all tasks concurrently
analyses, err := patterns.ProcessParallel(ctx, cfg, tasks, processor, nil)
```

**Pattern 4: Concurrent Vision Processing**
```go
// Process multiple images with vision agent concurrently
visionAgent, _ := agent.New(visionConfig)
images := []ImageData{img1, img2, img3, img4}

processor := func(ctx context.Context, img ImageData) (Description, error) {
    // Encode image
    encoded, err := encodeImage(img)
    if err != nil {
        return Description{}, err
    }
    
    // Direct agent.Vision call
    response, err := visionAgent.Vision(ctx, "Describe this image", []string{encoded})
    if err != nil {
        return Description{}, err
    }
    
    return Description{
        ImageID: img.ID,
        Text:    response.Content(),
    }, nil
}

// Process all images concurrently
descriptions, err := patterns.ProcessParallel(ctx, cfg, images, processor, nil)
```

#### Hub Integration Patterns (Optional Advanced Usage)

When you need **multi-agent coordination** or **message routing**, parallel execution can integrate with hub. This is optional - use only when orchestration complexity requires it.

**Pattern 1: Parallel Agent Requests via Hub**
```go
// Send requests to multiple agents concurrently through hub
// Useful when agents are registered in hub and you need message routing
agentIDs := []string{"analyzer-1", "analyzer-2", "analyzer-3"}

processor := func(ctx context.Context, agentID string) (Analysis, error) {
    // Hub request instead of direct agent call
    response, err := hub.Request(ctx, "coordinator", agentID, task)
    if err != nil {
        return Analysis{}, err
    }
    return response.Data.(Analysis), nil
}

analyses, err := patterns.ProcessParallel(ctx, cfg, agentIDs, processor, nil)

// Aggregate results
finalAnalysis := aggregateAnalyses(analyses)
```

**Pattern 2: Broadcast with Parallel Collection**
```go
// Broadcast to all agents, collect responses in parallel
hub.Broadcast(ctx, "coordinator", announcement)

// Collect responses concurrently
agentIDs := getAllRegisteredAgents(hub)

processor := func(ctx context.Context, agentID string) (Response, error) {
    // Wait for response from this agent (with timeout)
    return waitForResponse(ctx, hub, agentID, 5*time.Second)
}

responses, err := patterns.ProcessParallel(ctx, cfg, agentIDs, processor, nil)
```

**Pattern 3: Parallel Pub/Sub Distribution**
```go
// Publish to multiple topics concurrently through hub
topics := []string{"events.created", "events.updated", "events.deleted"}

processor := func(ctx context.Context, topic string) (PublishResult, error) {
    err := hub.Publish(ctx, "publisher", topic, event)
    if err != nil {
        return PublishResult{Topic: topic, Success: false}, err
    }
    return PublishResult{Topic: topic, Success: true}, nil
}

results, err := patterns.ProcessParallel(ctx, cfg, topics, processor, nil)
```

**When to Use Hub Integration:**

Use direct go-agents calls when:
- ✅ Parallel processing with single agent
- ✅ Concurrent data fetching/transformation
- ✅ Multiple agents without coordination needs
- ✅ Simple concurrent execution

Use hub integration when:
- ✅ Agents registered in hub with message routing
- ✅ Broadcast/collect response patterns
- ✅ Pub/sub message distribution
- ✅ Cross-hub coordination needed

#### Resilience Considerations

**Rate Limiting Challenges:**

The original classify-docs parallel processor was removed due to Azure rate limiting. Future resilience improvements:

**Option 1: Adaptive Worker Scaling**
```go
// Detect rate limit errors, reduce worker count dynamically
type RateLimitError struct { RetryAfter time.Duration }

// In worker loop:
if rateLimitErr, ok := err.(RateLimitError); ok {
    // Signal to reduce worker count
    // Backoff and retry
}
```

**Option 2: Backpressure Mechanisms**
```go
// Token bucket rate limiter
type RateLimiter interface {
    Wait(ctx context.Context) error
}

// Workers acquire token before processing
limiter.Wait(ctx)
result := processor(ctx, item)
```

**Option 3: Provider-Specific Rate Limit Detection**
```go
// Detect provider-specific rate limit responses
func IsRateLimited(err error) (bool, time.Duration) {
    // Check for Azure 429, OpenAI rate limit headers, etc.
}

// Graceful degradation: parallel → sequential on rate limits
```

**Implementation Note**: Initial parallel implementation focuses on architecture. Rate limit resilience added later based on actual production feedback.

#### Success Criteria

**Implementation Complete When:**
- ✅ Core `ProcessParallel` function implemented
- ✅ Worker pool with auto-detection (NumCPU * 2, capped at WorkerCap)
- ✅ Result ordering preserved despite concurrent execution
- ✅ Fail-fast error handling with context cancellation
- ✅ Background result collector prevents deadlocks
- ✅ Comprehensive tests achieve 80%+ coverage
- ✅ Hub integration examples demonstrate patterns
- ✅ Documentation covers architecture and usage

**Quality Validation:**
- ✅ Deadlock-free under all conditions (verified through stress tests)
- ✅ Context cancellation stops all workers immediately
- ✅ Order preservation verified through random processing delays
- ✅ Performance scales with worker count (up to optimal point)
- ✅ Progress callbacks provide accurate completion tracking

---

### Pattern: Conditional Routing

**Purpose**: State-based routing decisions for dynamic workflow execution.

**Primary Usage**: Direct go-agents calls with agent selection based on state. Hub integration is optional for hub-based message routing.

**Integration**: Can use state graph predicates or work independently.

#### Core Concept

```
              ┌─────────┐
              │ Predicate │
Input State → │ Evaluator │ → Route Decision
              └─────────┘
                    ↓
         ┌──────────┼──────────┐
         ▼          ▼          ▼
    Handler A   Handler B   Handler C
```

Evaluate predicates against state, select handler, execute selected path.

#### API Design

```go
// pkg/patterns/conditional.go

// RoutePredicate evaluates state and returns route decision.
type RoutePredicate[TState any] func(state TState) (route string, err error)

// RouteHandler processes state for a specific route.
type RouteHandler[TState any] func(
    ctx context.Context,
    state TState,
) (TState, error)

// RouteConfig maps route names to handlers.
type RouteConfig[TState any] struct {
    Routes map[string]RouteHandler[TState]
    Default RouteHandler[TState] // Fallback if no route matches
}

// ProcessConditional evaluates predicate and executes selected handler.
//
// The predicate evaluates the state and returns a route name.
// The corresponding handler is executed with the state.
// If route not found, default handler is used.
//
// Example - Multi-stage document processing:
//   predicate := func(doc Document) (string, error) {
//       if doc.NeedsReview { return "review", nil }
//       if doc.NeedsApproval { return "approval", nil }
//       return "complete", nil
//   }
//
//   config := RouteConfig{
//       Routes: map[string]RouteHandler{
//           "review": reviewHandler,
//           "approval": approvalHandler,
//           "complete": completeHandler,
//       },
//   }
//
//   result, err := ProcessConditional(ctx, doc, predicate, config)
func ProcessConditional[TState any](
    ctx context.Context,
    state TState,
    predicate RoutePredicate[TState],
    config RouteConfig[TState],
) (TState, error)
```

#### Implementation Steps

**Step 1: Implement basic routing**

Core logic:
1. Evaluate predicate to get route name
2. Lookup handler in route map
3. Execute handler with state
4. Return updated state

**Step 2: Add default handler support**

Fallback mechanism when route not found:
- Prevents errors for unexpected routes
- Enables graceful degradation

**Step 3: Add route validation**

Helper functions:
- Validate route configuration (no duplicate routes)
- Check for circular routing potential
- Verify default handler existence

**Step 4: Create integration examples**

Examples:
- Document approval workflow (review vs. approve vs. reject)
- Agent selection based on task type
- Hub routing based on message priority

#### Direct go-agents Usage (No Hub Required)

Conditional routing works directly with go-agents for state-based processing decisions without hub coordination.

**Pattern 1: Agent Selection Based on Content Type**
```go
// Route to different agents based on document type
technicalAgent, _ := agent.New(technicalConfig)
businessAgent, _ := agent.New(businessConfig)
legalAgent, _ := agent.New(legalConfig)

predicate := func(doc Document) (string, error) {
    switch doc.Type {
    case "technical":
        return "technical", nil
    case "business":
        return "business", nil
    case "legal":
        return "legal", nil
    default:
        return "general", nil
    }
}

config := RouteConfig{
    Routes: map[string]RouteHandler{
        "technical": func(ctx context.Context, doc Document) (Document, error) {
            // Direct agent call
            response, err := technicalAgent.Chat(ctx, doc.Content)
            doc.Analysis = response.Content()
            return doc, err
        },
        "business": func(ctx context.Context, doc Document) (Document, error) {
            response, err := businessAgent.Chat(ctx, doc.Content)
            doc.Analysis = response.Content()
            return doc, err
        },
        "legal": func(ctx context.Context, doc Document) (Document, error) {
            response, err := legalAgent.Chat(ctx, doc.Content)
            doc.Analysis = response.Content()
            return doc, err
        },
    },
    Default: func(ctx context.Context, doc Document) (Document, error) {
        response, err := technicalAgent.Chat(ctx, doc.Content)
        doc.Analysis = response.Content()
        return doc, err
    },
}

result, err := patterns.ProcessConditional(ctx, document, predicate, config)
```

**Pattern 2: Processing Pipeline Selection**
```go
// Choose processing pipeline based on data complexity
predicate := func(data Data) (string, error) {
    if data.Size > 1000000 {
        return "heavy", nil
    } else if data.Size > 10000 {
        return "medium", nil
    }
    return "light", nil
}

config := RouteConfig{
    Routes: map[string]RouteHandler{
        "heavy": func(ctx context.Context, data Data) (Data, error) {
            // Complex processing pipeline (no agents)
            return processHeavyWorkload(ctx, data)
        },
        "medium": func(ctx context.Context, data Data) (Data, error) {
            return processMediumWorkload(ctx, data)
        },
        "light": func(ctx context.Context, data Data) (Data, error) {
            return processLightWorkload(ctx, data)
        },
    },
}

result, err := patterns.ProcessConditional(ctx, data, predicate, config)
```

**Pattern 3: Capability-Based Agent Selection**
```go
// Route based on required agent capabilities
visionAgent, _ := agent.New(visionConfig)
chatAgent, _ := agent.New(chatConfig)
toolAgent, _ := agent.New(toolConfig)

predicate := func(task Task) (string, error) {
    if task.RequiresVision {
        return "vision", nil
    } else if task.RequiresTools {
        return "tools", nil
    }
    return "chat", nil
}

config := RouteConfig{
    Routes: map[string]RouteHandler{
        "vision": func(ctx context.Context, task Task) (Task, error) {
            response, err := visionAgent.Vision(ctx, task.Prompt, task.Images)
            task.Result = response.Content()
            return task, err
        },
        "tools": func(ctx context.Context, task Task) (Task, error) {
            response, err := toolAgent.Chat(ctx, task.Prompt) // Tools invoked automatically
            task.Result = response.Content()
            return task, err
        },
        "chat": func(ctx context.Context, task Task) (Task, error) {
            response, err := chatAgent.Chat(ctx, task.Prompt)
            task.Result = response.Content()
            return task, err
        },
    },
}

result, err := patterns.ProcessConditional(ctx, task, predicate, config)
```

**Pattern 4: Confidence-Based Routing**
```go
// Route based on state/confidence thresholds
simpleAgent, _ := agent.New(simpleConfig)
complexAgent, _ := agent.New(complexConfig)

predicate := func(state ProcessingState) (string, error) {
    if state.Confidence < 0.5 {
        return "complex-analysis", nil
    }
    return "simple-analysis", nil
}

config := RouteConfig{
    Routes: map[string]RouteHandler{
        "complex-analysis": func(ctx context.Context, state ProcessingState) (ProcessingState, error) {
            // Use more sophisticated agent
            response, err := complexAgent.Chat(ctx, state.Description)
            state.Analysis = response.Content()
            state.Confidence = calculateConfidence(response)
            return state, err
        },
        "simple-analysis": func(ctx context.Context, state ProcessingState) (ProcessingState, error) {
            response, err := simpleAgent.Chat(ctx, state.Description)
            state.Analysis = response.Content()
            return state, err
        },
    },
}

result, err := patterns.ProcessConditional(ctx, state, predicate, config)
```

#### Hub Integration Patterns (Optional Advanced Usage)

When you need **hub-based message routing** or **dynamic agent selection from hub registrations**, conditional routing can integrate with hub. This is optional.

**Pattern 1: Agent Selection Based on Task Type via Hub**
```go
predicate := func(task Task) (string, error) {
    switch task.Type {
    case "analysis":
        return "analyzer", nil
    case "review":
        return "reviewer", nil
    case "approval":
        return "approver", nil
    default:
        return "general", nil
    }
}

config := RouteConfig{
    Routes: map[string]RouteHandler{
        "analyzer": func(ctx context.Context, task Task) (Task, error) {
            // Hub request instead of direct agent call
            response, err := hub.Request(ctx, "coordinator", "analysis-agent", task)
            return response.Data.(Task), err
        },
        "reviewer": func(ctx context.Context, task Task) (Task, error) {
            response, err := hub.Request(ctx, "coordinator", "review-agent", task)
            return response.Data.(Task), err
        },
        // ... other handlers
    },
}

result, err := patterns.ProcessConditional(ctx, task, predicate, config)
```

**Pattern 2: Priority-Based Hub Routing**
```go
predicate := func(msg Message) (string, error) {
    if msg.Priority == PriorityCritical {
        return "urgent-hub", nil
    }
    return "normal-hub", nil
}

config := RouteConfig{
    Routes: map[string]RouteHandler{
        "urgent-hub": func(ctx context.Context, msg Message) (Message, error) {
            return processInUrgentHub(ctx, urgentHub, msg)
        },
        "normal-hub": func(ctx context.Context, msg Message) (Message, error) {
            return processInNormalHub(ctx, normalHub, msg)
        },
    },
}

result, err := patterns.ProcessConditional(ctx, message, predicate, config)
```

**When to Use Hub Integration:**

Use direct go-agents calls when:
- ✅ Agent selection based on task/content type
- ✅ Routing to different agent capabilities
- ✅ Simple conditional processing logic
- ✅ Single agent per route

Use hub integration when:
- ✅ Agents registered in hub with IDs
- ✅ Dynamic agent lookup from hub registry
- ✅ Cross-hub routing decisions
- ✅ Message routing patterns required

#### Integration with State Graphs (Future)

Conditional routing can use state graph predicates:

```go
// When Track A (state graphs) is complete:

// Use state graph transition predicates for routing
predicate := func(state State) (string, error) {
    // Leverage state graph predicate infrastructure
    return evaluateTransition(state)
}

// Or: Conditional routing becomes a state graph node
node := NewConditionalNode(predicate, config)
stateGraph.AddNode("conditional-step", node)
```

#### Success Criteria

**Implementation Complete When:**
- ✅ Core `ProcessConditional` function implemented
- ✅ Route configuration supports maps and default handlers
- ✅ Comprehensive tests achieve 80%+ coverage
- ✅ Hub integration examples demonstrate agent selection patterns
- ✅ Documentation covers routing strategies

**Quality Validation:**
- ✅ Route lookup is O(1) (map-based)
- ✅ Default handler fallback works correctly
- ✅ Error handling covers missing routes
- ✅ Predicate errors propagate correctly

---

## Track A: State Management

Track A establishes the foundation for stateful workflow orchestration through LangGraph-inspired state graphs adapted for Go.

### Architecture Overview

**Core Concepts:**

- **StateGraph**: Workflow definition as directed graph of nodes and edges
- **StateNode**: Computation step that transforms state
- **Edge**: Transition between nodes with optional predicates
- **State**: Data structure flowing through the graph (map[string]any)
- **Executor**: Engine that executes state graph traversal
- **Checkpoint**: State snapshot for recovery and rollback

**Design Principles:**

1. **Immutable State**: State transformations create new state (or copy-on-write)
2. **Explicit Transitions**: Edges define valid state graph paths
3. **Predicate-Based Routing**: Conditional edges enable dynamic paths
4. **Cycle Support**: Loops enabled through explicit edge definitions
5. **Checkpoint Recovery**: Save/restore state at any graph position

### Core Interfaces

```go
// pkg/state/graph.go

// State represents data flowing through the graph.
// Keys are string identifiers, values are any type.
type State map[string]any

// Clone creates a deep copy of the state.
func (s State) Clone() State

// Get retrieves a value from state with type assertion.
func (s State) Get(key string) (any, bool)

// Set updates or adds a key-value pair (returns new state).
func (s State) Set(key string, value any) State

// Merge combines another state into this state (returns new state).
func (s State) Merge(other State) State
```

```go
// StateNode represents a computation step in the graph.
type StateNode interface {
    // Execute transforms state based on node logic.
    // Returns updated state or error.
    Execute(ctx context.Context, state State) (State, error)
}

// StateGraph defines a workflow as a graph of nodes and edges.
type StateGraph interface {
    // AddNode registers a node in the graph.
    AddNode(name string, node StateNode) error
    
    // AddEdge creates a transition between nodes.
    // Predicate is optional (nil = always transition).
    AddEdge(from, to string, predicate TransitionPredicate) error
    
    // SetEntryPoint defines the starting node.
    SetEntryPoint(node string) error
    
    // SetExitPoint defines a terminal node (optional).
    SetExitPoint(node string) error
    
    // Execute runs the graph from entry point with initial state.
    Execute(ctx context.Context, initialState State) (State, error)
}

// TransitionPredicate determines if an edge can be traversed.
// Returns true if transition should occur, false otherwise.
type TransitionPredicate func(state State) bool
```

### Graph Execution Engine

**Execution Algorithm:**

```
1. Start at entry point node
2. Execute node with current state
3. Evaluate outgoing edges:
   - If single edge: follow it
   - If multiple edges: evaluate predicates, follow first true
   - If no valid edges: check if exit point, otherwise error
4. Repeat from step 2 with next node
5. Return final state when exit point reached
```

**Cycle Detection:**

- Track visited nodes per execution path
- Detect cycles by comparing current node to path history
- Allow cycles (loops) but prevent infinite loops through max iteration count
- Configuration: `MaxIterations` (default: 1000)

**Error Handling:**

- Node execution errors stop graph execution immediately
- Predicate evaluation errors stop execution
- Invalid transitions (edge not found) stop execution
- Timeout via context cancellation

### Implementation Steps

**Phase 2.1: Core State Structures**

```go
// pkg/state/state.go

// State implementation with immutable operations
type State map[string]any

func (s State) Clone() State {
    // Deep copy implementation
}

func (s State) Get(key string) (any, bool) {
    // Safe retrieval with exists check
}

func (s State) Set(key string, value any) State {
    // Copy-on-write update
}

func (s State) Merge(other State) State {
    // Combine states, other takes precedence
}
```

```go
// pkg/state/node.go

// StateNode interface (defined above)

// FunctionNode wraps a function as a StateNode
type FunctionNode struct {
    fn func(ctx context.Context, state State) (State, error)
}

func (n *FunctionNode) Execute(ctx context.Context, state State) (State, error) {
    return n.fn(ctx, state)
}

// Helper to create function nodes
func NewFunctionNode(fn func(context.Context, State) (State, error)) StateNode {
    return &FunctionNode{fn: fn}
}
```

```go
// pkg/state/edge.go

// Edge represents a transition between nodes
type Edge struct {
    From      string
    To        string
    Predicate TransitionPredicate // nil = always transition
}

// TransitionPredicate (defined above)

// Helper predicates for common patterns
func AlwaysTransition() TransitionPredicate {
    return func(state State) bool { return true }
}

func KeyExists(key string) TransitionPredicate {
    return func(state State) bool {
        _, exists := state.Get(key)
        return exists
    }
}

func KeyEquals(key string, value any) TransitionPredicate {
    return func(state State) bool {
        val, exists := state.Get(key)
        return exists && val == value
    }
}
```

**Tests (Phase 2.1):**
- State immutability (Clone, Set, Merge don't modify original)
- State operations (Get, Set, Merge) work correctly
- Function node execution
- Edge predicate evaluation
- Predicate helper functions

**Phase 2.2: Graph Executor**

```go
// pkg/state/graph.go

// StateGraph implementation
type stateGraph struct {
    nodes      map[string]StateNode
    edges      map[string][]Edge // from → []edges
    entryPoint string
    exitPoints map[string]bool
    
    maxIterations int
}

// New creates a StateGraph with default configuration
func New() StateGraph {
    return &stateGraph{
        nodes:         make(map[string]StateNode),
        edges:         make(map[string][]Edge),
        exitPoints:    make(map[string]bool),
        maxIterations: 1000,
    }
}

// AddNode, AddEdge, SetEntryPoint, SetExitPoint implementations

func (g *stateGraph) Execute(ctx context.Context, initialState State) (State, error) {
    if g.entryPoint == "" {
        return nil, fmt.Errorf("no entry point defined")
    }
    
    current := g.entryPoint
    state := initialState
    iterations := 0
    visited := make(map[string]int) // Track cycles
    
    for {
        // Check context cancellation
        if err := ctx.Err(); err != nil {
            return state, fmt.Errorf("execution cancelled: %w", err)
        }
        
        // Check iteration limit
        iterations++
        if iterations > g.maxIterations {
            return state, fmt.Errorf("max iterations (%d) exceeded", g.maxIterations)
        }
        
        // Track visited count for cycle detection
        visited[current]++
        
        // Get current node
        node, exists := g.nodes[current]
        if !exists {
            return state, fmt.Errorf("node not found: %s", current)
        }
        
        // Execute node
        newState, err := node.Execute(ctx, state)
        if err != nil {
            return state, fmt.Errorf("node %s failed: %w", current, err)
        }
        state = newState
        
        // Check if exit point
        if g.exitPoints[current] {
            return state, nil
        }
        
        // Find next transition
        edges, hasEdges := g.edges[current]
        if !hasEdges {
            return state, fmt.Errorf("node %s has no outgoing edges", current)
        }
        
        // Evaluate predicates to find next node
        nextNode := ""
        for _, edge := range edges {
            if edge.Predicate == nil || edge.Predicate(state) {
                nextNode = edge.To
                break
            }
        }
        
        if nextNode == "" {
            return state, fmt.Errorf("no valid transition from node %s", current)
        }
        
        current = nextNode
    }
}
```

**Tests (Phase 2.2):**
- Linear graph execution (A → B → C)
- Conditional routing (A → [predicate] → B or C)
- Cycle execution (A → B → C → B → exit)
- Max iterations protection
- Context cancellation
- Error propagation from nodes
- Exit point detection
- Missing node errors
- Missing edge errors

**Phase 2.3: Checkpointing**

```go
// pkg/state/checkpoint.go

// Checkpoint captures execution state for recovery
type Checkpoint struct {
    NodeName  string    // Current node position
    State     State     // State at this position
    Timestamp time.Time // When checkpoint was created
}

// CheckpointStore persists checkpoints
type CheckpointStore interface {
    Save(id string, checkpoint Checkpoint) error
    Load(id string) (Checkpoint, error)
    Delete(id string) error
    List() ([]string, error)
}

// MemoryCheckpointStore provides in-memory storage
type MemoryCheckpointStore struct {
    checkpoints map[string]Checkpoint
    mu          sync.RWMutex
}

func NewMemoryCheckpointStore() CheckpointStore {
    return &MemoryCheckpointStore{
        checkpoints: make(map[string]Checkpoint),
    }
}

// Save, Load, Delete, List implementations

// Add checkpointing to graph execution
type CheckpointConfig struct {
    Enabled  bool
    Store    CheckpointStore
    Interval int // Checkpoint every N nodes (0 = every node)
}

// Modify Execute to support checkpointing:
func (g *stateGraph) ExecuteWithCheckpoints(
    ctx context.Context,
    initialState State,
    checkpointID string,
    config CheckpointConfig,
) (State, error) {
    // Implementation that saves checkpoints at intervals
    // Enables recovery from checkpoint if needed
}

// Recovery helper
func (g *stateGraph) Resume(
    ctx context.Context,
    checkpointID string,
    config CheckpointConfig,
) (State, error) {
    checkpoint, err := config.Store.Load(checkpointID)
    if err != nil {
        return nil, fmt.Errorf("failed to load checkpoint: %w", err)
    }
    
    // Resume execution from checkpoint node
    return g.ExecuteFrom(ctx, checkpoint.NodeName, checkpoint.State, config)
}
```

**Tests (Phase 2.3):**
- Checkpoint save/load
- Resume from checkpoint
- Checkpoint interval configuration
- Memory checkpoint store operations
- Multiple checkpoint management

**Phase 2.4: Agent Integration (Both Direct and Hub-Based)**

State graph nodes can use agents in two ways:

**Direct go-agents Usage (Primary Pattern):**

```go
// Simple node with direct agent call
type DirectAgentNode struct {
    agent agent.Agent
}

func (n *DirectAgentNode) Execute(ctx context.Context, state State) (State, error) {
    data, _ := state.Get("data")
    
    // Direct agent.Chat call (no hub)
    response, err := n.agent.Chat(ctx, fmt.Sprintf("Process: %v", data))
    if err != nil {
        return state, err
    }
    
    return state.Set("result", response.Content()), nil
}

// Example: State graph with direct agent nodes
graph := state.New()
graph.AddNode("analyze", &DirectAgentNode{agent: analysisAgent})
graph.AddNode("review", &DirectAgentNode{agent: reviewAgent})
graph.AddNode("summarize", &DirectAgentNode{agent: summaryAgent})

graph.AddEdge("analyze", "review", nil)
graph.AddEdge("review", "summarize", KeyEquals("approved", true))
graph.SetEntryPoint("analyze")
graph.SetExitPoint("summarize")

result, err := graph.Execute(ctx, initialState)
```

**Hub-Based Coordination (When Needed):**

```go
// Hub node for multi-agent coordination
type HubNode struct {
    hub       hub.Hub
    fromAgent string
    toAgent   string
}

func (n *HubNode) Execute(ctx context.Context, state State) (State, error) {
    data, _ := state.Get("data")
    
    // Hub request for message routing
    response, err := n.hub.Request(ctx, n.fromAgent, n.toAgent, data)
    if err != nil {
        return state, err
    }
    
    return state.Set("result", response.Data), nil
}

// Example: State graph with hub-based nodes
graph := state.New()
graph.AddNode("analyze", NewHubNode(hub, "coordinator", "analyzer"))
graph.AddNode("review", NewHubNode(hub, "coordinator", "reviewer"))
graph.AddNode("approve", NewHubNode(hub, "coordinator", "approver"))

graph.AddEdge("analyze", "review", nil)
graph.AddEdge("review", "approve", KeyEquals("approved", true))
graph.SetEntryPoint("analyze")
graph.SetExitPoint("approve")

result, err := graph.Execute(ctx, initialState)
```

**When to Use Each Approach:**

Direct agent calls:
- ✅ Simple state transformation with single agent
- ✅ No message routing needed
- ✅ Clear sequential processing
- ✅ Minimal coordination overhead

Hub-based coordination:
- ✅ Multiple agents coordinate within a node
- ✅ Message routing required
- ✅ Cross-hub communication
- ✅ Dynamic agent selection

**Tests (Phase 2.4):**
- Direct agent node execution
- Hub node execution
- Multi-hub coordination in state graphs
- Both approaches update state correctly
- Error handling from agent calls
- Error handling from hub requests

### Success Criteria

**Track A Complete When:**
- ✅ Core state structures implemented (State, StateNode, Edge)
- ✅ Graph executor handles linear and conditional paths
- ✅ Cycle detection and max iteration protection
- ✅ Checkpointing enables recovery and rollback
- ✅ Hub integration validated through examples
- ✅ Comprehensive tests achieve 80%+ coverage
- ✅ Documentation covers state graph patterns

**Quality Validation:**
- ✅ State immutability preserved throughout execution
- ✅ Predicate evaluation is fast (pure functions preferred)
- ✅ Context cancellation stops execution immediately
- ✅ Checkpoint overhead is minimal
- ✅ Error messages are actionable

---

## Pattern: Stateful Workflows

**Purpose**: Complex workflows combining state graphs with workflow patterns.

**Dependencies**: Requires both Track A (state graphs) and Track B (patterns) complete.

### Integration Architecture

Stateful workflows combine:
- State graphs for overall workflow structure
- Patterns as state graph node implementations
- Hub for agent coordination within nodes

```
StateGraph
├── Node: Sequential Chain (pattern from Track B)
│   └── Steps orchestrate agents through hub
├── Node: Parallel Execution (pattern from Track B)
│   └── Fan-out to multiple agents
├── Node: Conditional Routing (pattern from Track B)
│   └── State-based agent selection
└── Node: Simple state transformations
```

### Example: Document Review Workflow

```go
// Create state graph
workflow := state.New()

// Node 1: Sequential analysis chain
analysisChain := func(ctx context.Context, state state.State) (state.State, error) {
    document, _ := state.Get("document").(Document)
    
    // Sequential chain of analysts
    analysts := []string{"technical-analyst", "business-analyst", "legal-analyst"}
    
    processor := func(ctx context.Context, analystID string, doc Document) (Document, error) {
        response, err := hub.Request(ctx, "coordinator", analystID, doc)
        return response.Data.(Document), err
    }
    
    result, err := patterns.ProcessChain(ctx, patterns.DefaultChainConfig(), analysts, document, processor, nil)
    if err != nil {
        return state, err
    }
    
    return state.Set("analyzed_document", result.Final), nil
}

workflow.AddNode("analyze", state.NewFunctionNode(analysisChain))

// Node 2: Parallel review
parallelReview := func(ctx context.Context, state state.State) (state.State, error) {
    document, _ := state.Get("analyzed_document").(Document)
    
    reviewers := []string{"reviewer-1", "reviewer-2", "reviewer-3"}
    
    processor := func(ctx context.Context, reviewerID string) (Review, error) {
        response, err := hub.Request(ctx, "coordinator", reviewerID, document)
        return response.Data.(Review), err
    }
    
    reviews, err := patterns.ProcessParallel(ctx, patterns.DefaultParallelConfig(), reviewers, processor, nil)
    if err != nil {
        return state, err
    }
    
    return state.Set("reviews", reviews), nil
}

workflow.AddNode("review", state.NewFunctionNode(parallelReview))

// Node 3: Conditional approval routing
approvalRouting := func(ctx context.Context, state state.State) (state.State, error) {
    reviews, _ := state.Get("reviews").([]Review)
    
    predicate := func(_ Review) (string, error) {
        allApproved := allReviewsApproved(reviews)
        if allApproved {
            return "final-approval", nil
        }
        return "revision", nil
    }
    
    config := patterns.RouteConfig{
        Routes: map[string]patterns.RouteHandler{
            "final-approval": func(ctx context.Context, _ Review) (Review, error) {
                response, err := hub.Request(ctx, "coordinator", "approver", state)
                return response.Data.(Review), err
            },
            "revision": func(ctx context.Context, _ Review) (Review, error) {
                return Review{Status: "needs_revision"}, nil
            },
        },
    }
    
    result, err := patterns.ProcessConditional(ctx, Review{}, predicate, config)
    if err != nil {
        return state, err
    }
    
    return state.Set("final_status", result.Status), nil
}

workflow.AddNode("approval", state.NewFunctionNode(approvalRouting))

// Connect nodes
workflow.AddEdge("analyze", "review", nil)
workflow.AddEdge("review", "approval", nil)
workflow.SetEntryPoint("analyze")
workflow.SetExitPoint("approval")

// Execute workflow
finalState, err := workflow.Execute(ctx, initialState)
```

### Implementation Steps

**Step 1: Validate integration points**

Ensure Track A and Track B components compose correctly:
- State graph nodes can execute patterns
- Patterns can use state for decisions
- Hub coordination works within patterns within state graphs

**Step 2: Create composition helpers**

Utility functions to simplify common compositions:

```go
// Helper: Create state node from sequential chain
func ChainNode[TItem any](
    hub hub.Hub,
    items []TItem,
    processor patterns.StepProcessor[TItem, state.State],
) state.StateNode

// Helper: Create state node from parallel execution
func ParallelNode[TItem, TResult any](
    hub hub.Hub,
    items []TItem,
    processor patterns.ItemProcessor[TItem, TResult],
) state.StateNode

// Helper: Create state node from conditional routing
func ConditionalNode(
    predicate patterns.RoutePredicate[state.State],
    config patterns.RouteConfig[state.State],
) state.StateNode
```

**Step 3: Create comprehensive examples**

Examples demonstrating real-world stateful workflows:
- Document approval workflow (above)
- Multi-stage data processing pipeline
- Agent collaboration workflow with feedback loops
- Incident response workflow with escalation

**Step 4: Add workflow-level testing**

Integration tests verifying:
- Patterns work correctly within state graphs
- State flows properly through pattern-based nodes
- Hub coordination works end-to-end
- Checkpointing works with pattern nodes
- Error propagation through composed workflows

### Success Criteria

**Stateful Workflows Complete When:**
- ✅ All Track A and Track B patterns integrate successfully
- ✅ Composition helpers simplify workflow construction
- ✅ Comprehensive examples demonstrate real-world use cases
- ✅ Integration tests achieve 80%+ coverage
- ✅ Documentation covers composition patterns
- ✅ Performance validated (workflows complete in reasonable time)

**Quality Validation:**
- ✅ Pattern composition is intuitive and type-safe
- ✅ Error handling works correctly across boundaries
- ✅ State transformations are predictable
- ✅ Hub integration is seamless

---

## Integration Points

### Track B Patterns Use Track A State Graphs

**Optional Integration**: Patterns can use state graphs internally for complex routing.

**Example**: Sequential chain step that uses state graph:

```go
// Step processor that internally uses state graph for complex logic
processor := func(ctx context.Context, item Item, current State) (State, error) {
    // Create mini state graph for this step
    subGraph := state.New()
    subGraph.AddNode("validate", validationNode)
    subGraph.AddNode("transform", transformationNode)
    subGraph.AddEdge("validate", "transform", nil)
    subGraph.SetEntryPoint("validate")
    subGraph.SetExitPoint("transform")
    
    // Execute sub-graph
    stepState := state.State{"item": item, "context": current}
    result, err := subGraph.Execute(ctx, stepState)
    if err != nil {
        return current, err
    }
    
    // Update current state with result
    return current.Merge(result), nil
}
```

### Track A State Graphs Use Track B Patterns

**Primary Integration**: State graph nodes execute patterns.

**Example**: State graph node that runs parallel execution:

```go
// Node that parallelizes agent requests
node := state.NewFunctionNode(func(ctx context.Context, state state.State) (state.State, error) {
    items, _ := state.Get("items").([]Item)
    
    results, err := patterns.ProcessParallel(ctx, cfg, items, processor, nil)
    if err != nil {
        return state, err
    }
    
    return state.Set("results", results), nil
})

graph.AddNode("parallel-processing", node)
```

### Composition Examples

**Example 1: Nested Patterns**
```go
// Sequential chain where each step runs parallel execution
chainProcessor := func(ctx context.Context, stage string, state State) (State, error) {
    agents := getAgentsForStage(stage)
    
    // Parallel execution within sequential step
    results, err := patterns.ProcessParallel(ctx, cfg, agents, agentProcessor, nil)
    if err != nil {
        return state, err
    }
    
    return state.Set(stage+"_results", results), nil
}

stages := []string{"analysis", "review", "approval"}
result, err := patterns.ProcessChain(ctx, cfg, stages, initialState, chainProcessor, nil)
```

**Example 2: State Graph with Pattern Nodes**
```go
graph := state.New()

// Sequential chain node
graph.AddNode("sequential", NewChainNode(hub, items, chainProcessor))

// Parallel execution node
graph.AddNode("parallel", NewParallelNode(hub, agents, parallelProcessor))

// Conditional routing node
graph.AddNode("conditional", NewConditionalNode(predicate, routeConfig))

// Connect with state-based transitions
graph.AddEdge("sequential", "parallel", state.KeyExists("chain_complete"))
graph.AddEdge("parallel", "conditional", state.KeyExists("parallel_complete"))

result, err := graph.Execute(ctx, initialState)
```

**Example 3: Pattern Using State Graph for Complex Logic**
```go
// Conditional routing that uses state graph for handler implementation
complexHandler := func(ctx context.Context, state State) (State, error) {
    // Handler is a state graph
    handlerGraph := state.New()
    handlerGraph.AddNode("validate", validationNode)
    handlerGraph.AddNode("process", processingNode)
    handlerGraph.AddNode("finalize", finalizationNode)
    
    handlerGraph.AddEdge("validate", "process", nil)
    handlerGraph.AddEdge("process", "finalize", nil)
    handlerGraph.SetEntryPoint("validate")
    handlerGraph.SetExitPoint("finalize")
    
    return handlerGraph.Execute(ctx, state)
}

config := patterns.RouteConfig{
    Routes: map[string]patterns.RouteHandler{
        "complex": complexHandler,
        "simple": simpleHandler,
    },
}

result, err := patterns.ProcessConditional(ctx, state, predicate, config)
```

---

### Milestones

**M1: Sequential Chains Complete**
- ✅ Generic sequential chain pattern extracted
- ✅ Tests achieve 80%+ coverage
- ✅ Hub integration examples documented

**M2: State Graph Core Complete**
- ✅ State structures and executor functional
- ✅ Linear and conditional graph execution validated
- ✅ Tests achieve 80%+ coverage

**M3: State Graph Advanced Complete**
- ✅ Checkpointing enables recovery
- ✅ Hub integration validated
- ✅ Tests achieve 80%+ coverage

**M4: Parallel Execution Complete**
- ✅ Parallel execution pattern implemented
- ✅ Tests achieve 80%+ coverage
- ✅ Hub integration examples documented

**M5: Full Pattern Suite Complete**
- ✅ Conditional routing implemented
- ✅ Track A + Track B integration validated
- ✅ Composition patterns documented

**M6: Stateful Workflows Complete**
- ✅ Complex workflow examples implemented
- ✅ End-to-end integration testing complete
- ✅ Documentation covers all composition patterns

### Dependencies

```
Parallel Dependencies (Can work simultaneously):
- Sequential Chains (Track B)
- Parallel Execution (Track B)
- State Graphs Phase 2.1-2.2 (Track A)

Sequential Dependencies:
- Conditional Routing requires: Sequential Chains + Parallel Execution
- State Graphs Phase 2.3-2.4 requires: Phase 2.1-2.2
- Stateful Workflows requires: All Track A + All Track B
```

### Risk Mitigation

**Risk 1: State Graph Design Complexity**
- **Mitigation**: Start with simple linear graphs, add complexity incrementally
- **Fallback**: Patterns provide value even without state graphs

**Risk 2: Integration Points Unclear**
- **Mitigation**: Create integration examples early (Week 3-4)
- **Fallback**: Keep tracks independent if integration issues arise

**Risk 3: Performance Issues**
- **Mitigation**: Add benchmarks for each pattern
- **Fallback**: Document performance characteristics and best practices

**Risk 4: Rate Limiting in Parallel Execution**
- **Mitigation**: Document limitations, plan resilience for future
- **Fallback**: Graceful degradation (parallel → sequential)

---

## Testing Strategy

### Test Organization

```
tests/
├── patterns/
│   ├── chain_test.go       # Sequential chain tests
│   ├── parallel_test.go    # Parallel execution tests
│   └── conditional_test.go # Conditional routing tests
├── state/
│   ├── state_test.go       # State operations tests
│   ├── node_test.go        # Node execution tests
│   ├── graph_test.go       # Graph execution tests
│   └── checkpoint_test.go  # Checkpointing tests
└── integration/
    ├── patterns_state_test.go  # Pattern + state graph integration
    ├── hub_patterns_test.go    # Hub + patterns integration
    └── workflow_test.go        # End-to-end workflow tests
```

### Test Coverage Requirements

- **Unit Tests**: 80%+ coverage per package
- **Integration Tests**: All integration points validated
- **Black-Box Testing**: All tests use public API only
- **Table-Driven Tests**: Multiple scenarios per test function

### Test Scenarios

**Sequential Chain Tests:**
- Basic chain (3-5 steps)
- Empty chain (0 items)
- Single item chain
- Error in middle step
- Context cancellation mid-chain
- Progress callback invocation
- Intermediate state capture

**Parallel Execution Tests:**
- Basic parallel (10-20 items)
- Worker count auto-detection
- Order preservation
- Error handling (fail-fast)
- Context cancellation
- Empty input
- Single item
- Large item sets (1000+)

**Conditional Routing Tests:**
- Basic routing (3-4 routes)
- Default handler fallback
- Predicate evaluation errors
- Missing route handling
- State-based routing decisions

**State Graph Tests:**
- Linear execution (A → B → C)
- Conditional edges (A → [predicate] → B or C)
- Cycle execution (A → B → C → B → exit)
- Max iterations protection
- Context cancellation
- Exit point detection
- Missing node/edge errors

**Integration Tests:**
- Pattern within state graph node
- State graph within pattern step
- Hub coordination through patterns
- Checkpointing with pattern nodes
- End-to-end workflow execution

### Performance Testing

**Benchmarks:**
- Sequential chain with 100/1000 items
- Parallel execution with varying worker counts
- State graph with 10/50/100 nodes
- Hub coordination overhead in patterns

**Performance Targets:**
- Sequential chains: < 10ms overhead per step
- Parallel execution: Scales with worker count up to optimal point
- State graph: < 100ms for 100-node graph
- Hub integration: < 50ms additional latency per hub call

---

## Success Criteria

### Track A: State Management

**Implementation Complete:**
- ✅ State structures support immutable operations
- ✅ Graph executor handles linear, conditional, and cyclic paths
- ✅ Checkpointing enables recovery from any point
- ✅ **State nodes work with direct go-agents calls** (primary pattern)
- ✅ Hub integration available for coordination (optional)
- ✅ Tests achieve 80%+ coverage
- ✅ Documentation covers state graph patterns

**Quality Metrics:**
- ✅ State operations are O(1) or O(n) (no exponential complexity)
- ✅ Graph execution is deterministic given same input
- ✅ Node execution is agnostic to agent/hub usage
- ✅ Error messages are actionable
- ✅ Context cancellation works immediately

### Track B: Workflow Patterns

**Implementation Complete:**
- ✅ Sequential chains extracted and generalized
- ✅ Parallel execution implemented with order preservation
- ✅ Conditional routing supports state-based decisions
- ✅ **All patterns work with direct go-agents calls** (primary pattern)
- ✅ Hub integration available for advanced coordination (optional)
- ✅ Tests achieve 80%+ coverage
- ✅ Documentation covers all patterns with examples

**Quality Metrics:**
- ✅ Pattern APIs are intuitive and type-safe
- ✅ Processor signatures are generic (not coupled to agents or hub)
- ✅ Error handling is consistent across patterns
- ✅ Performance meets targets
- ✅ Examples demonstrate both direct and hub-based usage

### Integration

**Implementation Complete:**
- ✅ Track A and Track B compose naturally
- ✅ Composition helpers simplify common patterns
- ✅ Stateful workflows demonstrate full integration
- ✅ Integration tests validate all boundaries
- ✅ Documentation covers composition strategies

**Quality Metrics:**
- ✅ No performance degradation from composition
- ✅ Error propagation works across boundaries
- ✅ Type safety preserved through composition
- ✅ Hub coordination works in composed workflows

### Overall Phase 2 & 3 Success

**Achieved When:**
- ✅ All Track A milestones complete
- ✅ All Track B milestones complete
- ✅ Integration validated through examples and tests
- ✅ Documentation comprehensive and accurate
- ✅ go-agents integration validated (patterns use agent.Agent interface)
- ✅ Performance acceptable for production use
- ✅ Test coverage exceeds 80% across all packages

**User Value Delivered:**
- Sequential workflow patterns available for immediate use
- Parallel execution enables concurrent agent coordination
- State graphs provide LangGraph-style workflow orchestration
- Stateful workflows enable complex multi-agent systems
- Clear path from simple patterns to complex orchestration

---

## Next Steps

### Immediate Actions

1. **Create Track B package structure:**
   ```bash
   mkdir -p pkg/patterns
   touch pkg/patterns/doc.go
   touch pkg/patterns/chain.go
   touch tests/patterns/chain_test.go
   ```

2. **Extract sequential chain from classify-docs:**
   - Copy core logic from `tools/classify-docs/pkg/processing/sequential.go`
   - Generalize types (remove document.Page dependency)
   - Adapt configuration structure
   - Create initial tests

3. **Start Track A planning:**
   - Design state graph interfaces in parallel
   - Create initial state.go structure
   - Plan graph executor algorithm

### Documentation Deliverables

- **Package Documentation**: Comprehensive godoc for all packages
- **Integration Examples**: Real-world usage patterns
- **Migration Guide**: Moving from simple to complex patterns
- **Performance Guide**: Optimization strategies
- **Troubleshooting Guide**: Common issues and solutions

---

## Conclusion

This consolidated approach delivers workflow capabilities faster while maintaining architectural integrity. Track B patterns provide immediate value, while Track A establishes the state management foundation. Integration happens naturally through clear interfaces and composition patterns.

**Key Design Achievement**: All patterns work directly with go-agents without requiring hub infrastructure. Hub integration is available when multi-agent orchestration is required, but it's optional, not mandatory.

**Pattern Flexibility Summary:**

| Pattern | Direct go-agents | Hub Integration | Pure Computation |
|---------|-----------------|-----------------|------------------|
| Sequential chains | ✅ Primary (classify-docs) | ✅ Optional | ✅ Supported |
| Parallel execution | ✅ Primary | ✅ Optional | ✅ Supported |
| Conditional routing | ✅ Primary | ✅ Optional | ✅ Supported |
| State graphs | ✅ Primary | ✅ Optional | ✅ Supported |

Success depends on clear separation of concerns, well-defined interfaces, and disciplined testing at each milestone. The generic processor signatures ensure patterns remain flexible and composable regardless of the underlying implementation approach.
