package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"path"
	"strings"
	gosync "sync"

	"github.com/google/uuid"
	"github.com/vine-io/vine/lib/sync"
	"go.etcd.io/etcd/client/v3"
	cc "go.etcd.io/etcd/client/v3/concurrency"
)

type EtcdSync struct {
	options sync.Options
	prefix  string
	client  *clientv3.Client

	mtx   gosync.Mutex
	locks map[string]*etcdLock
}

type etcdLock struct {
	s *cc.Session
	m *cc.Mutex
}

func configure(e *EtcdSync, client *clientv3.Client, opts ...sync.Option) error {
	options := e.options
	for _, o := range opts {
		o(&options)
	}

	if options.Prefix == "" {
		options.Prefix = "/vine/sync"
	}

	var err error
	if client == nil {
		var endpoints []string

		for _, addr := range options.Nodes {
			if len(addr) > 0 {
				if !strings.HasPrefix(addr, "http") {
					addr = "http://" + addr
				}
				endpoints = append(endpoints, addr)
			}
		}

		if len(endpoints) == 0 {
			endpoints = []string{"http://127.0.0.1:2379"}
		}

		client, err = clientv3.New(clientv3.Config{
			Endpoints: endpoints,
		})
	}
	if err != nil {
		return err
	}
	e.client = client

	return nil
}

func (e *EtcdSync) Init(opts ...sync.Option) error {
	for _, o := range opts {
		o(&e.options)
	}

	return configure(e, e.client, opts...)
}

func (e *EtcdSync) Options() sync.Options {
	return e.options
}

func (e *EtcdSync) Leader(ctx context.Context, name string, opts ...sync.LeaderOption) (sync.Leader, error) {
	var options sync.LeaderOptions
	for _, o := range opts {
		o(&options)
	}

	if options.Id == "" {
		options.Id = uuid.New().String()
	}

	if options.TTL == 0 {
		options.TTL = 30
	}
	if options.Namespace == "" {
		options.Namespace = "default"
	}

	// make path
	cpath := path.Join(e.prefix, "leaders", options.Namespace)

	s, err := cc.NewSession(e.client)
	if err != nil {
		return nil, err
	}

	l := cc.NewElection(s, cpath)

	leader := &etcdLeader{
		opts:    options,
		s:       s,
		e:       l,
		prefix:  e.prefix,
		name:    name,
		elected: make(chan struct{}, 1),
		stop:    make(chan struct{}, 1),
	}
	go leader.campaign()

	return leader, nil
}

func (e *EtcdSync) ListMembers(ctx context.Context, opts ...sync.ListMembersOption) ([]*sync.Member, error) {
	var options sync.ListMembersOptions
	for _, opt := range opts {
		opt(&options)
	}

	if options.Namespace == "" {
		options.Namespace = "default"
	}

	members := make([]*sync.Member, 0)

	key := path.Join(e.prefix, "leaders", options.Namespace)
	rsp, err := e.client.Get(ctx, key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	p, _ := e.client.Get(ctx, path.Join(e.prefix, "primary", options.Namespace))

	for _, kv := range rsp.Kvs {
		val := &sync.Member{}
		err = json.Unmarshal(kv.Value, &val)
		if err == nil {
			if p != nil && val.Id == string(p.Kvs[0].Value) {
				val.Role = sync.Primary
			} else {
				val.Role = sync.Follow
			}
			members = append(members, val)
		}
	}

	return members, nil
}

func (e *EtcdSync) WatchElect(ctx context.Context, opts ...sync.WatchElectOption) (sync.ElectWatcher, error) {
	return newEtcdWatcher(e, opts...)
}

func (e *EtcdSync) Lock(ctx context.Context, id string, opts ...sync.LockOption) error {
	var options sync.LockOptions
	for _, o := range opts {
		o(&options)
	}

	// make path
	key := path.Join(e.prefix, strings.Replace(e.options.Prefix+id, "/", "-", -1))

	var sopts []cc.SessionOption
	if options.TTL > 0 {
		sopts = append(sopts, cc.WithTTL(int(options.TTL.Seconds())))
	}

	s, err := cc.NewSession(e.client, sopts...)
	if err != nil {
		return err
	}

	m := cc.NewMutex(s, key)

	if options.Wait != 0 {
		ctx, _ = context.WithTimeout(ctx, options.Wait)
	}

	ech := make(chan error, 1)
	go func() {
		ech <- m.Lock(ctx)
	}()

	select {
	case <-ctx.Done():
		err = sync.ErrLockTimeout
	case err = <-ech:
	}

	if err != nil {
		_ = s.Close()
		return err
	}

	e.mtx.Lock()
	e.locks[id] = &etcdLock{
		s: s,
		m: m,
	}
	e.mtx.Unlock()
	return nil
}

func (e *EtcdSync) Unlock(ctx context.Context, id string) error {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	v, ok := e.locks[id]
	if !ok {
		return errors.New("lock not found")
	}
	defer v.s.Close()
	err := v.m.Unlock(context.Background())
	delete(e.locks, id)
	return err
}

func (e *EtcdSync) String() string {
	return "etcd"
}

func (e *EtcdSync) GetConn() *clientv3.Client {
	return e.client
}

func NewSync(opts ...sync.Option) sync.Sync {
	var options sync.Options
	for _, o := range opts {
		o(&options)
	}

	e := &EtcdSync{
		prefix:  options.Prefix,
		options: options,
		locks:   make(map[string]*etcdLock),
	}
	return e
}

func NewEtcdSync(client *clientv3.Client, opts ...sync.Option) sync.Sync {
	var options sync.Options
	for _, o := range opts {
		o(&options)
	}

	e := &EtcdSync{
		prefix:  options.Prefix,
		options: options,
		client:  client,
		locks:   make(map[string]*etcdLock),
	}
	return e
}
