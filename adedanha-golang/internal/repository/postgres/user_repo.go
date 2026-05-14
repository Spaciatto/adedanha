package postgres

import (
	"database/sql"
	"fmt"
	"strings"

	"adedanha-golang/internal/domain"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(user domain.User) error {
	_, err := r.db.Exec(
		"INSERT INTO users (id, name, email, avatar, created_at) VALUES ($1, $2, $3, '', $4)",
		user.ID, user.Name, user.Email, user.CreatedAt,
	)
	if err != nil {
		return domain.ErrEmailExists
	}
	return nil
}

func (r *UserRepo) GetByID(id string) (domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(
		"SELECT id, name, email, avatar, created_at FROM users WHERE id = $1", id,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Avatar, &u.CreatedAt)
	if err != nil {
		return u, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *UserRepo) GetByEmail(email string) (domain.User, error) {
	var u domain.User
	err := r.db.QueryRow(
		"SELECT id, name, email, avatar, created_at FROM users WHERE email = $1", email,
	).Scan(&u.ID, &u.Name, &u.Email, &u.Avatar, &u.CreatedAt)
	if err != nil {
		return u, domain.ErrUserNotFound
	}
	return u, nil
}

func (r *UserRepo) Update(id, name, email string) (domain.User, error) {
	result, err := r.db.Exec("UPDATE users SET name = $1, email = $2 WHERE id = $3", name, email, id)
	if err != nil {
		return domain.User{}, domain.ErrEmailExists
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.User{}, domain.ErrUserNotFound
	}
	return r.GetByID(id)
}

func (r *UserRepo) UpdateAvatar(id, avatar string) error {
	result, err := r.db.Exec("UPDATE users SET avatar = $1 WHERE id = $2", avatar, id)
	if err != nil {
		return err
	}
	rows, _ := result.RowsAffected()
	if rows == 0 {
		return domain.ErrUserNotFound
	}
	return nil
}

func (r *UserRepo) Exists(id string) (bool, error) {
	var exists bool
	err := r.db.QueryRow("SELECT EXISTS(SELECT 1 FROM users WHERE id = $1)", id).Scan(&exists)
	return exists, err
}

func (r *UserRepo) GetOnlineUsersByIDs(ids []string) ([]domain.OnlineUser, error) {
	if len(ids) == 0 {
		return []domain.OnlineUser{}, nil
	}
	placeholders, args := buildInClause(ids)
	rows, err := r.db.Query("SELECT id, name FROM users WHERE id IN ("+placeholders+")", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.OnlineUser
	for rows.Next() {
		var u domain.OnlineUser
		if rows.Scan(&u.ID, &u.Name) == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

func (r *UserRepo) GetAvailablePlayersByIDs(ids []string) ([]domain.OnlineUser, error) {
	if len(ids) == 0 {
		return []domain.OnlineUser{}, nil
	}
	placeholders, args := buildInClause(ids)
	query := `SELECT id, name FROM users WHERE id IN (` + placeholders + `)
		AND id NOT IN (
			SELECT mp.user_id FROM match_players mp
			JOIN matches m ON mp.match_id = m.id
			WHERE mp.active = TRUE AND m.status != 'finished'
		)`
	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []domain.OnlineUser
	for rows.Next() {
		var u domain.OnlineUser
		if rows.Scan(&u.ID, &u.Name) == nil {
			users = append(users, u)
		}
	}
	return users, nil
}

// buildInClause creates PostgreSQL placeholders ($1, $2, ...) for IN clauses
func buildInClause(ids []string) (string, []interface{}) {
	parts := make([]string, len(ids))
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		parts[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	return strings.Join(parts, ","), args
}

func itoa(i int) string {
	if i < 10 {
		return string(rune('0' + i))
	}
	return string(rune('0'+i/10)) + string(rune('0'+i%10))
}
