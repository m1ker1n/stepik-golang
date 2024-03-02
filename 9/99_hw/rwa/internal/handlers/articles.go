package handlers

import (
	"net/http"
)

type Articles struct {
}

func NewArticles() *Articles {
	//TODO add dependencies
	return &Articles{}
}

func (a *Articles) GetArticles(w http.ResponseWriter, r *http.Request) {
	//TODO implement me
	panic("implement me")
}

func (a *Articles) CreateArticle(w http.ResponseWriter, r *http.Request) {
	//TODO implement me
	panic("implement me")
}
