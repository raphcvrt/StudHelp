package models

import (
	"database/sql"
	"strings"
	"time"
)

type UserRole int

const (
	RoleGuest UserRole = iota
	RoleUser
	RoleModerator
	RoleAdmin
)

type User struct {
	ID        int64
	UUID      string
	Username  string
	Email     string
	Password  string
	AvatarURL string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type UserStore struct {
	DB *sql.DB
}

func NewUserStore(db *sql.DB) *UserStore {
	return &UserStore{DB: db}
}

func (s *UserStore) Create(user *User) error {
	// Version compatible SQLite
	res, err := s.DB.Exec(
		`INSERT INTO users (uuid, username, email, password, avatar_url, created_at) 
         VALUES (?, ?, ?, ?, ?, ?)`,
		user.UUID,
		user.Username,
		user.Email,
		user.Password,
		user.AvatarURL,
		time.Now(),
	)
	if err != nil {
		return err
	}

	// Récupérer l'ID généré
	id, err := res.LastInsertId()
	if err != nil {
		return err
	}
	user.ID = id

	return nil
}

func (s *UserStore) GetByID(id int64) (*User, error) {
	query := `SELECT id, uuid, username, email, password, avatar_url, created_at, updated_at 
              FROM users WHERE id = ?`

	var user User
	err := s.DB.QueryRow(query, id).Scan(
		&user.ID,
		&user.UUID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetByEmail(email string) (*User, error) {
	query := `SELECT id, uuid, username, email, password, avatar_url, created_at, updated_at 
              FROM users WHERE email = ?`

	var user User
	err := s.DB.QueryRow(query, email).Scan(
		&user.ID,
		&user.UUID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.AvatarURL,
		&user.CreatedAt,
		&user.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *UserStore) GetByUsername(username string) (*User, error) {
	query := `SELECT id, uuid, username, email, password, role, created_at FROM users WHERE username = ?`

	var user User
	err := s.DB.QueryRow(query, username).Scan(
		&user.ID,
		&user.UUID,
		&user.Username,
		&user.Email,
		&user.Password,
		&user.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *UserStore) UpdateRole(userID int64, role UserRole) error {
	query := `UPDATE users SET role = ? WHERE id = ?`

	_, err := s.DB.Exec(query, role, userID)
	return err
}

func (s *UserStore) GetAllModerators() ([]*User, error) {
	query := `SELECT id, uuid, username, email, role, created_at FROM users WHERE role = ?`

	rows, err := s.DB.Query(query, RoleModerator)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*User
	for rows.Next() {
		var user User
		err := rows.Scan(
			&user.ID,
			&user.UUID,
			&user.Username,
			&user.Email,
			&user.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

// GetFormattedJoinDate retourne la date d'inscription formatée
func (u *User) GetFormattedJoinDate() string {
	return u.CreatedAt.Format("January 2006")
}

// GetInitials retourne les initiales pour les avatars
func (u *User) GetInitials() string {
	if len(u.Username) >= 2 {
		return strings.ToUpper(u.Username[:2])
	}
	return "US"
}
func (u *User) GetAvatarURL() string {
	if u.AvatarURL == "" {
		return "/static/assets/pfp_placeholder.jpg"
	}
	return u.AvatarURL
}

// UpdateAvatar met à jour l'URL de l'avatar
func (s *UserStore) UpdateAvatar(userID int64, avatarURL string) error {
	_, err := s.DB.Exec(
		"UPDATE users SET avatar_url = ?, updated_at = ? WHERE id = ?",
		avatarURL,
		time.Now(),
		userID,
	)
	return err
}

// UpdateProfile met à jour les informations de base
func (s *UserStore) UpdateProfile(userID int64, username, email string) error {
	_, err := s.DB.Exec(
		"UPDATE users SET username = ?, email = ?, updated_at = ? WHERE id = ?",
		username,
		email,
		time.Now(),
		userID,
	)
	return err
}

// GetCommentsByUserID récupère tous les commentaires d'un utilisateur
func (s *CommentStore) GetCommentsByUserID(userID int64) ([]*Comment, error) {
	query := `SELECT id, post_id, user_id, content, created_at, updated_at, like_count, dislike_count 
             FROM comments 
             WHERE user_id = ? 
             ORDER BY created_at DESC`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*Comment
	for rows.Next() {
		var comment Comment
		err := rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.UserID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UpdatedAt,
			&comment.LikeCount,
			&comment.DislikeCount,
		)
		if err != nil {
			return nil, err
		}
		comments = append(comments, &comment)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return comments, nil
}
