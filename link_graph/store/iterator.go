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
