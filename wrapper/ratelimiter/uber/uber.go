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

package uber

import (
	"context"

	"github.com/lack-io/vine/service/client"
	"github.com/lack-io/vine/service/server"
	"go.uber.org/ratelimit"
)

type clientWrapper struct {
	r ratelimit.Limiter
	client.Client
}

func (c *clientWrapper) Call(ctx context.Context, req client.Request, rsp interface{}, opts ...client.CallOption) error {
	c.r.Take()
	return c.Client.Call(ctx, req, rsp, opts...)
}

// NewClientWrapper creates a blocking side rate limiter
func NewClientWrapper(rate int, opts ...ratelimit.Option) client.Wrapper {
	r := ratelimit.New(rate, opts...)

	return func(c client.Client) client.Client {
		return &clientWrapper{r, c}
	}
}

// NewHandlerWrapper creates a blocking server side rate limiter
func NewHandlerWrapper(rate int, opts ...ratelimit.Option) server.HandlerWrapper {
	r := ratelimit.New(rate, opts...)

	return func(h server.HandlerFunc) server.HandlerFunc {
		return func(ctx context.Context, req server.Request, rsp interface{}) error {
			r.Take()
			return h(ctx, req, rsp)
		}
	}
}
