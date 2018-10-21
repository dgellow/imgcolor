package main

import (
	"bytes"
	"fmt"
	"html/template"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"net/http"
	"os"
	"strconv"
)

const flashSession = "flash-session"

var indexPage *template.Template

// RGB ...
type RGB struct {
	R int `json:"r"`
	G int `json:"g"`
	B int `json:"b"`
}

// Result ...
type Result struct {
	RGB RGB    `json:"rgb"`
	Hex string `json:"hex"`
}

func main() {
	indexPage = template.Must(template.New("index.html.tmpl").Funcs(template.FuncMap{
		"n": func(n int) []struct{} {
			return make([]struct{}, n)
		},
		"inc": func(n int) int {
			return n + 1
		},
	}).ParseFiles("index.html.tmpl"))

	http.HandleFunc("/upload", uploadHandler)
	http.HandleFunc("/", indexHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
		log.Printf("Default to port %s", port)
	}

	log.Printf("Listening on port %s", port)
	if err := http.ListenAndServe(fmt.Sprintf(":%s", port), nil); err != nil {
		log.Fatal("error: failed to listen or serve:", err)
	}
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var data struct {
		Flash *flashMessage
	}

	flashMsg, err := flash(w, r, flashSession)
	switch err {
	case nil:
		data.Flash = &flashMsg
	case http.ErrNoCookie:
	default:
		log.Print("error: failed to read flash cookie:", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var b bytes.Buffer
	if err := indexPage.Execute(&b, &data); err != nil {
		log.Print("error: failed to execute template 'indexPage':", err)
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Write(b.Bytes())
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	maxResults, err := strconv.Atoi(r.FormValue("max-results"))
	if err != nil {
		log.Println("failed to parse max-results as int: error:", err)
		if err := writeFlash(w, flashSession, flashMessage{Error: "invalid setting max-results"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
		return
	}

	file, _, err := r.FormFile("file")
	if err == http.ErrMissingFile {
		log.Println("no file uploaded")
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
		return
	}
	if err != nil {
		log.Println("uploaded file: error:", err)
		if err := writeFlash(w, flashSession, flashMessage{Error: "invalid file"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.Header().Set("Location", "/")
		w.WriteHeader(http.StatusFound)
		return
	}
	defer file.Close()

	img, _, err := image.Decode(file)
	if err != nil {
		log.Println("cannot decode image: error:", err)
		if err := writeFlash(w, flashSession, flashMessage{Error: "invalid file"}); err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	res := mainColors(img, maxResults)
	if err := writeFlash(w, flashSession, flashMessage{Results: res}); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusFound)
}

func mainColors(img image.Image, maxResults int) []Result {
	q := newQuantizer(img, 14, 255.0)
	q.Quantize()
	freqs := q.MostFrequent(maxResults)
	res := make([]Result, len(freqs))
	for i := range freqs {
		res[i] = Result{
			RGB: RGB{
				R: freqs[i].r,
				G: freqs[i].g,
				B: freqs[i].b,
			},
		}
	}
	return res
}
