/*
 * @license
 * Copyright 2024 Dynatrace LLC
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package deployer

import (
	"errors"
	"sync"
)

type Runnable func(group *sync.WaitGroup, errCh chan error)

type Dispatcher struct {
	jobQueue   chan Runnable
	workerPool chan chan Runnable
	waitGroup  *sync.WaitGroup
	errCh      chan error
	maxWorkers int
	workers    []worker
}

// NewDispatcher creates a dispatcher that will use the specified amount of workers
// to dispatch its work loads. If maxWorkers is equal or less than 0, the dispatcher will dynamically
// create workers. Otherwise, there will be the specified fixed amount of workers available in the pool.
func NewDispatcher(maxWorkers int) *Dispatcher {
	return &Dispatcher{
		workerPool: make(chan chan Runnable),
		maxWorkers: maxWorkers,
		waitGroup:  &sync.WaitGroup{},
		errCh:      make(chan error),
		jobQueue:   make(chan Runnable),
	}
}

func (d *Dispatcher) Run() {
	if d.maxWorkers <= 0 {
		go d.dynamicDispatch()
	} else {
		for i := 0; i < d.maxWorkers; i++ {
			w := worker{
				workerPool: d.workerPool,
				jobs:       make(chan Runnable),
				waitGroup:  d.waitGroup,
				errCh:      d.errCh,
				quit:       make(chan bool),
			}
			d.workers = append(d.workers, w)
			w.start()
		}
		go d.dispatch()
	}
}

func (d *Dispatcher) Stop() {
	for _, w := range d.workers {
		w.stop()
	}
}

func (d *Dispatcher) AddJob(j Runnable) {
	d.waitGroup.Add(1)
	d.jobQueue <- j
}

func (d *Dispatcher) Wait() error {
	var ers []error
	waitForErrs := &sync.WaitGroup{}
	waitForErrs.Add(1)
	go func() {
		defer waitForErrs.Done()
		for err := range d.errCh {
			if err != nil {
				ers = append(ers, err)
			}
		}
	}()
	d.waitGroup.Wait()
	close(d.errCh)
	waitForErrs.Wait()
	return errors.Join(ers...)

}

func (d *Dispatcher) dispatch() {
	for job := range d.jobQueue {
		go func(job Runnable) {
			jobChannel := <-d.workerPool
			jobChannel <- job
		}(job)
	}
}

func (d *Dispatcher) dynamicDispatch() {
	for {
		job := <-d.jobQueue
		w := worker{
			workerPool: d.workerPool,
			jobs:       make(chan Runnable),
			waitGroup:  d.waitGroup,
			errCh:      d.errCh,
			quit:       make(chan bool),
			dynamic:    true,
		}
		w.start()
		go func(job Runnable) {
			jobChannel := <-d.workerPool
			jobChannel <- job
		}(job)
	}
}

type worker struct {
	workerPool chan chan Runnable
	jobs       chan Runnable
	waitGroup  *sync.WaitGroup
	errCh      chan error
	quit       chan bool
	dynamic    bool
}

func (w worker) start() {
	go func() {
		defer func() {
			if w.dynamic {
				close(w.jobs)
				w.quit <- true
			}
		}()

		for {
			w.workerPool <- w.jobs

			select {
			case job := <-w.jobs:
				job(w.waitGroup, w.errCh)
			case <-w.quit:
				return
			}
		}
	}()
}

func (w worker) stop() {
	go func() {
		w.quit <- true
	}()
}
