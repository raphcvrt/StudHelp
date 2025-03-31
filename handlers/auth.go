package handlers

import (
	"crypto/rand"
	"database/sql"
	"fmt"
	"forum/models"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	UserStore *models.UserStore
}

// installation de la base de donnée
var db *sql.DB

func SetDB(database *sql.DB) {
	db = database
}

func RegisterAuthRoutes(r *mux.Router) {
	h := NewAuthHandler(models.NewUserStore(db))

	// Routes d'authentification
	r.HandleFunc("/login", h.ShowLogin).Methods("GET")
	r.HandleFunc("/login", h.Login).Methods("POST")
	r.HandleFunc("/register", h.Register).Methods("POST")

	// Routes protégée
	r.HandleFunc("/logout", h.requireAuth(h.Logout)).Methods("GET")
}

// wrapper pour les handlers protégés
func (h *AuthHandler) requireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !h.isAuthenticated(r) {
			http.SetCookie(w, &http.Cookie{
				Name:  "redirect",
				Value: r.URL.Path,
				Path:  "/",
			})
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next(w, r)
	}
}
func NewAuthHandler(userStore *models.UserStore) *AuthHandler {
	return &AuthHandler{UserStore: userStore}
}

func (h *AuthHandler) ShowLogin(w http.ResponseWriter, r *http.Request) {
	action := r.URL.Query().Get("action")
	if action != "register" {
		action = "login"
	}

	data := map[string]interface{}{
		"Action":    action,
		"PageTitle": "Connexion",
	}

	if action == "register" {
		data["PageTitle"] = "Inscription"
	}

	RenderTemplate(w, "auth.html", data)
}

