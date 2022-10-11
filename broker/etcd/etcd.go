package etcd

import (
	"context"
	"errors"
	"net"
	"path"
	"strings"

	"github.com/vine-io/vine/core/broker"
	"github.com/vine-io/vine/core/codec"
	"github.com/vine-io/vine/core/codec/json"
	"github.com/vine-io/vine/lib/cmd"
	"go.etcd.io/etcd/client/v3"
)

func init() {
	cmd.DefaultBrokers["etcd"] = NewBroker
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
// is Etcd and therefore this is a no-op.
func (p *publication) Ack() error {
	return nil
}

func (p *publication) Error() error {
	return p.err
}

// subscribe proxies and handlers Redis messages as broker publications.
type subscriber struct {
	stop   chan struct{}
	codec  codec.Marshaler
	w      clientv3.WatchChan
	client *clientv3.Client
	topic  string
	handle broker.Handler
	opts   broker.SubscribeOptions
}

// recv loops to receive new messages from Redis and handle them
// as publications.
func (s *subscriber) recv() {
	// Close the connection once the subscriber stops receiving.

	for {

		select {
		case rsp := <-s.w:

			if rsp.Err() != nil || rsp.Canceled {
				return
			}

			for _, ev := range rsp.Events {
				if ev.Type != clientv3.EventTypePut {
					continue
				}

				var m broker.Message

				// Handle error? Only a log would be necessary since this type
				// of issue cannot be fixed.
				if err := s.codec.Unmarshal(ev.Kv.Value, &m); err != nil {
					continue
				}

				p := publication{
					topic:   s.topic,
					message: &m,
				}

				// Handle error? Retry?
				if p.err = s.handle(&p); p.err != nil {
					continue
				}

				// Added for posterity, however Ack is a no-op.
				if s.opts.AutoAck {
					if err := p.Ack(); err != nil {
						continue
					}
				}
			}

		case <-s.stop:
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
	select {
	case <-s.stop:
		return nil
	default:
		close(s.stop)
	}

	return nil
}

// broker implementation for Redis.
type etcdBroker struct {
	addr   string
	client *clientv3.Client
	opts   broker.Options
	bopts  *brokerOptions
}

// String returns the name of the broker implementation.
func (b *etcdBroker) String() string {
	return "etcd"
}

// Options returns the options defined for the broker.
func (b *etcdBroker) Options() broker.Options {
	return b.opts
}

// Address returns the address the broker will use to create new connections.
// This will be set only after Connect is called.
func (b *etcdBroker) Address() string {
	return b.addr
}

// Init sets or overrides broker options.
func (b *etcdBroker) Init(opts ...broker.Option) error {
	if b.client != nil {
		return errors.New("etcd: cannot init while connected")
	}

	for _, o := range opts {
		o(&b.opts)
	}

	return nil
}

// Connect establishes a connection to Redis which provides the
// pub/sub implementation.
func (b *etcdBroker) Connect() error {
	if b.client != nil {
		return nil
	}

	config := clientv3.Config{
		Endpoints: []string{"127.0.0.1:2379"},
		TLS:       b.opts.TLSConfig,
		Username:  b.bopts.username,
		Password:  b.bopts.password,
	}

	var cAddrs []string

	for _, address := range b.opts.Addrs {
		if len(address) == 0 {
			continue
		}
		addr, port, err := net.SplitHostPort(address)
		if ae, ok := err.(*net.AddrError); ok && ae.Err == "missing port in address" {
			port = "2379"
			addr = address
			cAddrs = append(cAddrs, net.JoinHostPort(addr, port))
		} else if err == nil {
			cAddrs = append(cAddrs, net.JoinHostPort(addr, port))
		}
	}

	// if we got addrs then we'll update
	if len(cAddrs) > 0 {
		config.Endpoints = cAddrs
	}
	b.addr = strings.Join(cAddrs, ",")

	var err error
	b.client, err = clientv3.New(config)
	if err != nil {
		return err
	}

	return nil
}

// Disconnect closes the connection pool.
func (b *etcdBroker) Disconnect() error {
	err := b.client.Close()
	b.client = nil
	b.addr = ""
	return err
}

// Publish publishes a message.
func (b *etcdBroker) Publish(ctx context.Context, topic string, msg *broker.Message, opts ...broker.PublishOption) error {
	v, err := b.opts.Codec.Marshal(msg)
	if err != nil {
		return err
	}

	var popts broker.PublishOptions
	for _, o := range opts {
		o(&popts)
	}

	opOpts := make([]clientv3.OpOption, 0)

	key := path.Join(b.bopts.prefix, topic)
	_, err = b.client.Put(ctx, key, string(v), opOpts...)
	if err != nil {
		return err
	}

	return nil
}

// Subscribe returns a subscriber for the topic and handler.
func (b *etcdBroker) Subscribe(topic string, handler broker.Handler, opts ...broker.SubscribeOption) (broker.Subscriber, error) {
	var options broker.SubscribeOptions
	for _, o := range opts {
		o(&options)
	}

	s := subscriber{
		codec:  b.opts.Codec,
		topic:  topic,
		handle: handler,
		opts:   options,
	}

	key := path.Join(b.bopts.prefix, topic)
	watcher := b.client.Watch(b.opts.Context, key, clientv3.WithPrefix())

	s.w = watcher
	s.client = b.client

	go s.recv()

	return &s, nil
}

// NewBroker returns a new broker implemented using the Etcd
func NewBroker(opts ...broker.Option) broker.Broker {
	// Default options
	bopts := &brokerOptions{
		timeout: DefaultTimeout,
		prefix:  DefaultPrefix,
	}

	// Initialize with empty broker options
	options := broker.Options{
		Codec:   json.Marshaler{},
		Context: context.WithValue(context.Background(), optionsKey, bopts),
	}

	for _, o := range opts {
		o(&options)
	}

	return &etcdBroker{
		opts:  options,
		bopts: bopts,
	}
}
