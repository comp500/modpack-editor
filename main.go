package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"

	"github.com/gobuffalo/packr"
)

const port = 8080

var staticFilesBox packr.Box
var blankPackBox packr.Box
var modpack Modpack

type postRequestData struct {
	Folder string
}

func ajaxHandler(w http.ResponseWriter, r *http.Request) {
	var data postRequestData
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil && err != io.EOF {
		writeError(w, err)
		return
	}

	switch r.URL.Path {
	case "/ajax/getCurrentPackDetails":
		getCurrentPackDetails(w)
	case "/ajax/loadModpackFolder":
		loadModpackFolder(w, data.Folder)
	case "/ajax/createModpackFolder":
		createModpackFolder(w, data.Folder)
	default:
		w.WriteHeader(404)
	}
}

func main() {
	port := flag.Int("port", 8080, "The port that the HTTP server listens on")
	ip := flag.String("ip", "127.0.0.1", "The ip that the HTTP server listens on")
	flag.Parse()

	staticFilesBox = packr.NewBox("./static")
	blankPackBox = packr.NewBox("./blankPack")

	fmt.Println("Welcome to modpack-editor!")
	fmt.Printf("Listening on port %d, accessible at http://127.0.0.1:%d/\n", *port, *port)
	fmt.Println("Press CTRL+C to exit.")

	http.Handle("/", http.FileServer(staticFilesBox))
	http.HandleFunc("/ajax/", ajaxHandler)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", *ip, *port), nil)
	if err != nil {
		log.Println("Error starting server:")
		log.Fatal(err)
	}
}

func writeError(w http.ResponseWriter, e error) {
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(struct {
		ErrorMessage string
	}{e.Error()})
}
