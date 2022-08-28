package etcd

import (
	"context"

	"github.com/vine-io/vine/lib/cache"
	"github.com/vine-io/vine/lib/cmd"
)

func init() {
	cmd.DefaultCaches["etcd"] = NewCache
}

var _ cache.Cache = (*etcd)(nil)

type etcd struct {
}

func (e *etcd) Init(opts ...cache.Option) error {
	//TODO implement me
	panic("implement me")
}

func (e *etcd) Options() cache.Options {
	//TODO implement me
	panic("implement me")
}

func (e *etcd) Get(ctx context.Context, key string, opts ...cache.GetOption) ([]*cache.Record, error) {
	//TODO implement me
	panic("implement me")
}

func (e *etcd) Put(ctx context.Context, r *cache.Record, opts ...cache.PutOption) error {
	//TODO implement me
	panic("implement me")
}

func (e *etcd) Del(ctx context.Context, key string, opts ...cache.DelOption) error {
	//TODO implement me
	panic("implement me")
}

func (e *etcd) List(ctx context.Context, opts ...cache.ListOption) ([]string, error) {
	//TODO implement me
	panic("implement me")
}

func (e *etcd) Close() error {
	//TODO implement me
	panic("implement me")
}

func (e *etcd) String() string {
	return "etcd"
}

func NewCache(opts ...cache.Option) cache.Cache {
	return &etcd{}
}
