package etcd

import (
	"github.com/lack-io/vine/core/registry"
	"github.com/lack-io/vine/core/registry/etcd"
	"github.com/lack-io/vine/lib/cmd"
)

func init() {
	cmd.DefaultRegistries["etcd"] = NewRegistry
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return etcd.NewRegistry(opts...)
}
