package models

import (
	"database/sql"
	"time"
)

type ActivityType string

const (
	ActivityCreatePost    ActivityType = "create_post"
	ActivityComment       ActivityType = "comment"
	ActivityLike          ActivityType = "like"
	ActivityDislike       ActivityType = "dislike"
	ActivityUpdateProfile ActivityType = "update_profile"
	ActivityDeletePost    ActivityType = "delete_post"
)

type Activity struct {
	ID          int64        `json:"id"`
	UserID      int64        `json:"user_id"`
	RecipientID int64        `json:"recipient_id"`
	Type        ActivityType `json:"type"`
	TargetID    int64        `json:"target_id"`
	CreatedAt   time.Time    `json:"created_at"`
	Content     string       `json:"content"`
	IsRead      bool         `json:"is_read"`
}

type ActivityStore struct {
	DB *sql.DB
}

// crée une nouvelle instance de ActivityStore
func NewActivityStore(db *sql.DB) *ActivityStore {
	return &ActivityStore{DB: db}
}

// récupère toutes les activités d'un utilisateur
func (s *ActivityStore) GetByUserID(userID int64) ([]*Activity, error) {
	query := `
		SELECT id, user_id, recipient_id, type, target_id, created_at, content, is_read
		FROM activities
		WHERE user_id = ?
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*Activity
	for rows.Next() {
		var activity Activity
		err := rows.Scan(
			&activity.ID,
			&activity.UserID,
			&activity.RecipientID,
			&activity.Type,
			&activity.TargetID,
			&activity.CreatedAt,
			&activity.Content,
			&activity.IsRead,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, &activity)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return activities, nil
}

// récupère toutes les notifications destinées à un utilisateur
func (s *ActivityStore) GetNotificationsForUser(userID int64) ([]*Activity, error) {
	query := `
		SELECT id, user_id, recipient_id, type, target_id, created_at, content, is_read
		FROM activities
		WHERE recipient_id = ? AND user_id != ? /* Exclut les activités de l'utilisateur lui-même */
		ORDER BY created_at DESC
	`

	rows, err := s.DB.Query(query, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*Activity
	for rows.Next() {
		var activity Activity
		err := rows.Scan(
			&activity.ID,
			&activity.UserID,
			&activity.RecipientID,
			&activity.Type,
			&activity.TargetID,
			&activity.CreatedAt,
			&activity.Content,
			&activity.IsRead,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, &activity)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return activities, nil
}

// récupère le nombre de notifications non lues
func (s *ActivityStore) GetUnreadNotificationsCount(userID int64) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM activities
		WHERE recipient_id = ? AND is_read = 0 AND user_id != ?
	`

	var count int
	err := s.DB.QueryRow(query, userID, userID).Scan(&count)
	return count, err
}

// récupère une activité par son ID
func (s *ActivityStore) GetByID(id int64) (*Activity, error) {
	query := `
		SELECT id, user_id, recipient_id, type, target_id, created_at, content, is_read
		FROM activities
		WHERE id = ?
	`

	var activity Activity
	err := s.DB.QueryRow(query, id).Scan(
		&activity.ID,
		&activity.UserID,
		&activity.RecipientID,
		&activity.Type,
		&activity.TargetID,
		&activity.CreatedAt,
		&activity.Content,
		&activity.IsRead,
	)

	if err != nil {
		return nil, err
	}

	return &activity, nil
}

// marque toutes les notifications d'un utilisateur comme lues
func (s *ActivityStore) MarkNotificationsAsRead(userID int64) error {
	query := `
		UPDATE activities
		SET is_read = 1
		WHERE recipient_id = ? AND is_read = 0
	`

	_, err := s.DB.Exec(query, userID)
	return err
}

// supprime une activité/notification
func (s *ActivityStore) Delete(id int64) error {
	query := `
		DELETE FROM activities
		WHERE id = ?
	`

	_, err := s.DB.Exec(query, id)
	return err
}

// ajoute une nouvelle activité
func (s *ActivityStore) Create(activity *Activity) error {
	query := `
		INSERT INTO activities (user_id, recipient_id, type, target_id, created_at, content, is_read)
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	err := s.DB.QueryRow(
		query,
		activity.UserID,
		activity.RecipientID,
		activity.Type,
		activity.TargetID,
		time.Now(),
		activity.Content,
		false, // Les nouvelles notifications sont non lues par défaut
	).Scan(&activity.ID)

	return err
}

// récupère les activités récentes globales
func (s *ActivityStore) GetRecentActivity(limit int) ([]*Activity, error) {
	query := `
		SELECT id, user_id, recipient_id, type, target_id, created_at, content, is_read
		FROM activities
		ORDER BY created_at DESC
		LIMIT ?
	`

	rows, err := s.DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var activities []*Activity
	for rows.Next() {
		var activity Activity
		err := rows.Scan(
			&activity.ID,
			&activity.UserID,
			&activity.RecipientID,
			&activity.Type,
			&activity.TargetID,
			&activity.CreatedAt,
			&activity.Content,
			&activity.IsRead,
		)
		if err != nil {
			return nil, err
		}
		activities = append(activities, &activity)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return activities, nil
}

// retourne le message formaté pour l'affichage
func (a *Activity) GetMessage(userStore *UserStore) string {
	user, err := userStore.GetByID(a.UserID)
	if err != nil {
		return "Unknown user " + a.Content
	}
	return user.Username + " " + a.Content
}

// retourne la date formatée
func (a *Activity) GetFormattedDate() string {
	return a.CreatedAt.Format("Jan 02, 2006 at 15:04")
}

// retourne une classe CSS en fonction du type de notification
func (a *Activity) GetNotificationTypeClass() string {
	switch a.Type {
	case ActivityLike:
		return "notification-like"
	case ActivityDislike:
		return "notification-dislike"
	case ActivityComment:
		return "notification-comment"
	default:
		return "notification-default"
	}
}

// GetNotificationIcon retourne l'icône à afficher en fonction du type de notification
func (a *Activity) GetNotificationIcon() string {
	switch a.Type {
	case ActivityLike:
		return "/static/assets/thumbup.svg"
	case ActivityDislike:
		return "/static/assets/thumbdown.svg"
	case ActivityComment:
		return "/static/assets/comment_bubble.svg"
	default:
		return "/static/assets/notifications.svg"
	}
}
