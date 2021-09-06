package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"path"
	"strings"
	gosync "sync"

	"github.com/vine-io/vine/lib/sync"
	"go.etcd.io/etcd/client/v3"
	cc "go.etcd.io/etcd/client/v3/concurrency"
)

type etcdSync struct {
	options sync.Options
	path    string
	client  *clientv3.Client

	leaderStore map[string]string
	leaderMtx   gosync.RWMutex

	mtx   gosync.Mutex
	locks map[string]*etcdLock
}

type etcdLock struct {
	s *cc.Session
	m *cc.Mutex
}

type etcdLeader struct {
	opts sync.LeaderOptions
	p    *etcdSync
	s    *cc.Session
	e    *cc.Election
	id   string
}

type Value struct {
	Namespace string `json:"namespace"`
	Id        string `json:"id"`
}

func (e *etcdLeader) Status() chan bool {
	ch := make(chan bool, 1)
	ech := e.e.Observe(context.Background())

	go func() {
		for r := range ech {
			v, val := r.Kvs[0].Value, &Value{}
			err := json.Unmarshal(v, &val)
			if err == nil && val.Id == e.id {
				e.p.leaderMtx.Lock()
				e.p.leaderStore[val.Namespace] = val.Id
				e.p.leaderMtx.Unlock()

				ch <- true
				close(ch)
				return
			}
		}
	}()

	return ch
}

func (e *etcdLeader) Resign() error {
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

func (e *etcdSync) Leader(id string, opts ...sync.LeaderOption) (sync.Leader, error) {
	var options sync.LeaderOptions
	for _, o := range opts {
		o(&options)
	}

	// make path
	path := path.Join(e.path, strings.Replace(path.Join(e.options.Prefix, options.Namespace, id), "/", "-", -1))

	s, err := cc.NewSession(e.client)
	if err != nil {
		return nil, err
	}

	l := cc.NewElection(s, path)

	val := &Value{
		Namespace: options.Namespace,
		Id:        id,
	}
	data, _ := json.Marshal(val)
	if err = l.Campaign(context.TODO(), string(data)); err != nil {
		return nil, err
	}

	e.leaderMtx.Lock()
	e.leaderStore[options.Namespace] = id
	e.leaderMtx.Unlock()

	return &etcdLeader{
		opts: options,
		p:    e,
		e:    l,
		id:   id,
	}, nil
}

func (e *etcdSync) ListMembers(opts ...sync.ListMembersOption) ([]*sync.Member, error) {
	var options sync.ListMembersOptions
	for _, opt := range opts {
		opt(&options)
	}

	members := make([]*sync.Member, 0)

	path := path.Join(e.path, strings.Replace(path.Join(e.options.Prefix, options.Namespace), "/", "-", -1))
	rsp, err := e.client.Get(context.TODO(), path, clientv3.WithPrefix())
	if err != nil {
		return nil, err
	}

	for _, kv := range rsp.Kvs {
		val := &Value{}
		err := json.Unmarshal(kv.Value, &val)
		if err != nil {
			continue
		}

		e.leaderMtx.RLock()
		id := e.leaderStore[val.Namespace]
		e.leaderMtx.RUnlock()

		role := sync.Follow
		if val.Id == id {
			role = sync.Primary
		}
		members = append(members, &sync.Member{Id: val.Id, Namespace: val.Namespace, Role: role})
	}

	return members, nil
}

func (e *etcdSync) Lock(id string, opts ...sync.LockOption) error {
	var options sync.LockOptions
	for _, o := range opts {
		o(&options)
	}

	// make path
	path := path.Join(e.path, strings.Replace(e.options.Prefix+id, "/", "-", -1))

	var sopts []cc.SessionOption
	if options.TTL > 0 {
		sopts = append(sopts, cc.WithTTL(int(options.TTL.Seconds())))
	}

	s, err := cc.NewSession(e.client, sopts...)
	if err != nil {
		return err
	}

	m := cc.NewMutex(s, path)

	if err = m.Lock(context.TODO()); err != nil {
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

func (e *etcdSync) Unlock(id string) error {
	e.mtx.Lock()
	defer e.mtx.Unlock()
	v, ok := e.locks[id]
	if !ok {
		return errors.New("lock not found")
	}
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
			endpoints = append(endpoints, addr)
		}
	}

	if len(endpoints) == 0 {
		endpoints = []string{"http://127.0.0.1:2379"}
	}

	// TODO: parse addresses
	c, err := clientv3.New(clientv3.Config{
		Endpoints: endpoints,
	})
	if err != nil {
		log.Fatal(err)
	}

	return &etcdSync{
		path:        "/vine/sync",
		client:      c,
		options:     options,
		leaderStore: map[string]string{},
		locks:       make(map[string]*etcdLock),
	}
}
