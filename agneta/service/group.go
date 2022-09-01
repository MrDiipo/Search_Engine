package service

import (
	"context"
	"github.com/hashicorp/go-multierror"
	"golang.org/x/xerrors"
	"sync"
)

// Service describes a service for the Agneta Search Engine monolithic application
type Service interface {
	// Name returns the service name.
	Name() string
	// Run executes the service and blocks until the context get cancelled
	// or an error occurs.
	Run(ctx context.Context) error
}

// Group is a list of Service instances that can execute in parallel.
type Group []Service

// Run executes all Services instances in the group using the provided context.
// Calls to Run block until all Services have completed executing either because
// the context was cancelled or any of the services reported an error.
func (g Group) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}
	runCtx, cancelFn := context.WithCancel(ctx)
	defer cancelFn()

	var wg sync.WaitGroup
	errCh := make(chan error, len(g))
	wg.Add(len(g))
	for _, s := range g {
		go func(s Service) {
			defer wg.Done()
			if err := s.Run(runCtx); err != nil {
				errCh <- xerrors.Errorf("%s : %w", s.Name(), err)
				cancelFn()
			}
		}(s)
	}

	// Keep running until the run context gets cancelled; then wait for all
	// spanned service go-routines to exit
	<-ctx.Done()
	wg.Wait()
	// Collect and accumulate any reported errors.
	var err error
	close(errCh)
	for srvErr := range errCh {
		err = multierror.Append(err, srvErr)
	}
	return err
}
