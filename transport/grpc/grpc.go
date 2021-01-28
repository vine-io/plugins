// Package grpc provides a grpc transport
package grpc

import (
	"github.com/lack-io/vine/service/config/cmd"
	"github.com/lack-io/vine/service/network/transport"
	"github.com/lack-io/vine/service/network/transport/grpc"
)

func init() {
	cmd.DefaultTransports["grpc"] = NewTransport
}

func NewTransport(opts ...transport.Option) transport.Transport {
	return grpc.NewTransport(opts...)
}
