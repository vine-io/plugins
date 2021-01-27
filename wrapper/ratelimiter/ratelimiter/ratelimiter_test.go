// Copyright 2021 lack
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     https://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ratelimiter

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/juju/ratelimit"
	"github.com/lack-io/vine/proto/errors"
	bmemory "github.com/lack-io/vine/service/broker/memory"
	"github.com/lack-io/vine/service/client"
	gclient "github.com/lack-io/vine/service/client/mucp"
	"github.com/lack-io/vine/service/client/selector"
	tmemory "github.com/lack-io/vine/service/network/transport/memory"
	rmemory "github.com/lack-io/vine/service/registry/memory"
	"github.com/lack-io/vine/service/server"
	gserver "github.com/lack-io/vine/service/server/mucp"
)

type testHandler struct{}
type TestRequest struct{}
type TestResponse struct{}

func (t *testHandler) Method(ctx context.Context, req *TestRequest, rsp *TestResponse) error {
	return nil
}

func TestRateClientLimit(t *testing.T) {
	// setup
	r := rmemory.NewRegistry()
	s := selector.NewSelector(selector.Registry(r))
	tr := tmemory.NewTransport()
	testRates := []int{1, 10, 20}

	for _, limit := range testRates {
		b := ratelimit.NewBucketWithRate(float64(limit), int64(limit))

		c := gclient.NewClient(
			// set the selector
			client.Selector(s),
			client.Transport(tr),
			// add the breaker wrapper
			client.Wrap(NewClientWrapper(b, false)),
		)

		req := c.NewRequest(
			"test.service",
			"Test.Method",
			&TestRequest{},
			client.WithContentType("application/json"),
		)
		rsp := TestResponse{}

		for j := 0; j < limit; j++ {
			err := c.Call(context.TODO(), req, &rsp)
			e := errors.Parse(err.Error())
			if e.Code == 429 {
				t.Errorf("Unexpected rate limit error: %v", err)
			}
		}

		err := c.Call(context.TODO(), req, rsp)
		e := errors.Parse(err.Error())
		if e.Code != 429 {
			t.Errorf("Expected rate limit error, got: %v", err)
		}
	}
}

func TestRateServerLimit(t *testing.T) {
	// setup
	testRates := []int{1, 5, 6, 10}

	for _, limit := range testRates {
		r := rmemory.NewRegistry()
		b := bmemory.NewBroker()
		tr := tmemory.NewTransport()
		s := selector.NewSelector(selector.Registry(r))

		br := ratelimit.NewBucketWithRate(float64(limit), int64(limit))
		c := gclient.NewClient(client.Selector(s), client.Transport(tr))

		name := fmt.Sprintf("test.service.%d", limit)

		srv := gserver.NewServer(
			server.Name(name),
			// add registry
			server.Registry(r),
			server.Transport(tr),
			// add broker
			server.Broker(b),
			// add the breaker wrapper
			server.WrapHandler(NewHandlerWrapper(br, false)),
		)

		type Test struct {
			*testHandler
		}

		srv.Handle(
			srv.NewHandler(&Test{new(testHandler)}),
		)

		if err := srv.Start(); err != nil {
			t.Fatalf("Unexpected error starting server: %v", err)
		}
		req := c.NewRequest(name, "Test.Method", &TestRequest{}, client.WithContentType("application/json"))
		rsp := TestResponse{}

		for j := 0; j < limit; j++ {
			if err := c.Call(context.TODO(), req, &rsp); err != nil {
				t.Fatalf("Unexpected request error: %v", err)
			}
		}

		err := c.Call(context.TODO(), req, &rsp)
		if err == nil {
			t.Fatalf("Expected rate limit error, got nil: rate %d, err %v", limit, err)
		}

		e := errors.Parse(err.Error())
		if e.Code != 429 {
			t.Fatalf("Expected rate limit error, got %v", err)
		}

		srv.Stop()

		// artificial test delay
		time.Sleep(500 * time.Millisecond)
	}
}
