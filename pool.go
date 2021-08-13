package gopool

import (
	"errors"
	"sync"
	"sync/atomic"
	"time"
)

const (
	// DEFAULT_POOL_CAPACITY default pool capacity
	DEFAULT_POOL_CAPACITY = 10
	// DEFAULT_WORKER_NUM the number of default workers
	DEFAULT_WORKER_NUM = 1
)

// PoolStatus job status
type PoolStatus int

var (
	// PoolWaiting wating for process job
	PoolWaiting PoolStatus = 0
	// PoolRunning there are jobs is running
	// or pool not exit
	PoolRunning PoolStatus = 1
	// PoolExit all jobs was processed
	// and all goroutine already exit
	PoolExit PoolStatus = 2
	// PoolExiting some jobs are running
	// or some jobs are waiting for run
	PoolExiting PoolStatus = 3
	// ErrPoolExit the error of pool already exit
	ErrPoolExit = errors.New("pool already exit")
)

// Pool pool define
type Pool struct {
	// workers consume job from this channel
	jobs chan interface{}
	// job handler callback function
	handler func(job interface{})
	// pool exit signal
	poolExit chan struct{}
	// worker exit signal
	workerExit chan struct{}
	m          sync.RWMutex
	// pool status
	status PoolStatus
	// the number of running goroutine
	workers uint64
	// the number of running jobs
	runners uint64
	// pool capacity
	capacity uint64
	// max number of goroutine in pool
	maxActive     uint64
	exitCallback  func()
	panicCallback func(r interface{})
}

// New generate a new pool with pool size and handler function
func New(capacity, maxActive uint64, handler func(task interface{})) (*Pool, error) {
	if capacity == 0 {
		capacity = DEFAULT_POOL_CAPACITY
	}
	if maxActive == 0 {
		maxActive = capacity / 2
	}

	pool := &Pool{
		capacity:   capacity,
		jobs:       make(chan interface{}, capacity),
		poolExit:   make(chan struct{}),
		workerExit: make(chan struct{}),
		status:     PoolRunning,
		handler:    handler,
		maxActive:  maxActive,
	}
	pool.start()
	return pool, nil
}

// WithExitCallback set exit callback handler
func (p *Pool) WithExitCallback(handler func()) {
	p.exitCallback = handler
}

// WithPanicCallback set panic callback handler
func (p *Pool) WithPanicCallback(handler func(r interface{})) {
	p.panicCallback = handler
}

// WithMaxActive set max actived goroutine
func (p *Pool) WithMaxActive(maxActive uint64) {
	if maxActive == 0 {
		return
	}
	p.maxActive = maxActive
}

// Status return pool status
func (p *Pool) Status() PoolStatus {
	p.m.RLock()
	defer p.m.RUnlock()
	return p.status
}

// Workers return pool workers
func (p *Pool) Workers() uint64 {
	return atomic.LoadUint64(&p.workers)
}

// MaxActive max actived goroutine in pool
func (p *Pool) MaxActive() uint64 {
	return p.maxActive
}

// Capacity return pool capacity
func (p *Pool) Capacity() uint64 {
	return p.capacity
}

// PenddingJobs the pendding jobs in pool
func (p *Pool) PenddingJobs() int {
	return len(p.jobs)
}

// RunningJobs the running jobs in pool
func (p *Pool) RunningJobs() uint64 {
	return atomic.LoadUint64(&p.runners)
}

// Add add new task into pool
func (p *Pool) Add(task interface{}) error {
	// if pool status is exit, stop add new task
	if p.Status() == PoolExit || p.Status() == PoolExiting {
		return ErrPoolExit
	}
	// add new task into jobs channel
	p.jobs <- task
	return nil
}

// Close stop add new task and wait all tasks are processed
func (p *Pool) Close() error {
	if p.Status() == PoolExit || p.Status() == PoolExiting {
		return ErrPoolExit
	}
	// send exit signal to notice stop add new task
	close(p.poolExit)
	return nil
}

// start a goroutine to waiting for recive job
// waiting for pool exit signal
func (p *Pool) start() {
	p.increaseWorker()
	go func() {
		for {
			select {
			case _, ok := <-p.poolExit:
				if !ok {
					// stop add new task
					p.setStatus(PoolExiting)
					// wait all task process finish
					p.waitTaskProcessed()
					// close job channel
					close(p.jobs)
					// close workers channel
					close(p.workerExit)
					// call exit callback func
					if p.exitCallback != nil {
						p.exitCallback()
					}
					p.setStatus(PoolExit)
					return
				}
			}
		}
	}()
}

// wait for all tasks already processed
func (p *Pool) waitTaskProcessed() {
	for {
		if p.RunningJobs() == 0 && p.PenddingJobs() == 0 {
			return
		}
	}
}

// start a worker to process job
func (p *Pool) startWorker(workerNum uint64) {
	d := 5 * time.Second
	ticker := time.NewTicker(d)
	defer ticker.Stop()
	defer func() {
		if r := recover(); r != nil {
			if p.panicCallback != nil {
				p.panicCallback(r)
			}
		}
	}()
	for {
		select {
		case <-ticker.C:
			// if this goroutine not used in some times then exit
			if p.Workers() > DEFAULT_WORKER_NUM {
				p.decreaseWorker(workerNum)
				return
			}
			ticker.Reset(d)
		case task, ok := <-p.jobs:
			p.increaseRunner()
			if !ok {
				// when channel is closed exit goroutine
				p.decreaseWorker(workerNum)
				return
			}
			p.handler(task)
			ticker.Reset(d)
			if p.Workers() < p.MaxActive() && p.PenddingJobs() > int(p.Capacity()/2) {
				p.increaseWorker()
			}
			p.decreaseRunner()
		}
	}
}

func (p *Pool) setStatus(status PoolStatus) {
	p.m.Lock()
	defer p.m.Unlock()
	p.status = status
}

// atomic add worker
func (p *Pool) increaseWorker() {
	workerNum := atomic.AddUint64(&p.workers, 1)
	go p.startWorker(workerNum)
}

// atomic delete worker
func (p *Pool) decreaseWorker(workerNum uint64) {
	atomic.AddUint64(&p.workers, ^uint64(0))
}

func (p *Pool) increaseRunner() {
	atomic.AddUint64(&p.runners, 1)
}

func (p *Pool) decreaseRunner() {
	atomic.AddUint64(&p.runners, ^uint64(0))
}
