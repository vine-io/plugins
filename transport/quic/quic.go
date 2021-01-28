// Package quic provides a QUIC based transport
package quic

import (
	"github.com/lack-io/vine/service/config/cmd"
	"github.com/lack-io/vine/service/network/transport"
	"github.com/lack-io/vine/service/network/transport/quic"
)

func init() {
	cmd.DefaultTransports["quic"] = NewTransport
}

func NewTransport(opts ...transport.Option) transport.Transport {
	return quic.NewTransport(opts...)
}
