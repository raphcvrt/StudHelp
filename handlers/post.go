package handlers

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"forum/models"

	"github.com/gorilla/mux"
)

type PostHandler struct {
	PostStore     *models.PostStore
	TagStore      *models.TagStore
	CommentStore  *models.CommentStore
	UserStore     *models.UserStore
	LikeStore     *models.LikeStore
	ActivityStore *models.ActivityStore
}

func NewPostHandler(postStore *models.PostStore, tagStore *models.TagStore, commentStore *models.CommentStore, userStore *models.UserStore, likeStore *models.LikeStore, activityStore *models.ActivityStore) *PostHandler {
	return &PostHandler{
		PostStore:     postStore,
		TagStore:      tagStore,
		CommentStore:  commentStore,
		UserStore:     userStore,
		LikeStore:     likeStore,
		ActivityStore: activityStore,
	}
}

func RegisterPostRoutes(r *mux.Router, h *PostHandler) {
	r.HandleFunc("/", h.HomePage).Methods("GET")
	r.HandleFunc("/post/{id}", h.ViewPost).Methods("GET")
	r.HandleFunc("/create-post", h.NewPostPage).Methods("GET")
	r.HandleFunc("/create-post", h.CreatePost).Methods("POST")
	r.HandleFunc("/edit-post/{id}", h.EditPostPage).Methods("GET")
	r.HandleFunc("/edit-post/{id}", h.UpdatePost).Methods("POST")
	r.HandleFunc("/delete-post/{id}", h.DeletePost).Methods("POST")
}

