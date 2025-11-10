package state

import (
	"context"
	"fmt"
	"time"

	"github.com/JaimeStill/go-agents-orchestration/pkg/config"
	"github.com/JaimeStill/go-agents-orchestration/pkg/observability"
)

// StateGraph defines a workflow as a directed graph of nodes and edges.
//
// State graphs enable LangGraph-style orchestration in Go with:
//   - Nodes: Computation steps that transform state
//   - Edges: Transitions between nodes with optional predicates
//   - Entry/Exit points: Define workflow start and end
//   - State flow: Immutable state flows through graph execution
//
// Example workflow structure:
//
//	graph, err := state.NewGraph(config)
//	graph.AddNode("analyze", analyzerNode)
//	graph.AddNode("review", reviewerNode)
//	graph.AddNode("approve", approverNode)
//	graph.AddEdge("analyze", "review", nil)
//	graph.AddEdge("review", "approve", state.KeyEquals("status", "approved"))
//	graph.SetEntryPoint("analyze")
//	graph.SetExitPoint("approve")
//	result, err := graph.Execute(ctx, initialState)
type StateGraph interface {
	// Name returns the graph identifier for event metadata
	Name() string

	// AddNode registers a computation step in the graph
	AddNode(name string, node StateNode) error

	// AddEdge creates a transition between nodes (predicate can be nil for unconditional)
	AddEdge(from, to string, predicate TransitionPredicate) error

	// SetEntryPoint defines the starting node for execution
	SetEntryPoint(node string) error

	// SetExitPoint defines a terminal node (execution stops here)
	SetExitPoint(node string) error

	// Execute runs the graph from entry point with initial state
	Execute(ctx context.Context, initialState State) (State, error)
}

// stateGraph implements StateGraph interface with concrete execution engine.
type stateGraph struct {
	name                 string
	nodes                map[string]StateNode
	edges                map[string][]Edge
	entryPoint           string
	exitPoints           map[string]bool
	maxIterations        int
	observer             observability.Observer
	checkpointStore      CheckpointStore
	checkpointInterval   int
	preserverCheckpoints bool
}

// Name returns the graph identifier for event metadata.
func (g *stateGraph) Name() string {
	return g.name
}

// NewGraph creates a new state graph from configuration.
//
// The constructor resolves the observer from the configuration registry
// and initializes the graph with empty node/edge collections.
//
// Example:
//
//	cfg := config.GraphConfig{
//	    Name:          "document-workflow",
//	    Observer:      "noop",
//	    MaxIterations: 1000,
//	}
//	graph, err := state.NewGraph(cfg)
//	if err != nil {
//	    // Handle observer resolution error
//	}
func NewGraph(cfg config.GraphConfig) (StateGraph, error) {
	observer, err := observability.GetObserver(cfg.Observer)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve observer: %w", err)
	}

	var checkpointStore CheckpointStore
	if cfg.Checkpoint.Interval > 0 {
		checkpointStore, err = GetCheckpointStore(cfg.Checkpoint.Store)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve checkpoint store: %w", err)
		}
	}

	return &stateGraph{
		name:                 cfg.Name,
		nodes:                make(map[string]StateNode),
		edges:                make(map[string][]Edge),
		exitPoints:           make(map[string]bool),
		maxIterations:        cfg.MaxIterations,
		observer:             observer,
		checkpointStore:      checkpointStore,
		checkpointInterval:   cfg.Checkpoint.Interval,
		preserverCheckpoints: cfg.Checkpoint.Preserve,
	}, nil
}

// AddNode registers a computation step in the graph.
//
// Nodes must have unique names. Adding a duplicate node returns an error.
func (g *stateGraph) AddNode(name string, node StateNode) error {
	if name == "" {
		return fmt.Errorf("node name cannot be empty")
	}

	if node == nil {
		return fmt.Errorf("node cannot be nil")
	}

	if _, exists := g.nodes[name]; exists {
		return fmt.Errorf("node %s already exists", name)
	}

	g.nodes[name] = node
	return nil
}

