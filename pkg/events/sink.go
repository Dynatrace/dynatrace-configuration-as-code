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
	"fmt"
	"github.com/spf13/afero"
	"os"
)

type Sink interface {
	Write([]byte) error

	Close() error
}
type FileSink struct {
	file afero.File
}

func NewFileSink(fs afero.Fs, filename string) (*FileSink, error) {
	f, err := fs.OpenFile(filename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("could not open file: %w", err)
	}
	return &FileSink{file: f}, nil
}

func (fs *FileSink) Write(buf []byte) error {
	if _, err := fs.file.Write(buf); err != nil {
		return err
	}

	if _, err := fs.file.Write([]byte("\n")); err != nil {
		return err
	}

	return nil
}

func (fs *FileSink) Close() error {
	return fs.file.Close()
}
