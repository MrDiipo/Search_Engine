package crawler

import (
	"Search_Engine/linkgraph/graph"
	"Search_Engine/pipeline"
	"Search_Engine/textindexer/index"
	"context"
	"github.com/google/uuid"
	"net/http"
	"time"
)

// URLGetter is implemented by objects that can perform HTTP GET requests.
type URLGetter interface {
	Get(url string) (*http.Response, error)
}

// PrivateNetworkDetector is implemented by object that can detect whether
// a host resolves to a private network address.
type PrivateNetworkDetector interface {
	IsPrivate(host string) (bool, error)
}

// Graph is implemented by object that can upsert links and edges into a link
// graph instance
type Graph interface {
	// UpsertLink creates a new link or updates an existing link.
	UpsertLink(link *graph.Link) error
	// UpsertEdge creates a new edge or updates an existing edge.
	UpsertEdge(edge *graph.Edge) error
	// RemoveStaleEdges removes any edge that originates from the
	// Specified link ID and was updated before the specified timestamp.
	RemoveStaleEdges(fromID uuid.UUID, updatedBefore time.Time) error
}

// Indexer is implement ed by objects that can index the contents of web-pages
// retrieved by the crawler pipeline
type Indexer interface {
	// Index inserts a new document to the index or updates the index entry
	// for an existing document
	Index(doc *index.Document) error
}

// Config encapsulates the configuration options for creating a new Crawler.
type Config struct {
	// A PrivateNetworkDetector instance
	PrivateNetworkDetector PrivateNetworkDetector
	// A URLGetter instance for fetching links.
	URLGetter URLGetter
	// A GraphUpdater instance for adding new link to the link graph.
	Graph Graph
	// A TextIndexer instance for indexing the content of each retrieved link.
	Indexer Indexer
	// The number of concurrent workers used for retrieving links
	FetchWorkers int
}

type Crawler struct {
	p *pipeline.Pipeline
}

// NewCrawler returns a new crawler instance.
func NewCrawler(cfg Config) *Crawler {
	return &Crawler{
		p: assembleCrawlerPipeline(cfg),
	}
}

// assembleCrawlerPipeline creates the various stages of a crawler pipeline
// using the options in cfg and assembles them into a pipeline instance.
func assembleCrawlerPipeline(cfg Config) *pipeline.Pipeline {
	return pipeline.New(
		pipeline.FixedWorkerPool(
			newLinkFetcher(cfg.URLGetter, cfg.PrivateNetworkDetector),
			cfg.FetchWorkers,
		),
		pipeline.NewFIFO(newLinkExtractor(cfg.PrivateNetworkDetector)),
		pipeline.NewFIFO(newTextExtractor()),
		pipeline.Broadcast(
			newGraphUpdater(cfg.Graph),
			newTextIndexer(cfg.Indexer),
		),
	)
}

// Crawl iterates linkIt and send each link through the crawler pipeline
// returning the total count of links that went through the pipeline.
func (c *Crawler) Crawl(ctx context.Context, linkIt graph.LinkIterator) (int, error) {
	sink := new(countingSink)
	err := c.p.Process(ctx, &linkSource{linkIt: linkIt}, sink)
	return sink.getCount(), err
}

type linkSource struct {
	linkIt graph.LinkIterator
}

func (ls *linkSource) Error() error {
	return ls.linkIt.Error()
}

func (ls *linkSource) Next(ctx context.Context) bool {
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
	// The broadcast split-stage sends out two payloads for each incoming link
	// so we need to divide the total count by 2.
	return s.count / 2
}