// AddEdge creates a transition between nodes.
//
// Both nodes must exist before adding an edge. Predicate can be nil for
// unconditional transitions. Multiple edges from the same node are allowed.
func (g *stateGraph) AddEdge(from, to string, predicate TransitionPredicate) error {
	if from == "" {
		return fmt.Errorf("from node cannot be empty")
	}

	if to == "" {
		return fmt.Errorf("to node cannot be empty")
	}

	if _, exists := g.nodes[from]; !exists {
		return fmt.Errorf("from node %s does not exist", from)
	}

	if _, exists := g.nodes[to]; !exists {
		return fmt.Errorf("to node %s does not exist", to)
	}

	edge := Edge{
		From:      from,
		To:        to,
		Predicate: predicate,
	}

	g.edges[from] = append(g.edges[from], edge)
	return nil
}

// SetEntryPoint defines the starting node for execution.
//
// The entry point node must exist. Only one entry point is allowed.
func (g *stateGraph) SetEntryPoint(node string) error {
	if node == "" {
		return fmt.Errorf("entry point cannot be empty")
	}

	if g.entryPoint != "" {
		return fmt.Errorf("entry point already set to %s", g.entryPoint)
	}

	if _, exists := g.nodes[node]; !exists {
		return fmt.Errorf("entry point node %s does not exist", node)
	}

	g.entryPoint = node
	return nil
}

// SetExitPoint defines a terminal node where execution stops.
//
// Multiple exit points are supported - call this method multiple times
// to register different termination conditions. The exit point node must exist.
func (g *stateGraph) SetExitPoint(node string) error {
	if node == "" {
		return fmt.Errorf("exit point cannot be empty")
	}

	if _, exists := g.nodes[node]; !exists {
		return fmt.Errorf("exit points node %s does not exist", node)
	}

	g.exitPoints[node] = true
	return nil
}

// Validate checks graph structure for common configuration errors.
//
// Validation ensures:
//   - At least one node exists
//   - Entry point is set and exists
//   - At least one exit point is set
//   - All exit points exist as nodes
//
// This method is called internally by Execute but can be called explicitly
// to validate graph structure before execution.
func (g *stateGraph) Validate() error {
	if len(g.nodes) == 0 {
		return fmt.Errorf("graph has no nodes")
	}

	if g.entryPoint == "" {
		return fmt.Errorf("entry point not set")
	}

	if _, exists := g.nodes[g.entryPoint]; !exists {
		return fmt.Errorf("entry point %s does not exist", g.entryPoint)
	}

	if len(g.exitPoints) == 0 {
		return fmt.Errorf("no exit points set")
	}

	for exitPoint := range g.exitPoints {
		if _, exists := g.nodes[exitPoint]; !exists {
			return fmt.Errorf("exit point %s does not exist", exitPoint)
		}
	}

	return nil
}

