package etcd

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/vine-io/vine/lib/sync"
)

func Test_etcdSync_Leader(t *testing.T) {
	s := NewSync(sync.Nodes("192.168.2.190:12379", "192.168.2.190:22379", "192.168.2.190:32379"))
	err := s.Init()
	if err != nil {
		t.Fatalf("sync init: %v", err)
	}

	ctx := context.TODO()
	id := "leader_test"
	_, err = s.Leader(ctx, id, sync.LeaderTTL(15))
	if err != nil {
		t.Fatalf("leader: %v", err)
	}

	go func() {
		_, err = s.Leader(ctx, id, sync.LeaderTTL(20))
		if err != nil {
			t.Fatalf("leader: %v", err)
		}
	}()

	select {}
}

func TestEtcdLeader_Resign(t *testing.T) {
	s := NewSync(sync.Nodes("192.168.2.190:12379"))
	err := s.Init()
	if err != nil {
		t.Fatalf("sync init: %v", err)
	}

	ctx := context.TODO()
	id := "lease_resign2"
	leader, err := s.Leader(ctx, id)
	if err != nil {
		t.Fatalf("leader: %v", err)
	}
	t.Logf("find new leader")

	err = leader.Resign()
	if err != nil {
		t.Fatalf("leader resign: %v", err)
	}
}

func TestEtcdLeader_Observe(t *testing.T) {
	s := NewSync(sync.Nodes("192.168.2.190:12379"))
	err := s.Init()
	if err != nil {
		t.Fatalf("sync init: %v", err)
	}

	ctx := context.TODO()
	id := "lease_resign"
	leader, err := s.Leader(ctx, id)
	if err != nil {
		t.Fatalf("leader: %v", err)
	}

	go func() {
		for v := range leader.Observe() {
			t.Logf("%v\n", v)
		}
	}()

	stop := make(chan struct{}, 1)
	go func() {
		l, _ := s.Leader(ctx, id)
		if l != nil {

			go func() {
				for v := range l.Observe() {
					t.Logf("%v\n", v)
				}
			}()

			t.Logf("leader %v", l.Id())
			time.Sleep(time.Second * 2)
			l.Resign()
		}
		stop <- struct{}{}
	}()

	time.Sleep(time.Second * 1)
	err = leader.Resign()
	if err != nil {
		t.Fatalf("leader resign: %v", err)
	}

	<-stop
}

func TestEtcdSync_ListMembers(t *testing.T) {
	s := NewSync(sync.Nodes("192.168.2.190:12379"))
	err := s.Init()
	if err != nil {
		t.Fatalf("sync init: %v", err)
	}

	ctx := context.TODO()
	id := "lease_member"
	l1, err := s.Leader(ctx, id, sync.LeaderNS("aa"))
	if err != nil {
		t.Fatalf("leader: %v", err)
	}
	//defer l1.Resign()

	l2, err := s.Leader(ctx, id, sync.LeaderNS("aa"))
	if err != nil {
		t.Fatalf("leader: %v", err)
	}
	defer l2.Resign()

	l3, err := s.Leader(ctx, id, sync.LeaderNS("aa"))
	if err != nil {
		t.Fatalf("leader: %v", err)
	}
	defer l3.Resign()

	<-l1.Status()
	time.Sleep(time.Second * 2)

	members, _ := s.ListMembers(ctx, sync.MemberNS("aa"))
	b, _ := json.Marshal(members)
	t.Log(string(b))
	if len(members) != 3 {
		t.Fatalf("member number expect %d, got %d", 3, len(members))
	}
	t.Logf("member: %v", members[0])

	_ = l1.Resign()
	select {
	case <-l2.Status():
	case <-l3.Status():
	}
	time.Sleep(time.Second * 1)

	members, _ = s.ListMembers(ctx, sync.MemberNS("aa"))
	b, _ = json.Marshal(members)
	t.Log(string(b))
	if len(members) != 2 {
		t.Fatalf("member number expect %d, got %d", 2, len(members))
	}
	t.Logf("member: %v", members[0])
}

func TestEtcdLeader_Watch(t *testing.T) {
	s := NewSync(sync.Nodes("192.168.2.190:12379"))
	err := s.Init()
	if err != nil {
		t.Fatalf("sync init: %v", err)
	}

	ctx := context.TODO()
	ns := "testns"
	watcher, _ := s.WatchElect(ctx, sync.WatchNS(ns))
	defer watcher.Close()
	go func() {
		for {
			m, _ := watcher.Next()
			t.Log(m)
		}
	}()

	id := "lease_resign2"
	leader, err := s.Leader(ctx, id, sync.LeaderNS(ns))
	if err != nil {
		t.Fatalf("leader: %v", err)
	}
	t.Logf("find new leader")

	err = leader.Resign()
	if err != nil {
		t.Fatalf("leader resign: %v", err)
	}

	time.Sleep(time.Second * 2)
}

func Test_etcdSync_Lock(t *testing.T) {
	s := NewSync()
	err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()
	lock := "lock23"
	err = s.Lock(ctx, lock, sync.LockTTL(time.Second*10))
	if err != nil {
		t.Fatalf("lock %s: %v", lock, err)
	}

	err = s.Lock(ctx, lock, sync.LockWait(time.Second*3))
	if err != sync.ErrLockTimeout {
		t.Fatalf("lock locked: %v", err)
	}

	err = s.Unlock(ctx, lock)
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
}
