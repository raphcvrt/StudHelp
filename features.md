# Forum Etudiant en Go

## Description

Ce projet est un forum en ligne développé en Go avec des templates HTML. Il permet aux étudiants de s'entraider et d'interragir en postant et en commentant des sujets variés. Le forum offre des fonctionnalités avancées de gestion de contenu et d'interactions sociales.

## Fonctionnalités

### Lorsque déconnecté

- Voir les posts et les commentaires.
- Consulter les profils des utilisateurs.
- Effectuer une recherche avancée avec :
  - Filtrage par tags.
  - Tri par nombre de likes, dislikes ou ordre chronologique.
- Créer un compte avec :
  - Critères spécifiques pour le mot de passe.
  - Option d'ajout d'une photo de profil (facultatif).

### Lorsque connecté

#### Gestion des posts

- Créer un post avec :
  - Insertion d'une image.
  - Sélection d'un tag existant.
  - Ajout d'un nouveau tag si nécessaire.
- Modifier ou supprimer ses propres posts.
- pouvoir liker/disliker/commenter les postes et commentaires

#### Profil utilisateur

- Accéder à sa page de profil.
- Modifier les informations du profil.
- Consulter la liste de ses posts et de ses likes.
- Voir ses statistiques de profil.
- Publier un post directement si aucun n'est encore présent.

#### Notifications

- Consulter ses notifications.
- Les notifications sont automatiquement marquées comme lues lors de l'actualisation de la page. mais il y a aussi un bouton qui le fait

## Technologies utilisées

- **Backend** : Go
- **Frontend** : Templates HTML
- **Base de données** : SQLite
- **Sécurité** : HTTPS, hashage des mots de passe
- **Gestion des images** : Stockage et affichage sécurisé
