// MIT License
//
// Copyright (c) 2020 Lack
//
// Permission is hereby granted, free of charge, to any person obtaining a copy
// of this software and associated documentation files (the "Software"), to deal
// in the Software without restriction, including without limitation the rights
// to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
// copies of the Software, and to permit persons to whom the Software is
// furnished to do so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package etcd

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"path"
	"sort"
	"strings"
	"sync"
	"time"

	json "github.com/json-iterator/go"
	hash "github.com/mitchellh/hashstructure/v2"
	"github.com/vine-io/vine/core/registry"
	"github.com/vine-io/vine/lib/cmd"
	log "github.com/vine-io/vine/lib/logger"
	"go.etcd.io/etcd/api/v3/v3rpc/rpctypes"
	"go.etcd.io/etcd/client/v3"
)

var (
	prefix = "/vine/registry/"
)

func init() {
	cmd.DefaultRegistries["etcd"] = NewRegistry

	registry.Flag.String("registry.etcd.username", "", "Sets the username for client of etcdv3")
	registry.Flag.String("registry.etcd.password", "", "Sets the password for client of etcdv3")
}

type Registry struct {
	client  *clientv3.Client
	options registry.Options

	sync.RWMutex
	register map[string]uint64
	leases   map[string]clientv3.LeaseID
}

func configure(e *Registry, client *clientv3.Client, opts ...registry.Option) error {

	var err error

	for _, o := range opts {
		o(&e.options)
	}

	if e.options.Timeout == 0 {
		e.options.Timeout = 10 * time.Second
	}

	if client == nil {
		config := clientv3.Config{
			Endpoints: []string{"127.0.0.1:2379"},
		}

		if e.options.Secure || e.options.TLSConfig != nil {
			tlsConfig := e.options.TLSConfig
			if tlsConfig == nil {
				tlsConfig = &tls.Config{
					InsecureSkipVerify: true,
				}
			}

			config.TLS = tlsConfig
		}

		if e.options.Context != nil {
			u, ok := e.options.Context.Value(authKey{}).(*authCreds)
			if ok {
				config.Username = u.Username
				config.Password = u.Password
			}
		}

		var cAddrs []string

		for _, address := range e.options.Addrs {
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

		client, err = clientv3.New(config)
	}

	if err != nil {
		return err
	}
	e.client = client
	return nil
}

func encode(s *registry.Service) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func decode(ds []byte) *registry.Service {
	var s *registry.Service
	json.Unmarshal(ds, &s)
	return s
}

func nodePath(ns, s, id string) string {
	service := strings.ReplaceAll(s, "/", "-")
	node := strings.ReplaceAll(id, "/", "-")
	return path.Join(prefix, ns, service, node)
}

func servicePath(ns, s string) string {
	return path.Join(prefix, ns, strings.Replace(s, "/", "-", -1))
}

func (e *Registry) Init(opts ...registry.Option) error {
	return configure(e, e.client, opts...)
}

func (e *Registry) Options() registry.Options {
	return e.options
}

func (e *Registry) registerNode(ctx context.Context, s *registry.Service, node *registry.Node, opts ...registry.RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("require at lease one node")
	}

	// check existing lease cache
	e.RLock()
	leaseID, ok := e.leases[s.Name+node.Id]
	e.RUnlock()

	var options registry.RegisterOptions
	for _, o := range opts {
		o(&options)
	}

	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	if !ok {
		// missing lease, check if the key exists

		// look for the existing key
		rsp, err := e.client.Get(ctx, nodePath(namespace, s.Name, node.Id), clientv3.WithSerializable())
		if err != nil {
			return err
		}

		// get the existing lease
		for _, kv := range rsp.Kvs {
			if kv.Lease > 0 {
				leaseID = clientv3.LeaseID(kv.Lease)

				// decode the existing node
				svc := decode(kv.Value)
				if svc == nil || len(svc.Nodes) == 0 {
					continue
				}

				// create hash of service; uint64
				h, err := hash.Hash(svc.Nodes[0], hash.FormatV2, nil)
				if err != nil {
					continue
				}

				// save the info
				e.Lock()
				e.leases[s.Name+node.Id] = leaseID
				e.register[s.Name+node.Id] = h
				e.Unlock()

				break
			}
		}
	}

	var leaseNotFound bool

	// renew the lease if it exists
	if leaseID > 0 {
		log.Debugf("Renewing existing lease for %s %d", s.Name, leaseID)

		if _, err := e.client.KeepAliveOnce(context.TODO(), leaseID); err != nil {
			if err != rpctypes.ErrLeaseNotFound {
				return err
			}

			log.Errorf("Lease not found for %s %d", s.Name, leaseID)
			// lease not found do register
			leaseNotFound = true
		}
	}

	// create hash of service; uint64
	h, err := hash.Hash(node, hash.FormatV2, nil)
	if err != nil {
		return err
	}

	// get existing hash for the service node
	e.Lock()
	v, ok := e.register[s.Name+node.Id]
	e.Unlock()

	// the service is unchanged, skip registering
	if ok && v == h && !leaseNotFound {
		log.Debugf("Service %s node %s unchanged skipping registration", s.Name, node.Id)
		return nil
	}

	if s.Namespace == "" {
		s.Namespace = namespace
	}

	service := &registry.Service{
		Name:      s.Name,
		Version:   s.Version,
		Namespace: s.Namespace,
		Metadata:  s.Metadata,
		Endpoints: s.Endpoints,
		Nodes:     []*registry.Node{node},
	}

	var lgr *clientv3.LeaseGrantResponse
	if options.TTL.Seconds() <= 0 {
		options.TTL = time.Second * 30
	}

	// get a lease used to expire keys since we have a ttl
	lgr, err = e.client.Grant(ctx, int64(options.TTL.Seconds()))
	if err != nil {
		return err
	}

	log.Infof("Registering %s namespace %s id %s with lease %v and leaseID %v and ttl %v", service.Name, service.Namespace, node.Id, lgr, lgr.ID, options.TTL)
	// create an entry for the node
	if lgr != nil {
		_, err = e.client.Put(ctx, nodePath(service.Namespace, service.Name, node.Id), encode(service), clientv3.WithLease(lgr.ID))
	} else {
		_, err = e.client.Put(ctx, nodePath(service.Namespace, service.Name, node.Id), encode(service))
	}
	if err != nil {
		return err
	}

	e.Lock()
	// save our hash of the service
	e.register[s.Name+node.Id] = h
	// save our leaseID of the service
	if lgr != nil {
		e.leases[s.Name+node.Id] = lgr.ID
	}
	e.Unlock()

	return nil
}