// Page d'accueil avec liste des posts
func (h *PostHandler) HomePage(w http.ResponseWriter, r *http.Request) {
	// Récupérer les paramètres de pagination
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1
	}
	perPage := 10 // nombre de posts par page

	// Construire le filtre de recherche
	filter := models.PostFilter{
		SortOrder: "desc", // Par défaut, ordre décroissant
	}
	filter.Pagination.Page = page
	filter.Pagination.PerPage = perPage

	// Récupérer le terme de recherche
	searchQuery := r.URL.Query().Get("search")
	if searchQuery != "" {
		filter.Search = searchQuery
	}

	// Récupérer le tri
	sortBy := r.URL.Query().Get("sort")
	switch sortBy {
	case "date_desc":
		filter.SortBy = "date"
		filter.SortOrder = "desc"
	case "date_asc":
		filter.SortBy = "date"
		filter.SortOrder = "asc"
	case "likes_desc":
		filter.SortBy = "likes"
		filter.SortOrder = "desc"
	case "likes_asc":
		filter.SortBy = "likes"
		filter.SortOrder = "asc"
	case "dislikes_desc":
		filter.SortBy = "dislikes"
		filter.SortOrder = "desc"
	case "dislikes_asc":
		filter.SortBy = "dislikes"
		filter.SortOrder = "asc"
	default:
		// Par défaut, trier par date (plus récent)
		sortBy = "date_desc"
		filter.SortBy = "date"
		filter.SortOrder = "desc"
	}

	// Filtrer par tag si spécifié
	var currentTagName string
	tagID := r.URL.Query().Get("tag")
	if tagID != "" {
		tagIDInt, err := strconv.ParseInt(tagID, 10, 64)
		if err == nil {
			filter.Tag = tagIDInt

			// Récupérer le nom du tag pour l'affichage
			tag, err := h.TagStore.GetByID(tagIDInt)
			if err == nil {
				currentTagName = tag.Name
			}
		}
	}

	// Récupération des posts
	posts, err := h.PostStore.FilterPosts(filter)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des posts", http.StatusInternalServerError)
		return
	}

	// Récupération des auteurs
	authors := make(map[int64]*models.User)
	for _, post := range posts {
		if _, exists := authors[post.UserID]; !exists {
			user, err := h.UserStore.GetByID(post.UserID)
			if err == nil {
				authors[post.UserID] = user
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
		count, err := h.CommentStore.GetCommentsByPostID(post.ID)
		if err == nil {
			commentCounts[post.ID] = len(count)
		} else {
			commentCounts[post.ID] = 0
		}
	}

	// Pagination
	totalPosts, _ := h.PostStore.CountPosts(filter)
	totalPages := (totalPosts + perPage - 1) / perPage

	// Récupérer tous les tags pour le filtre
	allTags, _ := h.TagStore.GetAllTags()
	popularTags, _ := h.TagStore.GetPopularTags(10)

	// Construire l'URL de base pour la pagination
	paginationBaseURL := "/?sort=" + sortBy
	if tagID != "" {
		paginationBaseURL += "&tag=" + tagID
	}
	if searchQuery != "" {
		paginationBaseURL += "&search=" + searchQuery
	}

	// Préparation des données pour le template
	data := map[string]interface{}{
		"Posts":             posts,
		"Authors":           authors,
		"PostTags":          postTags,
		"CommentCounts":     commentCounts,
		"AllTags":           allTags,
		"PopularTags":       popularTags,
		"CurrentPage":       page,
		"TotalPages":        totalPages,
		"CurrentTagID":      filter.Tag,
		"CurrentTagName":    currentTagName,
		"SearchQuery":       searchQuery,
		"SortBy":            sortBy,
		"PaginationBaseURL": paginationBaseURL,
	}

	// Vérification de l'authentification
	userID := getUserIDFromCookie(r)
	if userID > 0 {
		user, err := h.UserStore.GetByID(userID)
		if err == nil {
			data["CurrentUser"] = user
			data["IsAuthenticated"] = true
			data["User"] = user
		}
	} else {
		data["IsAuthenticated"] = false
	}

	// Rendu du template
	RenderTemplate(w, "index.html", data)
}

// Affichage d'un post spécifique
func (h *PostHandler) ViewPost(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'ID du post
	vars := mux.Vars(r)
	postID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "ID de post invalide", http.StatusBadRequest)
		return
	}

	// Récupérer le post et ses données associées
	post, err := h.PostStore.GetByID(postID)
	if err != nil {
		http.Error(w, "Post non trouvé", http.StatusNotFound)
		return
	}

	comments, _ := h.CommentStore.GetCommentsByPostID(postID)
	author, _ := h.UserStore.GetByID(post.UserID)
	tags, _ := h.TagStore.GetTagsByPostID(postID)

	// Récupérer les auteurs des commentaires
	commentAuthors := make(map[int64]*models.User)
	for _, comment := range comments {
		if _, exists := commentAuthors[comment.UserID]; !exists {
			user, err := h.UserStore.GetByID(comment.UserID)
			if err == nil {
				commentAuthors[comment.UserID] = user
			}
		}
	}

	// Récupérer les likes/dislikes
	userID := getUserIDFromCookie(r)
	var userLike *models.Like
	commentLikes := make(map[int64]*models.Like)

	if userID > 0 {
		userLike, _ = h.LikeStore.GetByPostAndUser(postID, userID)

		for _, comment := range comments {
			like, _ := h.LikeStore.GetByCommentAndUser(comment.ID, userID)
			if like != nil {
				commentLikes[comment.ID] = like
			}
		}
	}

	// Préparation des données pour le template
	data := map[string]interface{}{
		"Post":           post,
		"Author":         author,
		"Comments":       comments,
		"Tags":           tags,
		"CommentAuthors": commentAuthors,
		"UserLike":       userLike,
		"CommentLikes":   commentLikes,
	}

	// Vérification de l'authentification
	if userID > 0 {
		if user, err := h.UserStore.GetByID(userID); err == nil {
			data["CurrentUser"] = user
			data["IsAuthenticated"] = true
			data["User"] = user
		}
	} else {
		data["IsAuthenticated"] = false
	}

	RenderTemplate(w, "post_view.html", data)
}

// Page de création de post
func (h *PostHandler) NewPostPage(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, _ := h.UserStore.GetByID(userID)
	tags, _ := h.TagStore.GetAllTags()
	popularTags, _ := h.TagStore.GetPopularTags(20)

	data := map[string]interface{}{
		"Action":      "new",
		"Tags":        tags,
		"PopularTags": popularTags,
		"User":        user,
	}

	RenderTemplate(w, "post_forms.html", data)
}

