package pipeline

import (
	"context"
	"golang.org/x/xerrors"
	"sync"
)

// fifo processes payloads sequentially thereby maintaining their order
type fifo struct {
	proc Processor
}

// NewFIFO returns a StageRunner that processes incoming payloads in a
// first-in-first-out Fashion. Each input is passed to the specified processor
// and its output is emitted to the next stage
func NewFIFO(proc Processor) StageRunner {
	return fifo{proc: proc}
}

// Run implements StageRunner
func (f fifo) Run(ctx context.Context, params StageParams) {
	for {
		select {
		case <-ctx.Done():
			return
		case payloadIn, ok := <-params.Input():
			if !ok {
				return
			}
			payloadOut, err := f.proc.Process(ctx, payloadIn)
			if err != nil {
				wrappedErr := xerrors.Errorf("pipeline stage %d: %w", params.StageIndex(), err)
				maybeEmitError(wrappedErr, params.Error())
				return
			}
			// if processor did not output a payload for the next stage,
			// there is nothing we need to do
			if payloadOut == nil {
				payloadIn.MarkAsProcessed()
				continue
			}
			// output processed data
			select {
			case params.Output() <- payloadOut:
			case <-ctx.Done():
				return
			}
		}
	}
}

type fixedWorkerPool struct {
	fifos []StageRunner
}

// FixedWorkerPool returns a StageRunner that spins up a pool containing
// numWorkers to process incoming payloads in parallel and emit their outputs
// to the next stage
func FixedWorkerPool(proc Processor, numWorkers int) StageRunner {
	if numWorkers <= 0 {
		panic("FixedWorkerPool: numWorkers must be > 0")
	}
	fifos := make([]StageRunner, numWorkers)
	for i := 0; i < len(fifos); i++ {
		fifos[i] = NewFIFO(proc)
	}
	return &fixedWorkerPool{fifos: fifos}
}

// Run implements StageRunner
func (f *fixedWorkerPool) Run(ctx context.Context, params StageParams) {
	var wg sync.WaitGroup

	for i := 0; i < len(f.fifos); i++ {
		wg.Add(1)
		go func(fifoIndex int) {
			f.fifos[fifoIndex].Run(ctx, params)
			wg.Done()
		}(i)
	}
	wg.Wait()
}

type dynamicMakerPool struct {
	proc      Processor
	tokenPool chan struct{}
}

// DynamicWorkerPools returns a StageRunner that maintains a dynamic worker pool
// that can scale up to a maxWorkers for processing incoming inputs in parallel and
// emitting their outputs to the next stage
func DynamicWorkerPools(proc Processor, maxWorkers int) StageRunner {
	if maxWorkers <= 0 {
		panic("DynamicWorkerPool: maxWorkers must be > 0")
	}
	tokenPool := make(chan struct{}, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		tokenPool <- struct{}{}
	}
	return &dynamicMakerPool{proc: proc, tokenPool: tokenPool}
}

// Run implements StageRunner
func (d dynamicMakerPool) Run(ctx context.Context, params StageParams) {
stop:
	for {
		select {
		case <-ctx.Done():
			// Asked to cleanly shut down
			break stop
		case payloadIn, ok := <-params.Input():
			if !ok {
				break stop
			}
			var token struct{}
			select {
			case token = <-d.tokenPool:
			case <-ctx.Done():
				break stop
			}
			go func(payloadIn Payload, token struct{}) {
				defer func() { d.tokenPool <- token }()
				payloadOut, err := d.proc.Process(ctx, payloadIn)
				if err != nil {
					wrappedErr := xerrors.Errorf("pipeline stage %d: %w", params.StageIndex(), err)
					maybeEmitError(wrappedErr, params.Error())
					return
				}
				// If the processor did not output a payload for the
				// next stage there is nothing we need to be
				if payloadOut == nil {
					payloadIn.MarkAsProcessed()
					return
				}
				// Output processed data
				select {
				case params.Output() <- payloadOut:
				case <-ctx.Done():
				}
			}(payloadIn, token)
		}
		// wait for all workers to exit by trying to empty the token pool
		for i := 0; i < cap(d.tokenPool); i++ {
			<-d.tokenPool
		}
	}
}

type broadcast struct {
	fifos []StageRunner
}

func Broadcast(proc ...Processor) StageRunner {
	if len(proc) == 0 {
		panic("Broadcast : at least one processor must be specified")
	}
	fifos := make([]StageRunner, len(proc))
	for i, p := range proc {
		fifos[i] = NewFIFO(p)
	}
	return &broadcast{
		fifos: fifos,
	}
}

func (b broadcast) Run(ctx context.Context, params StageParams) {
	var wg sync.WaitGroup
	var inCh = make([]chan Payload, len(b.fifos))
	for i := 0; i < len(b.fifos); i++ {
		wg.Add(1)
		inCh[i] = make(chan Payload)
		go func(fifoIndex int) {
			fifoParams := &workerParams{
				stage: params.StageIndex(),
				inCh:  inCh[fifoIndex],
				outCh: params.Output(),
				errCh: params.Error(),
			}
			b.fifos[fifoIndex].Run(ctx, fifoParams)
			wg.Done()
		}(i)
	}
done:
	for {
		select {
		case <-ctx.Done():
			break done
		case payload, ok := <-params.Input():
			if !ok {
				break done
			}
			for i := len(b.fifos) - 1; i >= 0; i-- {
				var fifoPayload = payload
				if i != 0 {
					fifoPayload = payload.Clone()
				}
				select {
				case <-ctx.Done():
					break done
				case inCh[i] <- fifoPayload:
				}
			}
		}
	}
	// Close input channels and wait for FIFOs to exit
	for _, ch := range inCh {
		close(ch)
	}
	wg.Wait()
}
