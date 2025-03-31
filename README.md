# StudHelp

# Forum d'Entraide Étudiante

Un forum d'entraide destiné aux étudiants dans l'enseignement supérieur, permettant de partager des questions, des conseils et des ressources.

## Fonctionnalités

### Utilisateurs

* Inscription et connexion
* Création et gestion de profil
* Téléchargement d'avatar
* Suivi d'activité

### Publications

* Création, modification et suppression de posts
* Support d'images dans les posts
* Système de tags pour catégoriser les posts
* Système de likes/dislikes

### Commentaires

* Ajout de commentaires sur les posts
* Possibilité de liker/disliker les commentaires

### Recherche et Filtrage

* Barre de recherche par mots-clés
* Filtrage par tags
* Tri des résultats (date, likes, dislikes)
* Tags populaires en évidence

### Notifications

* Système de notifications pour les interactions (likes, commentaires)
* Interface de gestion des notifications

## Structure du Projet

```
├── data                # Base de données
├── database            # Configuration et schéma SQL
├── handlers            # Gestionnaires de requêtes HTTP
├── models              # Modèles de données
├── static              # Ressources statiques
│   ├── assets          # Images et icônes
│   ├── avatars         # Photos de profil utilisateurs
│   ├── css             # Styles CSS
│   └── uploads         # Images téléchargées par les utilisateurs
└── templates           # Templates HTML
```

## Technologies Utilisées

* **Backend**: Go (Golang)
* **Base de données**: SQLite
* **Frontend**: HTML, CSS, JavaScript
* **Templating**: Go templates
* **Routeur**: Gorilla Mux

## Installation et Démarrage

### Prérequis

* Go 1.17+
* SQLite

### Installation

1. Cloner le dépôt

```bash
git clone https://github.com/votre-nom/forum-entraide-etudiante.git
cd forum-entraide-etudiante
```

2. Installer les dépendances

```bash
go mod download
```

3. Démarrer le serveur

```bash
go run main.go
```

Le forum sera accessible à l'adresse `http://localhost:8080`

### Utilisation avec Docker

```bash
docker-compose up -d
```

## Fonctionnalités Principales en Détail

### Système de Posts

* Les utilisateurs peuvent créer des posts avec titre, contenu et images
* Possibilité d'ajouter des tags pour catégoriser les posts
* Interface intuitive pour la création et l'édition

### Système de Recherche

* Recherche textuelle dans les titres et contenus des posts
* Filtrage avancé par tags
* Options de tri multiples (date, popularité, etc.)

### Interface Utilisateur

* Design responsive adapté à tous les appareils
* Mode sombre automatique basé sur les préférences système
* Animations et transitions pour une expérience fluide

## Captures d'écran

*Captures d'écran à ajouter*

## Auteur

*Votre nom* - *Votre école*

## Licence

Ce projet est sous licence MIT - voir le fichier LICENSE pour plus de détails.

Forum d'entraide etudiante (fubu)
