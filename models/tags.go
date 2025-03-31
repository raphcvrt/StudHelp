package models

import (
	"database/sql"
	"time"
)

// Tag représente un tag thématique pour les posts
type Tag struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// TagStore gère les opérations CRUD pour les tags
type TagStore struct {
	DB *sql.DB
}

// NewTagStore crée une nouvelle instance de TagStore
func NewTagStore(db *sql.DB) *TagStore {
	return &TagStore{DB: db}
}

// Create ajoute un nouveau tag dans la base de données
func (s *TagStore) Create(tag *Tag) error {
	query := `
		INSERT INTO tags (name, description, created_at) 
		VALUES (?, ?, ?)
		RETURNING id
	`
	err := s.DB.QueryRow(
		query,
		tag.Name,
		tag.Description,
		time.Now(),
	).Scan(&tag.ID)

	return err
}

// GetByID récupère un tag par son ID
func (s *TagStore) GetByID(id int64) (*Tag, error) {
	query := `SELECT id, name, description, created_at FROM tags WHERE id = ?`

	var tag Tag
	err := s.DB.QueryRow(query, id).Scan(
		&tag.ID,
		&tag.Name,
		&tag.Description,
		&tag.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// GetByName récupère un tag par son nom
func (s *TagStore) GetByName(name string) (*Tag, error) {
	query := `SELECT id, name, description, created_at FROM tags WHERE name = ?`

	var tag Tag
	err := s.DB.QueryRow(query, name).Scan(
		&tag.ID,
		&tag.Name,
		&tag.Description,
		&tag.CreatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &tag, nil
}

// GetAllTags récupère tous les tags
func (s *TagStore) GetAllTags() ([]*Tag, error) {
	query := `SELECT id, name, description, created_at FROM tags ORDER BY name`

	rows, err := s.DB.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		var tag Tag
		err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Description,
			&tag.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

// Update met à jour un tag existant
func (s *TagStore) Update(tag *Tag) error {
	query := `
		UPDATE tags
		SET name = ?, description = ?
		WHERE id = ?
	`
	_, err := s.DB.Exec(
		query,
		tag.Name,
		tag.Description,
		tag.ID,
	)

	return err
}

// Delete supprime un tag
func (s *TagStore) Delete(id int64) error {
	query := `DELETE FROM tags WHERE id = ?`
	_, err := s.DB.Exec(query, id)
	return err
}

// GetTagsByPostID récupère tous les tags associés à un post
func (s *TagStore) GetTagsByPostID(postID int64) ([]*Tag, error) {
	query := `
		SELECT t.id, t.name, t.description, t.created_at
		FROM tags t
		JOIN post_tags pt ON t.id = pt.tag_id
		WHERE pt.post_id = ?
		ORDER BY t.name
	`

	rows, err := s.DB.Query(query, postID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		var tag Tag
		err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Description,
			&tag.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

// AddTagToPost associe un tag à un post
func (s *TagStore) AddTagToPost(postID, tagID int64) error {
	query := `INSERT INTO post_tags (post_id, tag_id) VALUES (?, ?)`
	_, err := s.DB.Exec(query, postID, tagID)
	return err
}

// RemoveTagFromPost retire l'association entre un tag et un post
func (s *TagStore) RemoveTagFromPost(postID, tagID int64) error {
	query := `DELETE FROM post_tags WHERE post_id = ? AND tag_id = ?`
	_, err := s.DB.Exec(query, postID, tagID)
	return err
}

// RemoveAllTagsFromPost retire toutes les associations de tags pour un post
func (s *TagStore) RemoveAllTagsFromPost(postID int64) error {
	query := `DELETE FROM post_tags WHERE post_id = ?`
	_, err := s.DB.Exec(query, postID)
	return err
}

// GetPopularTags récupère les tags les plus utilisés
func (s *TagStore) GetPopularTags(limit int) ([]*Tag, error) {
	query := `
		SELECT t.id, t.name, t.description, t.created_at, COUNT(pt.post_id) as usage_count
		FROM tags t
		JOIN post_tags pt ON t.id = pt.tag_id
		GROUP BY t.id
		ORDER BY usage_count DESC
		LIMIT ?
	`

	rows, err := s.DB.Query(query, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tags []*Tag
	for rows.Next() {
		var tag Tag
		var usageCount int
		err := rows.Scan(
			&tag.ID,
			&tag.Name,
			&tag.Description,
			&tag.CreatedAt,
			&usageCount,
		)
		if err != nil {
			return nil, err
		}
		tags = append(tags, &tag)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return tags, nil
}

// CreateOrGet récupère un tag existant par son nom ou en crée un nouveau
func (s *TagStore) CreateOrGet(name string, description string) (*Tag, error) {
	// D'abord, essayer de récupérer par nom
	tag, err := s.GetByName(name)
	if err == nil {
		// Le tag existe déjà
		return tag, nil
	}

	// Le tag n'existe pas, on le crée
	newTag := &Tag{
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}

	err = s.Create(newTag)
	if err != nil {
		return nil, err
	}

	return newTag, nil
}
