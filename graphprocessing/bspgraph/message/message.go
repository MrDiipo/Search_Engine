package message

// Message is implemented by types that can be processed bt a Queue.
type Message interface {
	// Type returns the type of this message
	Type() string
}

// Queue is implemented by types that can serve as a message queue
type Queue interface {
	// Close Cleanly shutdown the queue.
	Close() error
	// Enqueue inserts a message into the queue
	Enqueue(msg Message) error
	// PendingMessage returns true if the queue contains any message
	PendingMessage() bool
	// DiscardMessages drops all pending messages from the queue
	DiscardMessages() error
	// Messages returns an iterator for accessing the queued messages.
	Messages() Iterator
}

type Iterator interface {
	// Next advance the iterator so that the next message can be retrieved
	// via a call to Message(). If no more messages are available or an error
	// occurs, Next() returns false.
	Next() bool
	// Message returns the message currently pointed to by the iterator
	Message() Message
	// Error returns the last error the iterator encountered
	Error() error
}

// QueueFactory is a func that can create new Queue instances
type QueueFactory func() Queue
