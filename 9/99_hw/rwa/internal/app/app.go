package app

import (
	"github.com/gorilla/mux"
	"hw9/internal/handlers"
	"hw9/internal/repositories"
	"hw9/internal/services"
	"net/http"
)

type UserHandler interface {
	Login(http.ResponseWriter, *http.Request)
	Logout(http.ResponseWriter, *http.Request)
	Register(http.ResponseWriter, *http.Request)
	GetCurrentUser(http.ResponseWriter, *http.Request)
	UpdateCurrentUser(http.ResponseWriter, *http.Request)
}

type ArticlesHandler interface {
	// GetArticles responds with most recent articles globally.
	// GetArticles can be filtered via query parameters.
	GetArticles(http.ResponseWriter, *http.Request)
	CreateArticle(http.ResponseWriter, *http.Request)
}

func GetApp() http.Handler {
	r := mux.NewRouter()

	userRepo := repositories.NewUserMap(10)
	userService := services.NewUser(userRepo)
	userHandler := handlers.NewUser(userService)

	articlesHandler := handlers.NewArticles()

	registerRoutes(r, userHandler, articlesHandler)
	return r
}

func registerRoutes(r *mux.Router, user UserHandler, articles ArticlesHandler) {
	apiRouter := r.PathPrefix("/api").Subrouter()

	apiRouter.HandleFunc("/users", user.Register).Methods("POST")
	apiRouter.HandleFunc("/users/login", user.Login).Methods("POST")
	apiRouter.HandleFunc("/users/logout", user.Logout).Methods("POST")
	apiRouter.HandleFunc("/user", user.GetCurrentUser).Methods("GET")
	apiRouter.HandleFunc("/user", user.UpdateCurrentUser).Methods("PUT")

	apiRouter.HandleFunc("/articles", articles.CreateArticle).Methods("POST")
	apiRouter.HandleFunc("/articles", articles.GetArticles).Methods("GET")
}
