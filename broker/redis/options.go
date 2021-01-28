package redis

import (
	"time"

	"github.com/lack-io/vine/service/broker"
)

var (
	DefaultPoolSize       = 5
	DefaultIdleTimeout    = 2 * time.Minute
	DefaultConnectTimeout = 5 * time.Second
	DefaultReadTimeout    = 5 * time.Second
	DefaultWriteTimeout   = 5 * time.Second
	DefaultDB             = 0

	optionsKey = optionsKeyType{}
)

// options contain additional options for the broker.
type brokerOptions struct {
	poolSize       int
	idleTimeout    time.Duration
	connectTimeout time.Duration
	readTimeout    time.Duration
	writeTimeout   time.Duration
	username       string
	password       string
	db             int
}

type optionsKeyType struct{}

func ConnectTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.connectTimeout = d
	}
}

func ReadTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.readTimeout = d
	}
}

func WriteTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.writeTimeout = d
	}
}

func PoolSize(n int) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.poolSize = n
	}
}

func IdleTimeout(d time.Duration) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.idleTimeout = d
	}
}

func Auth(username, password string) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.username, bo.password = username, password
	}
}

func DB(db int) broker.Option {
	return func(o *broker.Options) {
		bo := o.Context.Value(optionsKey).(*brokerOptions)
		bo.db = db
	}
}
