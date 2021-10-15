package etcd

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/vine-io/vine/lib/sync"
)

func Test_etcdSync_Leader(t *testing.T) {
	s := NewSync(sync.Nodes("192.168.2.80:2379"))
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
	s := NewSync(sync.Nodes("http://192.168.2.80:2379"))
	err := s.Init()
	if err != nil {
		t.Fatal(err)
	}

	lock := "lock23"
	err = s.Lock(lock, sync.LockTTL(time.Second*10))
	if err != nil {
		t.Fatalf("lock %s: %v", lock, err)
	}

	//err = s.Lock(lock, sync.LockWait(time.Second * 3))
	//if err != sync.ErrLockTimeout {
	//	t.Fatalf("lock locked: %v", err)
	//}

	err = s.Unlock(lock)
	if err != nil {
		t.Fatalf("unlock: %v", err)
	}
}
