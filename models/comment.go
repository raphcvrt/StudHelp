package models

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type Comment struct {
	ID           int64
	PostID       int64
	UserID       int64
	Content      string
	CreatedAt    time.Time
	UpdatedAt    time.Time
	LikeCount    int
	DislikeCount int
	Status       PostStatus
}

type CommentStore struct {
	DB *sql.DB
}

func NewCommentStore(db *sql.DB) *CommentStore {
	return &CommentStore{DB: db}
}

func (s *CommentStore) Create(comment *Comment) error {
	now := time.Now()
	comment.CreatedAt = now
	comment.UpdatedAt = now // Assurez-vous que UpdatedAt est également défini

	query := `
        INSERT INTO comments (post_id, user_id, content, created_at, updated_at, status, like_count, dislike_count) 
        VALUES (?, ?, ?, ?, ?, ?, ?, ?)
        RETURNING id
    `

	err := s.DB.QueryRow(
		query,
		comment.PostID,
		comment.UserID,
		comment.Content,
		comment.CreatedAt,
		comment.UpdatedAt,
		comment.Status,
		comment.LikeCount,
		comment.DislikeCount,
	).Scan(&comment.ID)

	if err != nil {
		log.Printf("Erreur lors de l'insertion du commentaire: %v", err)
	}

	return err
}

func (s *CommentStore) GetByID(id int64) (*Comment, error) {
	query := `SELECT id, post_id, user_id, content, created_at, updated_at, like_count, dislike_count, status FROM comments WHERE id = ?`

	var comment Comment
	err := s.DB.QueryRow(query, id).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.UserID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
		&comment.LikeCount,
		&comment.DislikeCount,
		&comment.Status,
	)

	if err != nil {
		return nil, err
	}

	return &comment, nil
}

func (s *CommentStore) GetCommentsByPostID(postID int64) ([]*Comment, error) {
	log.Printf("Tentative de récupération des commentaires pour le post ID: %d", postID)

	query := `SELECT id, post_id, user_id, content, created_at, updated_at, like_count, dislike_count, status 
              FROM comments WHERE post_id = ?`

	// Vérifiez d'abord si des commentaires existent
	var count int
	err := s.DB.QueryRow("SELECT COUNT(*) FROM comments WHERE post_id = ?", postID).Scan(&count)
	if err != nil {
		log.Printf("Erreur lors du comptage des commentaires: %v", err)
		return nil, fmt.Errorf("failed to count comments: %w", err)
	}

	log.Printf("Nombre de commentaires trouvés: %d", count)

	if count == 0 {
		return []*Comment{}, nil // Retourner une slice vide si pas de commentaires
	}

	// Récupérer tous les commentaires
	rows, err := s.DB.Query(query, postID)
	if err != nil {
		log.Printf("Erreur lors de la requête des commentaires: %v", err)
		return nil, fmt.Errorf("failed to query comments: %w", err)
	}
	defer rows.Close()

	comments := make([]*Comment, 0, count)
	for rows.Next() {
		var c Comment
		err := rows.Scan(
			&c.ID,
			&c.PostID,
			&c.UserID,
			&c.Content,
			&c.CreatedAt,
			&c.UpdatedAt,
			&c.LikeCount,
			&c.DislikeCount,
			&c.Status,
		)
		if err != nil {
			log.Printf("Erreur lors du scan d'un commentaire: %v", err)
			return nil, fmt.Errorf("failed to scan comment: %w", err)
		}
		log.Printf("Commentaire récupéré: ID=%d, Content=%s", c.ID, c.Content)
		comments = append(comments, &c)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Erreur de rows: %v", err)
		return nil, fmt.Errorf("rows error: %w", err)
	}

	log.Printf("Total de %d commentaires récupérés avec succès", len(comments))
	return comments, nil
}
func (s *CommentStore) Update(comment *Comment) error {
	query := `
		UPDATE comments 
		SET content = ?, updated_at = ?, status = ?
		WHERE id = ?
	`

	_, err := s.DB.Exec(
		query,
		comment.Content,
		time.Now(),
		comment.Status,
		comment.ID,
	)

	return err
}

func (s *CommentStore) Delete(id int64) error {
	query := `DELETE FROM comments WHERE id = ?`

	_, err := s.DB.Exec(query, id)
	return err
}

// GetAuthorName retourne le nom de l'auteur
func (c *Comment) GetAuthorName(userStore *UserStore) string {
	user, err := userStore.GetByID(c.UserID)
	if err != nil {
		return "Unknown"
	}
	return user.Username
}

// GetFormattedDate retourne la date formatée
func (c *Comment) GetFormattedDate() string {
	return c.CreatedAt.Format("Jan 02, 2006 at 15:04")
}

// CanEdit détermine si l'utilisateur peut modifier ce commentaire
func (c *Comment) CanEdit(userID int64, userRole UserRole) bool {
	return userID == c.UserID || userRole >= RoleModerator
}

// UpdateLikeCount met à jour le nombre de likes d'un commentaire
func (s *CommentStore) UpdateLikeCount(commentID int64, delta int) error {
	query := `UPDATE comments SET like_count = like_count + ? WHERE id = ?`
	_, err := s.DB.Exec(query, delta, commentID)
	return err
}

// UpdateDislikeCount met à jour le nombre de dislikes d'un commentaire
func (s *CommentStore) UpdateDislikeCount(commentID int64, delta int) error {
	query := `UPDATE comments SET dislike_count = dislike_count + ? WHERE id = ?`
	_, err := s.DB.Exec(query, delta, commentID)
	return err
}
