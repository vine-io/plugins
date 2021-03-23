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

package mysql

import (
	"context"

	"github.com/lack-io/vine/service/dao"
)

type driverNameKey struct{}

func DriverName(name string) dao.Option {
	return func(o *dao.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, driverNameKey{}, name)
	}
}

type skipInitializeWithVersionKey struct{}

func SkipInitializeWithVersion(b bool) dao.Option {
	return func(o *dao.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, skipInitializeWithVersionKey{}, b)
	}
}

type stringSizeKey struct{}

func StringSize(size uint) dao.Option {
	return func(o *dao.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, stringSizeKey{}, size)
	}
}

type datetimePrecisionKey struct{}

func DatetimePrecision(d int) dao.Option {
	return func(o *dao.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, datetimePrecisionKey{}, d)
	}
}

type disableDatetimePrecisionKey struct{}

func DisableDatetimePrecision(b bool) dao.Option {
	return func(o *dao.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, disableDatetimePrecisionKey{}, b)
	}
}

//DontSupportForShareClause bool

type dontSupportRenameIndexKey struct{}

func DontSupportRenameIndex(b bool) dao.Option {
	return func(o *dao.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, dontSupportRenameIndexKey{}, b)
	}
}

type dontSupportRenameColumnKey struct{}

func DontSupportRenameColumn(b bool) dao.Option {
	return func(o *dao.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, dontSupportRenameColumnKey{}, b)
	}
}

type dontSupportForShareClauseKey struct{}

func DontSupportForShareClause(b bool) dao.Option {
	return func(o *dao.Options) {
		if o.Context == nil {
			o.Context = context.Background()
		}
		o.Context = context.WithValue(o.Context, dontSupportForShareClauseKey{}, b)
	}
}