// Création d'un nouveau post
func (h *PostHandler) CreatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Vérification de l'authentification
	userID := getUserIDFromCookie(r)
	if userID <= 0 {
		http.Error(w, "Non authentifié", http.StatusUnauthorized)
		return
	}

	// Traitement du formulaire
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Erreur lors du traitement du formulaire", http.StatusBadRequest)
		return
	}

	// Récupération des données
	title := r.FormValue("title")
	content := r.FormValue("content")
	tagIDs := r.Form["tags"]
	newTags := r.FormValue("newTags")

	if title == "" || content == "" {
		http.Error(w, "Le titre et le contenu sont obligatoires", http.StatusBadRequest)
		return
	}

	// Création du post
	post := &models.Post{
		UserID:    userID,
		Title:     title,
		Content:   content,
		CreatedAt: time.Now(),
		Status:    models.StatusApproved,
	}

	// Traitement de l'image
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		// Vérification du type de fichier
		contentType := handler.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			http.Error(w, "Type de fichier non supporté", http.StatusBadRequest)
			return
		}

		// Création du dossier si nécessaire
		uploadDir := "./static/uploads"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, 0755)
		}

		// Sauvegarde du fichier
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filepath := filepath.Join(uploadDir, filename)
		dst, err := os.Create(filepath)
		if err != nil {
			http.Error(w, "Erreur lors de la création du fichier", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		io.Copy(dst, file)
		post.ImageURL = "/static/uploads/" + filename
		post.ImageType = contentType
	}

	// Sauvegarde du post
	if err := h.PostStore.Create(post); err != nil {
		http.Error(w, "Erreur lors de la création du post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Association des tags
	for _, tagIDStr := range tagIDs {
		tagID, err := strconv.ParseInt(tagIDStr, 10, 64)
		if err == nil {
			h.PostStore.AddTag(post.ID, tagID)
		}
	}

	// Traitement des nouveaux tags
	if newTags != "" {
		tagNames := strings.Split(newTags, ",")
		for _, name := range tagNames {
			name = strings.TrimSpace(name)
			if name != "" {
				tag, err := h.TagStore.CreateOrGet(name, "")
				if err == nil {
					h.PostStore.AddTag(post.ID, tag.ID)
				}
			}
		}
	}

	// Création de l'activité
	activity := &models.Activity{
		UserID:      userID,
		RecipientID: userID,
		Type:        models.ActivityCreatePost,
		TargetID:    post.ID,
		CreatedAt:   time.Now(),
		Content:     fmt.Sprintf("a créé un nouveau post: %s", post.Title),
		IsRead:      true,
	}
	h.ActivityStore.Create(activity)

	http.Redirect(w, r, fmt.Sprintf("/post/%d", post.ID), http.StatusSeeOther)
}

// Page d'édition de post
func (h *PostHandler) EditPostPage(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	postID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "ID de post invalide", http.StatusBadRequest)
		return
	}

	// Récupération du post
	post, err := h.PostStore.GetByID(postID)
	if err != nil {
		http.Error(w, "Post non trouvé", http.StatusNotFound)
		return
	}

	// Vérification des permissions
	userID := getUserIDFromCookie(r)
	if userID != post.UserID {
		http.Error(w, "Vous n'êtes pas autorisé à modifier ce post", http.StatusForbidden)
		return
	}

	// Récupération des tags
	postTags, _ := h.TagStore.GetTagsByPostID(postID)
	allTags, _ := h.TagStore.GetAllTags()

	// Création du map des tags sélectionnés
	selectedTagIDs := make(map[int64]bool)
	for _, tag := range postTags {
		selectedTagIDs[tag.ID] = true
	}

	data := map[string]interface{}{
		"Action":         "edit",
		"Post":           post,
		"AllTags":        allTags,
		"PostTags":       postTags,
		"SelectedTagIDs": selectedTagIDs,
	}

	// Ajout de l'utilisateur connecté
	if userID > 0 {
		if user, err := h.UserStore.GetByID(userID); err == nil {
			data["User"] = user
		}
	}

	RenderTemplate(w, "post_forms.html", data)
}

