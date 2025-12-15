package queries

import (
	"context"
)

const delete_userSQL = `
DELETE FROM users WHERE id = $1;`

func (q *Queries) DeleteUser(ctx context.Context, id int64) (int64, error) {
	result, err := q.db.Exec(ctx, delete_userSQL, id)
	if err != nil {
		return 0, err
	}
	return result.RowsAffected(), nil
}
