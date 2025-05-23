package workers

import (
	"skilly/internal/infrastructure/dependencies"
	"sync"
)

func GetWorkers(deps *dependencies.Dependencies) []Worker {
	return []Worker{
		*NewWorker("profile-image-checker", ProfileImageChecker, *deps),
	}
}

type WorkerManager struct {
	workers []Worker
}

func NewWorkerManager(deps *dependencies.Dependencies) *WorkerManager {
	return &WorkerManager{
		workers: GetWorkers(deps),
	}
}

func (m *WorkerManager) Start() {
	wg := sync.WaitGroup{}
	for i := range m.workers {
		worker := &m.workers[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.Start()
		}()
	}
	wg.Wait()
}

func (m *WorkerManager) Stop() {
	wg := sync.WaitGroup{}
	for i := range m.workers {
		worker := &m.workers[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			worker.Stop()
		}()
	}
	wg.Wait()
}