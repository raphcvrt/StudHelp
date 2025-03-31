package handlers

import (
	"fmt"
	"forum/database"
	"forum/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

func RegisterCommentRoutes(r *mux.Router) {
	// r.HandleFunc("/comments", CreateCommentHandler).Methods("POST")
	r.HandleFunc("/post/{id}/comment", PostCommentHandler).Methods("POST")
}
func PostCommentHandler(w http.ResponseWriter, r *http.Request) {
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	vars := mux.Vars(r)
	postIDStr := vars["id"]

	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		http.Error(w, "Invalid post ID", http.StatusBadRequest)
		return
	}

	content := r.FormValue("content")
	if content == "" {
		http.Error(w, "Comment content is required", http.StatusBadRequest)
		return
	}

	// Initialiser tous les champs nécessaires
	now := time.Now()
	comment := &models.Comment{
		PostID:       postID,
		UserID:       userID,
		Content:      content,
		CreatedAt:    now,
		UpdatedAt:    now,
		LikeCount:    0,
		DislikeCount: 0,
	}

	log.Printf("Tentative de création d'un commentaire: PostID=%d, UserID=%d", postID, userID)

	if err := models.NewCommentStore(database.GetDB()).Create(comment); err != nil {
		log.Printf("Erreur lors de la création du commentaire: %v", err)
		http.Error(w, "Failed to create comment", http.StatusInternalServerError)
		return
	}

	log.Printf("Commentaire créé avec succès, ID=%d", comment.ID)

	// Récupérer les informations du post pour la notification
	postStore := models.NewPostStore(database.GetDB())
	post, err := postStore.GetByID(postID)
	if err != nil {
		log.Printf("Erreur lors de la récupération du post pour la notification: %v", err)
	} else if post.UserID != userID { // Ne pas notifier si l'utilisateur commente son propre post
		// Créer une notification pour le propriétaire du post
		activityStore := models.NewActivityStore(database.GetDB())

		activity := &models.Activity{
			UserID:      userID,      // L'auteur du commentaire
			RecipientID: post.UserID, // Le propriétaire du post
			Type:        models.ActivityComment,
			TargetID:    postID,
			CreatedAt:   now,
			Content:     "a commenté sur votre post",
			IsRead:      false,
		}

		if err := activityStore.Create(activity); err != nil {
			log.Printf("Failed to create notification: %v", err)
		} else {
			log.Printf("Notification créée avec succès pour l'utilisateur %d", post.UserID)
		}
	}

	// ajouter a l'activité (pour l'historique)
	activity := &models.Activity{
		UserID:      userID,
		RecipientID: userID, // L'utilisateur lui-même pour l'historique
		Type:        models.ActivityComment,
		TargetID:    postID,
		CreatedAt:   now,
		Content:     "a commenté sur un post de",
		IsRead:      true, // Déjà lue puisque c'est l'utilisateur qui l'a créée
	}

	if err := models.NewActivityStore(database.GetDB()).Create(activity); err != nil {
		log.Printf("Failed to create activity record: %v", err)
	}

	http.Redirect(w, r, fmt.Sprintf("/post/%d", postID), http.StatusSeeOther)
}
