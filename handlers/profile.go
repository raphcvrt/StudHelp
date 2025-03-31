package handlers

import (
	"fmt"
	"forum/database"
	"forum/models"
	"image"
	"image/jpeg"
	"image/png"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/nfnt/resize"
)

type ProfileHandler struct {
	UserStore *models.UserStore
	PostStore *models.PostStore
}

func NewProfileHandler(userStore *models.UserStore, postStore *models.PostStore) *ProfileHandler {
	return &ProfileHandler{
		UserStore: userStore,
		PostStore: postStore,
	}
}

// RegisterProfileRoutes enregistre toutes les routes liées au profil
func RegisterProfileRoutes(r *mux.Router, auth *AuthHandler) {
	userStore := models.NewUserStore(database.GetDB())
	postStore := models.NewPostStore(database.GetDB())
	h := NewProfileHandler(userStore, postStore)

	// Groupe de routes protégées par authentification
	profileRoutes := r.PathPrefix("").Subrouter()
	profileRoutes.Use(auth.authMiddleware())

	// Routes du profil
	profileRoutes.HandleFunc("/user/{id:[0-9]+}", h.ShowUserProfile).Methods("GET")
	profileRoutes.HandleFunc("/profile", h.ShowProfile).Methods("GET")
	profileRoutes.HandleFunc("/upload-avatar", h.UploadAvatar).Methods("POST")
	profileRoutes.HandleFunc("/update-profile", h.UpdateProfile).Methods("POST")
}

func (h *ProfileHandler) ShowProfile(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'ID de l'utilisateur connecté
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Récupérer les paramètres de pagination
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1 // Valeur par défaut
	}

	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 || perPage > 100 {
		perPage = 10 // Valeur par défaut, limitée à 100 pour éviter les abus
	}

	// Configuration du filtre pour les posts créés
	filter := models.PostFilter{
		UserID:    userID, // Filtrer les posts par l'ID de l'utilisateur connecté
		SortBy:    "date",
		SortOrder: "desc",
	}
	filter.Pagination.Page = page
	filter.Pagination.PerPage = perPage

	// Récupération des posts créés par l'utilisateur
	posts, err := models.NewPostStore(database.GetDB()).FilterPosts(filter)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des posts", http.StatusInternalServerError)
		return
	}

	// Récupération des posts aimés par l'utilisateur
	likeStore := models.NewLikeStore(database.GetDB())
	likedPostIDs, err := likeStore.GetLikedPostIDs(userID)
	if err != nil {
		log.Printf("Erreur lors de la récupération des posts aimés: %v", err)
		// Continuer même en cas d'erreur
		likedPostIDs = []int64{}
	}

	// Récupération des détails des posts aimés
	var likedPosts []*models.Post
	postStore := models.NewPostStore(database.GetDB())
	for _, postID := range likedPostIDs {
		post, err := postStore.GetByID(postID)
		if err == nil && post.Status == models.StatusApproved {
			likedPosts = append(likedPosts, post)
		}
	}

	// Récupérer les auteurs des posts aimés
	authors := make(map[int64]*models.User)
	for _, post := range likedPosts {
		if _, exists := authors[post.UserID]; !exists {
			user, err := h.UserStore.GetByID(post.UserID)
			if err == nil {
				authors[post.UserID] = user
			}
		}
	}

	// Récupérer le nombre total de posts
	totalPosts := len(posts)

	// Récupérer le nombre de commentaires pour chaque post (créés et aimés)
	commentCounts := make(map[int64]int)
	commentStore := models.NewCommentStore(database.GetDB())
	totalComments := 0

	// Pour les posts créés
	for _, post := range posts {
		comments, err := commentStore.GetCommentsByPostID(post.ID)
		if err == nil {
			commentCounts[post.ID] = len(comments)
		} else {
			commentCounts[post.ID] = 0
		}
	}

	// Pour les posts aimés
	for _, post := range likedPosts {
		if _, exists := commentCounts[post.ID]; !exists {
			comments, err := commentStore.GetCommentsByPostID(post.ID)
			if err == nil {
				commentCounts[post.ID] = len(comments)
			} else {
				commentCounts[post.ID] = 0
			}
		}
	}

	// Calculer le nombre total de commentaires faits par l'utilisateur
	userComments, err := commentStore.GetCommentsByUserID(userID)
	if err == nil {
		totalComments = len(userComments)
	}

	// Récupération des infos utilisateur
	user, err := models.NewUserStore(database.GetDB()).GetByID(userID)
	if err != nil {
		http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		return
	}

	// Préparation des données pour le template
	data := map[string]interface{}{
		"User":            user,
		"Posts":           posts,
		"LikedPosts":      likedPosts,
		"Authors":         authors,
		"CommentCounts":   commentCounts,
		"IsAuthenticated": true,
		"TotalPosts":      totalPosts,
		"TotalComments":   totalComments,
	}

	RenderTemplate(w, "profile.html", data)
}

