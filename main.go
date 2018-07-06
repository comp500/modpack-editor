package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"path/filepath"

	"github.com/gobuffalo/packr"
)

const port = 8080

var box packr.Box
var modpack Modpack

func staticHandler(w http.ResponseWriter, r *http.Request) {
	path := r.URL.Path
	if path == "/" {
		path = "/index.html"
	}

	io.WriteString(w, box.String(path))
}

func ajaxHandler(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		writeError(w, err)
	}

	switch r.URL.Path {
	case "/ajax/getCurrentPackDetails":
		getCurrentPackDetails(w)
	case "/ajax/loadModpackFolder":
		folder := r.PostFormValue("folder")
		loadModpackFolder(w, folder)
	default:
		w.WriteHeader(404)
	}
}

func main() {
	port := flag.Int("port", 8080, "The port that the HTTP server listens on")
	ip := flag.String("ip", "127.0.0.1", "The ip that the HTTP server listens on")
	flag.Parse()

	box = packr.NewBox("./static")

	fmt.Println("Welcome to modpack-editor!")
	fmt.Printf("Listening on port %d, accessible at http://127.0.0.1:%d/\n", *port, *port)
	fmt.Println("Press CTRL+C to exit.")

	http.HandleFunc("/", staticHandler)
	http.HandleFunc("/ajax/", ajaxHandler)
	err := http.ListenAndServe(fmt.Sprintf("%s:%d", *ip, *port), nil)
	if err != nil {
		log.Println("Error starting server:")
		log.Fatal(err)
	}
}

// Modpack is a modpack being edited by modpack-editor
type Modpack struct {
	Folder string
}

func (m *Modpack) loadConfigFiles() error {
	manifest, err := ioutil.ReadFile(m.Folder + "manifest.json")
	if err != nil {
		return err
	}
	log.Print(manifest)
	return nil
}

func loadModpackFolder(w http.ResponseWriter, folder string) {
	folderAbsolute, err := filepath.Abs(folder)
	if err != nil {
		writeError(w, err)
		return
	}

	modpack = Modpack{folderAbsolute}
	err = modpack.loadConfigFiles()
	if err != nil {
		writeError(w, err)
		return
	}
}

func getCurrentPackDetails(w io.Writer) {
	if modpack.Folder == "" { // Empty modpack
		json.NewEncoder(w).Encode(struct {
			Modpack []byte `json:"modpack"`
		}{nil})
	}
}

func writeError(w http.ResponseWriter, e error) {
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(struct {
		ErrorMessage string `json:"error"`
	}{e.Error()})
}