// s'inscrire
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	log.Println("[Register] Début de la fonction Register")
	log.Printf("[Register] Méthode HTTP: %s, URL: %s", r.Method, r.URL)

	if r.Method == http.MethodGet {
		// Rediriger vers la page de login avec option d'inscription
		http.Redirect(w, r, "/login?signup=true", http.StatusSeeOther)
		return
	}

	// CORRECTION : Utiliser ParseMultipartForm au lieu de ParseForm pour gérer les fichiers
	log.Println("[Register] Parsing du formulaire multipart...")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		// Si le formulaire n'est pas multipart, essayer avec ParseForm standard
		if err := r.ParseForm(); err != nil {
			log.Printf("[Register] Erreur parsing formulaire: %v", err)
			http.Error(w, "Could not parse form", http.StatusBadRequest)
			return
		}
	}
	log.Println("[Register] Formulaire parsé avec succès")

	// Récupération des valeurs du formulaire
	username := r.FormValue("username")
	email := r.FormValue("email")
	password := r.FormValue("password")
	confirmPassword := r.FormValue("confirm_password")

	log.Printf("[Register] Données reçues - Username: %s, Email: %s", username, email)
	log.Printf("[Register] Password length: %d, ConfirmPassword length: %d", len(password), len(confirmPassword))

	// Validation
	if username == "" || email == "" || password == "" || confirmPassword == "" {
		log.Printf("[Register] Champs manquants - Username: %t, Email: %t, Password: %t, ConfirmPassword: %t",
			username == "", email == "", password == "", confirmPassword == "")

		RenderTemplate(w, "auth.html", map[string]interface{}{
			"Action": "register",
			"Error":  "Tous les champs sont obligatoires",
			"FormData": map[string]string{
				"username": username,
				"email":    email,
			},
		})
		return
	}

	if password != confirmPassword {
		RenderTemplate(w, "auth.html", map[string]interface{}{
			"Action": "register",
			"Error":  "Les mots de passe ne correspondent pas",
			"FormData": map[string]string{
				"username": username,
				"email":    email,
			},
		})
		return
	}

	//est ce que l'username est unique
	if _, err := h.UserStore.GetByUsername(username); err == nil {
		RenderTemplate(w, "auth.html", map[string]interface{}{
			"Action": "register",
			"Error":  "Ce nom d'utilisateur est déjà pris",
			"FormData": map[string]string{
				"username": username,
				"email":    email,
			},
		})
		return
	}

	if _, err := h.UserStore.GetByEmail(email); err == nil {
		RenderTemplate(w, "auth.html", map[string]interface{}{
			"Action": "register",
			"Error":  "Cet email est déjà enregistré",
			"FormData": map[string]string{
				"username": username,
				"email":    email,
			},
		})
		return
	}

	// Hashage du mot de passe
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Printf("[Register] Erreur hachage mot de passe: %v", err)
		RenderTemplate(w, "auth.html", map[string]interface{}{
			"Action": "register",
			"Error":  "Erreur serveur - veuillez réessayer",
		})
		return
	}
	log.Printf("[Register] Hash généré pour nouvel utilisateur: %s", string(hashedPassword))

	// Génération UUID
	uuid, err := GenerateUUID()
	if err != nil {
		log.Printf("[Register] Erreur génération UUID: %v", err)
		RenderTemplate(w, "auth.html", map[string]interface{}{
			"Action": "register",
			"Error":  "Erreur serveur - veuillez réessayer",
		})
		return
	}

	// Création de l'user
	user := &models.User{
		UUID:      uuid,
		Username:  username,
		Email:     email,
		Password:  string(hashedPassword),
		AvatarURL: "/static/assets/pfp_placeholder.jpg",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := h.UserStore.Create(user); err != nil {
		log.Printf("[Register] Erreur création user: %v", err)
		RenderTemplate(w, "auth.html", map[string]interface{}{
			"Action": "register",
			"Error":  "Impossible de créer le compte utilisateur",
		})
		return
	}

	// Gestion de l'upload d'avatar
	file, handler, err := r.FormFile("avatar")
	if err == nil {
		defer file.Close()

		contentType := handler.Header.Get("Content-Type")
		if strings.HasPrefix(contentType, "image/") {
			uploadDir := "./static/avatars"
			if _, err := os.Stat(uploadDir); os.IsNotExist(err) {
				os.MkdirAll(uploadDir, 0755)
			}

			filename := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
			filepath := filepath.Join(uploadDir, filename)

			dst, err := os.Create(filepath)
			if err == nil {
				defer dst.Close()
				if _, err := io.Copy(dst, file); err == nil {
					user.AvatarURL = "/static/avatars/" + filename
					// Mise à jour de l'avatar dans la base de données
					if err := h.UserStore.UpdateAvatar(user.ID, "/static/avatars/"+filename); err != nil {
						log.Printf("[Register] Erreur mise à jour avatar: %v", err)
					}
				}
			}
		}
	} else {
		log.Printf("[Register] Pas d'avatar ou erreur: %v", err)
	}

	// On utilise directement l'ID retourné par Create()
	log.Printf("[Register] Création du cookie avec user_id=%d", user.ID)
	http.SetCookie(w, &http.Cookie{
		Name:     "user_id",
		Value:    strconv.FormatInt(user.ID, 10),
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	//log pour confirmer la redirection
	w.Header().Set("Cache-Control", "no-store")
	log.Println("[Register] Avant redirection - En-têtes:", w.Header())
	http.Redirect(w, r, "/", http.StatusSeeOther)
	log.Println("[Register] Après redirection")
}
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

// Login handles user authentication
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	log.Println("\n=== Début tentative de connexion ===")
	defer log.Println("=== Fin tentative de connexion ===")

	if r.Method != http.MethodPost {
		log.Println("Méthode non autorisée, redirection vers /login")
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Parse form data
	log.Println("Parsing du formulaire...")
	if err := r.ParseForm(); err != nil {
		log.Printf("Erreur parsing formulaire: %v\n", err)
		http.Error(w, "Could not parse form", http.StatusBadRequest)
		return
	}

	email := r.FormValue("email")
	password := r.FormValue("password")
	log.Printf("Données reçues - Email: [%s], Password: [%s]\n", email, password)

	// Validate input
	if email == "" || password == "" {
		log.Println("Email ou mot de passe vide")
		http.Error(w, "Email and password are required", http.StatusBadRequest)
		return
	}

	// Get user via l'email
	log.Printf("Recherche de l'utilisateur avec email: %s\n", email)
	user, err := h.UserStore.GetByEmail(email)
	if err != nil {
		log.Printf("ERREUR - Utilisateur non trouvé: %v\n", err)
		log.Printf("Détails DB: %+v\n", h.UserStore)
		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	log.Printf("Utilisateur trouvé: ID=%d, Email=%s\n", user.ID, user.Email)
	log.Printf("Hash stocké dans DB: %s\n", user.Password)

	// verif mdp
	log.Println("Comparaison du mot de passe avec le hash...")
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		log.Printf("ERREUR - Mot de passe invalide: %v\n", err)
		log.Println("=== DEBUG MOT DE PASSE ===")
		log.Printf("Longueur password reçu: %d\n", len(password))
		log.Printf("Longueur hash stocké: %d\n", len(user.Password))
		log.Println("=========================")

		http.Error(w, "Invalid credentials", http.StatusUnauthorized)
		return
	}

	log.Println("Authentification réussie, création du cookie...")
	http.SetCookie(w, &http.Cookie{
		Name:     "user_id",
		Value:    strconv.FormatInt(user.ID, 10),
		Expires:  time.Now().Add(24 * time.Hour),
		HttpOnly: true,
		Path:     "/",
		Secure:   r.TLS != nil,
		SameSite: http.SameSiteLaxMode,
	})

	log.Println("Redirection vers la page d'accueil")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:   "user_id",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) isAuthenticated(r *http.Request) bool {
	cookie, err := r.Cookie("user_id")
	if err != nil {
		return false
	}

	userID, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil || userID <= 0 {
		return false
	}

	// est ce que l'user existe
	_, err = h.UserStore.GetByID(userID)
	return err == nil
}
func (h *AuthHandler) authMiddleware() mux.MiddlewareFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !h.isAuthenticated(r) {
				// Stocke l'URL demandée pour redirection après login
				http.SetCookie(w, &http.Cookie{
					Name:  "redirect",
					Value: r.URL.Path,
					Path:  "/",
				})
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
func GenerateUUID() (string, error) {
	uuid := make([]byte, 16)
	_, err := rand.Read(uuid)
	if err != nil {
		return "", err
	}

	uuid[6] = (uuid[6] & 0x0f) | 0x40
	uuid[8] = (uuid[8] & 0x3f) | 0x80

	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:]), nil
}
func GetUserIDFromRequest(r *http.Request) int64 {
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
