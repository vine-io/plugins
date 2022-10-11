package etcd

import (
	"context"
	"testing"

	"github.com/vine-io/vine/core/broker"
)

var (
	b     broker.Broker
	topic = "test_broker"
)

func TestNewBroker(t *testing.T) {
	b := NewBroker()
	err := b.Init()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_etcdBroker_Connect(t *testing.T) {
	b := NewBroker()
	err := b.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = b.Connect()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_etcdBroker_Disconnect(t *testing.T) {
	b := NewBroker()
	err := b.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = b.Connect()
	if err != nil {
		t.Fatal(err)
	}
	err = b.Disconnect()
	if err != nil {
		t.Fatal(err)
	}
}

func Test_etcdBroker_Publish(t *testing.T) {
	b := NewBroker()
	err := b.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = b.Connect()
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.TODO()
	err = b.Publish(ctx, topic, &broker.Message{
		Header: map[string]string{},
		Body:   []byte("broker test message"),
	})
	if err != nil {
		t.Fatal(err)
	}
}

func Test_etcdBroker_Subscribe(t *testing.T) {
	b := NewBroker()
	err := b.Init()
	if err != nil {
		t.Fatal(err)
	}
	err = b.Connect()
	if err != nil {
		t.Fatal(err)
	}

	done := make(chan string, 1)
	go b.Subscribe(topic, func(event broker.Event) error {

		msg := string(event.Message().Body)
		t.Logf("get message: %v", msg)

		done <- msg
		return nil
	})

	ctx := context.TODO()
	msg := "broker test message2"
	err = b.Publish(ctx, topic, &broker.Message{
		Header: map[string]string{},
		Body:   []byte(msg),
	})
	if err != nil {
		t.Fatal(err)
	}

	result := <-done
	if result != msg {
		t.Fatal("pub/sub not matched")
	}
}
