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
)

type (
	builder struct {
		projectID string
	}
	handler struct {
		client string
		env    adapter.Env
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
	// h.client, err = newGoogleClient(b.projectID)
	// if err != nil {
	// 	return nil, err
	// }
	h.client = ""
	return h, nil
}

// adapter.HandlerBuilder#SetAdapterConfig
func (b *builder) SetAdapterConfig(cfg adapter.Config) {
	c := cfg.(*config.Params)
	b.projectID = c.ProjectId
}

// adapter.HandlerBuilder#Validate
func (b *builder) Validate() (ce *adapter.ConfigErrors) {
	// Error on empty projid
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

		//h.client.addToGraph(i.Source, i.Destination)

	}
	return nil
}

// adapter.Handler#Close
func (h *handler) Close() error {
	//h.client.Close
	return nil
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
