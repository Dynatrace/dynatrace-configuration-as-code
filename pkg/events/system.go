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

package events

import (
	"context"
	"errors"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/log"
	"reflect"
	"sync"
	"time"
)

type Event interface {
	CorrelationId() string
	Time() time.Time
}
type EventType reflect.Type

type EventSystem interface {
	Send(event Event)
	Start()
	Terminate()
}

type eventSystem struct {
	queue         chan Event
	subscriberMap map[EventType][]*subscriber
	subscribers   []*subscriber
	wg            sync.WaitGroup
}

func WithSubscribers(subscribers ...Subscriber) func(*eventSystem) {
	return func(eq *eventSystem) {
		eq.registerSub(createSubscribers(subscribers))
	}
}

type contextKey struct{}

func NewContext(ctx context.Context, e EventSystem) context.Context {
	return context.WithValue(ctx, contextKey{}, e)
}

func NewFromContextOrDiscard(ctx context.Context) EventSystem {
	v := ctx.Value(contextKey{})
	if v == nil {
		return &DiscardEventSystem{}
	}
	switch v := v.(type) {
	case EventSystem:
		return v
	default:
		panic(fmt.Sprintf("unexpected value type for event system context key: %T", v))
	}
}
func New(bufferSize int, options ...func(event *eventSystem)) (*eventSystem, error) {
	if bufferSize <= 0 {
		return nil, errors.New("buffer size cannot be <=0")
	}
	instance := &eventSystem{
		queue:         make(chan Event, bufferSize),
		subscriberMap: make(map[EventType][]*subscriber),
	}
	for _, o := range options {
		o(instance)
	}
	return instance, nil
}

func Discard() *DiscardEventSystem {
	return &DiscardEventSystem{}
}

func (eq *eventSystem) Send(event Event) {
	eq.wg.Add(1)
	eq.queue <- event
}

func (eq *eventSystem) Start() {
	sem := make(chan struct{}, cap(eq.queue))
	go func() {
		for event := range eq.queue {
			sem <- struct{}{}
			func(event Event) {
				defer func() {
					<-sem
				}()
				eq.processEvent(event)
				eq.wg.Done()
			}(event)
		}
	}()
}

func (eq *eventSystem) processEvent(event Event) {
	subscribers := eq.subscriberMap[reflect.TypeOf(event)]
	for _, subscr := range subscribers {
		curr := subscr
		for curr != nil {
			event = curr.subscriber.Receive(event)
			curr = curr.next
		}
	}
}

func (eq *eventSystem) Terminate() {
	log.Info("Waiting for  event system to finish...")
	eq.wg.Wait()
	close(eq.queue)

	for _, subscr := range eq.subscribers {
		curr := subscr
		for curr != nil {
			curr.subscriber.Stop()
			curr = curr.next
		}
	}
	log.Info("Event system terminated.")
}

func (eq *eventSystem) registerSub(c *subscriber) {
	for _, etype := range c.subscriber.EventTypes() {
		ettype := reflect.TypeOf(etype)
		if eq.subscriberMap[ettype] == nil {
			eq.subscriberMap[ettype] = make([]*subscriber, 0)
		}
		eq.subscriberMap[ettype] = append(eq.subscriberMap[ettype], c)
	}
	eq.subscribers = append(eq.subscribers, c)
}

type DiscardEventSystem struct {
}

func (d DiscardEventSystem) Send(Event) {
	// no-op
}

func (d DiscardEventSystem) Start() {
	// no-op
}

func (d DiscardEventSystem) Terminate() {
	// no-op
}