// ShowUserProfile - Mise à jour pour inclure les statistiques
func (h *ProfileHandler) ShowUserProfile(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'ID de l'utilisateur depuis l'URL
	vars := mux.Vars(r)
	userIDStr := vars["id"]

	userID, err := strconv.ParseInt(userIDStr, 10, 64)
	if err != nil {
		http.Error(w, "ID utilisateur invalide", http.StatusBadRequest)
		return
	}

	// Récupérer l'utilisateur
	user, err := h.UserStore.GetByID(userID)
	if err != nil {
		http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		return
	}

	// Récupérer les paramètres de pagination
	page, err := strconv.Atoi(r.URL.Query().Get("page"))
	if err != nil || page < 1 {
		page = 1 // Valeur par défaut
	}

	perPage, err := strconv.Atoi(r.URL.Query().Get("per_page"))
	if err != nil || perPage < 1 || perPage > 100 {
		perPage = 10 // Valeur par défaut, limitée à 100 pour éviter les abus
	}

	// Configuration du filtre
	filter := models.PostFilter{
		UserID:    userID,
		SortBy:    "date",
		SortOrder: "desc",
	}
	filter.Pagination.Page = page
	filter.Pagination.PerPage = perPage

	// Récupération des posts
	posts, err := h.PostStore.FilterPosts(filter)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des posts", http.StatusInternalServerError)
		return
	}

	// Récupérer le nombre total de posts
	totalPosts := len(posts)

	// Récupérer le nombre de commentaires pour chaque post et calculer le total
	commentCounts := make(map[int64]int)
	totalComments := 0
	commentStore := models.NewCommentStore(database.GetDB())

	// Compter les commentaires pour chaque post
	for _, post := range posts {
		comments, err := commentStore.GetCommentsByPostID(post.ID)
		if err == nil {
			commentCounts[post.ID] = len(comments)
		} else {
			commentCounts[post.ID] = 0
		}
	}

	// Calculer le nombre total de commentaires faits par l'utilisateur
	userComments, err := commentStore.GetCommentsByUserID(userID)
	if err == nil {
		totalComments = len(userComments)
	}

	// Vérifier si l'utilisateur actuel est authentifié
	currentUserID := getUserIDFromCookie(r)
	isAuthenticated := currentUserID > 0
	isOwnProfile := currentUserID == userID

	// Préparation des données pour le template
	data := map[string]interface{}{
		"User":            user,
		"Posts":           posts,
		"CommentCounts":   commentCounts,
		"IsAuthenticated": isAuthenticated,
		"IsOwnProfile":    isOwnProfile,
		"TotalPosts":      totalPosts,
		"TotalComments":   totalComments,
	}

	// Utiliser le template du profil public
	RenderTemplate(w, "user_profile.html", data)
}

