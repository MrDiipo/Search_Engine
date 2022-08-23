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

// AddVertex  inserts a new vertex with the specified id and initial value into
// the graph already exists, AddVertex will just overwrite its value with the provided initial value
func (g *Graph) AddVertex(id string, initValue interface{}) {
	v := g.vertices[id]
	if v == nil {
		v = &Vertex{
			id: id,
			msgQueue: [2]message.Queue{
				g.queueFactory(), g.queueFactory(),
			},
			active: true,
		}
		g.vertices[id] = v
	}
	v.SetValue(initValue)
}

func (g *Graph) AddEdge(srcID, dstID string, initialValue interface{}) error {
	srcVert := g.vertices[srcID]
	if srcVert == nil {
		return xerrors.Errorf("create edge from %q to %q: %w", srcID, dstID, ErrUnknownEdgeSource)
	}

	srcVert.edges = append(srcVert.edges, &Edge{
		dstID: dstID,
		value: initialValue,
	})
	return nil
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
	g.vertexCh = make(chan *Vertex)
	g.errCh = make(chan error)
	g.stepCompleteCh = make(chan struct{})

	g.wg.Add(workers)
	for i := 0; i < workers; i++ {
		go g.stepWorker()
	}
}

func (g *Graph) stepWorker() {
}

// RegisterAggregator adds an aggregator with the specified name into the graph.
func (g *Graph) RegisterAggregator(name string, aggr Aggregator) { g.aggregators[name] = aggr }

// Aggregator returns the aggregator with the specified name or nil if the aggregator
// does not exist
func (g *Graph) Aggregator(name string) Aggregator {
	return g.aggregators[name]
}

// Aggregators returns a map of all currently registered aggregators where the key is the
// aggregator's name
func (g *Graph) Aggregators() map[string]Aggregator {
	return g.aggregators
}
