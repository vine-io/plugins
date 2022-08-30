package etcd

import (
	"context"
	"testing"

	"github.com/vine-io/vine/lib/cache"
)

var testCache cache.Cache

func TestNewCache(t *testing.T) {
	options := []cache.Option{
		cache.Nodes("127.0.0.1:2379"),
		cache.WithContext(context.TODO()),
	}

	testCache = NewCache(options...)
	if err := testCache.Init(options...); err != nil {
		t.Fatal(err)
	}
}

func Test_etcdCache_Put(t *testing.T) {
	if testCache == nil {
		return
	}

	record := &cache.Record{
		Key:      "record",
		Value:    []byte("record value"),
		Metadata: map[string]interface{}{"label": "etcd"},
		Expiry:   0,
	}
	err := testCache.Put(context.TODO(), record)
	if err != nil {
		t.Fatal(err)
	}
}

func Test_etcdCache_Get(t *testing.T) {
	if testCache == nil {
		return
	}
	records, err := testCache.Get(context.TODO(), "record")
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatal("no record")
	}
	t.Log(records)
}

func Test_etcdCache_List(t *testing.T) {
	if testCache == nil {
		return
	}
	records, err := testCache.List(context.TODO())
	if err != nil {
		t.Fatal(err)
	}
	if len(records) == 0 {
		t.Fatal("no record")
	}
	t.Log(records)
}

func Test_etcdCache_Del(t *testing.T) {
	if testCache == nil {
		return
	}
	err := testCache.Del(context.TODO(), "record")
	if err != nil {
		t.Fatal(err)
	}
}

func Test_etcdCache_Close(t *testing.T) {
	if testCache == nil {
		return
	}
	err := testCache.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_etcdCache_String(t *testing.T) {
	if testCache == nil {
		return
	}
	if testCache.String() != "etcd" {
		t.Fatal("invalid string")
	}
}
