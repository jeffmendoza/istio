// Copyright 2018 Istio Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package stackdriver_test

import (
	"fmt"
	"testing"
	"time"

	"istio.io/istio/mixer/adapter/stackdriver"
	adapter_integration "istio.io/istio/mixer/pkg/adapter/test"
)

const newLogConf = `
apiVersion: "config.istio.io/v1alpha2"
kind: edge
metadata:
  name: default
  namespace: istio-system
spec:
  timestamp: request.time | context.time
  sourceUid: source.uid | "Unknown"
  sourceOwner: source.owner | "Unknown"
  sourceWorkloadName: source.workload.name | "Unknown"
  sourceWorkloadNamespace: source.workload.namespace | "Unknown"
  destinationUid: destination.uid | "Unknown"
  destinationOwner: destination.owner | "Unknown"
  destinationWorkloadName: destination.workload.name | "Unknown"
  destinationWorkloadNamespace: destination.workload.namespace | "Unknown"
  contextProtocol: context.protocol | "Unknown"
  apiProtocol: api.protocol | "Unknown"
`

const handlerConf = `
apiVersion: "config.istio.io/v1alpha2"
kind: stackdriver
metadata:
  name: handler
  namespace: istio-system
spec:
  project_id: %s
  service_account_path: %s
`

const ruleConf = `
apiVersion: "config.istio.io/v1alpha2"
kind: rule
metadata:
  name: edgetosd
  namespace: istio-system
spec:
  match: (context.reporter.kind | "inbound" == "inbound") && (context.protocol | "unknown" != "unknown")
  actions:
   - handler: handler.stackdriver
     instances:
     - default.edge
`

func setupTest() (interface{}, error) {
	// create GCP client save in retval
	return nil, nil
}
func teardownTest(ctx interface{}) {
	// close GCP client
}

type state struct {
	State string
}

func getState(ctx interface{}) (interface{}, error) {
	// use GCP client to get results, return in retval
	return state{"foo"}, nil
}

const wantTemplate = `
        {
         "AdapterState": {
          %s
         },
         "Returns": [
          {
           "Check": {
            "Status": {},
            "ValidDuration": 0,
            "ValidUseCount": 0
           },
           "Quota": null,
           "Error": null
          }
         ]
        }
`

type testCase struct {
	name   string
	report map[string]interface{}
	want   string
}

var testCases = []testCase{
	{
		"basic",
		map[string]interface{}{
			"request.time":                   time.Unix(1, 0),
			"source.uid":                     "foo",
			"source.owner":                   "bar",
			"source.workload.name":           "baz",
			"source.workload.namespace":      "baz",
			"destination.uid":                "foo",
			"destination.owner":              "bar",
			"destination.workload.name":      "baz",
			"destination.workload.namespace": "baz",
			"context.protocol":               "asdf",
			"api.protocol":                   "asdf",
		},
		`"State": "foo"`,
	},
}

func TestCG(t *testing.T) {

	h := fmt.Sprintf(handlerConf, projectID, saLocation)

	for _, c := range testCases {
		t.Run(c.name, func(t *testing.T) {
			w := fmt.Sprintf(wantTemplate, c.want)
			scenario := adapter_integration.Scenario{
				Configs: []string{newLogConf, h, ruleConf},
				ParallelCalls: []adapter_integration.Call{
					{
						CallKind: adapter_integration.REPORT,
						Attrs:    c.report,
					},
				},
				Want:     w,
				Setup:    setupTest,
				Teardown: teardownTest,
				GetState: getState,
			}
			adapter_integration.RunTest(t, stackdriver.GetInfo, scenario)
		})
	}
}
