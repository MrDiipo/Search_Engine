package pipeline

// Payload is implemented by values that can be sent through a pipeline
type Payload interface {
	Clone() Payload
	MarkAsProcessed()
}
