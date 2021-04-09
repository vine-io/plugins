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
	"regexp"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/stdlib"
	"github.com/lack-io/vine/service/dao"
	"github.com/lack-io/vine/service/dao/callbacks"
	"github.com/lack-io/vine/service/dao/clause"
	"github.com/lack-io/vine/service/dao/logger"
	"github.com/lack-io/vine/service/dao/migrator"
	"github.com/lack-io/vine/service/dao/schema"
)

const (
	// DefaultDriverName is the default driver name for SQLite.
	DefaultDriverName = "postgres"
)

type Dialect struct {
	DB                   *dao.DB
	Opts                 dao.Options
	DriverName           string
	Conn                 dao.ConnPool
	PreferSimpleProtocol bool
	WithOutReturning     bool
}

func newPGDialect(opts ...dao.Option) dao.Dialect {
	options := dao.NewOptions(opts...)

	for _, opt := range opts {
		opt(&options)
	}

	dialect := &Dialect{
		Opts: options,
		Conn: options.ConnPool,
	}

	if name, ok := options.Context.Value(driverNameKey{}).(string); ok {
		dialect.DriverName = name
	}

	if b, ok := options.Context.Value(preferSimpleProtocolKey{}).(bool); ok {
		dialect.PreferSimpleProtocol = b
	}

	if b, ok := options.Context.Value(withOutReturningKey{}).(bool); ok {
		dialect.WithOutReturning = b
	}

	return dialect
}

func (d *Dialect) Init(opts ...dao.Option) (err error) {
	for _, opt := range opts {
		opt(&d.Opts)
	}

	if name, ok := d.Opts.Context.Value(driverNameKey{}).(string); ok {
		d.DriverName = name
	} else {
		d.DriverName = DefaultDriverName
	}

	if d.DB == nil {
		d.DB, err = dao.Open(d)
		if err != nil {
			return err
		}
	}

	callbacks.RegisterDefaultCallbacks(d.DB, &callbacks.Options{
		WithReturning: !d.WithOutReturning,
	})

	if d.Conn != nil {
		d.DB.ConnPool = d.Conn
	} else {
		var config *pgx.ConnConfig

		config, err = pgx.ParseConfig(d.Opts.DSN)
		if err != nil {
			return err
		}
		if d.PreferSimpleProtocol {
			config.PreferSimpleProtocol = true
		}
		result := regexp.MustCompile("(time_zone|TimeZone)=(.*?)($|&| )").FindStringSubmatch(d.Opts.DSN)
		if len(result) > 2 {
			config.RuntimeParams["timezone"] = result[2]
		}
		d.DB.ConnPool = stdlib.OpenDB(*config)
	}

	d.DB.Statement.ConnPool = d.DB.ConnPool

	return nil
}

func (d *Dialect) Options() dao.Options {
	return d.Opts
}

func (d *Dialect) NewTx() *dao.DB {
	return d.DB.Session(&dao.Session{})
}

func (d *Dialect) Migrator() dao.Migrator {
	return Migrator{
		Migrator: migrator.Migrator{
			Options: migrator.Options{
				DB:                          d.DB,
				Dialect:                     d,
				CreateIndexAfterCreateTable: true,
			},
		},
		Dialect: d,
	}
}

func (d *Dialect) DataTypeOf(field *schema.Field) string {
	switch field.DataType {
	case schema.Bool:
		return "boolean"
	case schema.Int, schema.Uint:
		size := field.Size
		if field.DataType == schema.Uint {
			size++
		}
		if field.AutoIncrement {
			switch {
			case size <= 16:
				return "smallserial"
			case size <= 32:
				return "serial"
			default:
				return "bigserial"
			}
		} else {
			switch {
			case size <= 16:
				return "smallint"
			case size <= 32:
				return "integer"
			default:
				return "bigint"
			}
		}
	case schema.Float:
		if field.Precision > 0 {
			if field.Scale > 0 {
				return fmt.Sprintf("numeric(%d, %d)", field.Precision, field.Scale)
			}
			return fmt.Sprintf("numeric(%d, %d)", field.Precision, field.Scale)
		}

		return "decimal"
	case schema.String:
		if field.Size > 0 {
			return fmt.Sprintf("varchar(%d)", field.Size)
		}
		return "text"
	case schema.Time:
		if field.Precision > 0 {
			return fmt.Sprintf("timestamptz(%d)", field.Precision)
		}
		return "timestampz"
	case schema.Bytes:
		return "bytea"
	}

	return string(field.DataType)
}

func (d *Dialect) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (d *Dialect) BindVarTo(writer clause.Writer, stmt *dao.Statement, v interface{}) {
	writer.WriteByte('$')
	writer.WriteString(strconv.Itoa(len(stmt.Vars)))
}

func (d *Dialect) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('"')
	if strings.Contains(str, ".") {
		for idx, str := range strings.Split(str, ".") {
			if idx > 0 {
				writer.WriteString(`."`)
			}
			writer.WriteString(str)
			writer.WriteByte('"')
		}
	} else {
		writer.WriteString(str)
		writer.WriteByte('"')
	}
}

var numericPlaceholder = regexp.MustCompile("\\$(\\d+)")

func (d *Dialect) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, numericPlaceholder, `'`, vars...)
}

func (d *Dialect) SavePoint(tx *dao.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return nil
}

func (d *Dialect) RollbackTo(tx *dao.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return nil
}

func (d *Dialect) JSONDataType() string {
	return "JSONB"
}

func (d *Dialect) JSONBuild(column string) dao.JSONQuery {
	return JSONQuery(column)
}

func (d *Dialect) String() string {
	return "postgres"
}

func NewDialect(opts ...dao.Option) dao.Dialect {
	return newPGDialect(opts...)
}