func (e *Registry) Deregister(ctx context.Context, s *registry.Service, opts ...registry.DeregisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("required at lease one node")
	}

	var options registry.DeregisterOptions
	for _, o := range opts {
		o(&options)
	}

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	if s.Namespace == "" {
		s.Namespace = namespace
	}

	for _, node := range s.Nodes {
		e.Lock()
		// delete our hash of the service
		delete(e.register, s.Name+node.Id)
		// delete our lease of the service
		delete(e.leases, s.Name+node.Id)
		e.Unlock()

		ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)

		log.Infof("Deregistering %s id %s", s.Name, node.Id)
		_, err := e.client.Delete(ctx, nodePath(namespace, s.Name, node.Id))
		if err != nil {
			cancel()
			return err
		}
		cancel()
	}

	return nil
}

func (e *Registry) Register(ctx context.Context, s *registry.Service, opts ...registry.RegisterOption) error {
	if len(s.Nodes) == 0 {
		return errors.New("require at lease one node")
	}

	var grr error

	// registry each node individually
	for _, node := range s.Nodes {
		err := e.registerNode(ctx, s, node, opts...)
		if err != nil {
			grr = err
		}
	}

	return grr
}

func (e *Registry) GetService(ctx context.Context, name string, opts ...registry.GetOption) ([]*registry.Service, error) {
	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	var options registry.GetOptions
	for _, o := range opts {
		o(&options)
	}

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	rsp, err := e.client.Get(ctx, servicePath(namespace, name), clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return nil, registry.ErrNotFound
	}

	serviceMap := map[string]*registry.Service{}

	for _, n := range rsp.Kvs {
		if sn := decode(n.Value); sn != nil {
			s, ok := serviceMap[sn.Version]
			if !ok {
				s = &registry.Service{
					Name:      sn.Name,
					Version:   sn.Version,
					Namespace: sn.Namespace,
					Metadata:  sn.Metadata,
					Endpoints: sn.Endpoints,
				}
				serviceMap[s.Version] = s
			}

			s.Nodes = append(s.Nodes, sn.Nodes...)
		}
	}

	services := make([]*registry.Service, 0, len(serviceMap))
	for _, service := range serviceMap {
		services = append(services, service)
	}

	return services, nil
}

func (e *Registry) ListServices(ctx context.Context, opts ...registry.ListOption) ([]*registry.Service, error) {
	versions := make(map[string]*registry.Service)

	ctx, cancel := context.WithTimeout(ctx, e.options.Timeout)
	defer cancel()

	var options registry.ListOptions
	for _, o := range opts {
		o(&options)
	}

	namespace := e.options.Namespace
	if options.Namespace != "" {
		namespace = options.Namespace
	}

	key := path.Join(prefix, namespace) + "/"
	rsp, err := e.client.Get(ctx, key, clientv3.WithPrefix(), clientv3.WithSerializable())
	if err != nil {
		return nil, err
	}

	if len(rsp.Kvs) == 0 {
		return []*registry.Service{}, nil
	}

	for _, n := range rsp.Kvs {
		sn := decode(n.Value)
		if sn == nil {
			continue
		}
		v, ok := versions[sn.Name+sn.Version]
		if !ok {
			versions[sn.Name+sn.Version] = sn
			continue
		}
		// append to service:version nodes
		v.Nodes = append(v.Nodes, sn.Nodes...)
	}

	services := make([]*registry.Service, 0, len(versions))
	for _, service := range versions {
		services = append(services, &registry.Service{
			Name:      service.Name,
			Version:   service.Version,
			Namespace: service.Namespace,
			Metadata:  service.Metadata,
			Nodes:     service.Nodes,
			Endpoints: service.Endpoints,
		})
	}

	// sort the services
	sort.Slice(services, func(i, j int) bool { return services[i].Name < services[j].Name })

	return services, nil
}

func (e *Registry) Watch(ctx context.Context, opts ...registry.WatchOption) (registry.Watcher, error) {
	return newEtcdWatcher(e, e.options.Timeout, opts...)
}

func (e *Registry) String() string {
	return "etcd"
}

func (e *Registry) GetConn() *clientv3.Client {
	return e.client
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	options := registry.NewOptions(opts...)
	e := &Registry{
		options:  options,
		register: make(map[string]uint64),
		leases:   make(map[string]clientv3.LeaseID),
	}

	return e
}

func NewEtcdRegistry(client *clientv3.Client, opts ...registry.Option) registry.Registry {
	options := registry.NewOptions(opts...)
	e := &Registry{
		client:   client,
		options:  options,
		register: make(map[string]uint64),
		leases:   make(map[string]clientv3.LeaseID),
	}

	return e
}
