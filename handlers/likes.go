package handlers

import (
	"encoding/json"
	"forum/models"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"
)

type LikeHandler struct {
	LikeStore     *models.LikeStore
	PostStore     *models.PostStore
	CommentStore  *models.CommentStore
	ActivityStore *models.ActivityStore
	UserStore     *models.UserStore
}

// NewLikeHandler crée une nouvelle instance de LikeHandler
func NewLikeHandler(likeStore *models.LikeStore, postStore *models.PostStore, commentStore *models.CommentStore) *LikeHandler {
	// Initialiser également l'ActivityStore pour les notifications
	return &LikeHandler{
		LikeStore:     likeStore,
		PostStore:     postStore,
		CommentStore:  commentStore,
		ActivityStore: models.NewActivityStore(postStore.DB),
		UserStore:     models.NewUserStore(postStore.DB),
	}
}

// RegisterLikeRoutes enregistre les routes pour les likes/dislikes
func RegisterLikeRoutes(r *mux.Router, h *LikeHandler) {
	r.HandleFunc("/api/post/{id}/{action}", h.PostLikeDislikeHandler).Methods("POST")
	r.HandleFunc("/api/comment/{id}/{action}", h.CommentLikeDislikeHandler).Methods("POST")
}

// PostLikeDislikeHandler gère les likes/dislikes pour les posts
func (h *LikeHandler) PostLikeDislikeHandler(w http.ResponseWriter, r *http.Request) {
	// Vérifier si l'utilisateur est connecté
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Vous devez être connecté", http.StatusUnauthorized)
		return
	}

	// Récupérer les paramètres de l'URL
	vars := mux.Vars(r)
	postIDStr := vars["id"]
	action := vars["action"]

	// Valider l'action
	if action != "like" && action != "dislike" && action != "remove" {
		http.Error(w, "Action invalide", http.StatusBadRequest)
		return
	}

	// Convertir l'ID du post
	postID, err := strconv.ParseInt(postIDStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de post invalide", http.StatusBadRequest)
		return
	}

	// Récupérer les informations sur le post
	post, err := h.PostStore.GetByID(postID)
	if err != nil {
		http.Error(w, "Post non trouvé", http.StatusNotFound)
		return
	}

	// Vérifier si le post appartient à l'utilisateur qui like/dislike
	if post.UserID != userID {
		// Récupérer l'état actuel du like/dislike
		hasReaction, isLike, _ := h.LikeStore.GetUserLike(postID, userID)

		// Créer une notification seulement si c'est une nouvelle action ou un changement
		shouldNotify := false
		var notificationType models.ActivityType

		if action == "like" {
			if !hasReaction || !isLike {
				shouldNotify = true
				notificationType = models.ActivityLike
			}
		} else if action == "dislike" {
			if !hasReaction || isLike {
				shouldNotify = true
				notificationType = models.ActivityDislike
			}
		}

		if shouldNotify && action != "remove" {
			// Créer une notification pour le propriétaire du post
			now := time.Now()
			activity := &models.Activity{
				UserID:      userID,
				RecipientID: post.UserID,
				Type:        notificationType,
				TargetID:    postID,
				CreatedAt:   now,
				Content: func() string {
					if action == "like" {
						return "a aimé votre post"
					}
					return "n'a pas aimé votre post"
				}(),
				IsRead: false,
			}

			if h.ActivityStore != nil {
				if err := h.ActivityStore.Create(activity); err != nil {
					log.Printf("Erreur lors de la création de la notification: %v", err)
				} else {
					log.Printf("Notification de %s créée pour l'utilisateur %d", action, post.UserID)
				}
			}
		}
	}

	// Traiter l'action
	if action == "remove" {
		err = h.LikeStore.RemoveLike(postID, userID)
	} else {
		isLike := action == "like"
		err = h.LikeStore.AddOrUpdateLike(postID, userID, isLike)
	}

	if err != nil {
		log.Printf("Erreur lors du traitement de la réaction: %v", err)
		http.Error(w, "Erreur lors du traitement de la réaction", http.StatusInternalServerError)
		return
	}

	// Récupérer les nouveaux compteurs de likes/dislikes
	var likeCount, dislikeCount int
	err = h.PostStore.DB.QueryRow("SELECT like_count, dislike_count FROM posts WHERE id = ?", postID).Scan(&likeCount, &dislikeCount)
	if err != nil {
		log.Printf("Erreur lors de la récupération des compteurs: %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Récupérer l'état actuel de la réaction de l'utilisateur
	hasReaction, isLike, _ := h.LikeStore.GetUserLike(postID, userID)
	var userAction string
	if hasReaction {
		if isLike {
			userAction = "like"
		} else {
			userAction = "dislike"
		}
	} else {
		userAction = ""
	}

	// Renvoyer la réponse JSON
	response := map[string]interface{}{
		"success":    true,
		"likes":      likeCount,
		"dislikes":   dislikeCount,
		"userAction": userAction,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// CommentLikeDislikeHandler gère les likes/dislikes pour les commentaires
func (h *LikeHandler) CommentLikeDislikeHandler(w http.ResponseWriter, r *http.Request) {
	// Vérifier si l'utilisateur est connecté
	userID := getUserIDFromCookie(r)
	if userID == 0 {
		http.Error(w, "Vous devez être connecté", http.StatusUnauthorized)
		return
	}

	// Récupérer les paramètres de l'URL
	vars := mux.Vars(r)
	commentIDStr := vars["id"]
	action := vars["action"]

	// Valider l'action
	if action != "like" && action != "dislike" && action != "remove" {
		http.Error(w, "Action invalide", http.StatusBadRequest)
		return
	}

	// Convertir l'ID du commentaire
	commentID, err := strconv.ParseInt(commentIDStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de commentaire invalide", http.StatusBadRequest)
		return
	}

	// Récupérer le commentaire
	comment, err := h.CommentStore.GetByID(commentID)
	if err != nil {
		http.Error(w, "Commentaire non trouvé", http.StatusNotFound)
		return
	}

	// Récupérer le post associé au commentaire (pour ajouter plus de contexte dans la notification si nécessaire)
	_, err = h.PostStore.GetByID(comment.PostID)
	if err != nil {
		log.Printf("Erreur lors de la récupération du post pour la notification: %v", err)
	}

	// Vérifier si le commentaire appartient à l'utilisateur qui like/dislike
	if comment.UserID != userID {
		// Créer une notification pour le propriétaire du commentaire
		shouldNotify := false
		var notificationType models.ActivityType

		// Chercher si un like existe déjà
		like, err := h.LikeStore.GetByCommentAndUser(commentID, userID)
		if err != nil {
			log.Printf("Erreur lors de la récupération du like pour le commentaire: %v", err)
		}
		if action == "like" {
			if like == nil || !like.IsLike {
				shouldNotify = true
				notificationType = models.ActivityLike
			}
		} else if action == "dislike" {
			if like == nil || like.IsLike {
				shouldNotify = true
				notificationType = models.ActivityDislike
			}
		}

		if shouldNotify && action != "remove" && h.ActivityStore != nil {
			// Créer une notification pour le propriétaire du commentaire
			now := time.Now()
			activity := &models.Activity{
				UserID:      userID,
				RecipientID: comment.UserID,
				Type:        notificationType,
				TargetID:    comment.PostID, // On utilise l'ID du post comme cible pour le lien
				CreatedAt:   now,
				Content: func() string {
					if action == "like" {
						return "a aimé votre commentaire sur"
					}
					return "n'a pas aimé votre commentaire sur"
				}(),
				IsRead: false,
			}

			if err := h.ActivityStore.Create(activity); err != nil {
				log.Printf("Erreur lors de la création de la notification: %v", err)
			} else {
				log.Printf("Notification de %s créée pour l'utilisateur %d", action, comment.UserID)
			}
		}
	}

	// Chercher si un like existe déjà
	like, err := h.LikeStore.GetByCommentAndUser(commentID, userID)

	// Variables pour suivre l'état
	var userAction string

	// Déterminer l'action à effectuer
	if action == "like" {
		if like == nil {
			// Nouveau like
			like = &models.Like{
				CommentID: commentID,
				UserID:    userID,
				IsLike:    true,
			}
			err = h.LikeStore.CreateCommentLike(like)
			userAction = "like"
			// Incrémenter le nombre de likes
			err = h.CommentStore.UpdateLikeCount(commentID, 1)
		} else if !like.IsLike {
			// Changer dislike en like
			like.IsLike = true
			err = h.LikeStore.UpdateCommentLike(like)
			userAction = "like"
			// Incrémenter likes, décrémenter dislikes
			err = h.CommentStore.UpdateLikeCount(commentID, 1)
			err = h.CommentStore.UpdateDislikeCount(commentID, -1)
		} else {
			// Déjà un like, le retirer
			err = h.LikeStore.DeleteCommentLike(commentID, userID)
			userAction = "none"
			// Décrémenter le nombre de likes
			err = h.CommentStore.UpdateLikeCount(commentID, -1)
		}
	} else if action == "dislike" {
		if like == nil {
			// Nouveau dislike
			like = &models.Like{
				CommentID: commentID,
				UserID:    userID,
				IsLike:    false,
			}
			err = h.LikeStore.CreateCommentLike(like)
			userAction = "dislike"
			// Incrémenter le nombre de dislikes
			err = h.CommentStore.UpdateDislikeCount(commentID, 1)
		} else if like.IsLike {
			// Changer like en dislike
			like.IsLike = false
			err = h.LikeStore.UpdateCommentLike(like)
			userAction = "dislike"
			// Décrémenter likes, incrémenter dislikes
			err = h.CommentStore.UpdateLikeCount(commentID, -1)
			err = h.CommentStore.UpdateDislikeCount(commentID, 1)
		} else {
			// Déjà un dislike, le retirer
			err = h.LikeStore.DeleteCommentLike(commentID, userID)
			userAction = "none"
			// Décrémenter le nombre de dislikes
			err = h.CommentStore.UpdateDislikeCount(commentID, -1)
		}
	} else if action == "remove" {
		// Supprimer la réaction
		if like != nil {
			if like.IsLike {
				err = h.CommentStore.UpdateLikeCount(commentID, -1)
			} else {
				err = h.CommentStore.UpdateDislikeCount(commentID, -1)
			}
			err = h.LikeStore.DeleteCommentLike(commentID, userID)
		}
		userAction = "none"
	}

	if err != nil {
		log.Printf("Erreur lors du traitement de la réaction pour le commentaire: %v", err)
		http.Error(w, "Erreur lors du traitement de la réaction", http.StatusInternalServerError)
		return
	}

	// Récupérer les compteurs mis à jour
	comment, err = h.CommentStore.GetByID(commentID)
	if err != nil {
		http.Error(w, "Erreur lors de la récupération des compteurs", http.StatusInternalServerError)
		return
	}

	// Renvoyer la réponse JSON
	response := map[string]interface{}{
		"success":    true,
		"likes":      comment.LikeCount,
		"dislikes":   comment.DislikeCount,
		"userAction": userAction,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
