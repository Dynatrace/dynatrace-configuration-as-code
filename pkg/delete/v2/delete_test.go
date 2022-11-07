//go:build unit

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
package v2

import (
	"errors"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/api"
	"github.com/dynatrace-oss/dynatrace-monitoring-as-code/pkg/rest"
	"github.com/golang/mock/gomock"
	"gotest.tools/assert"
	"testing"
)

func TestSplitConfigsForDeletion(t *testing.T) {
	type expect struct {
		names   []string
		ids     []string
		numErrs int
	}

	type args struct {
		entries []DeletePointer
		values  []api.Value
	}

	d1 := DeletePointer{Name: "d1"}
	d2 := DeletePointer{Name: "d2"}
	d3 := DeletePointer{Name: "d3"}

	tests := []struct {
		name   string
		args   args
		expect expect
	}{
		{
			name: "Empty pointers",
			args: args{
				entries: []DeletePointer{},
			},
			expect: expect{
				names: []string{},
			},
		},
		{
			name: "Non unique - simple names, full overlap",
			args: args{
				entries: []DeletePointer{d1, d2, d3},
				values:  []api.Value{{Name: "d1"}, {Name: "d2"}, {Name: "d3"}},
			},
			expect: expect{
				names:   []string{"d1", "d2", "d3"},
				ids:     []string{},
				numErrs: 0,
			},
		},
		{
			name: "Non unique - simple names, more deletes",
			args: args{
				entries: []DeletePointer{d1, d2, d3},
				values:  []api.Value{{Name: "d1"}},
			},
			expect: expect{
				names:   []string{"d1"},
				ids:     []string{},
				numErrs: 0,
			},
		},
		{
			name: "Non unique - simple names, more values",
			args: args{
				entries: []DeletePointer{d1},
				values:  []api.Value{{Name: "d1"}, {Name: "d2"}, {Name: "d3"}},
			},
			expect: expect{
				names:   []string{"d1"},
				ids:     []string{},
				numErrs: 0,
			},
		},
		{
			name: "Non unique - fallback to id",
			args: args{
				entries: []DeletePointer{d1, {Name: "d2-id"}},
				values:  []api.Value{{Name: "d1"}, {Name: "d2", Id: "d2-id"}, {Name: "d3"}},
			},
			expect: expect{
				names:   []string{"d1"},
				ids:     []string{"d2-id"},
				numErrs: 0,
			},
		},
		{
			name: "Non unique - multiple same name",
			args: args{
				entries: []DeletePointer{d1, d2},
				values:  []api.Value{{Name: "d1"}, {Name: "d1"}, {Name: "d2"}, {Name: "d2"}},
			},
			expect: expect{
				names:   []string{},
				ids:     []string{},
				numErrs: 2,
			},
		},
		{
			name: "Non unique - combined",
			args: args{
				entries: []DeletePointer{d1, d2, d3, {Name: "d4-id"}},
				values:  []api.Value{{Name: "d1"}, {Name: "d2"}, {Name: "d2"}, {Name: "d3"}, {Id: "d4-id"}},
			},
			expect: expect{
				names:   []string{"d1", "d3"},
				ids:     []string{"d4-id"},
				numErrs: 1,
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			a := api.NewMockApi(gomock.NewController(t))
			a.EXPECT().GetId().AnyTimes().Return("some-id")

			apiMap := map[string]api.Api{a.GetId(): a}
			entriesToDelete := map[string][]DeletePointer{a.GetId(): tc.args.entries}

			client := rest.NewMockDynatraceClient(gomock.NewController(t))
			client.EXPECT().List(a).Return(tc.args.values, nil)

			client.EXPECT().BulkDeleteByName(a, gomock.InAnyOrder(tc.expect.names))

			for _, id := range tc.expect.ids {
				client.EXPECT().DeleteById(a, id)
			}

			errs := DeleteConfigs(client, apiMap, entriesToDelete)

			assert.Equal(t, len(errs), tc.expect.numErrs)
		})
	}
}

func TestSplitConfigsForDeletionClientReturnsError(t *testing.T) {
	a := api.NewMockApi(gomock.NewController(t))
	a.EXPECT().GetId().AnyTimes().Return("some-id")
	a.EXPECT().IsNonUniqueNameApi().Return(true)

	apiMap := map[string]api.Api{a.GetId(): a}
	entriesToDelete := map[string][]DeletePointer{}

	client := rest.NewMockDynatraceClient(gomock.NewController(t))
	client.EXPECT().List(a).Return(nil, errors.New("error"))

	errs := DeleteConfigs(client, apiMap, entriesToDelete)

	assert.Assert(t, len(errs) > 0, "an error should be returned")
}