// Mise à jour d'un post
func (h *PostHandler) UpdatePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Récupération du post
	vars := mux.Vars(r)
	postID, err := strconv.ParseInt(vars["id"], 10, 64)
	if err != nil {
		http.Error(w, "ID de post invalide", http.StatusBadRequest)
		return
	}

	post, err := h.PostStore.GetByID(postID)
	if err != nil {
		http.Error(w, "Post non trouvé", http.StatusNotFound)
		return
	}

	// Vérification des permissions
	userID := getUserIDFromCookie(r)
	if userID != post.UserID {
		http.Error(w, "Vous n'êtes pas autorisé à modifier ce post", http.StatusForbidden)
		return
	}

	// Traitement du formulaire
	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		http.Error(w, "Erreur lors du traitement du formulaire", http.StatusBadRequest)
		return
	}

	// Récupération des données
	title := r.FormValue("title")
	content := r.FormValue("content")
	tagIDs := r.Form["tags"]
	newTags := r.FormValue("newTags")

	if title == "" || content == "" {
		http.Error(w, "Le titre et le contenu sont obligatoires", http.StatusBadRequest)
		return
	}

	// Mise à jour des données du post
	post.Title = title
	post.Content = content
	post.UpdatedAt = time.Now()

	// Traitement de l'image
	file, handler, err := r.FormFile("image")
	if err == nil {
		defer file.Close()

		// Vérification du type de fichier
		contentType := handler.Header.Get("Content-Type")
		if !strings.HasPrefix(contentType, "image/") {
			http.Error(w, "Type de fichier non supporté", http.StatusBadRequest)
			return
		}

		// Suppression de l'ancienne image
		if post.ImageURL != "" {
			oldFilePath := "." + post.ImageURL
			if _, err := os.Stat(oldFilePath); err == nil {
				os.Remove(oldFilePath)
			}
		}

		// Création du dossier si nécessaire
		uploadDir := "./static/uploads"
		if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
			os.MkdirAll(uploadDir, 0755)
		}

		// Sauvegarde du fichier
		filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filepath := filepath.Join(uploadDir, filename)
		dst, err := os.Create(filepath)
		if err != nil {
			http.Error(w, "Erreur lors de la création du fichier", http.StatusInternalServerError)
			return
		}
		defer dst.Close()

		io.Copy(dst, file)
		post.ImageURL = "/static/uploads/" + filename
		post.ImageType = contentType
	}

	// Sauvegarde des modifications
	if err := h.PostStore.Update(post); err != nil {
		http.Error(w, "Erreur lors de la mise à jour du post: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Mise à jour des tags
	h.PostStore.RemoveAllTags(postID)

	for _, tagIDStr := range tagIDs {
		tagID, err := strconv.ParseInt(tagIDStr, 10, 64)
		if err == nil {
			h.PostStore.AddTag(postID, tagID)
		}
	}

	// Traitement des nouveaux tags
	if newTags != "" {
		tagNames := strings.Split(newTags, ",")
		for _, name := range tagNames {
			name = strings.TrimSpace(name)
			if name != "" {
				tag, err := h.TagStore.CreateOrGet(name, "")
				if err == nil {
					h.PostStore.AddTag(postID, tag.ID)
				}
			}
		}
	}

	// Création de l'activité
	activity := &models.Activity{
		UserID:      userID,
		RecipientID: userID,
		Type:        models.ActivityUpdateProfile,
		TargetID:    post.ID,
		CreatedAt:   time.Now(),
		Content:     fmt.Sprintf("a mis à jour le post: %s", post.Title),
		IsRead:      true,
	}
	h.ActivityStore.Create(activity)

	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}

// Suppression d'un post
func (h *PostHandler) DeletePost(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non autorisée", http.StatusMethodNotAllowed)
		return
	}

	// Récupération du post
	postID, err := strconv.ParseInt(mux.Vars(r)["id"], 10, 64)
	if err != nil {
		http.Error(w, "ID de post invalide", http.StatusBadRequest)
		return
	}

	// Vérification des permissions
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	post, err := h.PostStore.GetByID(postID)
	if err != nil || post.UserID != userID {
		http.Error(w, "Non autorisé", http.StatusForbidden)
		return
	}

	// Suppression du post
	if err := h.PostStore.Delete(postID); err != nil {
		http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
}
