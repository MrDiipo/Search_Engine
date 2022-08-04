package pipeline

import (
	"context"
	"golang.org/x/xerrors"
)

type workerParams struct {
	stage int
	inCh  chan Payload
	outCh chan<- Payload
	errCh chan<- error
}

func (w workerParams) StageIndex() int {
	return w.stage
}

func (w workerParams) Input() <-chan Payload {
	return w.inCh
}

func (w workerParams) Output() chan<- Payload {
	return w.outCh
}

func (w workerParams) Error() chan<- error {
	return w.errCh
}

func maybeEmitError(err error, errCh chan<- error) {
	select {
	case errCh <- err:
	default:
	}
}

func sourceWorker(ctx context.Context, source Source, outCh chan<- Payload, errCh chan<- error) {
	for source.Next(ctx) {
		payload := source.Payload()
		select {
		case outCh <- payload:
		case <-ctx.Done():
			return
		}
	}
	// check for errors
	if err := source.Error(); err != nil {
		wrappedErr := xerrors.Errorf("pipeline source: %w", err)
		maybeEmitError(wrappedErr, errCh)
	}
}

func sinkWorker(ctx context.Context, sink Sink, inCh <-chan Payload, errCh chan<- error) {
	for {
		select {
		case payload, ok := <-inCh:
			if !ok {
				return
			}
			if err := sink.Consume(ctx, payload); err != nil {
				wrappedErr := xerrors.Errorf("pipeline sink: %w", err)
				maybeEmitError(wrappedErr, errCh)
				return
			}
			payload.MarkAsProcessed()
		case <-ctx.Done():
			return
		}
	}
}
