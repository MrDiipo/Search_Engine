package graph

import "golang.org/x/xerrors"

var (
	// ErrNotFound is returned when a link or edge lookup fails
	ErrNotFound = xerrors.New("not found")

	ErrUnknownEdgeLinks = xerrors.New("unknown source and/or destination or edge")
)
