//go:build unit

package runner

import (
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/cmd/monaco/download"
	"io"

	"github.com/golang/mock/gomock"
	"github.com/spf13/afero"
	"gotest.tools/assert"

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
			[]string{"either '--environments' or '--url' has to be provided"},
		},
		{
			"both --url and --environments provided",
			"--url test --environments test --specific-environment test --environment-name test --token-name test --",
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
			[]string{"environment-name", "token-name"},
		},
		{
			"--environments is missing argument",
			"--environments",
			[]string{"needs an argument"},
		},
		{
			"--environments is missing other required parameters",
			"--environments test.yaml",
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

			cmd := getDownloadCommand(afero.NewOsFs(), commandMock)
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
		setup func(command *download.MockCommand)
	}{
		{
			"direct download no specific apis",
			"--url test --environment-name test --token-name token",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{})
			},
		},
		{
			"direct download with specific apis (multiple flags)",
			"--url test --environment-name test --token-name token --specific-api test --specific-api test2",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{"test", "test2"})
			},
		},
		{
			"direct download with specific apis (single flag)",
			"--url test --environment-name test --token-name token --specific-api test,test2",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{"test", "test2"})
			},
		},
		{
			"direct download with specific apis (mixed flags)",
			"--url test --environment-name test --token-name token --specific-api test,test2 --specific-api test3",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{"test", "test2", "test3"})
			},
		},
		{
			"direct download with specific apis (mixed flags) and verbose",
			"--url test --environment-name test --token-name token --specific-api test,test2 --specific-api test3 -v",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{"test", "test2", "test3"})
			},
		},
		{
			"direct download with specific apis (mixed flags) and verbose",
			"--url test --environment-name test --token-name token --specific-api test,test2 --specific-api test3 -v",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigs(gomock.Any(), "test", "test", "token", []string{"test", "test2", "test3"})
			},
		},
		{
			"manifest download no specific apis",
			"--environments test.yaml --specific-environment test",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{})
			},
		},
		{
			"manifest download with specific apis (multiple flags)",
			"--environments test.yaml --specific-environment test --specific-api test --specific-api test2",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{"test", "test2"})
			},
		},
		{
			"manifest download with specific apis (single flag)",
			"--environments test.yaml --specific-environment test --specific-api test,test2",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{"test", "test2"})
			},
		},
		{
			"manifest download with specific apis (mixed flags)",
			"--environments test.yaml --specific-environment test --specific-api test,test2 --specific-api test3",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{"test", "test2", "test3"})
			},
		},
		{
			"manifest download with specific apis (mixed flags) and verbose",
			"--environments test.yaml --specific-environment test --specific-api test,test2 --specific-api test3 -v",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{"test", "test2", "test3"})
			},
		},
		{
			"manifest download with specific apis (mixed flags) and verbose",
			"--environments test.yaml --specific-environment test --specific-api test,test2 --specific-api test3 -v",
			func(cmd *download.MockCommand) {
				cmd.EXPECT().DownloadConfigsBasedOnManifest(gomock.Any(), "test.yaml", "test", []string{"test", "test2", "test3"})
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			commandMock := createDownloadCommandMock(t)
			test.setup(commandMock)

			cmd := getDownloadCommand(afero.NewOsFs(), commandMock)
			cmd.SetArgs(strings.Split(test.args, " "))
			cmd.SetOut(io.Discard) // skip output to ensure that the error message contains the error, not the help message
			err := cmd.Execute()

			assert.NilError(t, err, "no error expected")
		})
	}
}

// createDownloadCommandMock creates the mock for the download command
// finish is called automatically by the testing system to ensure expected calls
func createDownloadCommandMock(t *testing.T) *download.MockCommand {
	mockCtrl := gomock.NewController(t)
	commandMock := download.NewMockCommand(mockCtrl)
	return commandMock
}
