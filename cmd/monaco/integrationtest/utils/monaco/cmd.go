//go:build unit || integration || download_restore || cleanup || nightly

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

package monaco

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http/httptrace"
	"net/textproto"
	"regexp"
	"strings"
	"testing"

	"github.com/spf13/afero"

	"github.com/dynatrace/dynatrace-configuration-as-code/v2/cmd/monaco/runner"
)

func NewTestFs() afero.Fs { return afero.NewCopyOnWriteFs(afero.NewOsFs(), afero.NewMemMapFs()) }

// spacesRegex finds all sequential spaces
var spacesRegex = regexp.MustCompile(`\s+`)

// Run is the entrypoint to run monaco for all integration tests.
// It requires to specify the full command (`monaco [deploy]....`) and sets up the runner.
func Run(t *testing.T, fs afero.Fs, command string) error {
	// remove multiple spaces
	c := spacesRegex.ReplaceAllString(command, " ")
	c = strings.Trim(c, " ")

	const prefix = "monaco "

	if !strings.HasPrefix(c, prefix) {
		return fmt.Errorf("command must start with '%s'", prefix)
	}
	t.Logf("Running command: %s", command)
	c = strings.TrimPrefix(c, prefix)

	args := strings.Split(c, " ")

	cmd := runner.BuildCmd(fs)
	cmd.SetArgs(args)

	// explicit cancel for each monaco run invocation

	ctx, cancel := context.WithCancel(t.Context())
	defer cancel()

	trace := httptrace.ClientTrace{
		GetConn: func(hostPort string) {
			t.Log("GetConn")
		},
		GotConn: func(info httptrace.GotConnInfo) {
			t.Log("GotConn")
		},
		PutIdleConn: func(err error) {
			t.Log("PutIdleConn")
		},
		GotFirstResponseByte: func() {
			t.Log("GotFirstResponseByte")
		},
		Got100Continue: func() {
			t.Log("Got100Continue")
		},
		Got1xxResponse: func(code int, header textproto.MIMEHeader) error {
			t.Log("Got1xxResponse", code, header)
			return nil
		},
		DNSStart: func(info httptrace.DNSStartInfo) {
			t.Log("DNSStart", info)
		},
		DNSDone: func(info httptrace.DNSDoneInfo) {
			t.Log("DNSDone", info)
		},
		ConnectStart: func(network, addr string) {
			t.Log("ConnectStart", network, addr)
		},
		ConnectDone: func(network, addr string, err error) {
			t.Log("ConnectDone", network, addr, err)
		},
		TLSHandshakeStart: func() {
			t.Log("TLSHandshakeStart")
		},
		TLSHandshakeDone: func(state tls.ConnectionState, err error) {
			t.Log("TLSHandshakeDone", state, err)
		},
		WroteHeaderField: func(key string, value []string) {
			if key == "authorization" {
				value = []string{"****HIDDEN****"}
			}
			t.Log("WroteHeaderField", key, value)
		},
		WroteHeaders: func() {
			t.Log("WroteHeaders")
		},
		Wait100Continue: func() {
			t.Log("Wait100Continue")
		},
		WroteRequest: func(info httptrace.WroteRequestInfo) {
			t.Log("WroteRequest", info)
		},
	}

	ctx = httptrace.WithClientTrace(ctx, &trace)

	return runner.RunCmd(ctx, cmd)
}
