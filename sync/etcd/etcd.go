package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"path"
	"strings"
	gosync "sync"

	"github.com/google/uuid"
	"github.com/vine-io/vine/lib/sync"
	"go.etcd.io/etcd/client/v3"
	cc "go.etcd.io/etcd/client/v3/concurrency"
)

type etcdSync struct {
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

type Value struct {
	Namespace string `json:"namespace"`
	Id        string `json:"id"`
}

type etcdLeader struct {
	opts   sync.LeaderOptions
	s      *cc.Session
	e      *cc.Election
	prefix string
	ns     string
	id     string
}

func (e *etcdLeader) Id() string {
	return e.id
}

func (e *etcdLeader) Status() chan bool {
	ch := make(chan bool, 1)
	ech := e.e.Observe(context.Background())

	go func() {
		for r := range ech {
			if string(r.Kvs[0].Value) == e.id {
				ch <- true
				close(ch)
				return
			}
		}
	}()

	return ch
}

func (e *etcdLeader) Resign() error {
	key := path.Join(e.prefix, "leaders", e.ns, e.id)
	e.s.Client().Delete(context.TODO(), key)
	return e.e.Resign(context.Background())
}

func (e *etcdSync) Init(opts ...sync.Option) error {
	for _, o := range opts {
		o(&e.options)
	}
	return nil
}

func (e *etcdSync) Options() sync.Options {
	return e.options
}

func (e *etcdSync) Leader(name string, opts ...sync.LeaderOption) (sync.Leader, error) {
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

	// make path
	cpath := path.Join(e.prefix, strings.Replace(path.Join(e.options.Prefix, options.Namespace, name), "/", "-", -1))

	s, err := cc.NewSession(e.client)
	if err != nil {
		return nil, err
	}

	ctx := context.TODO()
	l := cc.NewElection(s, cpath)

	member := &sync.Member{
		Leader:    name,
		Id:        options.Id,
		Namespace: options.Namespace,
		Role:      sync.Follow,
	}

	key := path.Join(e.prefix, "leaders", options.Namespace, options.Id)
	val, _ := json.Marshal(member)
	_, _ = e.client.Put(ctx, key, string(val))

	if err = l.Campaign(ctx, options.Id); err != nil {
		return nil, err
	}

	member.Role = sync.Primary
	val, _ = json.Marshal(member)
	_, _ = e.client.Put(ctx, key, string(val))

	return &etcdLeader{
		opts:   options,
		s:      s,
		e:      l,
		prefix: e.prefix,
		ns:     options.Namespace,
		id:     options.Id,
	}, nil
}

func (e *etcdSync) ListMembers(opts ...sync.ListMembersOption) ([]*sync.Member, error) {
	var options sync.ListMembersOptions
	for _, opt := range opts {
		opt(&options)
	}

	members := make([]*sync.Member, 0)

	key := path.Join(e.prefix, "leaders", options.Namespace)
	rsp, err := e.client.Get(context.TODO(), key, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	for _, kv := range rsp.Kvs {
		val := &sync.Member{}
		err = json.Unmarshal(kv.Value, &val)
		if err == nil {
			members = append(members, val)
		}
	}

	return members, nil
}

func (e *etcdSync) Lock(id string, opts ...sync.LockOption) error {
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

	ctx := context.TODO()
	if options.Wait != 0 {
		ctx, _ = context.WithTimeout(ctx, options.Wait)
	}

	ech := make(chan error, 1)
	go func() {
		ech <- m.Lock(ctx)
	}()

	select {
	case <-ctx.Done():
		return sync.ErrLockTimeout
	case err = <-ech:
		if err != nil {
			return err
		}
	}

	e.mtx.Lock()
	e.locks[id] = &etcdLock{
		s: s,
		m: m,
	}
	e.mtx.Unlock()
	return nil
}

func (e *etcdSync) Unlock(id string) error {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	v, ok := e.locks[id]
	if !ok {
		return errors.New("lock not found")
	}
	defer v.s.Client()
	err := v.m.Unlock(context.Background())
	delete(e.locks, id)
	return err
}

func (e *etcdSync) String() string {
	return "etcd"
}

func NewSync(opts ...sync.Option) sync.Sync {
	var options sync.Options
	for _, o := range opts {
		o(&options)
	}

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

	if options.Prefix == "" {
		options.Prefix = "/vine/sync"
	}

	// TODO: parse addresses
	c, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &etcdSync{
		prefix:  options.Prefix,
		client:  c,
		options: options,
		locks:   make(map[string]*etcdLock),
	}
}
