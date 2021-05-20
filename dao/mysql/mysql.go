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

package mysql

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/lack-io/vine/lib/dao/callbacks"
	"github.com/lack-io/vine/lib/dao"
	"github.com/lack-io/vine/lib/dao/clause"
	"github.com/lack-io/vine/lib/dao/logger"
	"github.com/lack-io/vine/lib/dao/migrator"
	"github.com/lack-io/vine/lib/dao/schema"
)

const (
	// DefaultDriverName is the default driver name for SQLite.
	DefaultDriverName = "mysql"
	// DefaultStringSize is the default string size for mysql
	DefaultStringSize uint = 255
)

type Dialect struct {
	once                      sync.Once
	DB                        *dao.DB
	Opts                      dao.Options
	DriverName                string
	Conn                      dao.ConnPool
	SkipInitializeWithVersion bool
	DefaultStringSize         uint
	DefaultDatetimePrecision  *int
	DisableDatetimePrecision  bool
	DontSupportRenameIndex    bool
	DontSupportRenameColumn   bool
	DontSupportForShareClause bool
}

func newMysqlDialect(opts ...dao.Option) dao.Dialect {
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
	} else {
		dialect.DriverName = DefaultDriverName
	}

	if b, ok := options.Context.Value(skipInitializeWithVersionKey{}).(bool); ok {
		dialect.SkipInitializeWithVersion = b
	}

	if size, ok := options.Context.Value(stringSizeKey{}).(uint); ok {
		dialect.DefaultStringSize = size
	} else {
		dialect.DefaultStringSize = DefaultStringSize
	}

	if p, ok := options.Context.Value(datetimePrecisionKey{}).(int); ok {
		dialect.DefaultDatetimePrecision = &p
	}

	if b, ok := options.Context.Value(disableDatetimePrecisionKey{}).(bool); ok {
		dialect.DisableDatetimePrecision = b
	}

	if b, ok := options.Context.Value(dontSupportRenameIndexKey{}).(bool); ok {
		dialect.DontSupportRenameIndex = b
	}

	if b, ok := options.Context.Value(dontSupportRenameColumnKey{}).(bool); ok {
		dialect.DontSupportRenameColumn = b
	}

	if b, ok := options.Context.Value(dontSupportForShareClauseKey{}).(bool); ok {
		dialect.DontSupportForShareClause = b
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

	d.once.Do(func() {
		callbacks.RegisterDefaultCallbacks(d.DB, &callbacks.Options{})
		_ = d.DB.Callback().Update().Replace("dao:update", Update)
	})

	if d.Conn != nil {
		d.DB.ConnPool = d.Conn
	} else {
		d.DB.ConnPool, err = sql.Open(d.DriverName, d.Opts.DSN)
		if err != nil {
			return err
		}
	}

	d.DB.Statement.ConnPool = d.DB.ConnPool

	if !d.SkipInitializeWithVersion {
		var version string
		err = d.DB.ConnPool.QueryRowContext(d.Opts.Context, "SELECT VERSION()").Scan(&version)
		if err != nil {
			return err
		}

		if strings.Contains(version, "MariaDB") {
			d.DontSupportRenameIndex = true
			d.DontSupportRenameColumn = true
			d.DontSupportForShareClause = true
		} else if strings.HasPrefix(version, "5.6.") {
			d.DontSupportRenameIndex = true
			d.DontSupportRenameColumn = true
			d.DontSupportForShareClause = true
		} else if strings.HasPrefix(version, "5.7.") {
			d.DontSupportRenameColumn = true
			d.DontSupportForShareClause = true
		} else if strings.HasPrefix(version, "5.") {
			d.DisableDatetimePrecision = true
			d.DontSupportRenameIndex = true
			d.DontSupportRenameColumn = true
			d.DontSupportForShareClause = true
		}
	}

	for k, v := range d.ClauseBuilders() {
		d.DB.ClauseBuilders[k] = v
	}
	return nil
}

func (d *Dialect) Options() dao.Options {
	return d.Opts
}

