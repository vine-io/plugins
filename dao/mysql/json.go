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
	"fmt"
	"strings"

	"github.com/lack-io/vine/service/dao"
	"github.com/lack-io/vine/service/dao/clause"
)

type jsonQueryExpression struct {
	tx          *dao.DB
	op          dao.JSONOp
	column      string
	contains    bool
	keys        []string
	equalsValue interface{}
}

func JSONQuery(column string) *jsonQueryExpression {
	return &jsonQueryExpression{column: column}
}

func (j *jsonQueryExpression) Tx(tx *dao.DB) dao.JSONQuery {
	j.tx = tx
	return j
}

func (j *jsonQueryExpression) Contains(op dao.JSONOp, value interface{}, keys ...string) dao.JSONQuery {
	j.contains = true
	j.op = op
	j.keys = keys
	j.equalsValue = value
	return j
}

func (j *jsonQueryExpression) Op(op dao.JSONOp, value interface{}, keys ...string) dao.JSONQuery {
	j.op = op
	j.keys = keys
	j.equalsValue = value
	return j
}

func (j *jsonQueryExpression) Build(builder clause.Builder) {
	if stmt, ok := builder.(*dao.Statement); ok {
		if j.contains {
			if len(j.keys) == 0 {
				builder.WriteString(fmt.Sprintf("JSON_CONTAINS(%s, '?', '$')", stmt.Quote(j.column)))
			} else {
				builder.WriteString(fmt.Sprintf("JSON_CONTAINS(%s, '?', '$.%s')", stmt.Quote(j.column), strings.Join(j.keys, ".")))
			}
			stmt.AddVar(builder, j.equalsValue)
		} else {
			if len(j.keys) > 0 {
				if j.op == dao.JSONHasKey {
					builder.WriteString(fmt.Sprintf("JSON_EXTRACT(%s, '$.%s') IS NOT NULL", stmt.Quote(j.column), strings.Join(j.keys, ".")))
				} else {
					if len(j.keys) > 0 {
						builder.WriteString(fmt.Sprintf("JSON_EXTRACT(%s, '$.%s') %s ", stmt.Quote(j.column), strings.Join(j.keys, "."), j.op.String()))
						stmt.AddVar(builder, j.equalsValue)
					}
				}
			}
		}
	}
}
