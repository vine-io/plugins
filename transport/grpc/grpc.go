// Package grpc provides a grpc transport
package grpc

import (
	"github.com/lack-io/vine/core/transport"
	"github.com/lack-io/vine/core/transport/grpc"
	"github.com/lack-io/vine/lib/cmd"
)

func init() {
	cmd.DefaultTransports["grpc"] = NewTransport
}

func NewTransport(opts ...transport.Option) transport.Transport {
	return grpc.NewTransport(opts...)
}
