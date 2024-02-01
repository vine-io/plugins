package uber

import (
	"context"

	"github.com/vine-io/vine/core/client"
	"github.com/vine-io/vine/core/server"
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
