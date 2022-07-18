// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package redis

import (
	"context"
	"errors"

	"github.com/go-redis/redis/v8"
	"github.com/vine-io/vine/core/broker"
	"github.com/vine-io/vine/core/codec"
	"github.com/vine-io/vine/core/codec/json"
	"github.com/vine-io/vine/lib/cmd"
)

func init() {
	cmd.DefaultBrokers["redis"] = NewBroker
}

// publication is an internal publication for the broker.
type publication struct {
	topic   string
	message *broker.Message
	err     error
}

// Topic returns the topic this publication applies to.
func (p *publication) Topic() string {
	return p.topic
}

// Message returns the broker message of the publication
func (p *publication) Message() *broker.Message {
	return p.message
}

// Ack sends an acknowledgement to the broker. However this is not supported
// is Redis and therefore this is a no-op.
func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}

// subscribe proxies and handlers Redis messages as broker publications.
type subscriber struct {
	codec  codec.Marshaler
	conn   *redis.PubSub
	topic  string
	handle broker.Handler
	opts   broker.SubscribeOptions
}

// recv loops to receive new messages from Redis and handle them
// as publications.
func (s *subscriber) recv() {
	// Close the connection once the subscriber stops receiving.
	defer s.conn.Close()

	for {
		v, err := s.conn.Receive(s.opts.Context)
		if err != nil {
			return
		}
		switch x := v.(type) {
		case redis.Message:
			var m broker.Message

			// Handle error? Only a log would be necessary since this type
			// of issue cannot be fixed.
			if err := s.codec.Unmarshal([]byte(x.Payload), &m); err != nil {
				break
			}

			p := publication{
				topic:   x.Channel,
				message: &m,
			}

			// Handle error? Retry?
			if p.err = s.handle(&p); p.err != nil {
				break
			}

			// Added for posterity, however Ack is a no-op.
			if s.opts.AutoAck {
				if err := p.Ack(); err != nil {
					break
				}
			}

		case redis.Subscription:
			if x.Count == 0 {
				return
			}

		case error:
			return
		}
	}
}

// Options returns the subscriber options
func (s *subscriber) Options() broker.SubscribeOptions {
	return s.opts
}

// Topic returns the topic of the subscriber.
func (s *subscriber) Topic() string {
	return s.topic
}

// Unsubscribe unsubscribes the subscriber and frees the connection.
func (s *subscriber) Unsubscribe() error {
	return s.conn.Unsubscribe(context.TODO(), s.topic)
}

// broker implementation for Redis.
type redisBroker struct {
	addr   string
	client *redis.Client
	opts   broker.Options
	bopts  *brokerOptions
}

// String returns the name of the broker implementation.
func (b *redisBroker) String() string {
	return "redis"
}

// Options returns the options defined for the broker.
func (b *redisBroker) Options() broker.Options {
	return b.opts
}

// Address returns the address the broker will use to create new connections.
// This will be set only after Connect is called.
func (b *redisBroker) Address() string {
	return b.addr
}

// Init sets or overrides broker options.
func (b *redisBroker) Init(opts ...broker.Option) error {
	if b.client != nil {
		return errors.New("redis: cannot init while connected")
	}

	for _, o := range opts {
		o(&b.opts)
	}

	return nil
}

// Connect establishes a connection to Redis which provides the
// pub/sub implementation.
func (b *redisBroker) Connect() error {
	if b.client != nil {
		return nil
	}

	var addr string

	if len(b.opts.Addrs) == 0 || b.opts.Addrs[0] == "" {
		addr = "127.0.0.1:6379"
	} else {
		addr = b.opts.Addrs[0]
	}

	b.addr = addr

	opts := &redis.Options{
		Addr:         b.addr,
		Username:     b.bopts.username,
		Password:     b.bopts.password,
		DB:           b.bopts.db,
		PoolSize:     b.bopts.poolSize,
		IdleTimeout:  b.bopts.idleTimeout,
		DialTimeout:  b.bopts.connectTimeout,
		ReadTimeout:  b.bopts.readTimeout,
		WriteTimeout: b.bopts.writeTimeout,
		OnConnect: func(ctx context.Context, cn *redis.Conn) error {
			return cn.Ping(ctx).Err()
		},
	}

	b.client = redis.NewClient(opts)

	return nil
}

// Disconnect closes the connection pool.
func (b *redisBroker) Disconnect() error {
	err := b.client.Close()
	b.client = nil
	b.addr = ""
	return err
}

// Publish publishes a message.
func (b *redisBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	v, err := b.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}

	conn := b.client.Conn(ctx)
	err = conn.Publish(ctx, topic, v).Err()
	conn.Close()

	return err
}

// Subscribe returns a subscriber for the topic and handler.
func (b *redisBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	s := subscriber{
		codec:  b.opts.Codec,
		conn:   b.client.Subscribe(b.opts.Context, topic),
		topic:  topic,
		handle: handler,
		opts:   options,
	}

	go s.recv()

	return &s, nil
}

// NewBroker returns a new broker implemented using the Redis pub/sub
// protocol. The connection address may be a fully qualified IANA address
func NewBroker(opts ...broker.Option) broker.Broker {
	// Default options
	bopts := &brokerOptions{
		poolSize:       DefaultPoolSize,
		connectTimeout: DefaultConnectTimeout,
		idleTimeout:    DefaultIdleTimeout,
		readTimeout:    DefaultReadTimeout,
		writeTimeout:   DefaultWriteTimeout,
		db:             DefaultDB,
	}

	// Initialize with empty broker options
	options := broker.Options{
		Codec:   json.Marshaler{},
		Context: context.WithValue(context.Background(), optionsKey, bopts),
	}

	for _, o := range opts {
		o(&options)
	}

	return &redisBroker{
		opts:  options,
		bopts: bopts,
	}
}
