package etcd

import (
	"encoding/json"
	"testing"

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
