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

// Package memory provides a sync.Mutex implementation of the lock for local use
package memory

import (
	gosync "sync"
	"time"

	"github.com/vine-io/vine/lib/sync"
)

type memorySync struct {
	options sync.Options

	mtx   gosync.RWMutex
	locks map[string]*memoryLock
}

type memoryLock struct {
	id      string
	time    time.Time
	ttl     time.Duration
	release chan bool
}

type memoryLeader struct {
	opts   sync.LeaderOptions
	id     string
	resign func(id string) error
	status chan bool
}

func (m *memoryLeader) Resign() error {
	return m.resign(m.id)
}

func (m *memoryLeader) Status() chan bool {
	return m.status
}

func (m *memorySync) Leader(id string, opts ...sync.LeaderOption) (sync.Leader, error) {
	var once gosync.Once
	var options sync.LeaderOptions
	for _, o := range opts {
		o(&options)
	}

	// acquire a lock for the id
	if err := m.Lock(id); err != nil {
		return nil, err
	}

	// return the leader
	return &memoryLeader{
		opts: options,
		id:   id,
		resign: func(id string) error {
			once.Do(func() {
				m.Unlock(id)
			})
			return nil
		},
		// TODO: signal when Unlock is called
		status: make(chan bool, 1),
	}, nil
}

func (m *memorySync) Init(opts ...sync.Option) error {
	for _, o := range opts {
		o(&m.options)
	}
	return nil
}

func (m *memorySync) Options() sync.Options {
	return m.options
}

func (m *memorySync) Lock(id string, opts ...sync.LockOption) error {
	// lock our access
	m.mtx.Lock()

	var options sync.LockOptions
	for _, o := range opts {
		o(&options)
	}

	lk, ok := m.locks[id]
	if !ok {
		m.locks[id] = &memoryLock{
			id:      id,
			time:    time.Now(),
			ttl:     options.TTL,
			release: make(chan bool),
		}
		// unlock
		m.mtx.Unlock()
		return nil
	}

	m.mtx.Unlock()

	// set wait time
	var wait <-chan time.Time
	var ttl <-chan time.Time

	// decide if we should wait
	if options.Wait > time.Duration(0) {
		wait = time.After(options.Wait)
	}

	// check the ttl of the lock
	if lk.ttl > time.Duration(0) {
		// time lived for the lock
		live := time.Since(lk.time)

		// set a timer for the leftover ttl
		if live > lk.ttl {
			// release the lock if it expired
			_ = m.Unlock(id)
		} else {
			ttl = time.After(live)
		}
	}

lockLoop:
	for {
		// wait for the lock to be released
		select {
		case <-lk.release:
			m.mtx.Lock()

			// someone locked before us
			lk, ok = m.locks[id]
			if ok {
				m.mtx.Unlock()
				continue
			}

			// got chance to lock
			m.locks[id] = &memoryLock{
				id:      id,
				time:    time.Now(),
				ttl:     options.TTL,
				release: make(chan bool),
			}

			m.mtx.Unlock()

			break lockLoop
		case <-ttl:
			// ttl exceeded
			_ = m.Unlock(id)
			// TODO: check the ttl again above
			ttl = nil
			// try acquire
			continue
		case <-wait:
			return sync.ErrLockTimeout
		}
	}

	return nil
}

func (m *memorySync) Unlock(id string) error {
	m.mtx.Lock()
	defer m.mtx.Unlock()

	lk, ok := m.locks[id]
	// no lock exists
	if !ok {
		return nil
	}

	// delete the lock
	delete(m.locks, id)

	select {
	case <-lk.release:
		return nil
	default:
		close(lk.release)
	}

	return nil
}

func (m *memorySync) String() string {
	return "memory"
}

func NewSync(opts ...sync.Option) sync.Sync {
	var options sync.Options
	for _, o := range opts {
		o(&options)
	}

	return &memorySync{
		options: options,
		locks:   make(map[string]*memoryLock),
	}
}