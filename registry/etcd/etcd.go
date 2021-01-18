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

package etcd

import (
	"github.com/lack-io/vine/service/config/cmd"
	"github.com/lack-io/vine/service/registry"
	"github.com/lack-io/vine/service/registry/etcd"
)

func init() {
	cmd.DefaultRegistries["etcd"] = NewRegistry
}

func NewRegistry(opts ...registry.Option) registry.Registry {
	return etcd.NewRegistry(opts...)
}
