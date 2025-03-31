package handlers

import (
	"forum/models"
	"log"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"
)

type NotificationHandler struct {
	ActivityStore *models.ActivityStore
	UserStore     *models.UserStore
	PostStore     *models.PostStore
	CommentStore  *models.CommentStore
}

// Créer une nouvelle instance de NotificationHandler
func NewNotificationHandler(activityStore *models.ActivityStore, userStore *models.UserStore, postStore *models.PostStore, commentStore *models.CommentStore) *NotificationHandler {
	return &NotificationHandler{
		ActivityStore: activityStore,
		UserStore:     userStore,
		PostStore:     postStore,
		CommentStore:  commentStore,
	}
}

// ShowNotifications affiche toutes les notifications d'un utilisateur
func (h *NotificationHandler) ShowNotifications(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'ID de l'utilisateur connecté
	userID := GetUserIDFromRequest(r)
	if userID == 0 {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	// Récupérer l'utilisateur
	user, err := h.UserStore.GetByID(userID)
	if err != nil {
		http.Error(w, "Utilisateur non trouvé", http.StatusNotFound)
		return
	}

	// Récupérer les notifications (activités) pour cet utilisateur
	// Ces activités concernent des actions sur les posts/commentaires de l'utilisateur
	activities, err := h.ActivityStore.GetNotificationsForUser(userID)
	if err != nil {
		log.Printf("Erreur lors de la récupération des notifications: %v", err)
		http.Error(w, "Erreur lors de la récupération des notifications", http.StatusInternalServerError)
		return
	}

	// Récupérer les détails nécessaires pour chaque notification
	notifications := make([]map[string]interface{}, 0)
	for _, activity := range activities {
		// Récupérer l'utilisateur qui a effectué l'action
		actorUser, err := h.UserStore.GetByID(activity.UserID)
		if err != nil {
			continue
		}

		// Préparer les données de base pour chaque notification
		notification := map[string]interface{}{
			"ID":            activity.ID,
			"ActorUser":     actorUser,
			"Type":          activity.Type,
			"CreatedAt":     activity.CreatedAt,
			"FormattedDate": activity.GetFormattedDate(),
			"Content":       activity.Content,
			"IsRead":        activity.IsRead,
		}

		// Ajouter des informations spécifiques en fonction du type d'activité
		switch activity.Type {
		case models.ActivityComment, models.ActivityLike, models.ActivityDislike:
			// Récupérer les détails du post concerné
			post, err := h.PostStore.GetByID(activity.TargetID)
			if err == nil {
				notification["Post"] = post
			}
		}

		notifications = append(notifications, notification)
	}

	// Marquer toutes les notifications comme lues
	go func() {
		if err := h.ActivityStore.MarkNotificationsAsRead(userID); err != nil {
			log.Printf("Erreur lors de la mise à jour des notifications: %v", err)
		}
	}()

	// Préparer les données pour le template
	data := map[string]interface{}{
		"User":            user,
		"Notifications":   notifications,
		"IsAuthenticated": true,
	}

	// Servir le template de notifications
	RenderTemplate(w, "notifications.html", data)
}

// GetUnreadNotificationsCount retourne le nombre de notifications non lues
func (h *NotificationHandler) GetUnreadNotificationsCount(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'ID de l'utilisateur connecté
	userID := GetUserIDFromRequest(r)
	if userID == 0 {
		http.Error(w, "Non authentifié", http.StatusUnauthorized)
		return
	}

	// Récupérer le nombre de notifications non lues
	count, err := h.ActivityStore.GetUnreadNotificationsCount(userID)
	if err != nil {
		log.Printf("Erreur lors de la récupération du nombre de notifications: %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Retourner le résultat en JSON
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"count": ` + strconv.Itoa(count) + `}`))
}

// DeleteNotification supprime une notification
func (h *NotificationHandler) DeleteNotification(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'ID de l'utilisateur connecté
	userID := GetUserIDFromRequest(r)
	if userID == 0 {
		http.Error(w, "Non authentifié", http.StatusUnauthorized)
		return
	}

	// Récupérer l'ID de la notification à supprimer
	vars := mux.Vars(r)
	notificationIDStr := vars["id"]
	notificationID, err := strconv.ParseInt(notificationIDStr, 10, 64)
	if err != nil {
		http.Error(w, "ID de notification invalide", http.StatusBadRequest)
		return
	}

	// Vérifier que l'utilisateur est autorisé à supprimer cette notification
	activity, err := h.ActivityStore.GetByID(notificationID)
	if err != nil || activity.RecipientID != userID {
		http.Error(w, "Notification non trouvée ou accès non autorisé", http.StatusNotFound)
		return
	}

	// Supprimer la notification
	if err := h.ActivityStore.Delete(notificationID); err != nil {
		log.Printf("Erreur lors de la suppression de la notification: %v", err)
		http.Error(w, "Erreur lors de la suppression", http.StatusInternalServerError)
		return
	}

	// Rediriger vers la page des notifications
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}

// MarkAllNotificationsAsRead marque toutes les notifications comme lues
func (h *NotificationHandler) MarkAllNotificationsAsRead(w http.ResponseWriter, r *http.Request) {
	// Récupérer l'ID de l'utilisateur connecté
	userID := GetUserIDFromRequest(r)
	if userID == 0 {
		http.Error(w, "Non authentifié", http.StatusUnauthorized)
		return
	}

	// Marquer toutes les notifications comme lues
	if err := h.ActivityStore.MarkNotificationsAsRead(userID); err != nil {
		log.Printf("Erreur lors de la mise à jour des notifications: %v", err)
		http.Error(w, "Erreur serveur", http.StatusInternalServerError)
		return
	}

	// Rediriger vers la page des notifications
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}

// register les routes pour les notifs
func RegisterNotificationRoutes(r *mux.Router, h *NotificationHandler) {
	r.HandleFunc("/notifications", h.ShowNotifications).Methods("GET")
	r.HandleFunc("/api/notifications/count", h.GetUnreadNotificationsCount).Methods("GET")
	r.HandleFunc("/notifications/{id}/delete", h.DeleteNotification).Methods("POST")
	r.HandleFunc("/notifications/mark-read", h.MarkAllNotificationsAsRead).Methods("POST")
}
