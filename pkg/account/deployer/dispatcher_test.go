//go:build unit

/*
 * @license
 * Copyright 2023 Dynatrace LLC
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
	"fmt"
	"github.com/stretchr/testify/assert"
	"sync"
	"testing"
	"time"
)

func TestDispatcher(t *testing.T) {
	mockJobFunc := func(group *sync.WaitGroup, errCh chan error) {
		time.Sleep(100 * time.Millisecond)
		group.Done()
	}

	job := mockJobFunc
	dispatcher := NewDispatcher(2)
	dispatcher.Run()

	dispatcher.AddJob(job)
	err := dispatcher.Wait()
	assert.NoError(t, err, "No errors should be returned")
}

func TestDispatcher_Err(t *testing.T) {
	mockJobFunc := func(group *sync.WaitGroup, errCh chan error) {
		time.Sleep(100 * time.Millisecond)
		errCh <- fmt.Errorf("an error occured")
		group.Done()
	}

	job := mockJobFunc
	dispatcher := NewDispatcher(2)

	dispatcher.Run()

	dispatcher.AddJob(job)
	err := dispatcher.Wait()
	assert.Error(t, err)
}

func TestDispatcherConcurrency(t *testing.T) {
	mockJobFunc := func(group *sync.WaitGroup, errCh chan error) {
		time.Sleep(100 * time.Millisecond)
		group.Done()
	}

	job := mockJobFunc
	dispatcher := NewDispatcher(1)

	dispatcher.Run()

	numJobs := 5
	for i := 0; i < numJobs; i++ {
		dispatcher.AddJob(job)
	}

	err := dispatcher.Wait()
	assert.NoError(t, err, "No errors should be returned")
}
