package main

import (
	"encoding/base64"
	"encoding/json"
	"log"
	"net/http"
	"time"
)

type flashMessage struct {
	Error   string   `json:"error,omitempty"`
	Results []Result `json:"results,omitempty"`
}

func writeFlash(w http.ResponseWriter, name string, v interface{}) error {
	b, err := json.Marshal(v)
	if err != nil {
		log.Println("error: failed to write flash message:", err)
		return err
	}

	c := http.Cookie{Name: name, Value: base64.URLEncoding.EncodeToString(b)}
	http.SetCookie(w, &c)
	return nil
}

func flash(w http.ResponseWriter, r *http.Request, name string) (flashMessage, error) {
	c, err := r.Cookie(name)
	if err != nil {
		return flashMessage{}, err
	}

	b, err := base64.URLEncoding.DecodeString(c.Value)
	if err != nil {
		return flashMessage{}, err
	}

	expired := http.Cookie{Name: name, MaxAge: -1, Expires: time.Unix(1, 0)}
	http.SetCookie(w, &expired)

	var flashMsg flashMessage
	if err := json.Unmarshal(b, &flashMsg); err != nil {
		return flashMessage{}, err
	}
	return flashMsg, nil
}
