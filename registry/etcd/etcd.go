package etcd

import (
	"github.com/vine-io/vine/core/registry"
	"github.com/vine-io/vine/core/registry/etcd"
	"github.com/vine-io/vine/lib/cmd"
)

func init() {
	cmd.DefaultRegistries["etcd"] = NewRegistry
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return etcd.NewRegistry(opts...)
}
