package state

import "context"

// StateGraph defines a workflow as a directed graph of nodes and edges.
//
// State graphs enable LangGraph-style orchestration in Go with:
//   - Nodes: Computation steps that transform state
//   - Edges: Transitions between nodes with optional predicates
//   - Entry/Exit points: Define workflow start and end
//   - State flow: Immutable state flows through graph execution
//
// Phase 3 will provide the executor implementation. This interface establishes
// the contract for graph construction and execution.
//
// Example workflow structure:
//
//	graph := state.NewGraph(config, observer)
//	graph.AddNode("analyze", analyzerNode)
//	graph.AddNode("review", reviewerNode)
//	graph.AddNode("approve", approverNode)
//	graph.AddEdge("analyze", "review", nil)  // Unconditional
//	graph.AddEdge("review", "approve", state.KeyEquals("status", "approved"))
//	graph.SetEntryPoint("analyze")
//	graph.SetExitPoint("approve")
//	result, err := graph.Execute(ctx, initialState)
type StateGraph interface {
	// AddNode registers a computation step in the graph
	AddNode(name string, node StateNode) error

	// AddEdge creates a transition between nodes (predicate can be nil for unconditional)
	AddEdge(from, to string, predicate TransitionPredicate) error

	// SetEntryPoint defines the starting node for execution
	SetEntryPoint(node string) error

	// SetExitPoint defines a terminal node (execution stops here)
	SetExitPoint(node string) error

	// Execute runs the graph from entry point with initial state (Phase 3)
	Execute(ctx context.Context, initialState State) (State, error)
}
