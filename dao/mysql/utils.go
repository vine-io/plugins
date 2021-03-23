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
	"github.com/lack-io/vine/service/dao"
	"github.com/lack-io/vine/service/dao/callbacks"
	"github.com/lack-io/vine/service/dao/clause"
)

func Update(db *dao.DB) {
	if db.Error == nil {
		if db.Statement.Schema != nil && !db.Statement.Unscoped {
			for _, c := range db.Statement.Schema.UpdateClauses {
				db.Statement.AddClause(c)
			}
		}

		if db.Statement.SQL.String() == "" {
			db.Statement.SQL.Grow(180)
			db.Statement.AddClauseIfNotExists(clause.Update{})
			if set := callbacks.ConvertToAssignments(db.Statement); len(set) != 0 {
				db.Statement.AddClause(set)
			} else {
				return
			}
			db.Statement.Build("UPDATE", "SET", "WHERE", "ORDER BY", "LIMIT")
		}

		if _, ok := db.Statement.Clauses["WHERE"]; !db.AllowGlobalUpdate && !ok {
			db.AddError(dao.ErrMissingWhereClause)
			return
		}

		if !db.DryRun && db.Error == nil {
			result, err := db.Statement.ConnPool.ExecContext(db.Statement.Context, db.Statement.SQL.String(), db.Statement.Vars...)

			if err == nil {
				db.RowsAffected, _ = result.RowsAffected()
			} else {
				db.AddError(err)
			}
		}
	}
}