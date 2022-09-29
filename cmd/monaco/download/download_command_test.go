// @license
// Copyright 2021 Dynatrace LLC
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build unit

package download

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/download"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"gotest.tools/assert"

	"io"
	"strings"
	"testing"
)

// TestInvalidCliCommands is a very basic test testing that invalid commands error.
// It is not the goal to test the exact message that cobra generates, except if we supply the message.
// Otherwise, we would run into issues upon upgrading.
// On th other hand, we could use the exact message to review the exact messages customers will see.
func TestInvalidCliCommands(t *testing.T) {
	tests := []struct {
		name                  string
		args                  string
		errorContainsExpected []string
	}{
		{
			"neither env or url provided",
			"",
			[]string{"either '--manifest' or '--url' has to be provided"},
		},
		{
			"both --url and --manifest provided",
			"--url test --manifest manifest.yaml --specific-environment test --project test --token test --",
			[]string{"none of the others can be"},
		},
		{
			"url without arg",
			"--url",
			[]string{"--url"},
		},
		{
			"--url is missing other required parameters",
			"--url test",
			[]string{"project", "token"},
		},
		{
			"--manifest is missing argument",
			"--manifest",
			[]string{"needs an argument"},
		},
		{
			"--manifest is missing other required parameters",
			"--manifest test.yaml",
			[]string{"specific-environment"},
		},
		{
			"unknown flag",
			"--test",
			[]string{"--test"},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)

			cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(strings.Split(test.args, " "))
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			// for all test cases there should be at least an error
			assert.Assert(t, err != nil, "there should be an error")

			// for most cases we can test the message in more detail
			for _, expected := range test.errorContainsExpected {
				assert.ErrorContains(t, err, expected)
			}

			// for testing not to forget adding expectations
			assert.Assert(t, len(test.errorContainsExpected) > 0, "no error conditions specified")
		})
	}
}

func TestValidCommands(t *testing.T) {
	tests := []struct {
		name  string
		args  string
		setup func(command *MockCommand)
	}{
		{
			"direct download no specific apis",
			"--url test --project test --token token",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{})
			},
		},
		{
			"direct download with specific apis (multiple flags)",
			"--url test --project test --token token --specific-api test --specific-api test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{"test", "test2"})
			},
		},
		{
			"direct download with specific apis (single flag)",
			"--url test --project test --token token --specific-api test,test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{"test", "test2"})
			},
		},
		{
			"direct download with specific apis (mixed flags)",
			"--url test --project test --token token --specific-api test,test2 --specific-api test3",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{"test", "test2", "test3"})
			},
		},
		{
			"manifest download no specific apis",
			"--manifest test.yaml --specific-environment test",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{})
			},
		},
		{
			"manifest download with specific apis (multiple flags)",
			"--manifest test.yaml --specific-environment test --specific-api test --specific-api test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{"test", "test2"})
			},
		},
		{
			"manifest download with specific apis (single flag)",
			"--manifest test.yaml --specific-environment test --specific-api test,test2",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{"test", "test2"})
			},
		},
		{
			"manifest download with specific apis (mixed flags)",
			"--manifest test.yaml --specific-environment test --specific-api test,test2 --specific-api test3",
			func(cmd *MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{"test", "test2", "test3"})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)
			test.setup(commandMock)

			cmd := GetDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(strings.Split(test.args, " "))
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			assert.NilError(t, err, "no error expected")
		})
	}
}

// createDownloadCommandMock creates the mock for the download command
// finish is called automatically by the testing system to ensure expected calls
func createDownloadCommandMock(t *testing.T) *MockCommand {
	mockCtrl := gomock.NewController(t)
	commandMock := NewMockCommand(mockCtrl)
	return commandMock
}
