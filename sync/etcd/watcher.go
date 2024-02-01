package etcd

import (
	"context"
	"encoding/json"
	"errors"
	"path"

	"github.com/vine-io/vine/lib/sync"
	clientv3 "go.etcd.io/etcd/client/v3"
)

type etcdWatcher struct {
	stop   chan struct{}
	w      clientv3.WatchChan
	client *clientv3.Client
}

func newEtcdWatcher(e *EtcdSync, opts ...sync.WatchElectOption) (sync.ElectWatcher, error) {
	var wo sync.WatchElectOptions
	for _, o := range opts {
		o(&wo)
	}

	stop := make(chan struct{}, 1)

	key := path.Join(e.prefix, "leaders", wo.Namespace)
	watchChan := e.client.Watch(context.TODO(), key, clientv3.WithPrefix())

	return &etcdWatcher{
		stop: stop,
		w:    watchChan,
	}, nil
}

func (ew *etcdWatcher) Next() (*sync.Member, error) {
	for wresp := range ew.w {
		if wresp.Err() != nil {
			return nil, wresp.Err()
		}
		if wresp.Canceled {
			return nil, errors.New("could not get next, watch is canceled")
		}
		for _, ev := range wresp.Events {
			if ev.Type == clientv3.EventTypeDelete {
				continue
			}
			member := &sync.Member{}
			err := json.Unmarshal(ev.Kv.Value, &member)
			if err != nil {
				return nil, err
			}
			return member, nil
		}
	}

	return nil, errors.New("could not get next")
}

func (ew *etcdWatcher) Close() {
	select {
	case <-ew.stop:
		return
	default:
		close(ew.stop)
	}
}
