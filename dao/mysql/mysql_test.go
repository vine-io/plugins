package mysql_test

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/vine-io/vine/lib/dao"
	"github.com/vine-io/vine/lib/dao/clause"

	"github.com/vine-io/plugins/dao/mysql"
)

const dns = "root:123456@tcp(192.168.3.111:3306)/vine?charset=utf8&parseTime=True&loc=Local"

func TestNewDialect(t *testing.T) {

	dao.DefaultDialect = mysql.NewDialect(dao.DSN(dns))
	err := dao.DefaultDialect.Init()
	if err != nil {
		t.Fatal(err)
	}

	if err := dao.DefaultDialect.Migrator().AutoMigrate(&UserS{}); err != nil {
		t.Fatal(err)
	}

	u1 := &UserS{
		Others: []*Other{{Name: "u1", Age: 23}},
		Sli:    []string{"aa", "bb"},
		D1: (*UserD1)(&D1{
			Name: "aa",
			D2: struct {
				BB string `json:"bb"`
			}{BB: "bb"},
		}),
	}

	u2 := &UserS{
		Others: []*Other{{Name: "u2", Age: 24}},
		Sli:    []string{"cc", "bb"},
		D1: (*UserD1)(&D1{
			Name: "bb",
			D2: struct {
				BB string `json:"bb"`
			}{BB: "bbc"},
		}),
	}

	if err := dao.DefaultDialect.NewTx().Create(u1).Error; err != nil {
		t.Log(err)
	}

	if err := dao.DefaultDialect.NewTx().Create(u2).Error; err != nil {
		t.Log(err)
	}

	//tx := dao.DefaultDialect.NewTx()
	//u1 := &UserS{}
	//
	//clauses := []clause.Expression{
	//	dao.DefaultDialect.JSONBuild("others").Tx(tx).Contains(dao.JSONLike, "u%", "name"),
	//	//dao.DefaultDialect.JSONBuild("d1").Tx(tx).Op(dao.JSONHasKey, nil, "d2"),
	//}
	//tx.Model(&UserS{}).Clauses(clauses...).First(&u1)
	//
	//t.Log(u1)
}

func TestQuery(t *testing.T) {

	dao.DefaultDialect = mysql.NewDialect(dao.DSN(dns))
	err := dao.DefaultDialect.Init()
	if err != nil {
		t.Fatal(err)
	}

	tx := dao.DefaultDialect.NewTx()
	u1 := &UserS{}

	clauses := []clause.Expression{
		dao.DefaultDialect.JSONBuild("others").Tx(tx).Contains(dao.JSONLike, "u1", "name"),
		dao.DefaultDialect.JSONBuild("d1").Tx(tx).Op(dao.JSONHasKey, nil, "d2"),
	}
	tx.Model(&UserS{}).Clauses(clauses...).First(&u1)

	t.Log(u1)
}

type Other struct {
	Name string `json:"name"`
	Age  int64  `json:"age"`
}

// UserOthers the alias of []*Other
type UserOthers []*Other

// Value return json value, implement driver.Valuer interface
func (m UserOthers) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *UserOthers) Scan(value interface{}) error {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	return json.Unmarshal(bytes, &m)
}

func (m *UserOthers) DaoDataType() string {
	return dao.DefaultDialect.JSONDataType()
}

// UserSli the alias of []string
type UserSli []string

// Value return json value, implement driver.Valuer interface
func (m UserSli) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *UserSli) Scan(value interface{}) error {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	return json.Unmarshal(bytes, &m)
}

func (m *UserSli) DaoDataType() string {
	return dao.DefaultDialect.JSONDataType()
}

type D1 struct {
	Name string `json:"name"`

	D2 struct {
		BB string `json:"bb"`
	} `json:"d2"`
}

// UserD1 the alias of D1
type UserD1 D1

// Value return json value, implement driver.Valuer interface
func (m *UserD1) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	return string(b), err
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (m *UserD1) Scan(value interface{}) error {
	var bytes []byte
	switch v := value.(type) {
	case []byte:
		bytes = v
	case string:
		bytes = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	return json.Unmarshal(bytes, &m)
}

func (m *UserD1) DaoDataType() string {
	return dao.DefaultDialect.JSONDataType()
}

// UserS the Schema for User
type UserS struct {
	Id                int64      `json:"id,omitempty" dao:"column:id;autoIncrement;primaryKey"`
	Others            UserOthers `json:"others,omitempty" dao:"column:others"`
	Sli               UserSli    `json:"sli,omitempty" dao:"column:sli"`
	D1                *UserD1    `json:"d1,omitempty" dao:"column:d1"`
	DeletionTimestamp int64      `json:"deletionTimestamp,omitempty" dao:"column:deletion_timestamp"`
}

func (u *UserS) TableName() string {
	return "users"
}