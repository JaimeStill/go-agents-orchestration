package workflows

import "fmt"

// ChainError provides rich error context for chain execution failures.
//
// Generic over both TItem and TContext to preserve complete error state including
// the item being processed and the accumulated state at failure point. This enables
// detailed debugging and recovery strategies.
//
// The error implements standard error unwrapping via Unwrap(), enabling errors.Is
// and errors.As for error chain inspection.
//
// Example usage:
//
//	result, err := workflows.ProcessChain(ctx, cfg, items, initial, processor, nil)
//	if err != nil {
//	    if chainErr, ok := err.(*ChainError[Item, State]); ok {
//	        fmt.Printf("Failed at step %d\n", chainErr.StepIndex)
//	        fmt.Printf("Failed item: %v\n", chainErr.Item)
//	        fmt.Printf("State at failure: %v\n", chainErr.State)
//	    }
//	}
type ChainError[TItem, TContext any] struct {
	// StepIndex is the 0-based index of the step that failed
	StepIndex int

	// Item is the item being processed when the error occurred
	Item TItem

	// State is the accumulated context at the time of failure
	State TContext

	// Err is the underlying error that caused the failure
	Err error
}

// Error returns a formatted error message with step index context.
// Implements the standard error interface.
func (e *ChainError[TItem, TContext]) Error() string {
	return fmt.Sprintf("chain failed at step %d: %v", e.StepIndex, e.Err)
}

// Unwrap returns the underlying error, enabling errors.Is and errors.As.
// This supports standard Go error unwrapping patterns.
func (e *ChainError[TItem, TContext]) Unwrap() error {
	return e.Err
}
