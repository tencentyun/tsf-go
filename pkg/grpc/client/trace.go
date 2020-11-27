package client

// Copyright 2019 The OpenZipkin Authors
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

import (
	"context"
	"strings"

	"github.com/openzipkin/zipkin-go"
	"github.com/openzipkin/zipkin-go/model"
	"google.golang.org/grpc/peer"
)

func spanName(method string) string {
	name := strings.TrimPrefix(method, "/")
	name = strings.Replace(name, "/", ".", -1)
	return name
}

func remoteEndpointFromContext(ctx context.Context, name string) *model.Endpoint {
	remoteAddr := ""

	p, ok := peer.FromContext(ctx)
	if ok {
		remoteAddr = p.Addr.String()
	}

	ep, _ := zipkin.NewEndpoint(name, remoteAddr)
	return ep
}
