package pipeline

import "context"

// Payload is implemented by values that can be sent through a pipeline
type Payload interface {
	Clone() Payload
	MarkAsProcessed()
}

// ProcessorFunc is an adapter that allow the use of plain functions as Processor instance
type ProcessorFunc func(context.Context, Payload) (Payload, error)

func (f ProcessorFunc) Process(ctx context.Context, p Payload) (Payload, error) {
	return f(ctx, p)
}

// Processor implement types that can process Payloads as part of a pipeline stage
type Processor interface {
	// Process operates on the input payload and returns a new payload
	// to be forwarded to the next pipeline stage. Processors may also opt to
	// prevent the  payload from reaching the rest of the pipeline by returning
	// a nil payload value instead.
	Process(ctx context.Context, payload Payload) (Payload, error)
}

//StageParams encapsulates the information required for
// executing a pipeline stage. The pipeline passes a StageParams
// instance to the Run() method of each stage.
type StageParams interface {
	// StageIndex returns the position of this stage in the pipeline
	StageIndex() int
	// Input returns a channel for reading the input payloads for a stage
	Input() <-chan Payload
	// Output returns a channel for writing the output payload to a stage
	Output() chan<- Payload
	// Error returns a channel for writing the errors encountered in a stage
	// while processing payloads.
	Error() chan<- Payload
}

// StageRunner is implemented by types that can be strung together to
// form a multi-stage pipeline
type StageRunner interface {
	// Run implements the processing logic for this stage by reading incoming payloads
	// from an input channel, Processing them and outputting the results to an output channel
	Run(context.Context, StageParams)
}

type Source interface {
	//Next fetches the next payload from the source. Returns false if no items exists
	Next() bool
}
