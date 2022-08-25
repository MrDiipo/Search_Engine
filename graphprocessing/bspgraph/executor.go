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

// ExecutorFactory is a function that creates new executor instance
type ExecutorFactory func(*Graph, ExecutorCallbacks) *Executor

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

// RunToCompletion keeps executing supersteps until the context expires, an
// error occurs or one of the Pre/PostStepKeepRunning callbacks specified at
// configuration time returns false,
func (ex *Executor) RunToCompletion(ctx context.Context) error {
	return ex.run(ctx, -1)
}

// RunSteps executes at most numStep stepsteps unless the context expires,
// an error occurs or one of the Pre/PostStepKeepRunning callbacks specifies at
// configuration time returns false
func (ex *Executor) RunSteps(ctx context.Context, numSteps int) error {
	return ex.run(ctx, numSteps)
}

// Graph returns the graph instance associated with this executor
func (ex *Executor) Graph() *Graph {
	return ex.g
}

// Superstep returns the current graph superstep
func (ex *Executor) Superstep() int {
	return ex.g.superstep
}

func (ex *Executor) run(ctx context.Context, maxSteps int) error {
	var (
		activeInStep int
		err          error
		keepRunning  bool
		cb           = ex.cb
	)
	for ; maxSteps != 0; ex.g.superstep, maxSteps = ex.g.superstep+1, maxSteps-1 {
		if err = ensureContextNotExpired(ctx); err != nil {
			break
		} else if err = cb.PreStep(ctx, ex.g); err != nil {
			break
		} else if activeInStep, err = ex.g.step(); err != nil {
			break
		} else if err = cb.PostStep(ctx, ex.g, activeInStep); err != nil {
			break
		} else if keepRunning, err = cb.PostStepKeepRunning(ctx, ex.g, activeInStep); !keepRunning || err != nil {
			break
		}
	}
	return err
}

func ensureContextNotExpired(ctx context.Context) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
