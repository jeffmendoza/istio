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

package fluentd

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	adapter_integration "istio.io/istio/mixer/pkg/adapter/test"

	dc "github.com/fsouza/go-dockerclient"
	dockertest "gopkg.in/ory-am/dockertest.v3"
)

const newLogConf = `
apiVersion: "config.istio.io/v1alpha2"
kind: logentry
metadata:
  name: newlog
  namespace: istio-system
spec:
  severity: '"warning"'
  timestamp: request.time
  variables:
    source: source.labels["app"] | source.service | "unknown"
    user: source.user | "unknown"
    destination: destination.labels["app"] | destination.service | "unknown"
    responseCode: response.code | 0
    responseSize: response.size | 0
    latency: response.duration | "0ms"
  monitored_resource_type: '"UNSPECIFIED"'
`

const handlerConf = `
apiVersion: "config.istio.io/v1alpha2"
kind: fluentd
metadata:
  name: handler
  namespace: istio-system
spec:
  address: "localhost:24224"
`

const ruleConf = `
apiVersion: "config.istio.io/v1alpha2"
kind: rule
metadata:
  name: newlogtofluentd
  namespace: istio-system
spec:
  match: "true" # match for all requests
  actions:
   - handler: handler.fluentd
     instances:
     - newlog.logentry
`

const want = `
		{
		 "AdapterState": {
		  "Date": "1970-01-01T00:00:01+00:00",
		  "Message": {
		   "destination": "dest",
		   "latency": "1m0s",
		   "responseCode": 200,
		   "responseSize": 1337,
		   "severity": "warning",
		   "source": "src1",
		   "user": "me"
		  },
		  "Tag": "newlog.logentry.istio-system"
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

type state struct {
	Date    string
	Tag     string
	Message interface{}
}

type testContext struct {
	Pool      *dockertest.Pool
	Container *dockertest.Resource
}

func testDataDir() string {
	dir, _ := os.Getwd()
	for !strings.HasSuffix(dir, filepath.Join("istio.io", "istio")) {
		dir = filepath.Dir(dir)
	}
	dir = filepath.Join(dir, "mixer", "adapter", "fluentd", "testdata")
	return dir
}

func setupTest() (interface{}, error) {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return nil, fmt.Errorf("Could not connect to docker daemon: %s", err)
	}

	mnt := testDataDir()

	// Remove pre-existing output logs.
	if matches, err := filepath.Glob(filepath.Join(mnt, "out.*")); err == nil {
		for _, v := range matches {
			log.Printf("Found old log to remove: %v", v)
			if err := os.Remove(v); err != nil {
				return nil, fmt.Errorf("Could not remove pre-existing out log %v: %s", v, err)
			}
		}
	}

	// Run Fluentd docker container
	o := dockertest.RunOptions{
		Repository: "fluent/fluentd",
		Tag:        "latest",
		Entrypoint: []string{"/usr/bin/dumb-init", "--"},
		Cmd:        []string{"fluentd", "-c", "/conf/fluentd.conf"},
		Mounts:     []string{fmt.Sprintf("%v:/conf", mnt)},
		PortBindings: map[dc.Port][]dc.PortBinding{
			"24224/tcp": []dc.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: "24224",
				},
			},
		},
	}
	c, err := pool.RunWithOptions(&o)
	if err != nil {
		return nil, fmt.Errorf("Could not start fluentd container: %s", err)
	}

	// Check logs until fluentd started
	if err := pool.Retry(func() error {
		var b bytes.Buffer
		if err := pool.Client.Logs(dc.LogsOptions{
			Container:    c.Container.ID,
			OutputStream: &b,
			Stdout:       true,
		}); err != nil {
			return err
		}
		if !strings.Contains(b.String(), "fluentd worker is now running") {
			return fmt.Errorf("Fluentd not started yet")
		}
		return nil
	}); err != nil {
		return nil, fmt.Errorf("Could not determine fluentd has started: %s", err)
	}

	ctx := testContext{
		Pool:      pool,
		Container: c,
	}
	return ctx, nil
}

func teardownTest(ctx interface{}) {
	tc := ctx.(testContext)

	if err := tc.Pool.Client.StopContainer(tc.Container.Container.ID, 15); err != nil {
		log.Printf("Could not stop fluentd container: %s", err)
		return
	}

	if err := tc.Pool.Client.RemoveContainer(dc.RemoveContainerOptions{
		ID:            tc.Container.Container.ID,
		Force:         true,
		RemoveVolumes: true,
	}); err != nil {
		log.Printf("Could not remove fluentd container: %s", err)
		return
	}
}

func getState(ctx interface{}) (interface{}, error) {
	// Fluentd config is set to flush to file every 1s. Need to wait.
	time.Sleep(3 * time.Second)
	matches, err := filepath.Glob(filepath.Join(testDataDir(), "out.*"))
	if err != nil {
		log.Printf("Can't get output log, error: %s", err)
		return nil, fmt.Errorf("Can't get output log, error: %s", err)
	}
	if len(matches) != 1 {
		log.Printf("Matches not 1: %v", matches)
		return nil, fmt.Errorf("Matches not 1: %v", matches)
	}

	out, err := ioutil.ReadFile(matches[0])
	if err != nil {
		log.Printf("Can't read output log, error: %s", err)
		return nil, fmt.Errorf("Can't read output log, error: %s", err)
	}

	output := strings.TrimSpace(string(out))

	parts := strings.Split(output, "\t")
	if len(parts) != 3 {
		return parts, nil
	}

	var message interface{}
	if err := json.Unmarshal([]byte(parts[2]), &message); err != nil {
		return parts, nil
	}
	s := state{
		Date:    parts[0],
		Tag:     parts[1],
		Message: message,
	}

	return s, nil
}

func TestLog(t *testing.T) {
	scenario := adapter_integration.Scenario{
		Configs: []string{newLogConf, handlerConf, ruleConf},
		ParallelCalls: []adapter_integration.Call{
			{
				CallKind: adapter_integration.REPORT,
				Attrs: map[string]interface{}{
					"source.service":      "src1",
					"request.time":        time.Unix(1, 0),
					"source.user":         "me",
					"destination.service": "dest",
					"response.code":       int64(200),
					"response.size":       int64(1337),
					"response.duration":   time.Minute,
				},
			},
		},
		Want:     want,
		Setup:    setupTest,
		Teardown: teardownTest,
		GetState: getState,
	}

	adapter_integration.RunTest(t, GetInfo, scenario)
}
