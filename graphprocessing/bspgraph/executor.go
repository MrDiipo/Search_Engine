package bspgraph

import "context"

// ExecutorCallbacks encapsulates a series of callbacks that are invoked by an
// Executor instance on a graph. Callbacks are optional if not specified
type ExecutorCallbacks struct {

	// PreStep if defined, is invoked before running the next superstep.
	// This serves as a place to initialize variables, aggregators etc. that will
	// be used for the next superstep
	PreStep func(ctx context.Context, g *Graph) error

	// PostStep, if define	d, is invoked after running a superstep.
	PostStep func(ctx context.Context, g *Graph, activeInStep int) error

	// PostStepKeepRunning, if defined, is invoked after running a superstep
	// to decide whether the stop condition for terminating the run has been met
	PostStepKeepRunning func(ctx context.Context, g *Graph, activeInStep int) (bool, error)
}

func patchEmptyCallbacks(e *ExecutorCallbacks) {
	if e.PreStep == nil {
		e.PreStep = func(ctx context.Context, g *Graph) error { return nil }
	}
	if e.PostStep == nil {
		e.PostStep = func(ctx context.Context, g *Graph, activeInStep int) error { return nil }
	}
	if e.PostStepKeepRunning == nil {
		e.PostStepKeepRunning = func(ctx context.Context, g *Graph, activeInStep int) (bool, error) { return true, nil }
	}
}

// Executor wraps a Graph instance and provides an orchestration layer
// for executing super-steps until an error occurs or an exit condition is met.
// Clients can provide their own ExecutorCallbacks
type Executor struct {
	g  *Graph
	cb ExecutorCallbacks
}

// NewExecutor returns an Executor instance for graph g that invokes
// the provided list of callbacks inside each execution loop.
func NewExecutor(g *Graph, cb ExecutorCallbacks) *Executor {
	patchEmptyCallbacks(&cb)
	g.superstep = 0
	return &Executor{
		g:  g,
		cb: cb,
	}
}
