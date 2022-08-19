package crawler

import (
	"Search_Engine/pipeline"
	"Search_Engine/textindexer/index"
	"context"
	"time"
)

type textIndexer struct {
	indexer index.Indexer
}

func newTextIndexer(indexer index.Indexer) *textIndexer {
	return &textIndexer{
		indexer: indexer,
	}
}

func (t *textIndexer) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)

	doc := &index.Document{
		LinkID:    payload.LinkID,
		URL:       payload.URL,
		Title:     payload.Title,
		Content:   payload.TextContent,
		IndexedAt: time.Now(),
	}

	if err := t.indexer.Index(doc); err != nil {
		return nil, err
	}
	return p, nil
}
