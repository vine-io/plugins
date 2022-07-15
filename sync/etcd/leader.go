package etcd

import (
	"context"
	"encoding/json"
	"path"

	"github.com/vine-io/vine/lib/sync"
	cc "go.etcd.io/etcd/client/v3/concurrency"
)

type Value struct {
	Namespace string `json:"namespace"`
	Id        string `json:"id"`
}

type etcdLeader struct {
	opts   sync.LeaderOptions
	s      *cc.Session
	e      *cc.Election
	prefix string
	name   string

	err     error
	elected chan struct{}
	stop    chan struct{}
}

func (e *etcdLeader) Id() string {
	return e.opts.Id
}

func (e *etcdLeader) Primary() (*sync.Member, error) {
	if e.err != nil {
		return nil, e.err
	}

	rsp, err := e.e.Leader(context.TODO())
	if err != nil {
		return nil, err
	}
	member := &sync.Member{}
	_ = json.Unmarshal(rsp.Kvs[0].Value, member)
	member.Role = sync.Primary
	return member, nil
}

func (e *etcdLeader) campaign() {
	member := &sync.Member{
		Leader:    e.name,
		Id:        e.opts.Id,
		Namespace: e.opts.Namespace,
	}

	text, _ := json.Marshal(member)

	ctx := context.Background()
	if err := e.e.Campaign(ctx, string(text)); err != nil {
		e.err = err
		return
	}
	key := path.Join(e.prefix, "primary", e.opts.Namespace)
	_, _ = e.s.Client().Put(context.TODO(), key, e.opts.Id)
	e.elected <- struct{}{}
}

func (e *etcdLeader) Resign() error {
	select {
	case <-e.stop:
		return nil
	case <-e.elected:
	default:
		return nil
	}

	close(e.stop)
	return e.e.Resign(context.Background())
}

func (e *etcdLeader) Observe() chan sync.ObserveResult {
	ch := make(chan sync.ObserveResult, 1)
	ech := e.e.Observe(context.Background())

	go func() {
		for {
			select {
			case <-e.stop:
				close(ch)
				return
			case r := <-ech:
				v := &sync.Member{}
				err := json.Unmarshal(r.Kvs[0].Value, &v)
				if err == nil {
					ch <- sync.ObserveResult{
						Namespace: v.Namespace,
						Id:        v.Id,
					}
				}
			}
		}
	}()

	return ch
}

func (e *etcdLeader) Status() chan bool {
	ch := make(chan bool, 1)
	ech := e.e.Observe(context.Background())

	go func() {
		for {
			select {
			case <-e.stop:
				close(ch)
				return
			case r := <-ech:
				v := &sync.Member{}
				err := json.Unmarshal(r.Kvs[0].Value, &v)
				if err == nil && v.Id == e.Id() {
					ch <- true
					close(ch)
					return
				}
			}

		}
	}()

	return ch
}
