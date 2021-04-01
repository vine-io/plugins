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

package postgres

import (
	"fmt"
	"strings"

	"github.com/lack-io/vine/service/dao"
	"github.com/lack-io/vine/service/dao/clause"
)

type jsonQueryExpression struct {
	tx          *dao.DB
	op          dao.JSONOp
	contains    bool
	column      string
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

func (j *jsonQueryExpression) Op(op dao.JSONOp, value interface{}, keys ...string) dao.JSONQuery {
	j.op = op
	j.keys = keys
	j.equalsValue = value
	return j
}

func (j *jsonQueryExpression) Contains(op dao.JSONOp, value interface{}, keys ...string) dao.JSONQuery {
	if j.tx != nil {
		j.tx.Statement.Join(fmt.Sprintf("CROSS JOIN LATERAL jsonb_array_elements(%s) o%s", j.tx.Statement.Quote(j.column), j.column))
	}
	j.contains = true
	j.op = op
	j.keys = keys
	j.equalsValue = value
	return j
}

func (j *jsonQueryExpression) Build(builder clause.Builder) {
	if stmt, ok := builder.(*dao.Statement); ok {
		if j.contains {
			if len(j.keys) == 0 {
				builder.WriteString(fmt.Sprintf("%s ? ", stmt.Quote("o"+j.column)))
			} else {
				builder.WriteString(fmt.Sprintf("%s %s ", join("o"+j.column, j.keys...), j.op.String()))
			}
			stmt.AddVar(builder, j.equalsValue)
		} else {
			if len(j.keys) > 0 {
				if j.op == dao.JSONHasKey {
					stmt.WriteQuoted(j.column)
					for _, key := range j.keys[0 : len(j.keys)-1] {
						stmt.WriteString("->")
						stmt.AddVar(builder, "'"+key+"'")
					}

					stmt.WriteString(" ? ")
					stmt.AddVar(builder, j.keys[len(j.keys)-1])
				} else {
					builder.WriteString(fmt.Sprintf("%s %s ", join(j.column, j.keys...), j.op.String()))
					stmt.AddVar(builder, j.equalsValue)
				}
			}

		}
	}
}

func join(column string, keys ...string) string {
	if len(keys) == 1 {
		return column + "->>" + "'" + keys[0] + "'"
	}
	outs := []string{column}
	for item, key := range keys {
		if item == len(keys)-1 {
			outs = append(outs, "->>")
		} else {
			outs = append(outs, "->")
		}
		outs = append(outs, "'"+key+"'")
	}
	return strings.Join(outs, "")
}
