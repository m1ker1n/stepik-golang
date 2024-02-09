package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/vk"
)

const (
	APP_ID     = "7065390"              // вам надо заменить это значение на свое на https://vk.com/apps?act=manage https://dev.vk.com/ru/api/getting-started
	APP_KEY    = "cQZe3Vvo4mHotmetUdXK" // вам надо заменить это значение на свое на https://vk.com/apps?act=manage https://dev.vk.com/ru/api/getting-started
	APP_SECRET = "1bbf49951bbf49951bbf49953b1bd486bb11bbf1bbf4995468b3d76e2cb2114610654e0"
	API_URL    = "https://api.vk.com/method/users.get?fields=email,photo_50&access_token=%s&v=5.131"

	AUTH_URL = "https://oauth.vk.com/authorize?client_id=7065390&redirect_uri=http://localhost:8080/&response_type=code&scope=email"
)

type Response struct {
	Response []struct {
		FirstName string `json:"first_name"`
		Photo     string `json:"photo_50"`
	}
}

// https://oauth.vk.com/authorize?client_id=7065390&redirect_uri=http://localhost:8080/&response_type=code&scope=email

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		code := r.FormValue("code")

		if code == "" {
			w.Write([]byte(`<div><a href="` + AUTH_URL + `">authorize</a></div>`))
			return
		}

		conf := oauth2.Config{
			ClientID:     APP_ID,
			ClientSecret: APP_KEY,
			RedirectURL:  "http://localhost:8080/",
			Endpoint:     vk.Endpoint,
		}

		token, err := conf.Exchange(ctx, code)
		if err != nil {
			log.Println("cannot exchange", err)
			http.Error(w, err.Error(), 500)
			return
		}

		email := token.Extra("email").(string)
		userIDraw := token.Extra("user_id").(float64)
		userID := int(userIDraw)

		w.Write([]byte(`
		<div> Oauth token:<br>
			` + fmt.Sprintf("%#v", token) + `
		</div>
		<div>Email: ` + email + `</div>
		<div>UserID: ` + strconv.Itoa(userID) + `</div>
		<br>
		`))

		client := conf.Client(ctx, token)
		resp, err := client.Get(fmt.Sprintf(API_URL, token.AccessToken))
		if err != nil {
			log.Println("cannot request data", err)
			http.Error(w, err.Error(), 500)
			return
		}

		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			log.Println("cannot read buffer", err)
			http.Error(w, err.Error(), 500)
			return
		}

		data := &Response{}
		err = json.Unmarshal(body, data)
		if err != nil {
			log.Println("cannot json.Unmarshal", err)
			http.Error(w, err.Error(), 500)
			return
		}

		w.Write([]byte(`
		<div>
			<img src="` + data.Response[0].Photo + `"/>
			` + data.Response[0].FirstName + `
		</div>
		`))
	})

	log.Println("starting server at :8080")
	http.ListenAndServe(":8080", nil)
}
