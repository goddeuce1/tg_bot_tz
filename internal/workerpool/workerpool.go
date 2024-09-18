package workerpool

import (
	"context"
	"github.com/sirupsen/logrus"
	"sync"
)

type JobRunner interface {
	Run(ctx context.Context, gracefulWg *sync.WaitGroup)
	AddJob(job Job)
}

type Job func(ctx context.Context) error

type WorkerPool struct {
	logger       *logrus.Logger
	workersCount uint32
	queue        chan Job
}

func NewWorkerPool(logger *logrus.Logger, workersCount uint32, queue chan Job) *WorkerPool {
	return &WorkerPool{
		logger:       logger,
		workersCount: workersCount,
		queue:        queue,
	}
}

func (pool *WorkerPool) Run(ctx context.Context, gracefulWg *sync.WaitGroup) {
	defer func() {
		pool.logger.Info("Shutting down worker pool")

		gracefulWg.Done()
	}()

	wg := &sync.WaitGroup{}
	for i := uint32(0); i < pool.workersCount; i++ {
		wg.Add(1)
		go func(ctx context.Context) {
			defer wg.Done()

			for {
				select {
				case <-ctx.Done():
					return

				case job, ok := <-pool.queue:
					if !ok {
						return
					}

					if err := job(ctx); err != nil {
						pool.logger.Errorf("failed to process job: %s", err)
					}
				}
			}
		}(ctx)
	}

	wg.Wait()
}

func (pool *WorkerPool) AddJob(job Job) {
	pool.queue <- job
}
