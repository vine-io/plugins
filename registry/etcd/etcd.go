package etcd

import (
	"github.com/lack-io/vine/service/config/cmd"
	"github.com/lack-io/vine/service/registry"
	"github.com/lack-io/vine/service/registry/etcd"
)

func init() {
	cmd.DefaultRegistries["etcd"] = NewRegistry
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return etcd.NewRegistry(opts...)
}
