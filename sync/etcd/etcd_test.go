package etcd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/vine-io/vine/lib/sync"
)

func Test_etcdSync_Leader(t *testing.T) {
	s := NewSync()
	err := s.Init()
	if err != nil {
		t.Fatalf("sync init: %v", err)
	}

	id := "leader_test"
	_, err = s.Leader(id)
	if err != nil {
		t.Fatalf("leader: %v", err)
	}
}

func TestEtcdLeader_Resign(t *testing.T) {
	s := NewSync()
	err := s.Init()
	if err != nil {
		t.Fatalf("sync init: %v", err)
	}

	id := "lease_resign"
	leader, err := s.Leader(id)
	if err != nil {
		t.Fatalf("leader: %v", err)
	}

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

	id := "lease_resign"
	leader, err := s.Leader(id)
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
		l, _ := s.Leader(id)
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
	s := NewSync()
	err := s.Init()
	if err != nil {
		t.Fatalf("sync init: %v", err)
	}

	id := "lease_member"
	l, err := s.Leader(id, sync.LeaderNS("aa"))
	if err != nil {
		t.Fatalf("leader: %v", err)
	}
	defer l.Resign()

	members, _ := s.ListMembers(sync.MemberNS("aa"))
	b, _ := json.Marshal(members)
	t.Log(string(b))
	if len(members) != 1 {
		t.Fatalf("member number expect %d, got %d", 1, len(members))
	}
	t.Logf("member: %v", members[0])
}

func Test_etcdSync_Lock(t *testing.T) {
	s := NewSync()
	err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	lock := "lock23"
	err = s.Lock(lock, sync.LockTTL(time.Second*10))
	if err != nil {
		t.Fatalf("lock %s: %v", lock, err)
	}

	err = s.Lock(lock, sync.LockWait(time.Second*3))
	if err != sync.ErrLockTimeout {
		t.Fatalf("lock locked: %v", err)
	}

	err = s.Unlock(lock)
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
}
