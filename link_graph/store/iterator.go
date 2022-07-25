package store

import "Web_Crawler/link_graph/graph"

type linkIterator struct {
	s *InMemoryGraph

	links    []*graph.Link
	curIndex int
}

func (i *linkIterator) Next() bool {
	if i.curIndex >= len(i.links) {
		return false
	}
	i.curIndex++
	return true
}

func (i linkIterator) Link() *graph.Link {
	i.s.mu.RLock()
	link := new(graph.Link)
	*link = *i.links[i.curIndex-1]
	i.s.mu.Unlock()
	return link
}

// Error implements graph.LinkIterator.
func (i *linkIterator) Error() error {
	return nil
}

// Close implements graph.LinkIterator.
func (i *linkIterator) Close() error {
	return nil
}

type edgeIterator struct {
	s        *InMemoryGraph
	edges    []*graph.Edge
	curIndex int
}

func (e edgeIterator) Next() bool {

	if e.curIndex >= len(e.edges) {
		return false
	}
	e.curIndex++
	return true
}

func (e edgeIterator) Error() error {
	return nil
}

func (e edgeIterator) Close() error {
	return nil
}

func (e edgeIterator) Edge() *graph.Edge {
	e.s.mu.RLock()
	edge := new(graph.Edge)
	*edge = *e.edges[e.curIndex-1]
	e.s.mu.Unlock()
	return edge
}
