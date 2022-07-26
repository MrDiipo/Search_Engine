package index

import (
	"github.com/google/uuid"
	"time"
)

const (
	QueryTypeMatch QueryType = iota
	QueryTypePhrase
)

type QueryType uint8

type Query struct {
	Type       QueryType
	Expression string
	Offset     string
}

type Iterator interface {
	// Close the iterator amd release any allocated  resources
	Close() error
	// Next loads the next document matching the search query.
	// It returns false if no more documents are available
	Next() bool
	// Error returns the last error encountered by the iterator
	Error() error
	// Document returns the current document from the results
	Document() *Document
	// TotalCount returns the approximate number of search results
	TotalCount() uint64
}

type Document struct {
	LinkID uuid.UUID

	URL     string
	Title   string
	Content string

	IndexedAt time.Time
	PageRank  float64
}

type Indexer interface {
	Index(doc *Document) error
	FindByID(linkID uuid.UUID) (*Document, error)
	Search(query Query) (Iterator, error)
	UpdateScore(linkID uuid.UUID, score float64) error
}