// UploadAvatar gère le téléchargement d'avatar et le recadrage en carré
func (h *ProfileHandler) UploadAvatar(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Non autorisé", http.StatusUnauthorized)
		return
	}

	// Limite à 10MB
	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		log.Printf("Erreur lors du parsing du formulaire: %v", err)
		http.Error(w, "Erreur de lecture du formulaire", http.StatusBadRequest)
		return
	}

	file, handler, err := r.FormFile("avatar")
	if err != nil {
		log.Printf("Erreur lors de la récupération du fichier: %v", err)
		http.Error(w, "Erreur de lecture du fichier", http.StatusBadRequest)
		return
	}
	defer file.Close()

	// Vérifier le type MIME
	contentType := handler.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "image/") {
		http.Error(w, "Seules les images sont autorisées", http.StatusBadRequest)
		return
	}

	// Créer le dossier si inexistant
	uploadDir := "./static/avatars"
	if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
		err := os.MkdirAll(uploadDir, 0755)
		if err != nil {
			log.Printf("Erreur lors de la création du dossier: %v", err)
			http.Error(w, "Erreur système", http.StatusInternalServerError)
			return
		}
	}

	// Générer un nom de fichier unique avec timestamp
	filename := fmt.Sprintf("%d_%d_%s", userID, time.Now().Unix(), handler.Filename)
	filepath := filepath.Join(uploadDir, filename)

	// Décoder l'image
	img, _, err := image.Decode(file)
	if err != nil {
		log.Printf("Erreur lors du décodage de l'image: %v", err)
		http.Error(w, "Format d'image non supporté", http.StatusBadRequest)
		return
	}

	// Recadrer l'image en carré
	squareImg := cropToSquare(img)

	// Redimensionner à une taille standard (par exemple 200x200 pixels)
	resizedImg := resize.Resize(200, 200, squareImg, resize.Lanczos3)

	// Sauvegarder l'image recadrée
	dst, err := os.Create(filepath)
	if err != nil {
		log.Printf("Erreur lors de la création du fichier: %v", err)
		http.Error(w, "Erreur de sauvegarde", http.StatusInternalServerError)
		return
	}
	defer dst.Close()

	// Encoder et sauvegarder l'image en fonction du type d'origine
	if strings.Contains(contentType, "jpeg") || strings.Contains(contentType, "jpg") {
		jpeg.Encode(dst, resizedImg, &jpeg.Options{Quality: 90})
	} else if strings.Contains(contentType, "png") {
		png.Encode(dst, resizedImg)
	} else {
		// Par défaut, enregistrer en JPEG
		jpeg.Encode(dst, resizedImg, &jpeg.Options{Quality: 90})
	}

	// Mettre à jour l'avatar dans la base
	avatarURL := "/static/avatars/" + filename
	if err := h.UserStore.UpdateAvatar(userID, avatarURL); err != nil {
		log.Printf("Erreur lors de la mise à jour de l'avatar dans la base: %v", err)
		http.Error(w, "Erreur de base de données", http.StatusInternalServerError)
		return
	}

	log.Printf("Avatar mis à jour avec succès pour l'utilisateur %d: %s", userID, avatarURL)
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}

// cropToSquare recadre une image pour la rendre carrée
func cropToSquare(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Déterminer la plus petite dimension
	size := width
	if height < width {
		size = height
	}

	// Calculer les coordonnées de recadrage pour centrer le carré
	x0 := (width - size) / 2
	y0 := (height - size) / 2
	x1 := x0 + size
	y1 := y0 + size

	// Créer une nouvelle image carrée
	square := image.NewRGBA(image.Rect(0, 0, size, size))

	// Copier la partie centrale de l'image originale en utilisant x1 et y1
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			square.Set(x-x0, y-y0, img.At(x, y))
		}
	}

	return square
}

// UpdateProfile gère la mise à jour des infos
func (h *ProfileHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Non autorisé", http.StatusUnauthorized)
		return
	}

	if err := r.ParseForm(); err != nil {
		log.Printf("Erreur lors du parsing du formulaire: %v", err)
		http.Error(w, "Formulaire invalide", http.StatusBadRequest)
		return
	}

	username := r.FormValue("username")
	email := r.FormValue("email")

	// Récupérer les valeurs actuelles pour comparer
	currentUser, err := h.UserStore.GetByID(userID)
	if err != nil {
		log.Printf("Erreur lors de la récupération de l'utilisateur: %v", err)
		http.Error(w, "Utilisateur non trouvé", http.StatusInternalServerError)
		return
	}

	// Vérification unicité username si modifié
	if username != currentUser.Username {
		if _, err := h.UserStore.GetByUsername(username); err == nil {
			http.Error(w, "Ce nom d'utilisateur est déjà utilisé", http.StatusBadRequest)
			return
		}
	}

	// Vérification unicité email si modifié
	if email != currentUser.Email {
		if _, err := h.UserStore.GetByEmail(email); err == nil {
			http.Error(w, "Cette adresse email est déjà utilisée", http.StatusBadRequest)
			return
		}
	}

	if err := h.UserStore.UpdateProfile(userID, username, email); err != nil {
		log.Printf("Erreur lors de la mise à jour du profil: %v", err)
		http.Error(w, "Erreur de mise à jour", http.StatusInternalServerError)
		return
	}

	log.Printf("Profil mis à jour avec succès pour l'utilisateur %d", userID)
	http.Redirect(w, r, "/profile", http.StatusSeeOther)
}
