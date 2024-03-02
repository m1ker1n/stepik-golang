package app

import (
	"github.com/gorilla/mux"
	"hw9/internal/handlers"
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
	userHandler := handlers.NewUser()
	articlesHandler := handlers.NewArticles()
	registerHandlers(r, userHandler, articlesHandler)
	return r
}

func registerHandlers(r *mux.Router, user UserHandler, articles ArticlesHandler) {
	r.HandleFunc("/users", user.Register).Methods("POST")
	r.HandleFunc("/users/login", user.Login).Methods("POST")
	r.HandleFunc("/users/logout", user.Logout).Methods("POST")
	r.HandleFunc("/user", user.GetCurrentUser).Methods("GET")
	r.HandleFunc("/user", user.UpdateCurrentUser).Methods("PUT")

	r.HandleFunc("/articles", articles.CreateArticle).Methods("POST")
	r.HandleFunc("/articles", articles.GetArticles).Methods("GET")
}
