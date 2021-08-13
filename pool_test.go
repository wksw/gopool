package gopool

import (
	"sync"
	"testing"
)

func taskHandler(task interface{}) {
	return
}

func poolStatus(p *Pool, t *testing.T) {
	t.Logf("pool capacity:%d", p.Capacity())
	t.Logf("pool maxactive:%d", p.MaxActive())
	t.Logf("running jobs:%d", p.RunningJobs())
	t.Logf("running workers:%d", p.Workers())
	t.Logf("pending jobs:%d", p.PenddingJobs())
}

func TestPool(t *testing.T) {
	p, err := New(testCapacity, testMaxActive, taskHandler)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	p.WithExitCallback(func() {
		t.Log("pool exit")
		poolStatus(p, t)
	})
	var wg sync.WaitGroup
	for i := 0; i < 10000; i++ {
		wg.Add(1)
		p.Add(struct{}{})
		wg.Done()
	}
	wg.Wait()
	poolStatus(p, t)
	p.Close()
	// waiting for all task processed
	for {
		if p.Status() == PoolExit {
			return
		}
	}
}

func TestPoolWithNoTask(t *testing.T) {
	p, err := New(testCapacity, testMaxActive, taskHandler)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	p.Close()
	for {
		if p.Status() == PoolExit {
			return
		}
	}
}

func TestPoolWithExit(t *testing.T) {
	p, err := New(testCapacity, testMaxActive, taskHandler)
	if err != nil {
		t.Error(err.Error())
		t.FailNow()
	}
	p.Close()
	for {
		if p.Status() == PoolExit {
			break
		}
	}
	err = p.Add(struct{}{})
	if err != ErrPoolExit {
		t.Error("pool already exit, can not add task")
	}
}
