package cockroachdb

import (
	"Web_Crawler/link_graph/graph"
	"database/sql"
	"golang.org/x/xerrors"
)

// linkIterator is a graph.LinkIterator implementation for the cdb graph
type linkIterator struct {
	rows        *sql.Rows
	lastErr     error
	latchedLink *graph.Link
}

func (i *linkIterator) Next() bool {
	if i.lastErr != nil || !i.rows.Next() {
		return false
	}

	l := new(graph.Link)
	i.lastErr = i.rows.Scan(&l.ID, &l.URL, &l.RetrievedAt)
	if i.lastErr != nil {
		return false
	}
	l.RetrievedAt = l.RetrievedAt.UTC()
	i.latchedLink = l
	return true
}

func (i *linkIterator) Error() error {
	return i.lastErr
}

func (i *linkIterator) Close() error {
	err := i.rows.Close()
	if err != nil {
		return xerrors.Errorf("link iterator: %w", err)
	}
	return nil
}

func (i *linkIterator) Link() *graph.Link {
	return i.latchedLink
}

// edgeIterator is a graph.EdgeIterator implementation for the cdb graph.
type edgeIterator struct {
	rows        *sql.Rows
	lastErr     error
	latchedEdge *graph.Edge
}
