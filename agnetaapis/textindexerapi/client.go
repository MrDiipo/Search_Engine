package textindexerapi

import (
	"Search_Engine/agnetaapis/textindexerapi/proto/generated"
	"Search_Engine/textindexer/index"
	"context"
	"github.com/golang/protobuf/ptypes"
	"github.com/google/uuid"
	"golang.org/x/xerrors"
	"io"
)

// TextIndexerClient provides an API compatible with the index.Indexer interface
// for accessing text indexer instances exposed by a remote gRPC server.
type TextIndexerClient struct {
	ctx context.Context
	cli generated.TextIndexerClient
}

// NewTextIndexerClient returns a new client that implements a subset of the index.Indexer interface by
// delegating methods to an indexer instance exposed by a remote gRPC server.
func NewTextIndexerClient(ctx context.Context, rpcClient generated.TextIndexerClient) *TextIndexerClient {
	return &TextIndexerClient{ctx: ctx, cli: rpcClient}
}

// Index inserts a new document into the index or updates the index entry
// for an existing document
func (c *TextIndexerClient) Index(doc *index.Document) error {
	req := &generated.Document{
		LinkId:  doc.LinkID[:],
		Url:     doc.URL,
		Title:   doc.Title,
		Content: doc.Content,
	}
	res, err := c.cli.Index(c.ctx, req)
	if err != nil {
		return err
	}
	t := res.IndexedAt.AsTime()
	//  ptypes.Timestamp(res.IndexedAt.AsTime())
	doc.IndexedAt = t
	return nil
}

// UpdateScore updates the PageRank score for a document with the specified
// link ID.
func (c *TextIndexerClient) UpdateScore(linkID uuid.UUID, score float64) error {
	req := &generated.UpdateScoreRequest{
		LinkId:        linkID[:],
		PageRankScore: score,
	}
	_, err := c.cli.UpdateScore(c.ctx, req)
	return err
}

// Search the index for a particular query and return back a result iterator.
func (c *TextIndexerClient) Search(query index.Query) (index.Iterator, error) {
	ctx, cancelFn := context.WithCancel(c.ctx)
	req := &generated.Query{
		Type:       generated.Query_Type(query.Type),
		Expression: query.Expression,
		Offset:     query.Offset,
	}
	stream, err := c.cli.Search(ctx, req)
	if err != nil {
		cancelFn()
		return nil, err
	}
	// Read result count
	res, err := stream.Recv()
	if err != nil {
		cancelFn()
		return nil, err
	} else if res.GetDoc() != nil {
		cancelFn()
		return nil, xerrors.Errorf("expected server to report the result count before sending any documents")
	}
	return &resultIterator{
		total:    res.GetDocCount(),
		stream:   stream,
		cancelFn: cancelFn,
	}, nil

}

type resultIterator struct {
	total   uint64
	stream  generated.TextIndexer_SearchClient
	lastErr error
	next    *index.Document

	// A function to cancel the context to perform the streaming RPC.
	// It allows us to abort server-streaming calls from the client side
	cancelFn func()
}

// Close releases any resources associated with an iterator.
func (r *resultIterator) Close() error {
	r.cancelFn()
	return nil
}

func (r *resultIterator) Next() bool {
	res, err := r.stream.Recv()
	if err != nil {
		if err != io.EOF {
			r.lastErr = err
		}
		r.cancelFn()
		return false
	}
	resDoc := res.GetDoc()
	if resDoc == nil {
		r.cancelFn()
		r.lastErr = xerrors.Errorf("received nil document in search result list")
		return false
	}
	linkID := uuidFromBytes(resDoc.LinkId)
	t, err := ptypes.Timestamp(resDoc.IndexedAt)
	if err != nil {
		r.cancelFn()
		r.lastErr = xerrors.Errorf("unable to decode indexedAt attribute of document %q: %w", linkID, err)
		return false
	}

	r.next = &index.Document{
		LinkID:    linkID,
		URL:       resDoc.Url,
		Title:     resDoc.Title,
		Content:   resDoc.Content,
		IndexedAt: t,
	}
	return true
}

func (r *resultIterator) Error() error {
	return r.lastErr
}

func (r *resultIterator) Document() *index.Document {
	return r.next
}

func (r *resultIterator) TotalCount() uint64 {
	return r.total
}