// Execute runs the graph from entry point with initial state.
//
// Execution follows this algorithm:
//  1. Validate graph structure
//  2. Start at entry point node
//  3. Execute current node with state
//  4. Check if current node is an exit point
//  5. Evaluate outgoing edges to find next node
//  6. Repeat from step 3 with next node
//  7. Return final state when exit point reached
//
// Cycle detection and iteration limits prevent infinite loops.
// Observer receives events for all execution milestones.
//
// Returns ExecutionError with full context on failure.
func (g *stateGraph) Execute(ctx context.Context, initialState State) (State, error) {
	if err := g.Validate(); err != nil {
		return initialState, fmt.Errorf("graph validation failed: %w", err)
	}

	g.observer.OnEvent(ctx, observability.Event{
		Type:      observability.EventGraphStart,
		Timestamp: time.Now(),
		Source:    g.name,
		Data: map[string]any{
			"entry_point": g.entryPoint,
			"run_id":      initialState.RunID(),
			"exit_points": len(g.exitPoints),
		},
	})

	current := g.entryPoint
	state := initialState
	iterations := 0
	visited := make(map[string]int)
	path := make([]string, 0, g.maxIterations)

	for {
		if err := ctx.Err(); err != nil {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("execution cancelled: %w", err),
			}
		}

		iterations++
		if iterations > g.maxIterations {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("max iterations (%d) exceeded", g.maxIterations),
			}
		}

		visited[current]++
		path = append(path, current)

		if visited[current] > 1 {
			g.observer.OnEvent(ctx, observability.Event{
				Type:      observability.EventCycleDetected,
				Timestamp: time.Now(),
				Source:    g.name,
				Data: map[string]any{
					"node":        current,
					"visit_count": visited[current],
					"iteration":   iterations,
					"path_length": len(path),
				},
			})
		}

		node, exists := g.nodes[current]
		if !exists {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("node %s not found", current),
			}
		}

		g.observer.OnEvent(ctx, observability.Event{
			Type:      observability.EventNodeStart,
			Timestamp: time.Now(),
			Source:    g.name,
			Data: map[string]any{
				"node":      current,
				"iteration": iterations,
			},
		})

		newState, err := node.Execute(ctx, state)

		g.observer.OnEvent(ctx, observability.Event{
			Type:      observability.EventNodeComplete,
			Timestamp: time.Now(),
			Source:    g.name,
			Data: map[string]any{
				"node":      current,
				"iteration": iterations,
				"error":     err != nil,
			},
		})

		if err != nil {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("node execution failed: %w", err),
			}
		}

		state = newState.SetCheckpointNode(current)

		if g.checkpointInterval > 0 && iterations%g.checkpointInterval == 0 {
			if err := state.Checkpoint(g.checkpointStore); err != nil {
				return state, &ExecutionError{
					NodeName: current,
					State:    state,
					Path:     path,
					Err:      fmt.Errorf("checkpoint save failed: %w", err),
				}
			}

			g.observer.OnEvent(ctx, observability.Event{
				Type:      observability.EventCheckpointSave,
				Timestamp: time.Now(),
				Source:    g.name,
				Data: map[string]any{
					"node":   current,
					"run_id": state.RunID(),
				},
			})
		}

		if g.exitPoints[current] {
			g.observer.OnEvent(ctx, observability.Event{
				Type:      observability.EventGraphComplete,
				Timestamp: time.Now(),
				Source:    g.name,
				Data: map[string]any{
					"exit_point":  current,
					"iterations":  iterations,
					"path_length": len(path),
				},
			})

			if !g.preserverCheckpoints && g.checkpointInterval > 0 {
				g.checkpointStore.Delete(state.RunID())
			}

			return state, nil
		}

		edges, hasEdges := g.edges[current]
		if !hasEdges {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("node %s has not outgoing edges and is not an exit point", current),
			}
		}

		nextNode := ""
		for i, edge := range edges {
			g.observer.OnEvent(ctx, observability.Event{
				Type:      observability.EventEdgeEvaluate,
				Timestamp: time.Now(),
				Source:    g.name,
				Data: map[string]any{
					"from":          edge.From,
					"to":            edge.To,
					"edge_index":    i,
					"has_predicate": edge.Predicate != nil,
				},
			})

			if edge.Predicate == nil || edge.Predicate(state) {
				nextNode = edge.To

				g.observer.OnEvent(ctx, observability.Event{
					Type:      observability.EventEdgeTransition,
					Timestamp: time.Now(),
					Source:    g.name,
					Data: map[string]any{
						"from":       edge.From,
						"to":         edge.To,
						"edge_index": i,
					},
				})

				break
			}
		}

		if nextNode == "" {
			return state, &ExecutionError{
				NodeName: current,
				State:    state,
				Path:     path,
				Err:      fmt.Errorf("no valid transition from node %s", current),
			}
		}

		current = nextNode
	}
}
