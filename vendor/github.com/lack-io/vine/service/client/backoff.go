// Copyright 2020 The vine Authors
//
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

package client

import (
	"context"
	"time"

	"github.com/lack-io/vine/util/backoff"
)

type BackoffFunc func(ctx context.Context, req Request, attemps int) (time.Duration, error)

func exponentialBackoff(ctx context.Context, req Request, attempts int) (time.Duration, error) {
	return backoff.Do(attempts), nil
}
