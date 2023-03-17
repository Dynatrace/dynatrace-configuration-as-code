/**
 * @license
 * Copyright 2022 Dynatrace LLC
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
	"math"
	"sync"
)

// Limiter is used to limit a number of goroutines.
// Instead of starting the goroutines using `go func()...`, use `limiter.Execute()`
// After usage, close the limiter using `limiter.Close()`
//
// Limiter will not limit the number of goroutines, but the number of callbacks that are executed at once.
type Limiter struct {
	waitChan chan struct{}
}

var (
	_ sync.Locker = (*Limiter)(nil) // implement the Locker interface to let 'go vet' find copies by value that we don't want to have.
)

func (l *Limiter) Lock()   {}
func (l *Limiter) Unlock() {}

// NewLimiter creates a new limiter with the given amount of max concurrent running functions.
// Note, that if maxConcurrent <= 0 is equivalent of constructing a Limiter with maxConcurrent=MAX_INT
func NewLimiter(maxConcurrent int) *Limiter {
	if maxConcurrent <= 0 {
		return &Limiter{
			waitChan: make(chan struct{}, math.MaxInt),
		}
	}
	return &Limiter{
		waitChan: make(chan struct{}, maxConcurrent),
	}
}

// Execute runs the passed function in a goroutines.
// If the maximum number of goroutines is reached, it is waited until a previously started goroutine (using this limiter) is done.
// If the limiter is closed and more functions are started, Execute will panic.
func (l *Limiter) Execute(callback func()) {
	go func() {
		l.waitChan <- struct{}{}

		defer func() {
			<-l.waitChan
		}()

		callback()
	}()
}

// ExecuteBlocking runs the passed function blocking.
// If the maximum number of parallel running functions is reached, the function does not execute the callback and does not return
// until a slot is free.
func (l *Limiter) ExecuteBlocking(callback func()) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	l.Execute(func() {
		callback()
		wg.Done()
	})

	wg.Wait()
}

// Close cleans up the limiter. All running goroutines will be finished, but no new ones can be started.
// Closing the limiter multiple times will cause a panic
func (l *Limiter) Close() {
	close(l.waitChan)
}
