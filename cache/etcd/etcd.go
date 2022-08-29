package etcd

import (
	"context"
	"crypto/tls"
	"net"
	"path"
	"time"

	json "github.com/json-iterator/go"
	"github.com/vine-io/vine/lib/cache"
	"github.com/vine-io/vine/lib/cmd"
	"go.etcd.io/etcd/client/v3"
)

func init() {
	cmd.DefaultCaches["etcd"] = NewCache
}

var (
	_      cache.Cache = (*etcdCache)(nil)
	prefix             = "/vine/cache"
)

type etcdCache struct {
	client  *clientv3.Client
	options cache.Options

	timeout time.Duration
}

func configure(e *etcdCache, client *clientv3.Client, opts ...cache.Option) error {

	var err error

	for _, o := range opts {
		o(&e.options)
	}

	if e.options.Context != nil {
		u, ok := e.options.Context.Value(timeKey{}).(*timeoutValue)
		if ok {
			e.timeout = u.duration
		}
	}

	if client == nil {
		config := clientv3.Config{
			Endpoints: []string{"127.0.0.1:2379"},
		}

		if e.options.Context != nil {
			tv, ok := e.options.Context.Value(tlsKey{}).(*tlsValue)
			if ok {
				config.TLS = tv.cfg
				if tv.cfg == nil {
					config.TLS = &tls.Config{InsecureSkipVerify: true}
				}
			}

			u, ok := e.options.Context.Value(authKey{}).(*authCreds)
			if ok {
				config.Username = u.Username
				config.Password = u.Password
			}
		}

		var cAddrs []string

		for _, address := range e.options.Nodes {
			if len(address) == 0 {
				continue
			}
			addr, port, err := net.SplitHostPort(address)
			if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
				port = "2379"
				addr = address
				cAddrs = append(cAddrs, net.JoinHostPort(addr, port))
			} else if err == nil {
				cAddrs = append(cAddrs, net.JoinHostPort(addr, port))
			}
		}

		// if we got addrs then we'll update
		if len(cAddrs) > 0 {
			config.Endpoints = cAddrs
		}

		client, err = clientv3.New(config)
	}

	if err != nil {
		return err
	}
	e.client = client
	return nil
}

func (e *etcdCache) Init(opts ...cache.Option) error {
	return configure(e, e.client, opts...)
}

func (e *etcdCache) Options() cache.Options {
	return e.options
}

func (e *etcdCache) Get(ctx context.Context, key string, opts ...cache.GetOption) ([]*cache.Record, error) {
	var options cache.GetOptions
	for _, o := range opts {
		o(&options)
	}

	opOpts := make([]clientv3.OpOption, 0)
	if options.Prefix {
		opOpts = append(opOpts, clientv3.WithPrefix())
	}

	pre := prefix
	if options.Database != "" {
		pre = path.Join(pre, options.Database)
	}
	if options.Table != "" {
		pre = path.Join(pre, options.Table)
	}
	if options.Limit != 0 {
		opOpts = append(opOpts, clientv3.WithLimit(int64(options.Limit)))
	}

	key = path.Join(pre, key)
	rsp, err := e.client.Get(ctx, key, opOpts...)
	if err != nil {
		return nil, err
	}

	records := make([]*cache.Record, 0, rsp.Count)
	for i, kv := range rsp.Kvs {
		record := cache.Record{}
		_ = json.Unmarshal(kv.Value, &record)
		records[i] = &record
	}

	return records, nil
}

func (e *etcdCache) Put(ctx context.Context, r *cache.Record, opts ...cache.PutOption) error {
	//TODO implement me
	panic("implement me")
}

func (e *etcdCache) Del(ctx context.Context, key string, opts ...cache.DelOption) error {
	//TODO implement me
	panic("implement me")
}

func (e *etcdCache) List(ctx context.Context, opts ...cache.ListOption) ([]string, error) {
	var options cache.ListOptions
	for _, o := range opts {
		o(&options)
	}

	key := prefix
	opOpts := make([]clientv3.OpOption, 0)
	opOpts = append(opOpts, clientv3.WithKeysOnly())

	if options.Prefix != "" {
		opOpts = append(opOpts, clientv3.WithPrefix())
		key = path.Join(key, options.Prefix)
	}

	if options.Database != "" {
		key = path.Join(key, options.Database)
	}
	if options.Table != "" {
		key = path.Join(key, options.Table)
	}
	if options.Limit != 0 {
		opOpts = append(opOpts, clientv3.WithLimit(int64(options.Limit)))
	}

	rsp, err := e.client.Get(ctx, key, opOpts...)
	if err != nil {
		return nil, err
	}

	outs := make([]string, 0, rsp.Count)
	for i, kv := range rsp.Kvs {
		outs[i] = string(kv.Key)
	}

	return outs, nil
}

func (e *etcdCache) Close() error {
	if e.client == nil {
		return nil
	}
	return e.client.Close()
}

func (e *etcdCache) String() string {
	return "etcd"
}

func NewCache(opts ...cache.Option) cache.Cache {
	e := &etcdCache{}
	for _, o := range opts {
		o(&e.options)
	}

	return e
}
