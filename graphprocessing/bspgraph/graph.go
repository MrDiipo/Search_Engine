package bspgraph

import (
	"Search_Engine/graphprocessing/bspgraph/message"
	"errors"
	"golang.org/x/xerrors"
	"sync"
	"sync/atomic"
)

var (
	// ErrUnknownEdgeSource is returned by AddEdge when the source vertex
	// is not present in the graph
	ErrUnknownEdgeSource = xerrors.New("source vertex is not part of the graph")
	// ErrDestinationIsLocal is returned by Relayer instances to indicate
	// that a message destination is actually owned the local graph
	ErrDestinationIsLocal = xerrors.New("message destination is assigned to the local graph")
	// ErrInvalidMessageDestination is returned by calls to SendMessage and
	// BroadCastToNeighbors when the destination cannot be resolved to any
	// (local or remote) vertex.
	ErrInvalidMessageDestination = xerrors.New("invalid message destination")
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

// Vertices returns the graph vertices as a map where the key is the vertex ID.
func (g *Graph) Vertices() map[string]*Vertex { return g.vertices }

// AddEdge inserts a directed edge from src to destination and annotates it with the
// specified initialValue.
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

// BroadcastToNeighbors is a helper function that broadcasts a single message
// to each neighbor of a particular vertex. Messages are queued for delivery
// and will be processed by recipients in the next super-step.
func (g *Graph) BroadcastToNeighbors(v *Vertex, msg message.Message) error {
	for _, e := range v.edges {
		if err := g.SendMessage(e.dstID, msg); err != nil {
			return err
		}
	}
	return nil
}

// SendMessage  attemps to deliver a message to the vertex with the specified
// destination ID. Messages are queued up for delivery and will be processed by
// recipients in the next super-step.
//
// If the destination ID is not known in this graph, it might still be a valid ID for a vertex
// that is owned by a remote graph instance. If the client has provided a Relayer when configuring the graph,
// SendMessage will delegate message delivery to it.
//
// On the other hand. if no Relayer is defined or the configured RemoteMessageSender
// returns a ErrDestinationIsLocal error, SendMessage wil first check  whether an UnknownVertexHandler
// has been provided at configuration time and invoke it. Otherwise and ErrInvalidMessageDestination
// is returned to the caller.
func (g *Graph) SendMessage(dstID string, msg message.Message) error {
	// If the vertex is known to the local graph instance, queue the message
	// directly, so it can be delivered at the next superstep.
	dstVert := g.vertices[dstID]
	if dstVert != nil {
		queueIndex := (g.superstep + 1) % 2
		return dstVert.msgQueue[queueIndex].Enqueue(msg)
	}
	// The vertex is not known locally but might be known to a partition
	// that is processed at another node. If a remote relayer has been
	// configured,  delegate the message send operation to it.
	if g.relayer != nil {
		if err := g.relayer.Relay(dstID, msg); !errors.Is(err, ErrDestinationIsLocal) {
			return err
		}
	}
	return xerrors.Errorf("message cannot be delivered to %q: %w", dstID, ErrInvalidMessageDestination)
}

// Superstep returns the current superstep value.
func (g *Graph) Superstep() int { return g.superstep }

// Step executes the next superstep and returns back the number of vertices
// that were processed either because they were still active or because they
// received a message.
func (g *Graph) step() (int, error) {
	g.activeInStep = 0
	g.pendingInStep = int64(len(g.vertices))

	// No work required.
	if g.pendingInStep == 0 {
		return 0, nil
	}

	for _, v := range g.vertices {
		g.vertexCh <- v
	}
	// Block until worker pool has finished processing all vertices.
	<-g.stepCompleteCh
	// Dequeue any errors
	var err error
	select {
	case err = <-g.errCh:
	default: // no error available
	}
	return int(g.activeInStep), err
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

// stepWorker polls vertexCh for incoming vertices and executes the configured
// ComputeFunc for each one. The worker automatically exits when vertexCh gets closed.
func (g *Graph) stepWorker() {
	for v := range g.vertexCh {
		buffer := g.superstep % 2
		if v.active || v.msgQueue[buffer].PendingMessage() {
			_ = atomic.AddInt64(&g.activeInStep, 1)
			v.active = true
			if err := g.computeFn(g, v, v.msgQueue[buffer].Messages()); err != nil {
				tryEmitError(g.errCh, xerrors.Errorf("running compute function for vertex %q failed: %w", v.ID(), err))
			} else if err := v.msgQueue[buffer].DiscardMessages(); err != nil {
				tryEmitError(g.errCh, xerrors.Errorf("discarding unprocessed messages for vertex %q failed: %w", v.ID(), err))
			}
		}
		if atomic.AddInt64(&g.pendingInStep, -1) == 0 {
			g.stepCompleteCh <- struct{}{}
		}
	}
	g.wg.Done()
}

// Reset the state of the graph
func (g *Graph) Reset() error {
	g.superstep = 0
	for _, v := range g.vertices {
		for i := 0; i < 2; i++ {
			if err := v.msgQueue[i].Close(); err != nil {
				return xerrors.Errorf("closing message queue #%d for vertex %v: %w", i, v.ID(), err)
			}
		}
	}
	g.vertices = make(map[string]*Vertex)
	g.aggregators = make(map[string]Aggregator)
	return nil
}

// Close shuts down the long-running go-routines
func (g *Graph) Close() error {
	close(g.vertexCh)
	g.wg.Wait()
	return g.Reset()
}

func tryEmitError(errCh chan<- error, err error) {
	select {
	case errCh <- err: // enqueue error
	default: // channel already contains another error
	}
}