func (d *Dialect) Apply(options *dao.Options) error {
	if options.NowFunc == nil {
		if d.DefaultDatetimePrecision == nil {
			var defaultDatetimePrecision = 3
			d.DefaultDatetimePrecision = &defaultDatetimePrecision
		}

		round := time.Second / time.Duration(math.Pow10(*d.DefaultDatetimePrecision))
		options.NowFunc = func() time.Time { return time.Now().Local().Round(round) }
	}
	return nil
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
		sqlType := "bigint"
		switch {
		case field.Size <= 8:
			sqlType = "tinyint"
		case field.Size <= 16:
			sqlType = "smallint"
		case field.Size <= 24:
			sqlType = "mediumint"
		case field.Size <= 32:
			sqlType = "int"
		}

		if field.DataType == schema.Uint {
			sqlType += " unsigned"
		}

		if field.AutoIncrement {
			sqlType += " AUTO_INCREMENT"
		}
		return sqlType
	case schema.Float:
		if field.Precision > 0 {
			return fmt.Sprintf("decimal(%d, %d)", field.Precision, field.Scale)
		}

		if field.Size <= 32 {
			return "float"
		}
		return "double"
	case schema.String:
		size := field.Size
		defaultSize := d.DefaultStringSize

		if size == 0 {
			if defaultSize > 0 {
				size = int(defaultSize)
			} else {
				hasIndex := field.TagSettings["INDEX"] != "" || field.TagSettings["UNIQUE"] != ""
				// TEXT, GEOMETRY or JSON column can't have a default value
				if field.PrimaryKey || field.HasDefaultValue || hasIndex {
					size = 191 // utf8mb4
				}
			}
		}

		if size >= 65536 && size <= int(math.Pow(2, 24)) {
			return "mediumtext"
		} else if size > int(math.Pow(2, 24)) || size <= 0 {
			return "longtext"
		}
		return fmt.Sprintf("varchar(%d)", size)
	case schema.Time:
		precision := ""

		if !d.DisableDatetimePrecision && field.Precision == 0 {
			field.Precision = *d.DefaultDatetimePrecision
		}

		if field.Precision > 0 {
			precision = fmt.Sprintf("(%d)", field.Precision)
		}

		if field.NotNull || field.PrimaryKey {
			return "datetime" + precision
		}
		return "datetime" + precision + " NULL"
	case schema.Bytes:
		if field.Size > 0 && field.Size < 65536 {
			return fmt.Sprintf("varbinary(%d)", field.Size)
		}

		if field.Size >= 65536 && field.Size <= int(math.Pow(2, 24)) {
			return "mediumblob"
		}

		return "longblob"
	}

	return string(field.DataType)
}

func (d *Dialect) DefaultValueOf(field *schema.Field) clause.Expression {
	return clause.Expr{SQL: "DEFAULT"}
}

func (d *Dialect) BindVarTo(writer clause.Writer, stmt *dao.Statement, v interface{}) {
	writer.WriteByte('?')
}

func (d *Dialect) QuoteTo(writer clause.Writer, str string) {
	writer.WriteByte('`')
	if strings.Contains(str, ".") {
		for idx, str := range strings.Split(str, ".") {
			if idx > 0 {
				writer.WriteString(".`")
			}
			writer.WriteString(str)
			writer.WriteByte('`')
		}
	} else {
		writer.WriteString(str)
		writer.WriteByte('`')
	}
}

func (d *Dialect) Explain(sql string, vars ...interface{}) string {
	return logger.ExplainSQL(sql, nil, `'`, vars...)
}

func (d *Dialect) SavePoint(tx *dao.DB, name string) error {
	tx.Exec("SAVEPOINT " + name)
	return nil
}

func (d *Dialect) RollbackTo(tx *dao.DB, name string) error {
	tx.Exec("ROLLBACK TO SAVEPOINT " + name)
	return nil
}

func (d *Dialect) ClauseBuilders() map[string]clause.ClauseBuilder {
	clauseBuilders := map[string]clause.ClauseBuilder{
		"ON CONFLICT": func(c clause.Clause, builder clause.Builder) {
			if onConflict, ok := c.Expression.(clause.OnConflict); ok {
				builder.WriteString("ON DUPLICATE KEY UPDATE ")
				if len(onConflict.DoUpdates) == 0 {
					if s := builder.(*dao.Statement).Schema; s != nil {
						var column clause.Column
						onConflict.DoNothing = false

						if s.PrioritizedPrimaryField != nil {
							column = clause.Column{Name: s.PrioritizedPrimaryField.DBName}
						} else if len(s.DBNames) > 0 {
							column = clause.Column{Name: s.DBNames[0]}
						}

						if column.Name != "" {
							onConflict.DoUpdates = []clause.Assignment{{Column: column, Value: column}}
						}
					}
				}

				for idx, assignment := range onConflict.DoUpdates {
					if idx > 0 {
						builder.WriteByte(',')
					}

					builder.WriteQuoted(assignment.Column)
					builder.WriteByte('=')
					if column, ok := assignment.Value.(clause.Column); ok && column.Table == "excluded" {
						column.Table = ""
						builder.WriteString("VALUES(")
						builder.WriteQuoted(column)
						builder.WriteByte(')')
					} else {
						builder.AddVar(builder, assignment.Value)
					}
				}
			} else {
				c.Build(builder)
			}
		},
		"VALUES": func(c clause.Clause, builder clause.Builder) {
			if values, ok := c.Expression.(clause.Values); ok && len(values.Columns) == 0 {
				builder.WriteString("VALUES()")
				return
			}
			c.Build(builder)
		},
	}

	if d.DontSupportForShareClause {
		clauseBuilders["FOR"] = func(c clause.Clause, builder clause.Builder) {
			if values, ok := c.Expression.(clause.Locking); ok && strings.EqualFold(values.Strength, "SHARE") {
				builder.WriteString("LOCK IN SHARE MODE")
				return
			}
			c.Build(builder)
		}
	}

	return clauseBuilders
}

func (d *Dialect) JSONDataType() string {
	return "JSON"
}

func (d *Dialect) JSONBuild(column string) dao.JSONQuery {
	return JSONQuery(column)
}

func (d *Dialect) String() string {
	return "mysql"
}

// Example:
//	mysql.NewDialect(dao.DSN("dao:dao@tcp(localhost:9910)/dao?charset=utf8&parseTime=True&loc=Local"))
func NewDialect(opts ...dao.Option) dao.Dialect {
	return newMysqlDialect(opts...)
}
