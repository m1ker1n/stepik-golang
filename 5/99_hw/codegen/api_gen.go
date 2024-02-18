package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

var (
	errUnauthorized        = errors.New("unauthorized")
	errNotFound            = errors.New("unknown method")
	errStatusNotAcceptable = errors.New("bad method")
)

type httpResponse struct {
	Err      string `json:"error"`
	Response any    `json:"response,omitempty"`
}

func (r httpResponse) write(w http.ResponseWriter, status int) {
	marshal, _ := json.Marshal(r)
	w.WriteHeader(status)
	_, _ = w.Write(marshal)
}

func auth(r *http.Request) bool {
	return r.Header.Get("X-Auth") == "100500"
}

func (srv *MyApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		switch r.Method {
		case "POST":
			srv.wrapperCreate(w, r)
		default:
			httpResponse{Err: errStatusNotAcceptable.Error()}.write(w, http.StatusNotAcceptable)
		}
	case "/user/profile":
		switch r.Method {
		case "GET":
			srv.wrapperProfile(w, r)
		default:
			httpResponse{Err: errStatusNotAcceptable.Error()}.write(w, http.StatusNotAcceptable)
		}
	default:
		httpResponse{Err: errNotFound.Error()}.write(w, http.StatusNotFound)
	}
}

func (srv *MyApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {
	if authorized := auth(r); !authorized {
		httpResponse{Err: errUnauthorized.Error()}.write(w, http.StatusForbidden)
		return
	}

	in, err := getAndValidateCreateParams(r)
	if err != nil {
		httpResponse{Err: err.Error()}.write(w, http.StatusBadRequest)
		return
	}

	result, err := srv.Create(r.Context(), in)
	if err != nil {
		var ae ApiError
		if errors.As(err, &ae) {
			httpResponse{Err: ae.Error()}.write(w, ae.HTTPStatus)
			return
		}
		httpResponse{Err: err.Error()}.write(w, http.StatusInternalServerError)
		return
	}

	httpResponse{Response: result}.write(w, http.StatusOK)
}

func getAndValidateCreateParams(r *http.Request) (CreateParams, error) {
	if err := r.ParseForm(); err != nil {
		return CreateParams{}, err
	}

	login := r.Form.Get("login")
	if login == "" {
		return CreateParams{}, fmt.Errorf("login must me not empty")
	}
	if len(login) < 10 {
		return CreateParams{}, fmt.Errorf("login len must be >= 10")
	}
	name := r.Form.Get("full_name")
	status := r.Form.Get("status")
	if status == "" {
		status = "user"
	}
	switch status {
	case "user", "moderator", "admin":
	default:
		return CreateParams{}, fmt.Errorf("status must be one of [user, moderator, admin]")
	}
	ageRaw := r.Form.Get("age")
	// to set default int value without error
	if ageRaw == "" {
		ageRaw = "0"
	}
	age, err := strconv.Atoi(ageRaw)
	if err != nil {
		return CreateParams{}, fmt.Errorf("age must be int")
	}
	if age < 0 {
		return CreateParams{}, fmt.Errorf("age must be >= 0")
	}
	if age > 128 {
		return CreateParams{}, fmt.Errorf("age must be <= 128")
	}

	in := CreateParams{
		Login:  login,
		Name:   name,
		Status: status,
		Age:    age,
	}

	return in, nil
}

func (srv *MyApi) wrapperProfile(w http.ResponseWriter, r *http.Request) {
	in, err := getAndValidateProfileParams(r)
	if err != nil {
		httpResponse{Err: err.Error()}.write(w, http.StatusBadRequest)
		return
	}

	result, err := srv.Profile(r.Context(), in)
	if err != nil {
		var ae ApiError
		if errors.As(err, &ae) {
			httpResponse{Err: ae.Error()}.write(w, ae.HTTPStatus)
			return
		}
		httpResponse{Err: err.Error()}.write(w, http.StatusInternalServerError)
		return
	}

	httpResponse{Response: result}.write(w, http.StatusOK)
}

func getAndValidateProfileParams(r *http.Request) (ProfileParams, error) {
	if err := r.ParseForm(); err != nil {
		return ProfileParams{}, err
	}

	login := r.Form.Get("login")
	if login == "" {
		return ProfileParams{}, fmt.Errorf("login must me not empty")
	}

	in := ProfileParams{
		Login: login,
	}

	return in, nil
}

func (srv *OtherApi) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/user/create":
		switch r.Method {
		case "POST":
			srv.wrapperCreate(w, r)
		default:
			httpResponse{Err: errStatusNotAcceptable.Error()}.write(w, http.StatusNotAcceptable)
		}
	default:
		httpResponse{Err: errNotFound.Error()}.write(w, http.StatusNotFound)
	}
}

func (srv *OtherApi) wrapperCreate(w http.ResponseWriter, r *http.Request) {
	if authorized := auth(r); !authorized {
		httpResponse{Err: errUnauthorized.Error()}.write(w, http.StatusForbidden)
		return
	}

	in, err := getAndValidateOtherCreateParams(r)
	if err != nil {
		httpResponse{Err: err.Error()}.write(w, http.StatusBadRequest)
		return
	}

	result, err := srv.Create(r.Context(), in)
	if err != nil {
		var ae ApiError
		if errors.As(err, &ae) {
			httpResponse{Err: ae.Error()}.write(w, ae.HTTPStatus)
			return
		}
		httpResponse{Err: err.Error()}.write(w, http.StatusInternalServerError)
		return
	}

	httpResponse{Response: result}.write(w, http.StatusOK)
}

func getAndValidateOtherCreateParams(r *http.Request) (OtherCreateParams, error) {
	if err := r.ParseForm(); err != nil {
		return OtherCreateParams{}, err
	}

	username := r.Form.Get("username")
	if username == "" {
		return OtherCreateParams{}, fmt.Errorf("username must me not empty")
	}
	if len(username) < 3 {
		return OtherCreateParams{}, fmt.Errorf("username len must be >= 3")
	}
	name := r.Form.Get("account_name")
	class := r.Form.Get("class")
	if class == "" {
		class = "warrior"
	}
	switch class {
	case "warrior", "sorcerer", "rouge":
	default:
		return OtherCreateParams{}, fmt.Errorf("class must be one of [warrior, sorcerer, rouge]")
	}
	levelRaw := r.Form.Get("level")
	// to set default int value without error
	if levelRaw == "" {
		levelRaw = "0"
	}
	level, err := strconv.Atoi(levelRaw)
	if err != nil {
		return OtherCreateParams{}, fmt.Errorf("level must be int")
	}
	if level < 1 {
		return OtherCreateParams{}, fmt.Errorf("level must be >= 1")
	}
	if level > 50 {
		return OtherCreateParams{}, fmt.Errorf("level must be <= 50")
	}

	in := OtherCreateParams{
		Username: username,
		Name:     name,
		Class:    class,
		Level:    level,
	}

	return in, nil
}
