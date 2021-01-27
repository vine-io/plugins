// Copyright 2021 lack
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package validator

import (
	"context"

	"github.com/lack-io/vine/proto/errors"
	"github.com/lack-io/vine/service/client"
	"github.com/lack-io/vine/service/server"
)

type Validator interface {
	Validate() error
}

func NewHandlerWrapper() server.HandlerWrapper {
	return func(fn server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if v, ok := req.Body().(Validator); ok {
				if err := v.Validate(); err != nil {
					return errors.BadRequest(req.Service(), "%v", err)
				}
			}
			return fn(ctx, req, rsp)
		}
	}
}

type clientWrapper struct {
	client.Client
}

func (c clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	if v, ok := req.Body().(Validator); ok {
		if err := v.Validate(); err != nil {
			return errors.BadRequest(req.Service(), "%v", err)
		}
	}
	return c.Client.Call(ctx, req, rsp, opts...)
}

func (c clientWrapper) Stream(ctx context.Context, req client.Request, opts ...client.CallOption) (client.Stream, error) {
	if v, ok := req.Body().(Validator); ok {
		if err := v.Validate(); err != nil {
			return nil, errors.BadRequest(req.Service(), "%v", err)
		}
	}
	return c.Client.Stream(ctx, req, opts...)
}

func NewClientWrapper() client.Wrapper {
	return func(c client.Client) client.Client {
		return &clientWrapper{Client: c}
	}
}
