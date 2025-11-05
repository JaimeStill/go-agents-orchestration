// Package workflows provides composable workflow patterns for orchestrating multi-step processes.
//
// This package implements generic workflow primitives extracted from real-world usage
// (classify-docs) that work with any item and context types. All patterns support
// direct go-agents usage as the primary approach, with optional hub coordination
// for multi-agent orchestration.
//
// # Sequential Chain Pattern
//
// The sequential chain pattern implements a fold/reduce operation where items are
// processed in order with state accumulation between steps. Each step receives the
// current accumulated state and returns an updated state.
//
// Example with direct agent usage:
//
//	questions := []string{"What is AI?", "What is ML?"}
//	initial := Conversation{}
//
//	processor := func(ctx context.Context, question string, conv Conversation) (Conversation, error) {
//	    response, err := agent.Chat(ctx, question)
//	    if err != nil {
//	        return conv, err
//	    }
//	    conv.AddExchange(question, response.Content())
//	    return conv, nil
//	}
//
//	result, err := workflows.ProcessChain(ctx, config.DefaultChainConfig(), questions, initial, processor, nil)
//
// # Pattern Independence
//
// All workflow patterns are agnostic about processing approach:
//   - Direct go-agents usage (primary pattern)
//   - Hub orchestration (optional for multi-agent coordination)
//   - Pure data transformation (no agents required)
//   - Mixed approaches (some steps with agents, some without)
//
// The processor function signatures intentionally don't constrain implementation,
// enabling maximum flexibility.
//
// # Observer Integration
//
// All patterns emit events at key execution points for observability:
//   - Chain/workflow start and completion
//   - Step start and completion
//   - Error conditions
//
// Observer configuration is provided via config package structures, following
// the configuration lifecycle principle (config used only during initialization).
//
// # Error Handling
//
// Errors are wrapped in rich error types (e.g., ChainError) that provide complete
// context for debugging:
//   - Step/position where failure occurred
//   - Item being processed
//   - State at time of failure
//   - Underlying error with unwrap support
//
// # Integration with State Package
//
// The state package's State type works naturally as TContext for stateful workflows,
// enabling composition of workflow patterns with state graph execution (Phase 7).
package workflows
