package crawler

import (
	"Search_Engine/linkgraph/graph"
	"Search_Engine/pipeline"
	"context"
	"time"
)

type graphUpdater struct {
	updater graph.Graph
}

func newGraphUpdater(updater graph.Graph) *graphUpdater {
	return &graphUpdater{
		updater: updater,
	}
}

func (gu *graphUpdater) Process(ctx context.Context, p pipeline.Payload) (pipeline.Payload, error) {
	payload := p.(*crawlerPayload)

	src := &graph.Link{
		ID:          payload.LinkID,
		URL:         payload.URL,
		RetrievedAt: time.Now(),
	}

	if err := gu.updater.UpsertLink(src); err != nil {
		return nil, err
	}

	// Upsert discovered no-follow links without creating an edge
	for _, dstLink := range payload.NoFollowLinks {
		dst := &graph.Link{URL: dstLink}
		if err := gu.updater.UpsertLink(dst); err != nil {
			return nil, err
		}
	}

	// Upsert discovered links and create edges for them. Keep track of them
	// the current time so we can drop stale edges that have not been update after this loop
	removeEdgesOlderThan := time.Now()
	for _, dstLink := range payload.Links {
		dst := &graph.Link{URL: dstLink}
		if err := gu.updater.UpsertLink(src); err != nil {
			return nil, err
		}
		if err := gu.updater.UpsertEdge(&graph.Edge{Src: src.ID, Dst: dst.ID}); err != nil {
			return nil, err
		}
	}
	// Drop stale edges that were not touched while upserting the outgoing edges.
	if err := gu.updater.RemoveStaleEdges(src.ID, removeEdgesOlderThan); err != nil {
		return nil, err
	}
	return p, nil
}
