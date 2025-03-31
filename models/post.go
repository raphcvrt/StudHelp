package models

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"
	"time"
)

type PostStatus string

const (
	PostStatusActive   PostStatus = "active"
	PostStatusDeleted  PostStatus = "deleted"
	PostStatusHidden   PostStatus = "hidden"
	PostStatusReported PostStatus = "reported"
	StatusPending      PostStatus = "pending"
	StatusApproved     PostStatus = "approved"
	StatusRejected     PostStatus = "rejected"
)

// Post représente un article du forum
type Post struct {
	ID           int64      `json:"id"`
	UserID       int64      `json:"user_id"`
	Title        string     `json:"title"`
	Content      string     `json:"content"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LikeCount    int        `json:"like_count"`
	DislikeCount int        `json:"dislike_count"`
	Status       PostStatus `json:"status"`
	Tags         []*Tag     `json:"tags"` // Utilise Tag au lieu de Category
	ImageURL     string     `json:"image_url"`
	ImageType    string     `json:"image_type"`
}

// PostFilter contient les critères de filtrage pour les posts
type PostFilter struct {
	Tag        int64            `json:"tag"`
	Search     string           `json:"search"`
	SortBy     string           `json:"sort_by"`
	SortOrder  string           `json:"sort_order"`
	Status     PostStatus       `json:"status"`
	UserID     int64            `json:"user_id"`
	DateFrom   time.Time        `json:"date_from"`
	DateTo     time.Time        `json:"date_to"`
	Pagination PaginationParams `json:"pagination"`
}

type PaginationParams struct {
	Page    int `json:"page"`
	PerPage int `json:"per_page"`
}

// PostStore gère les opérations CRUD pour les posts
type PostStore struct {
	DB *sql.DB
}

// NewPostStore crée une nouvelle instance de PostStore
func NewPostStore(db *sql.DB) *PostStore {
	return &PostStore{DB: db}
}

// Create ajoute un nouveau post
func (s *PostStore) Create(post *Post) error {
	query := `
		INSERT INTO posts (user_id, title, content, created_at, status, image_url, image_type) 
		VALUES (?, ?, ?, ?, ?, ?, ?)
		RETURNING id
	`

	err := s.DB.QueryRow(
		query,
		post.UserID,
		post.Title,
		post.Content,
		time.Now(),
		post.Status,
		post.ImageURL,
		post.ImageType,
	).Scan(&post.ID)

	return err
}

// GetByID récupère un post par son ID
func (s *PostStore) GetByID(id int64) (*Post, error) {
	query := `SELECT id, user_id, title, content, created_at, updated_at, like_count, dislike_count, status, image_url, image_type FROM posts WHERE id = ?`

	var post Post
	err := s.DB.QueryRow(query, id).Scan(
		&post.ID,
		&post.UserID,
		&post.Title,
		&post.Content,
		&post.CreatedAt,
		&post.UpdatedAt,
		&post.LikeCount,
		&post.DislikeCount,
		&post.Status,
		&post.ImageURL,
		&post.ImageType,
	)

	if err != nil {
		return nil, err
	}

	return &post, nil
}

// GetAllPosts récupère tous les posts avec pagination
func (s *PostStore) GetAllPosts(page, perPage int) ([]*Post, error) {
	query := `SELECT id, user_id, title, content, created_at, updated_at, like_count, dislike_count, status, image_url, image_type FROM posts LIMIT ? OFFSET ?`

	rows, err := s.DB.Query(query, perPage, (page-1)*perPage)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.LikeCount,
			&post.DislikeCount,
			&post.Status,
			&post.ImageURL,
			&post.ImageType,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

// GetPostsByUserID récupère tous les posts d'un utilisateur
func (s *PostStore) GetPostsByUserID(userID int64) ([]*Post, error) {
	query := `SELECT id, user_id, title, content, created_at, updated_at, like_count, dislike_count, status, image_url, image_type FROM posts WHERE user_id = ?`

	rows, err := s.DB.Query(query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.LikeCount,
			&post.DislikeCount,
			&post.Status,
			&post.ImageURL,
			&post.ImageType,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

// GetPostsByTag récupère tous les posts associés à un tag spécifique
func (s *PostStore) GetPostsByTag(tagID int64) ([]*Post, error) {
	query := `
		SELECT p.id, p.user_id, p.title, p.content, p.created_at, p.updated_at, p.like_count, p.dislike_count, p.status, p.image_url, p.image_type 
		FROM posts p
		JOIN post_tags pt ON p.id = pt.post_id
		WHERE pt.tag_id = ?
	`

	rows, err := s.DB.Query(query, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.LikeCount,
			&post.DislikeCount,
			&post.Status,
			&post.ImageURL,
			&post.ImageType,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

// FilterByTag est un alias de GetPostsByTag pour maintenir la compatibilité
func (s *PostStore) FilterByTag(tagID int64) ([]*Post, error) {
	return s.GetPostsByTag(tagID)
}

// Update met à jour un post existant
func (s *PostStore) Update(post *Post) error {
	query := `
		UPDATE posts 
		SET title = ?, content = ?, updated_at = ?, status = ?, image_url = ?, image_type = ?
		WHERE id = ?
	`

	_, err := s.DB.Exec(
		query,
		post.Title,
		post.Content,
		time.Now(),
		post.Status,
		post.ImageURL,
		post.ImageType,
		post.ID,
	)

	return err
}

func (s *PostStore) Delete(id int64) error {
	_, err := s.DB.Exec("DELETE FROM posts WHERE id = ?", id)
	return err
}

// FilterPosts applique des filtres dynamiques aux posts
func (s *PostStore) FilterPosts(filter PostFilter) ([]*Post, error) {
	log.Println("Filtrage des posts avec les critères fournis")

	query := `
        SELECT p.id, p.user_id, p.title, p.content, p.created_at, p.updated_at, p.like_count, p.dislike_count, p.status, p.image_url, p.image_type 
        FROM posts p
        WHERE 1=1
    `
	var params []interface{}

	// Appliquer les filtres
	if filter.UserID != 0 {
		query += " AND p.user_id = ?"
		params = append(params, filter.UserID)
	}

	if filter.Tag != 0 {
		query += " AND p.id IN (SELECT post_id FROM post_tags WHERE tag_id = ?)"
		params = append(params, filter.Tag)
	}

	if filter.Search != "" {
		query += " AND (p.title LIKE ? OR p.content LIKE ?)"
		searchTerm := "%" + filter.Search + "%"
		params = append(params, searchTerm, searchTerm)
	}

	if filter.Status != "" {
		query += " AND p.status = ?"
		params = append(params, filter.Status)
	}

	if !filter.DateFrom.IsZero() {
		query += " AND p.created_at >= ?"
		params = append(params, filter.DateFrom)
	}

	if !filter.DateTo.IsZero() {
		query += " AND p.created_at <= ?"
		params = append(params, filter.DateTo)
	}

	// Tri et pagination
	switch filter.SortBy {
	case "date":
		query += " ORDER BY p.created_at"
	case "likes":
		query += " ORDER BY p.like_count"
	case "title":
		query += " ORDER BY p.title"
	default:
		query += " ORDER BY p.created_at"
	}

	if filter.SortOrder == "asc" {
		query += " ASC"
	} else {
		query += " DESC"
	}

	if filter.Pagination.PerPage > 0 {
		query += " LIMIT ? OFFSET ?"
		offset := (filter.Pagination.Page - 1) * filter.Pagination.PerPage
		params = append(params, filter.Pagination.PerPage, offset)
	}

	// Journalisation pour le débogage
	log.Printf("Requête SQL: %s", query)
	log.Printf("Paramètres: %+v", params)

	rows, err := s.DB.Query(query, params...)
	if err != nil {
		log.Printf("Erreur SQL: %v", err)
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		var post Post
		err := rows.Scan(
			&post.ID,
			&post.UserID,
			&post.Title,
			&post.Content,
			&post.CreatedAt,
			&post.UpdatedAt,
			&post.LikeCount,
			&post.DislikeCount,
			&post.Status,
			&post.ImageURL,
			&post.ImageType,
		)
		if err != nil {
			log.Printf("Erreur de scan: %v", err)
			return nil, err
		}
		posts = append(posts, &post)
	}

	if err = rows.Err(); err != nil {
		log.Printf("Erreur de rows: %v", err)
		return nil, err
	}

	log.Printf("Nombre de posts trouvés: %d", len(posts))
	return posts, nil
}

// CountPosts compte le nombre total de posts selon les filtres
func (s *PostStore) CountPosts(filter PostFilter) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM posts p
		WHERE 1=1
	`
	var params []interface{}

	// Appliquer les filtres
	if filter.UserID != 0 {
		query += " AND p.user_id = ?"
		params = append(params, filter.UserID)
	}

	if filter.Tag != 0 {
		query += " AND p.id IN (SELECT post_id FROM post_tags WHERE tag_id = ?)"
		params = append(params, filter.Tag)
	}

	if filter.Search != "" {
		query += " AND (p.title LIKE ? OR p.content LIKE ?)"
		searchTerm := "%" + filter.Search + "%"
		params = append(params, searchTerm, searchTerm)
	}

	if !filter.DateFrom.IsZero() {
		query += " AND p.created_at >= ?"
		params = append(params, filter.DateFrom)
	}

	if !filter.DateTo.IsZero() {
		query += " AND p.created_at <= ?"
		params = append(params, filter.DateTo)
	}

	var count int
	err := s.DB.QueryRow(query, params...).Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// AddTag ajoute un tag à un post
func (s *PostStore) AddTag(postID, tagID int64) error {
	query := `
		INSERT INTO post_tags (post_id, tag_id) 
		VALUES (?, ?)
	`

	_, err := s.DB.Exec(query, postID, tagID)
	return err
}

// RemoveTag supprime un tag d'un post
func (s *PostStore) RemoveTag(postID, tagID int64) error {
	query := `
		DELETE FROM post_tags 
		WHERE post_id = ? AND tag_id = ?
	`

	_, err := s.DB.Exec(query, postID, tagID)
	return err
}

// RemoveAllTags supprime tous les tags d'un post
func (s *PostStore) RemoveAllTags(postID int64) error {
	query := `DELETE FROM post_tags WHERE post_id = ?`
	_, err := s.DB.Exec(query, postID)
	return err
}

// Like ajoute un like à un post
func (s *PostStore) Like(postID, userID int64) error {
	query := `
		UPDATE posts 
		SET like_count = like_count + 1 
		WHERE id = ?
	`

	_, err := s.DB.Exec(query, postID)
	return err
}

// Unlike supprime un like d'un post
func (s *PostStore) Unlike(postID, userID int64) error {
	query := `
		UPDATE posts 
		SET like_count = like_count - 1 
		WHERE id = ?
	`

	_, err := s.DB.Exec(query, postID)
	return err
}

// Dislike ajoute un dislike à un post
func (s *PostStore) Dislike(postID, userID int64) error {
	query := `
		UPDATE posts 
		SET dislike_count = dislike_count + 1 
		WHERE id = ?
	`

	_, err := s.DB.Exec(query, postID)
	return err
}

// Undislike supprime un dislike d'un post
func (s *PostStore) Undislike(postID, userID int64) error {
	query := `
		UPDATE posts 
		SET dislike_count = dislike_count - 1 
		WHERE id = ?
	`

	_, err := s.DB.Exec(query, postID)
	return err
}

// GetTagNames récupère les noms des tags d'un post
func (p *Post) GetTagNames(tagStore *TagStore) []string {
	tags, err := tagStore.GetTagsByPostID(p.ID)
	if err != nil {
		return nil
	}

	var names []string
	for _, tag := range tags {
		names = append(names, tag.Name)
	}
	return names
}

// GetFormattedDate retourne la date formatée pour l'affichage
func (p *Post) GetFormattedDate() string {
	return p.CreatedAt.Format("Jan 02, 2006")
}

// LoadTags charge les tags associés au post
func (p *Post) LoadTags(tagStore *TagStore) error {
	tags, err := tagStore.GetTagsByPostID(p.ID)
	if err != nil {
		return err
	}
	p.Tags = tags
	return nil
}

// Fonction utilitaire pour extraire l'ID utilisateur d'un cookie
func getUserIDFromCookie(r *http.Request) int64 {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return 0
	}

	userID, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		return 0
	}

	return userID
}
func (s *PostStore) GetCommentCount(postID int64) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM comments WHERE post_id = ?`
	err := s.DB.QueryRow(query, postID).Scan(&count)
	return count, err
}

// récupère tous les posts associés à un tag spécifique
func (s *PostStore) GetPostsByTagID(tagID int64) ([]*Post, error) {
	query := `
	SELECT p.id, p.title, p.content, p.user_id, p.image_url, p.like_count, p.dislike_count, p.created_at, p.updated_at 
	FROM posts p
	JOIN post_tags pt ON p.id = pt.post_id
	WHERE pt.tag_id = ? AND p.status = 'approved'
	ORDER BY p.created_at DESC
	`

	rows, err := s.DB.Query(query, tagID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var posts []*Post
	for rows.Next() {
		post := &Post{}
		err := rows.Scan(
			&post.ID,
			&post.Title,
			&post.Content,
			&post.UserID,
			&post.ImageURL,
			&post.LikeCount,
			&post.DislikeCount,
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		posts = append(posts, post)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}
