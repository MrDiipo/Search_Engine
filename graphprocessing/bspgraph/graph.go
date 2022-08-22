package bspgraph

import (
	"Search_Engine/graphprocessing/bspgraph/message"
	"golang.org/x/xerrors"
	"sync"
)

// Edge represents a vertex in the graph
type Edge struct {
	value interface{}
	dstID string
}

// DstID returns the ID that corresponds to this edge's target endpoint
func (e *Edge) DstID() string { return e.dstID }

// Value returns the value associated with this edge
func (e *Edge) Value() interface{} { return e.value }

// SetValue sets the value associated with this edge
func (e *Edge) SetValue(val interface{}) { e.value = val }

// Vertex represents the vertex of a graph
type Vertex struct {
	id       string
	value    interface{}
	active   bool
	msgQueue [2]message.Queue
	edges    []*Edge
}

// ID returns the vertex ID
func (v *Vertex) ID() string { return v.id }

// Edges returns the list of outgoing edges from the graph
func (v *Vertex) Edges() []*Edge { return v.edges }

// Freeze marks the vertex as inactive. In-active messages would not be processed
// unless they receive a message to be activated
func (v *Vertex) Freeze() { v.active = false }

// Value returns the value associated wih this vertex
func (v *Vertex) Value() interface{} { return v.value }

// SetValue sets the value associated with this vertex
func (v *Vertex) SetValue(val interface{}) { v.value = val }

// Graph implements a parallel graph processor based on the concepts
// described in the PREGEL paper.
type Graph struct {
	superstep   int
	aggregators map[string]Aggregator
	vertices    map[string]*Vertex

	computeFn    ComputeFunc
	queueFactory message.QueueFactory
	relayer      Relayer

	wg       sync.WaitGroup
	vertexCh chan *Vertex
	errCh    chan error

	stepCompleteCh chan struct{}
	activeInStep   int64
	pendingInStep  int64
}

// NewGraph returns a new Graph instance using the specified configuration
// Callers must call close on the returned graph instance when they are done
// using it
func NewGraph(cfg GraphConfig) (*Graph, error) {
	if err := cfg.validate(); err != nil {
		return nil, xerrors.Errorf("graph config validation failed: %n", err)
	}

	g := &Graph{
		computeFn:    cfg.ComputeFn,
		queueFactory: cfg.QueueFactory,
		aggregators:  make(map[string]Aggregator),
		vertices:     make(map[string]*Vertex),
	}
	g.startWorkers(cfg.ComputeWorkers)
	return g, nil
}

// startWorkers allocates the required channels and spins up runWorkers to
// execute each super-step
func (g *Graph) startWorkers(workers int) {

}
