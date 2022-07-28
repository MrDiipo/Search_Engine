package pipeline

import "context"

type Processor interface {
	// Process operates on the input payload and returns a new payload
	// to be forwarded to the next pipeline stage. Processors may also opt to
	// prevent the  payload from reaching the rest of the pipeline by returning
	// a nil payload value instead.
	Process(ctx context.Context, payload Payload) (Payload, error)
}
