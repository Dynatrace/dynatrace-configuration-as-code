//go:build unit

/**
 * @license
 * Copyright 2020 Dynatrace LLC
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

package concurrency

import (
	"gotest.tools/assert"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewLimiter(t *testing.T) {
	limiter := NewLimiter(47)

	capacity := cap(limiter.waitChan)
	length := len(limiter.waitChan)

	assert.Equal(t, capacity, 47, "Capacity is not correct")
	assert.Equal(t, length, 0, "Limiter should be empty")
}

func TestCallbackExecutions(t *testing.T) {
	tests := []struct {
		name           string
		cap, callbacks int
	}{
		{
			"More callbacks than capacity (sequential)",
			1,
			100,
		},
		{
			"More callbacks than capacity (parallel)",
			100,
			10000,
		},
		{
			"More capacity than callbacks",
			10000,
			10,
		},
		{
			"Same amount",
			10,
			10,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			limiter := NewLimiter(test.cap)

			wg := sync.WaitGroup{}
			wg.Add(test.callbacks)

			counter := int32(test.callbacks)

			for i := 0; i < test.callbacks; i++ {

				limiter.Execute(func() {
					atomic.AddInt32(&counter, -1)

					wg.Done()
				})
			}

			wg.Wait()

			assert.Equal(t, int32(0), counter, "Counter should be 0")
		})
	}
}

// tests that callbacks are actually started in parallel.
func TestExecutionsAreActuallyParallel(t *testing.T) {
	limiter := NewLimiter(1000)

	m := sync.Mutex{} // mutex to lock children to not run
	m.Lock()

	wgPre := sync.WaitGroup{}  // started callbacks
	wgPost := sync.WaitGroup{} // finished callbacks

	for i := 0; i < 10; i++ {
		wgPre.Add(1)
		wgPost.Add(1)
		limiter.Execute(func() {
			wgPre.Done() // goroutine started

			m.Lock() // wait for signal to 'do stuff'
			m.Unlock()

			wgPost.Done() // goroutine end
		})
	}

	wgPre.Wait() // wait all goroutines started
	assert.Equal(t, len(limiter.waitChan), 10, "goroutines should be running")

	m.Unlock() // unlock to run all goroutines
}

func TestLimiterWithZeroLimit(t *testing.T) {
	limiter := NewLimiter(0)
	wg := sync.WaitGroup{}
	wg.Add(1)
	limiter.ExecuteBlocking(func() {
		wg.Done()
	})
	wg.Wait()
}

func TestExecuteBlockingDoesNotReturnImmediately(t *testing.T) {
	limiter := NewLimiter(1) // 1 callback allowed, so we set up 2 callbacks to test that they have been called after one another

	firstCallbackWaitUntilTestSetupIsComplete := sync.Mutex{} // block the first callback until the test setup is done
	firstCallbackWaitUntilTestSetupIsComplete.Lock()

	firstCallbackCalled := atomic.Bool{}
	secondCallbackDone := atomic.Bool{}

	wgBothGoroutinesStarted := sync.WaitGroup{}
	wgBothGoroutinesStarted.Add(2)

	wgBothGoroutinesDone := sync.WaitGroup{}
	wgBothGoroutinesDone.Add(2)

	wgFirstGoroutineStarted := sync.WaitGroup{}
	wgFirstGoroutineStarted.Add(1)

	go func() {
		limiter.ExecuteBlocking(func() {
			wgFirstGoroutineStarted.Done() // allow second goroutine to start
			wgBothGoroutinesStarted.Done()

			firstCallbackWaitUntilTestSetupIsComplete.Lock() // blocking wait for test to start

			time.Sleep(time.Second)
			firstCallbackCalled.Store(true) // store true to verify in second goroutine that this one is done
		})

		wgBothGoroutinesDone.Done()
	}()

	go func() {
		wgFirstGoroutineStarted.Wait() // continue after first goroutine is inside the Execute block
		wgBothGoroutinesStarted.Done()

		limiter.ExecuteBlocking(func() {})
		assert.Equal(t, firstCallbackCalled.Load(), true)

		secondCallbackDone.Store(true)
		wgBothGoroutinesDone.Done()
	}()

	wgBothGoroutinesStarted.Wait()

	firstCallbackWaitUntilTestSetupIsComplete.Unlock()

	wgBothGoroutinesDone.Wait()
	assert.Equal(t, secondCallbackDone.Load(), true)
}
