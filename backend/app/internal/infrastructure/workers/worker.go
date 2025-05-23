package workers

import (
	"context"
	"log/slog"
	"sync"

	"skilly/internal/infrastructure/dependencies"
)

type Worker struct {
	deps       *dependencies.Dependencies
	runFunc    func(context.Context, *dependencies.Dependencies)
	name       string
	wg         sync.WaitGroup
	cancelFunc context.CancelFunc
}

func NewWorker(name string, runFunc func(context.Context, *dependencies.Dependencies), deps dependencies.Dependencies) *Worker {
	deps.Logger = deps.Logger.With(slog.String("worker", name))
	return &Worker{
		deps:       &deps,
		name:       name,
		runFunc:    runFunc,
	}
}

func (w *Worker) Start() {
	ctx, cancel := context.WithCancel(context.Background())
	w.cancelFunc = cancel

	w.wg.Add(1)
	go func () {
		defer w.wg.Done()
		w.deps.Logger.Info("worker started")
		w.runFunc(ctx, w.deps)
		w.deps.Logger.Info("worker exiting")
	}()
}

func (w *Worker) Stop() {
	w.deps.Logger.Info("Worker: stop signal received.")
	if w.cancelFunc != nil {
		w.cancelFunc()
	}
	w.wg.Wait()
	w.deps.Logger.Info("Worker: worker stopped gracefully.")
}
