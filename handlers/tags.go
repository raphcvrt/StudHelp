package handlers

import (
	"forum/models"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

// TagHandler gère les opérations liées aux tags
type TagHandler struct {
	TagStore     *models.TagStore
	PostStore    *models.PostStore
	UserStore    *models.UserStore
	CommentStore *models.CommentStore
}

// NewTagHandler crée une nouvelle instance de TagHandler
func NewTagHandler(tagStore *models.TagStore, postStore *models.PostStore, userStore *models.UserStore, commentStore *models.CommentStore) *TagHandler {
	return &TagHandler{
		TagStore:     tagStore,
		PostStore:    postStore,
		UserStore:    userStore,
		CommentStore: commentStore,
	}
}

// RegisterTagRoutes enregistre les routes liées aux tags
func RegisterTagRoutes(r *mux.Router, h *TagHandler) {
	r.HandleFunc("/tag/{id:[0-9]+}", h.ShowTagPosts).Methods("GET")
}

// ShowTagPosts affiche tous les posts associés à un tag spécifique
func (h *TagHandler) ShowTagPosts(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'ID du tag depuis l'URL
	vars := mux.Vars(r)
	tagIDStr := vars["id"]

	tagID, err := strconv.ParseInt(tagIDStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de tag invalide", http.StatusBadRequest)
		return
	}

	// Récupérer les informations du tag
	tag, err := h.TagStore.GetByID(tagID)
	if err != nil {
		http.Error(w, "Tag non trouvé", http.StatusNotFound)
		return
	}

	// Récupérer les posts associés à ce tag
	posts, err := h.PostStore.GetPostsByTagID(tagID)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des posts", http.StatusInternalServerError)
		return
	}

	// Récupérer les auteurs des posts
	authors := make(map[int64]*models.User)
	for _, post := range posts {
		if _, exists := authors[post.UserID]; !exists {
			author, err := h.UserStore.GetByID(post.UserID)
			if err == nil {
				authors[post.UserID] = author
			}
		}
	}

	// Récupérer les tags pour chaque post
	postTags := make(map[int64][]*models.Tag)
	for _, post := range posts {
		tags, err := h.TagStore.GetTagsByPostID(post.ID)
		if err == nil {
			postTags[post.ID] = tags
		}
	}

	// Récupérer le nombre de commentaires pour chaque post
	commentCounts := make(map[int64]int)
	for _, post := range posts {
		comments, err := h.CommentStore.GetCommentsByPostID(post.ID)
		if err == nil {
			commentCounts[post.ID] = len(comments)
		} else {
			commentCounts[post.ID] = 0
		}
	}

	// Vérifier si l'utilisateur est authentifié
	userID := getUserIDFromCookie(r)
	isAuthenticated := userID > 0

	// Préparation des données pour le template
	data := map[string]interface{}{
		"Tag":             tag,
		"Posts":           posts,
		"Authors":         authors,
		"PostTags":        postTags,
		"CommentCounts":   commentCounts,
		"IsAuthenticated": isAuthenticated,
	}

	// Rendre le template
	RenderTemplate(w, "tag_view.html", data)
}
