// Package http returns a http2 transport using net/http
package http

import (
	"github.com/lack-io/vine/core/transport"
	"github.com/lack-io/vine/core/transport/http"
	"github.com/lack-io/vine/lib/cmd"
)

func init() {
	cmd.DefaultTransports["http"] = NewTransport
}

// NewTransport returns a new http transport using net/http and supporting http2
func NewTransport(opts ...transport.Option) transport.Transport {
	return http.NewTransport(opts...)
}
