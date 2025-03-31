package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"forum/database"
	"forum/handlers"
	"forum/models"

	"github.com/gorilla/mux"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	// Configuration des logs
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Démarrage de l'application Forum...")

	// Chargement des templates
	err := handlers.LoadTemplates("templates")
	if err != nil {
		log.Fatalf("Échec du chargement des templates: %v", err)
	}

	// Vérification des templates essentiels
	requiredTemplates := []string{
		"base.html", "auth.html", "post_forms.html", "post_view.html",
		"profile.html", "index.html", "notifications.html",
	}

	for _, tmpl := range requiredTemplates {
		if handlers.Templates.Lookup(tmpl) == nil {
			log.Fatalf("Template essentiel manquant: %s", tmpl)
		}
	}

	// Initialisation de la base de données
	db, err := database.InitDB()
	if err != nil {
		log.Fatalf("Échec d'initialisation de la DB: %v", err)
	}
	defer db.Close()

	// Application des migrations
	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Échec des migrations: %v", err)
	}

	// Configuration du routeur
	r := mux.NewRouter()
	r.Use(loggingMiddleware)

	// Fichiers statiques
	fs := http.FileServer(http.Dir("./static"))
	r.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fs))

	// Initialisation des stores
	userStore := models.NewUserStore(db)
	postStore := models.NewPostStore(db)
	tagStore := models.NewTagStore(db)
	commentStore := models.NewCommentStore(db)
	likeStore := models.NewLikeStore(db)
	activityStore := models.NewActivityStore(db)

	// Initialisation des handlers
	likeHandler := handlers.NewLikeHandler(likeStore, postStore, commentStore)
	postHandler := handlers.NewPostHandler(postStore, tagStore, commentStore, userStore, likeStore, activityStore)
	tagHandler := handlers.NewTagHandler(tagStore, postStore, userStore, commentStore)
	authHandler := handlers.NewAuthHandler(userStore)
	profileHandler := handlers.NewProfileHandler(userStore, postStore)
	notificationHandler := handlers.NewNotificationHandler(activityStore, userStore, postStore, commentStore)

	// Enregistrement des routes spécifiques à chaque domaine
	handlers.RegisterCommentRoutes(r)
	handlers.RegisterLikeRoutes(r, likeHandler)
	handlers.RegisterTagRoutes(r, tagHandler)
	handlers.RegisterNotificationRoutes(r, notificationHandler)

	// Routes d'authentification
	r.HandleFunc("/login", authHandler.ShowLogin).Methods("GET")
	r.HandleFunc("/login", authHandler.Login).Methods("POST")
	r.HandleFunc("/register", authHandler.Register).Methods("POST")
	r.HandleFunc("/logout", authHandler.Logout).Methods("GET")

	// Routes pour les posts
	r.HandleFunc("/", postHandler.HomePage).Methods("GET")
	r.HandleFunc("/post/{id}", postHandler.ViewPost).Methods("GET")
	r.HandleFunc("/create-post", postHandler.NewPostPage).Methods("GET")
	r.HandleFunc("/create-post", postHandler.CreatePost).Methods("POST")
	r.HandleFunc("/edit-post/{id}", postHandler.EditPostPage).Methods("GET")
	r.HandleFunc("/edit-post/{id}", postHandler.UpdatePost).Methods("POST")
	r.HandleFunc("/delete-post/{id}", postHandler.DeletePost).Methods("POST")

	// Routes pour les profils
	r.HandleFunc("/user/{id:[0-9]+}", profileHandler.ShowUserProfile).Methods("GET")
	r.HandleFunc("/profile", func(w http.ResponseWriter, r *http.Request) {
		if _, err := r.Cookie("user_id"); err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		profileHandler.ShowProfile(w, r)
	}).Methods("GET")

	// Middleware pour l'authentification
	authMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if _, err := r.Cookie("user_id"); err != nil {
				http.Redirect(w, r, "/login", http.StatusSeeOther)
				return
			}
			next.ServeHTTP(w, r)
		})
	}

	// Routes protégées par authentification
	protected := r.PathPrefix("/").Subrouter()
	protected.Use(authMiddleware)
	protected.HandleFunc("/profile", profileHandler.ShowProfile).Methods("GET")
	protected.HandleFunc("/upload-avatar", profileHandler.UploadAvatar).Methods("POST")
	protected.HandleFunc("/update-profile", profileHandler.UpdateProfile).Methods("POST")
	protected.HandleFunc("/notifications", notificationHandler.ShowNotifications).Methods("GET")

	// Configuration du serveur HTTP
	srv := &http.Server{
		Addr:         ":8080",
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Println("Serveur prêt à écouter sur http://localhost:8080")
	if err := srv.ListenAndServe(); err != nil {
		log.Fatalf("Erreur du serveur: %v", err)
	}
}

// Middleware de logging pour les requêtes
func loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		log.Printf("%s %s (Durée: %v)", r.Method, r.URL.Path, time.Since(start))
	})
}
