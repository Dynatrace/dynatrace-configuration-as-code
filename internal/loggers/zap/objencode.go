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

package zap

import (
	"go.uber.org/zap/zapcore"
	"sync"
	"time"
)

type concurrentMapObjectEncoder struct {
	mu  sync.RWMutex
	moe *zapcore.MapObjectEncoder
}

func (c *concurrentMapObjectEncoder) Fields() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.moe.Fields
}
func (c *concurrentMapObjectEncoder) AddArray(key string, marshaler zapcore.ArrayMarshaler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.moe.AddArray(key, marshaler)
}

func (c *concurrentMapObjectEncoder) AddObject(key string, marshaler zapcore.ObjectMarshaler) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.moe.AddObject(key, marshaler)
}

func (c *concurrentMapObjectEncoder) AddBinary(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddBinary(key, value)
}

func (c *concurrentMapObjectEncoder) AddByteString(key string, value []byte) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddByteString(key, value)
}

func (c *concurrentMapObjectEncoder) AddBool(key string, value bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddBool(key, value)
}

func (c *concurrentMapObjectEncoder) AddComplex128(key string, value complex128) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddComplex128(key, value)
}

func (c *concurrentMapObjectEncoder) AddComplex64(key string, value complex64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddComplex64(key, value)
}

func (c *concurrentMapObjectEncoder) AddDuration(key string, value time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddDuration(key, value)
}

func (c *concurrentMapObjectEncoder) AddFloat64(key string, value float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddFloat64(key, value)
}

func (c *concurrentMapObjectEncoder) AddFloat32(key string, value float32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddFloat32(key, value)
}

func (c *concurrentMapObjectEncoder) AddInt(key string, value int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddInt(key, value)
}

func (c *concurrentMapObjectEncoder) AddInt64(key string, value int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddInt64(key, value)
}

func (c *concurrentMapObjectEncoder) AddInt32(key string, value int32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddInt32(key, value)
}

func (c *concurrentMapObjectEncoder) AddInt16(key string, value int16) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddInt16(key, value)
}

func (c *concurrentMapObjectEncoder) AddInt8(key string, value int8) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddInt8(key, value)
}

func (c *concurrentMapObjectEncoder) AddString(key, value string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddString(key, value)
}

func (c *concurrentMapObjectEncoder) AddTime(key string, value time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddTime(key, value)
}

func (c *concurrentMapObjectEncoder) AddUint(key string, value uint) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddUint(key, value)
}

func (c *concurrentMapObjectEncoder) AddUint64(key string, value uint64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddUint64(key, value)
}

func (c *concurrentMapObjectEncoder) AddUint32(key string, value uint32) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddUint32(key, value)
}

func (c *concurrentMapObjectEncoder) AddUint16(key string, value uint16) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddUint16(key, value)
}

func (c *concurrentMapObjectEncoder) AddUint8(key string, value uint8) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddUint8(key, value)
}

func (c *concurrentMapObjectEncoder) AddUintptr(key string, value uintptr) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.AddUintptr(key, value)
}

func (c *concurrentMapObjectEncoder) AddReflected(key string, value interface{}) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.moe.AddReflected(key, value)
}

func (c *concurrentMapObjectEncoder) OpenNamespace(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.moe.OpenNamespace(key)
}
