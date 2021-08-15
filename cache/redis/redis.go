package redis

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
	"github.com/vine-io/vine/lib/cache"
	log "github.com/vine-io/vine/lib/logger"
)

type rkv struct {
	ctx     context.Context
	options cache.Options
	Client  *redis.Client
}

func (r *rkv) Init(opts ...cache.Option) error {
	for _, o := range opts {
		o(&r.options)
	}

	return r.configure()
}

func (r *rkv) Close() error {
	return r.Client.Close()
}

func (r *rkv) Get(key string, opts ...cache.GetOption) ([]*cache.Record, error) {
	options := cache.GetOptions{}
	options.Table = r.options.Table

	for _, o := range opts {
		o(&options)
	}

	var keys []string

	rkey := fmt.Sprintf("%s%s", options.Table, key)
	// Handle Prefix
	// TODO suffix
	if options.Prefix {
		prefixKey := fmt.Sprintf("%s*", rkey)
		fkeys, err := r.Client.Keys(r.ctx, prefixKey).Result()
		if err != nil {
			return nil, err
		}
		// TODO Limit Offset

		keys = append(keys, fkeys...)

	} else {
		keys = []string{rkey}
	}

	records := make([]*cache.Record, 0, len(keys))

	for _, rkey = range keys {
		val, err := r.Client.Get(r.ctx, rkey).Bytes()

		if err != nil && err == redis.Nil {
			return nil, cache.ErrNotFound
		} else if err != nil {
			return nil, err
		}

		if val == nil {
			return nil, cache.ErrNotFound
		}

		d, err := r.Client.TTL(r.ctx, rkey).Result()
		if err != nil {
			return nil, err
		}

		records = append(records, &cache.Record{
			Key:    key,
			Value:  val,
			Expiry: d,
		})
	}

	return records, nil
}

func (r *rkv) Del(key string, opts ...cache.DelOption) error {
	options := cache.DelOptions{}
	options.Table = r.options.Table

	for _, o := range opts {
		o(&options)
	}

	rkey := fmt.Sprintf("%s%s", options.Table, key)
	return r.Client.Del(r.ctx, rkey).Err()
}

func (r *rkv) Put(record *cache.Record, opts ...cache.PutOption) error {
	options := cache.PutOptions{}
	options.Table = r.options.Table

	for _, o := range opts {
		o(&options)
	}

	rkey := fmt.Sprintf("%s%s", options.Table, record.Key)
	return r.Client.Set(r.ctx, rkey, record.Value, record.Expiry).Err()
}

func (r *rkv) List(opts ...cache.ListOption) ([]string, error) {
	options := cache.ListOptions{}
	options.Table = r.options.Table

	for _, o := range opts {
		o(&options)
	}

	keys, err := r.Client.Keys(r.ctx, "*").Result()
	if err != nil {
		return nil, err
	}

	return keys, nil
}

func (r *rkv) Options() cache.Options {
	return r.options
}

func (r *rkv) String() string {
	return "redis"
}

func NewCache(opts ...cache.Option) cache.Cache {
	var options cache.Options
	for _, o := range opts {
		o(&options)
	}

	s := &rkv{
		ctx:     context.Background(),
		options: options,
	}

	if err := s.configure(); err != nil {
		log.Fatal(err)
	}

	return s
}

func (r *rkv) configure() error {
	var redisOptions *redis.Options
	nodes := r.options.Nodes

	if len(nodes) == 0 {
		nodes = []string{"redis://127.0.0.1:6379"}
	}

	redisOptions, err := redis.ParseURL(nodes[0])
	if err != nil {
		//Backwards compatibility
		redisOptions = &redis.Options{
			Addr:     nodes[0],
			Password: "", // no password set
			DB:       0,  // use default DB
		}
	}

	r.Client = redis.NewClient(redisOptions)

	return nil
}
