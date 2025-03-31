package models

import (
	"database/sql"
	"log"
	"time"
)

type LikeStore struct {
	DB *sql.DB
}
type Like struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id,omitempty"`
	CommentID int64     `json:"comment_id,omitempty"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	IsLike    bool      `json:"is_like"`
}

// AddOrUpdateLike ajoute ou met à jour un like/dislike pour un post
func (s *LikeStore) AddOrUpdateLike(postID, userID int64, isLike bool) error {
	// D'abord, vérifier si un like existe déjà et récupérer son état
	var existingLike struct {
		ID     int64
		IsLike bool
	}

	err := s.DB.QueryRow("SELECT id, is_like FROM likes WHERE post_id = ? AND user_id = ?",
		postID, userID).Scan(&existingLike.ID, &existingLike.IsLike)

	// Si erreur = pas de résultat, créer un nouveau like
	if err == sql.ErrNoRows {
		// Insérer un nouveau like
		_, err = s.DB.Exec("INSERT INTO likes (post_id, user_id, is_like, created_at) VALUES (?, ?, ?, ?)",
			postID, userID, isLike, time.Now())
		if err != nil {
			return err
		}

		// Mettre à jour le compteur du post
		if isLike {
			_, err = s.DB.Exec("UPDATE posts SET like_count = like_count + 1 WHERE id = ?", postID)
		} else {
			_, err = s.DB.Exec("UPDATE posts SET dislike_count = dislike_count + 1 WHERE id = ?", postID)
		}
		return err
	} else if err != nil {
		// Une autre erreur s'est produite
		return err
	}

	// Si l'état n'a pas changé, ne rien faire
	if existingLike.IsLike == isLike {
		return nil
	}

	// L'état a changé, mettre à jour la réaction et les compteurs
	_, err = s.DB.Exec("UPDATE likes SET is_like = ?, created_at = ? WHERE id = ?",
		isLike, time.Now(), existingLike.ID)
	if err != nil {
		return err
	}

	// Mettre à jour les compteurs
	if isLike {
		// Changé de dislike à like
		_, err = s.DB.Exec("UPDATE posts SET like_count = like_count + 1, dislike_count = dislike_count - 1 WHERE id = ?", postID)
	} else {
		// Changé de like à dislike
		_, err = s.DB.Exec("UPDATE posts SET like_count = like_count - 1, dislike_count = dislike_count + 1 WHERE id = ?", postID)
	}

	return err
}

// RemoveLike supprime un like/dislike
func (s *LikeStore) RemoveLike(postID, userID int64) error {
	// D'abord, vérifier si c'était un like ou un dislike
	var isLike bool
	err := s.DB.QueryRow("SELECT is_like FROM likes WHERE post_id = ? AND user_id = ?",
		postID, userID).Scan(&isLike)
	if err != nil {
		return err
	}

	// Supprimer la réaction
	_, err = s.DB.Exec("DELETE FROM likes WHERE post_id = ? AND user_id = ?", postID, userID)
	if err != nil {
		return err
	}

	// Mettre à jour le compteur
	if isLike {
		_, err = s.DB.Exec("UPDATE posts SET like_count = like_count - 1 WHERE id = ?", postID)
	} else {
		_, err = s.DB.Exec("UPDATE posts SET dislike_count = dislike_count - 1 WHERE id = ?", postID)
	}
	return err
}

// GetUserLike récupère la réaction d'un utilisateur
func (s *LikeStore) GetUserLike(postID, userID int64) (bool, bool, error) {
	var isLike bool
	err := s.DB.QueryRow("SELECT is_like FROM likes WHERE post_id = ? AND user_id = ?",
		postID, userID).Scan(&isLike)
	if err == sql.ErrNoRows {
		// Aucune réaction trouvée
		return false, false, nil
	}
	if err != nil {
		return false, false, err
	}
	// Réaction trouvée
	return true, isLike, nil
}
func NewLikeStore(db *sql.DB) *LikeStore {
	return &LikeStore{DB: db}
}

// GetByPostAndUser récupère un like pour un post et un utilisateur spécifiques
func (s *LikeStore) GetByPostAndUser(postID, userID int64) (*Like, error) {
	query := `SELECT id, post_id, user_id, created_at, is_like FROM likes WHERE post_id = ? AND user_id = ?`

	var like Like
	err := s.DB.QueryRow(query, postID, userID).Scan(
		&like.ID,
		&like.PostID,
		&like.UserID,
		&like.CreatedAt,
		&like.IsLike,
	)

	if err != nil {
		return nil, err
	}

	return &like, nil
}

// GetByCommentAndUser récupère un like pour un commentaire et un utilisateur spécifiques
func (s *LikeStore) GetByCommentAndUser(commentID, userID int64) (*Like, error) {
	query := `SELECT id, comment_id, user_id, created_at, is_like FROM likes WHERE comment_id = ? AND user_id = ?`

	var like Like
	err := s.DB.QueryRow(query, commentID, userID).Scan(
		&like.ID,
		&like.CommentID,
		&like.UserID,
		&like.CreatedAt,
		&like.IsLike,
	)

	if err != nil {
		return nil, err
	}

	return &like, nil
}

// CreateCommentLike crée un nouveau like pour un commentaire
func (s *LikeStore) CreateCommentLike(like *Like) error {
	query := `INSERT INTO likes (comment_id, user_id, is_like, created_at) VALUES (?, ?, ?, ?)`

	res, err := s.DB.Exec(query, like.CommentID, like.UserID, like.IsLike, time.Now())
	if err != nil {
		log.Printf("Erreur lors de la création du like de commentaire: %v", err)
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	like.ID = id
	return nil
}

// UpdateCommentLike met à jour un like existant pour un commentaire
func (s *LikeStore) UpdateCommentLike(like *Like) error {
	query := `UPDATE likes SET is_like = ?, created_at = ? WHERE comment_id = ? AND user_id = ?`

	_, err := s.DB.Exec(query, like.IsLike, time.Now(), like.CommentID, like.UserID)
	if err != nil {
		log.Printf("Erreur lors de la mise à jour du like de commentaire: %v", err)
		return err
	}

	return nil
}

// DeleteCommentLike supprime un like pour un commentaire
func (s *LikeStore) DeleteCommentLike(commentID, userID int64) error {
	query := `DELETE FROM likes WHERE comment_id = ? AND user_id = ?`

	_, err := s.DB.Exec(query, commentID, userID)
	if err != nil {
		log.Printf("Erreur lors de la suppression du like de commentaire: %v", err)
		return err
	}

	return nil
}

// GetLikedPostIDs récupère les IDs de tous les posts qu'un utilisateur a aimés
func (s *LikeStore) GetLikedPostIDs(userID int64) ([]int64, error) {
	query := `
		SELECT post_id FROM likes 
		WHERE user_id = ? AND post_id IS NOT NULL AND is_like = 1
	`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var postIDs []int64
	for rows.Next() {
		var postID int64
		if err := rows.Scan(&postID); err != nil {
			return nil, err
		}
		postIDs = append(postIDs, postID)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return postIDs, nil
}
