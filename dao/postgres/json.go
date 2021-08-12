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

package postgres

import (
	"fmt"
	"strings"

	"github.com/vine-io/vine/lib/dao"
	"github.com/vine-io/vine/lib/dao/clause"
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
