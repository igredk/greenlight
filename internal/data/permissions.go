package data

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Slice to hold the permission codes like "movies:read" and "movies:write" for a single user.
type Permissions []string

// Method to check whether the Permissions slice contains a specific permission code.
func (p Permissions) Include(code string) bool {
	for i := range p {
		if code == p[i] {
			return true
		}
	}
	return false
}

// A PermissionModel struct type which wraps a connection pool.
type PermissionModel struct {
	DB *pgxpool.Pool
}

// Returns all permission codes for a specific user in a Permissions slice.
func (m PermissionModel) GetAllForUser(userID int64) (Permissions, error) {
	query := `
        SELECT permissions.code
        FROM permissions
        INNER JOIN users_permissions ON permissions.id = users_permissions.permission_id
        INNER JOIN users ON users_permissions.user_id = users.id
        WHERE users.id = $1`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	rows, err := m.DB.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}

	permissions, err := pgx.CollectRows(rows, pgx.RowTo[string])

	return Permissions(permissions), nil
}

// Add the provided permission codes for a specific user.
func (m PermissionModel) AddForUser(userID int64, codes ...string) error {
	query := `
        INSERT INTO users_permissions
        SELECT $1, permissions.id FROM permissions WHERE permissions.code = ANY($2)`

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	_, err := m.DB.Exec(ctx, query, userID, codes)
	return err
}
