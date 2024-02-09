package main

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net/http"

	jwt "github.com/golang-jwt/jwt"

	"github.com/gorilla/mux"
)

/*

curl -X POST -H "Content-Type: application/json" -d '{"login": "rvasily", "password": "love"}' http://localhost:8080/login

curl -H "AccessToken: eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpZCI6NDUwLCJsb2dpbiI6InJ2YXNpbHkiLCJuYW1lIjoiVmFzaWx5IFJvbWFub3YiLCJyb2xlIjoidXNlciJ9.Y0FJFm8fSbjc4nzBa1LHJSxNRRYp-chOZLr26sOJSgo" http://localhost:8080/profile



*/

type User struct {
	ID       int
	FullName string
	Role     string
}

var (
	users = map[string]User{
		"rvasily":        {450, "Vasily Romanov", "user"},
		"romanov.vasily": {42, "Василий Романов", "admin"},
	}

	ExamplePassword    = "love"
	ExampleTokenSecret = []byte("супер секретный ключ")
)

func profilePage(w http.ResponseWriter, r *http.Request) {

	inToken := r.Header.Get("AccessToken")

	hashSecretGetter := func(token *jwt.Token) (interface{}, error) {
		method, ok := token.Method.(*jwt.SigningMethodHMAC)
		if !ok || method.Alg() != "HS256" {
			return nil, fmt.Errorf("bad sign method")
		}
		return ExampleTokenSecret, nil
	}
	token, err := jwt.Parse(inToken, hashSecretGetter)
	if err != nil || !token.Valid {
		jsonError(w, http.StatusUnauthorized, "bad token")
		return
	}

	payload, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		jsonError(w, http.StatusUnauthorized, "no payload")
	}

	resp, _ := json.Marshal(map[string]interface{}{
		"status": http.StatusOK,
		"data": map[string]interface{}{
			"login": payload["login"],
			"name":  payload["name"],
			"role":  payload["role"],
			"id":    payload["id"],
		},
	})
	w.Write(resp)
	w.Write([]byte("\n\n"))
}

type LoginForm struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

func loginPage(w http.ResponseWriter, r *http.Request) {

	if r.Header.Get("Content-Type") != "application/json" {
		jsonError(w, http.StatusBadRequest, "unknown payload")
		return
	}

	body, _ := ioutil.ReadAll(r.Body)
	r.Body.Close()

	fd := &LoginForm{}
	err := json.Unmarshal(body, fd)
	if err != nil {
		jsonError(w, http.StatusBadRequest, "cant unpack payload")
		return
	}

	user, exist := users[fd.Login]
	if !exist || fd.Password != ExamplePassword {
		jsonError(w, http.StatusUnauthorized, "bad login or password")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login": fd.Login,
		"name":  user.FullName,
		"role":  user.Role,
		"id":    user.ID,
	})
	tokenString, err := token.SignedString(ExampleTokenSecret)
	if err != nil {
		jsonError(w, http.StatusInternalServerError, err.Error())
		return
	}

	resp, _ := json.Marshal(map[string]interface{}{
		"status": http.StatusOK,
		"data": map[string]interface{}{
			"token": tokenString,
		},
	})
	w.Write(resp)
	w.Write([]byte("\n\n"))
}

func main() {
	rand.Seed(42)

	r := mux.NewRouter()
	r.HandleFunc("/login", loginPage).Methods("POST")
	r.HandleFunc("/profile", profilePage)

	fmt.Println("starting server at :8080")
	http.ListenAndServe(":8080", r)
}

func jsonError(w io.Writer, status int, msg string) {
	resp, _ := json.Marshal(map[string]interface{}{
		"status": status,
		"error":  msg,
	})
	w.Write(resp)
}
