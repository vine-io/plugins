package etcd

import (
	"time"

	"github.com/vine-io/vine/core/broker"
)

var (
	DefaultTimeout = 15 * time.Second
	DefaultPrefix  = "/vine.etcd.broker/"
	optionsKey     = optionsKeyType{}
)

// options contain additional options for the broker.
type brokerOptions struct {
	timeout  time.Duration
	prefix   string
	username string
	password string
}

type optionsKeyType struct{}

func Timeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.timeout = d
	}
}

func Auth(username, password string) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.username, bo.password = username, password
	}
}

func Prefix(prefix string) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.prefix = prefix
	}
}
