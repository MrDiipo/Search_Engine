package textindexerapi

import (
	"Search_Engine/agnetaapis/textindexerapi/proto/generated"
	"Search_Engine/textindexer/index"
	"context"
	"github.com/golang/protobuf/ptypes/empty"
	"github.com/golang/protobuf/ptypes/timestamp"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
	"time"
)

var _ generated.TextIndexerServer = (*TextIndexerServer)(nil)

// TextIndexerServer provides a gRPC layer for indexing and querying documents
type TextIndexerServer struct {
	i index.Indexer
}

func NewTextIndexerServer(i index.Indexer) *TextIndexerServer {
	return &TextIndexerServer{i: i}
}

// Index inserts a new document to the index or updates the index entry for
// an existing document
func (t *TextIndexerServer) Index(ctx context.Context, req *generated.Document) (*generated.Document, error) {
	doc := &index.Document{
		LinkID:  uuidFromBytes(req.LinkId),
		URL:     req.Url,
		Title:   req.Title,
		Content: req.Content,
	}
	err := t.i.Index(doc)
	if err != nil {
		return nil, err
	}
	req.IndexedAt = timeToProto(doc.IndexedAt)
	return req, nil
}

func (t *TextIndexerServer) Search(req *generated.Query, server generated.TextIndexer_SearchServer) error {
	query := index.Query{
		Type:       index.QueryType(req.Type),
		Expression: req.Expression,
		Offset:     req.Offset,
	}
	it, err := t.i.Search(query)
	if err != nil {
		return err
	}
	// Send back the total document count
	countRes := &generated.QueryResult{
		Result: &generated.QueryResult_DocCount{
			DocCount: it.TotalCount(),
		},
	}
	if err = server.Send(countRes); err != nil {
		_ = it.Close()
		return err
	}
	// Start streaming
	for it.Next() {
		doc := it.Document()
		res := generated.QueryResult{
			Result: &generated.QueryResult_Doc{
				Doc: &generated.Document{
					LinkId:    doc.LinkID[:],
					Url:       doc.URL,
					Title:     doc.Title,
					Content:   doc.Content,
					IndexedAt: timeToProto(doc.IndexedAt),
				},
			},
		}
		if err = server.SendMsg(&res); err != nil {
			_ = it.Close()
			return err
		}
	}
	if err = it.Error(); err != nil {
		_ = it.Close()
		return err
	}
	return it.Close()
}

// UpdateScore updates the PageRank score for a document with the specified link ID.
func (t *TextIndexerServer) UpdateScore(ctx context.Context, req *generated.UpdateScoreRequest) (*emptypb.Empty, error) {
	linkID := uuidFromBytes(req.LinkId)
	return new(empty.Empty), t.i.UpdateScore(linkID, req.PageRankScore)
}

func uuidFromBytes(id []byte) uuid.UUID {
	if len(id) != 16 {
		return uuid.Nil
	}
	var dst uuid.UUID
	copy(dst[:], id)
	return dst
}

func timeToProto(t time.Time) *timestamp.Timestamp {
	//ts, _ := ptypes.TimestampProto(t)
	//timestamppb.New(t)
	return timestamppb.New(t)
}
