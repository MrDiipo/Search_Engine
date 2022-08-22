package message

import "sync"

// inMemoryQueue implements the a queue that stores messages in memory
// Messages can be enqueued concurrently but the returned iterator is not safe for
// concurrent access
type inMemoryQueue struct {
	mu         sync.Mutex
	msgs       []Message
	latchedMsg Message
}

func NewInMemoryQueue() Queue {
	return new(inMemoryQueue)
}

func (q *inMemoryQueue) Enqueue(msg Message) error {
	q.mu.Lock()
	q.msgs = append(q.msgs, msg)
	q.mu.Unlock()
	return nil
}

func (q *inMemoryQueue) PendingMessage() bool {
	q.mu.Lock()
	pending := len(q.msgs) != 0
	q.mu.Unlock()
	return pending
}

func (q *inMemoryQueue) DiscardMessages() error {
	q.mu.Lock()
	q.msgs = q.msgs[:0]
	q.latchedMsg = nil
	q.mu.Unlock()
	return nil
}

func (q *inMemoryQueue) Close() error {
	return nil
}

func (q *inMemoryQueue) Messages() Iterator {
	return q
}

func (q *inMemoryQueue) Next() bool {
	q.mu.Lock()
	qLen := len(q.msgs)
	if qLen == 0 {
		q.mu.Unlock()
		return false
	}
	// Dequeue messages from the tail of the queue
	q.latchedMsg = q.msgs[qLen-1]
	q.msgs = q.msgs[:qLen-1]
	q.mu.Unlock()
	return true
}

func (q *inMemoryQueue) Message() Message {
	q.mu.Lock()
	msg := q.latchedMsg
	q.mu.Unlock()
	return msg
}

func (q *inMemoryQueue) Error() error {
	return nil
}
