package pipeline

import (
	"context"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/xerrors"
	"sync"
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

type Pipeline struct {
	stages []StageRunner
}

// New returns a new pipeline instance where input payloads will traverse each
// one of the specified stages.
func New(stages ...StageRunner) *Pipeline {
	return &Pipeline{
		stages: stages,
	}
}

// Process reads from the contents of the specified source, sends them through the
// various stages of the pipeline and directs the results of the specified sink
// and returns any errors that may hae occurred.
//
// Calls to Process blocks until:
// - all data from the source has been processed OR
// - an error occurs OR
// - the supplied context expires
//
// It is safe to call Process concurrently with different sources and sinks
func (p *Pipeline) Process(ctx context.Context, source Source, sink Sink) error {
	var wg sync.WaitGroup
	pCtx, ctxCancelFn := context.WithCancel(ctx)

	// Allocate channels for wiring together the source, the pipeline stages
	// and the output sink. The output of the i_the stage is used as an input
	// for the i+1_the stage. We need to allocate one extra channel than the number
	// of stages, so we can wire the source/sink.
	stageCh := make([]chan Payload, len(p.stages)+1)
	errCh := make(chan error, len(p.stages)+2)
	for i := 0; i < len(stageCh); i++ {
		stageCh[i] = make(chan Payload)
	}

	// Start a  worker for each stage
	for i := 0; i < len(p.stages); i++ {
		wg.Add(1)
		go func(stageIndex int) {
			p.stages[stageIndex].Run(pCtx, &workerParams{
				stage: stageIndex,
				inCh:  stageCh[stageIndex],
				outCh: stageCh[stageIndex+1],
				errCh: errCh,
			})

			// Signal next stage that no more data is available.
			close(stageCh[stageIndex+1])
			wg.Done()
		}(i)
	}

	// Start source and sink workers
	wg.Add(2)
	go func() {
		sourceWorker(pCtx, source, stageCh[0], errCh)
		// Signal next stage that no more data s available
		close(stageCh[0])
		wg.Done()
	}()

	go func() {
		sinkWorker(pCtx, sink, stageCh[len(stageCh)-1], errCh)
		wg.Done()
	}()

	// Close the error channel once all workers exit.
	go func() {
		wg.Wait()
		close(errCh)
		ctxCancelFn()
	}()

	// Collect any emitted errors and wrap them in a multi-error
	var err error
	for pErr := range errCh {
		err = multierror.Append(err, pErr)
		ctxCancelFn()
	}
	return err
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

func maybeEmitError(err error, errCh chan<- error) {
	select {
	case errCh <- err:
	default:
	}
}
