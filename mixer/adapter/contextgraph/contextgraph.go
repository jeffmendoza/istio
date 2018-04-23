// Copyright 2017 Istio Authors
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

//go:generate $GOPATH/src/istio.io/istio/bin/mixer_codegen.sh -f mixer/adapter/contextgraph/config/config.proto

// Package contextgraph adapter for Stackdriver Context API.
package contextgraph

import (
	"context"

	"istio.io/istio/mixer/adapter/contextgraph/config"
	"istio.io/istio/mixer/pkg/adapter"
	"istio.io/istio/mixer/template/edge"
	"k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type (
	builder struct {
		projectID string
	}
	handler struct {
		client    string
		env       adapter.Env
		clientset *kubernetes.Clientset
		watchInt  watch.Interface
	}
)

// ensure types implement the requisite interfaces
var _ edge.HandlerBuilder = &builder{}
var _ edge.Handler = &handler{}

///////////////// Configuration-time Methods ///////////////

// adapter.HandlerBuilder#Build
func (b *builder) Build(ctx context.Context, env adapter.Env) (adapter.Handler, error) {
	h := &handler{
		env: env,
	}

	// Whatever needed to talk to contextgraph API
	// h.client, err = newGoogleClient(b.projectID)
	// if err != nil {
	// 	return nil, err
	// }
	h.client = ""

	// Fixme use in-cluster config
	// This is setup for running mixs through 'kubectl proxy'
	config := &rest.Config{
		Host: "localhost:8001",
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	h.clientset = clientset

	// Do not use go routines
	env.ScheduleDaemon(h.serviceWatch)

	return h, nil
}

// adapter.HandlerBuilder#SetAdapterConfig
func (b *builder) SetAdapterConfig(cfg adapter.Config) {
	c := cfg.(*config.Params)
	b.projectID = c.ProjectId
}

// adapter.HandlerBuilder#Validate
func (b *builder) Validate() (ce *adapter.ConfigErrors) {
	// FIXME Error on empty projid
	return
}

// edge.HandlerBuilder#SetEdgeTypes
func (b *builder) SetEdgeTypes(types map[string]*edge.Type) {
}

////////////////// Request-time Methods //////////////////////////

// edge.Handler#HandleEdge
func (h *handler) HandleEdge(ctx context.Context, insts []*edge.Instance) error {
	for _, i := range insts {
		h.env.Logger().Debugf("Connection Reported: %v to %v", i.Source, i.Destination)

		// Would add a graph edge here
		//h.client.addToGraph(i.Source, i.Destination)

	}
	return nil
}

// adapter.Handler#Close
func (h *handler) Close() error {
	// close API client here
	//h.client.Close

	if h.watchInt != nil {
		h.watchInt.Stop()
	}

	return nil
}

func (h *handler) serviceWatch() {
	// Watch Services
	w, err := h.clientset.CoreV1().Services("").Watch(metav1.ListOptions{})
	if err != nil {
		h.env.Logger().Errorf("Error starting service watch: %s", err)
		return
	}
	h.watchInt = w
	c := w.ResultChan()
	for {
		e, more := <-c
		if !more {
			h.env.Logger().Debugf("Service watch channel closed, exiting")
			break
		}

		//FIXME do something with the service info, send to contextgraph
		h.env.Logger().Debugf("Event: %s\n", e.Type)
		s := e.Object.(*v1.Service)
		h.env.Logger().Debugf("Service: %v \tNamespace: %v\n", s.Name, s.Namespace)
	}
}

////////////////// Bootstrap //////////////////////////

// GetInfo returns the adapter.Info specific to this adapter.
func GetInfo() adapter.Info {
	return adapter.Info{
		Name:        "contextgraph",
		Description: "Sends graph to Stackdriver Context API",
		SupportedTemplates: []string{
			edge.TemplateName,
		},
		NewBuilder: func() adapter.HandlerBuilder { return &builder{} },
		DefaultConfig: &config.Params{
			ProjectId: "",
		},
	}
}
