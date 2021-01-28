// Package http returns a http2 transport using net/http
package http

import (
	"github.com/lack-io/vine/service/config/cmd"
	"github.com/lack-io/vine/service/network/transport"
)

func init() {
	cmd.DefaultTransports["http"] = NewTransport
}

// NewTransport returns a new http transport using net/http and supporting http2
func NewTransport(opts ...transport.Option) transport.Transport {
	return transport.NewTransport(opts...)
}
