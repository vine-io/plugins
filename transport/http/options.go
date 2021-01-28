package http

import (
	"net/http"

	"github.com/lack-io/vine/service/network/transport"
	thttp "github.com/lack-io/vine/service/network/transport/http"
)

// Handle registers the handler for the given pattern.
func Handle(pattern string, handler http.Handler) transport.Option {
	return thttp.Handle(pattern, handler)
}
