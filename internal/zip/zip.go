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

package zip

import (
	"archive/zip"
	"fmt"
	"github.com/dynatrace/dynatrace-configuration-as-code/v2/internal/multierror"
	"github.com/spf13/afero"
	"io"
	"path/filepath"
)

func Create(fs afero.Fs, zipFileName string, files []string, preservePath bool) error {
	zipFile, err := fs.Create(zipFileName)
	if err != nil {
		return err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	var errs error
	for _, f := range files {
		err = addFileToZip(fs, zipWriter, f, preservePath)
		if err != nil {
			errs = multierror.New(errs, fmt.Errorf("unable to add %s file to archive %s: %w", f, zipFileName, err))
		}
	}
	return errs
}
func addFileToZip(fs afero.Fs, zipWriter *zip.Writer, file string, preservePath bool) error {
	fileToZip, err := fs.Open(file)
	if err != nil {
		return err
	}
	defer fileToZip.Close()

	fileInfo, err := fileToZip.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(fileInfo)
	if err != nil {
		return err
	}

	if preservePath {
		header.Name = file
	} else {
		header.Name = filepath.Base(file)
	}

	zippedFile, err := zipWriter.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(zippedFile, fileToZip)
	if err != nil {
		return err
	}

	return nil
}
