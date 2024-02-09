package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	textTemplate "text/template"
)

type Storage interface {
	Add(*Photo) error
	GetPhotos(uint32) ([]*Photo, error)
	Rate(uint32, int) error
}

// -----------------------------

type PhotolistHandler struct {
	St   Storage
	Tmpl *textTemplate.Template
}

func (h *PhotolistHandler) List(w http.ResponseWriter, r *http.Request) {
	sess, _ := SessionFromContext(r.Context())
	items, err := h.St.GetPhotos(sess.UserID)
	if err != nil {
		log.Println("cant get items", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	err = h.Tmpl.Execute(w,
		struct {
			Items []*Photo
		}{
			items,
		})
	if err != nil {
		log.Println("cant execute template", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
}

func (h *PhotolistHandler) Upload(w http.ResponseWriter, r *http.Request) {
	uploadData, _, err := r.FormFile("my_file")
	if err != nil {
		log.Println("cant parse file", err)
		http.Error(w, "request error", http.StatusInternalServerError)
		return
	}
	defer uploadData.Close()

	comment := r.FormValue("comment")

	md5Sum, err := SaveFile(uploadData)
	if err != nil {
		log.Println("cant save file", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	realFile := "./images/" + md5Sum + ".jpg"
	err = MakeThumbnails(realFile, md5Sum)
	if err != nil {
		log.Println("cant resize file", err)
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	sess, _ := SessionFromContext(r.Context())
	err = h.St.Add(&Photo{
		UserID:  sess.UserID,
		Path:    md5Sum,
		Comment: comment,
	})
	if err != nil {
		log.Println("cant store item", err)
		http.Error(w, "storage error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/photos", 302)
}

func (h *PhotolistHandler) Rate(w http.ResponseWriter, r *http.Request) {
	// if r.Method != http.MethodPost {
	// 	// ....
	// }

	// if strings.Contains(r.Referer(), "vasya.ru") {
	// 	// ...
	// }

	w.Header().Add("Content-Type", "application/json")

	id, err := strconv.Atoi(r.FormValue("id"))
	if err != nil {
		http.Error(w, `{"err": "bad id"}`, http.StatusBadRequest)
		return
	}
	vote := r.FormValue("vote")
	rate := 0
	switch vote {
	case "up":
		rate = 1
	case "down":
		rate = -1
	default:
		http.Error(w, `{"err": "bad vote"}`, http.StatusBadRequest)
		return
	}

	err = h.St.Rate(uint32(id), rate)
	if err != nil {
		log.Println("rate err: ", err)
		http.Error(w, `{"err": "db err"}`, http.StatusBadRequest)
		return
	}

	result := map[string]interface{}{
		"id": id,
	}
	resp, _ := json.Marshal(result)
	w.Write(resp)
}
