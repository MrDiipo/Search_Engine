package graph

import (
	"github.com/google/uuid"
	"time"
)

type Iterator interface {
	// Next advances the iterator. If no more items are available or an
	// error occurs, calls to Next() returns false.
	Next() bool

	// Error returns the last error encountered by the iterator
	Error() error

	// Close releases any resources associated with an iterator.
	Close() error
}

// LinkIterator is implemented by object that can iterate the graph links.
type LinkIterator interface {
	Iterator

	// Link returns the currently fetched link object
	Link() *Link
}

// EdgeIterator is implemented by object that can iterate the graph edges
type EdgeIterator interface {
	Iterator

	//Edge returns the currently fetched edge objects
	Edge() *Edge
}

type Link struct {
	ID          uuid.UUID
	URL         string
	RetrievedAt time.Time
}

type Edge struct {
	ID        uuid.UUID
	Src       uuid.UUID
	Dst       uuid.UUID
	UpdatedAt time.Time
}

type Graph interface {
	UpsertLink(link *Link) error
	FindLink(id uuid.UUID) (*Link, error)

	UpsertEdge(edge *Edge) error
	RemoveStaleEdges(fromID uuid.UUID, updateBefore time.Time) error

	Links(fromID, toID uuid.UUID, retrievedBefore time.Time) (LinkIterator, error)
	Edges(fromID, toID uuid.UUID, updatedBefore time.Time) (EdgeIterator, error)
}
