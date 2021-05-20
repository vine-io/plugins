// Package quic provides a QUIC based transport
package quic

import (
	"github.com/lack-io/vine/core/transport"
	"github.com/lack-io/vine/core/transport/quic"
	"github.com/lack-io/vine/lib/cmd"
)

func init() {
	cmd.DefaultTransports["quic"] = NewTransport
}

func NewTransport(opts ...transport.Option) transport.Transport {
	return quic.NewTransport(opts...)
}
