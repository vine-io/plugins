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

package ratelimiter

import (
	"context"
	"time"

	"github.com/juju/ratelimit"
	"github.com/lack-io/vine/proto/errors"
	"github.com/lack-io/vine/service/server"

	"github.com/lack-io/vine/service/client"
)

type clientWrapper struct {
	fn func() error
	client.Client
}

func limit(b *ratelimit.Bucket, wait bool, errId string) func() error {
	return func() error {
		if wait {
			time.Sleep(b.Take(1))
		} else if b.TakeAvailable(1) == 0 {
			return errors.New(errId, "too many request", 429)
		}
		return nil
	}
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	if err := c.fn(); err != nil {
		return err
	}
	return c.Client.Call(ctx, req, rsp, opts...)
}

// NewClientWrapper takes a rate limiter and wait flag and returns a client wrapper.
func NewClientWrapper(b *ratelimit.Bucket, wait bool) client.Wrapper {
	fn := limit(b, wait, "go.vine.client")

	return func(c client.Client) client.Client {
		return &clientWrapper{fn, c}
	}
}

// NewHandlerWrapper takes a rate limiter and wait flag and returns a client Wrapper.
func NewHandlerWrapper(b *ratelimit.Bucket, wait bool) server.HandlerWrapper {
	fn := limit(b, wait, "go.vine.server")

	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			if err := fn(); err != nil {
				return err
			}
			return h(ctx, req, rsp)
		}
	}
}
