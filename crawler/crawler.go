package crawler

import (
	"Search_Engine/linkgraph/graph"
	"Search_Engine/pipeline"
	"context"
	"net/http"
)

// URLGetter is implemented by objects that can perform HTTP GET requests.
type URLGetter interface {
	Get(url string) (*http.Response, error)
}

// PrivateNetworkDetector is implemented by object that can detect whether
// a host resolves to a private network address.
type PrivateNetworkDetector interface {
	isPrivate(host string) (bool, error)
}

type linkSource struct {
	linkIt graph.LinkIterator
}

func (ls *linkSource) Error() error {
	return ls.linkIt.Error()
}

func (ls *linkSource) Next() bool {
	return ls.linkIt.Next()
}

func (ls *linkSource) Payload() pipeline.Payload {
	link := ls.linkIt.Link()
	p := payloadPool.Get().(*crawlerPayload)

	p.LinkID = link.ID
	p.URL = link.URL
	p.RetrievedAt = link.RetrievedAt
	return p
}

type countingSink struct {
	count int
}

func (s *countingSink) Consume(_ context.Context, p pipeline.Payload) error {
	s.count++
	return nil
}

func (s *countingSink) getCount() int {
	return s.count / 2
}
