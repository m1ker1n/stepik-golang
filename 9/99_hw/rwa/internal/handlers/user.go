package handlers

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
)

type User struct {
	registrar UserRegistrar
}

func NewUser(registrar UserRegistrar) *User {
	//TODO add dependencies
	return &User{
		registrar: registrar,
	}
}

type Profile struct {
	ID        string `json:"id"`
	Email     string `json:"email"`
	CreatedAt string `json:"createdAt"`
	UpdatedAt string `json:"updatedAt,omitempty"`
	Username  string `json:"username"`
	Bio       string `json:"bio"`
	Image     string `json:"image"`
	Token     string `json:"token"`
	Following bool
}

// requestResponseUser is a type which wraps requests and response while (un-)marshalling.
type requestResponseUser[T any] struct {
	User T `json:"user"`
}

func (u *User) Login(w http.ResponseWriter, r *http.Request) {
	//TODO implement me
	panic("implement me")
}

func (u *User) Logout(w http.ResponseWriter, r *http.Request) {
	//TODO implement me
	panic("implement me")
}

type RegistrationArgs struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Username string `json:"username"`
}

type UserRegistrar interface {
	RegisterUser(ctx context.Context, args RegistrationArgs) (Profile, error)
}

func (u *User) Register(w http.ResponseWriter, r *http.Request) {
	rawBody, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "body is not read", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	var body requestResponseUser[RegistrationArgs]
	err = json.Unmarshal(rawBody, &body)
	if err != nil {
		http.Error(w, "could not unmarshal body", http.StatusInternalServerError)
		return
	}

	profile, err := u.registrar.RegisterUser(r.Context(), body.User)
	if err != nil {
		//TODO add handling error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	response := requestResponseUser[Profile]{profile}
	rawResponse, err := json.Marshal(response)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	w.Write(rawResponse)
}

func (u *User) GetCurrentUser(w http.ResponseWriter, r *http.Request) {
	//TODO implement me
	panic("implement me")
}

func (u *User) UpdateCurrentUser(w http.ResponseWriter, r *http.Request) {
	//TODO implement me
	panic("implement me")
}
