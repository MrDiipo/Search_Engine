package bspgraph

import "Search_Engine/graphprocessing/bspgraph/message"

// Aggregator is implemented by type that provide concurrent-safe
// aggregation primitives (e.g counters, min/max, topN).
type Aggregator interface {
	// Type returns the type of this aggregator.
	Type() string
	// Set the value of the aggregator to the specified value
	Set(val interface{})
	// Get the current aggregator value.
	Get() interface{}
	// Aggregate updates the aggregator's value based on the provided value
	Aggregate(val interface{})
	// Delta returns the change int the aggregator's value since the last call to Delta or Set
	Delta() interface{}
}

// Relayer is implemented by types that can relay messages to vertices that are managed
// by a remote graph instance
type Relayer interface {
	// Relay a message to a vertex that is not known locally
	Relay(dst string, msg message.Message) error
}

// RelayerFunc is an adapter to allow the use of ordinary functions as Relayers.
// If f is a function with the appropriate signature, RelaterFunc(f) is a relayer that calls f.
type RelayerFunc func(string, message.Message) error

// Relay calls f(dst, msg).
func (f RelayerFunc) Relay(dst string, msg message.Message) error {
	return f(dst, msg)
}

// ComputeFunc is a function that a graph instance invokes on each vertex when
// executing a super-step.
type ComputeFunc func(g *Graph, v *Vertex, msgIt message.Iterator) error
