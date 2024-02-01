package validator

import (
	"context"

	"github.com/vine-io/vine/core/client"
	"github.com/vine-io/vine/core/server"
	"github.com/vine-io/vine/lib/errors"
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
