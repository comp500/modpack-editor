package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"sync"

	"github.com/gobuffalo/packr"
)

var staticFilesBox packr.Box
var blankPackBox packr.Box
var modpack Modpack
var modpackMutex sync.RWMutex
var disableCacheStore bool

type postRequestData struct {
	Folder  string
	Modpack Modpack
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
	case "/ajax/saveModpack":
		saveModpack(w, data.Modpack)
	default:
		w.WriteHeader(404)
	}
}

func addonHandlerSlug(w http.ResponseWriter, r *http.Request) {
	// Get addon slug from /addonSlug/mod-name
	slug := r.URL.Path[7:]

	data, err := requestAddonDataFromSlug(slug)
	if err != nil {
		writeError(w, err)
		return
	}

	// Update cache
	// TODO: do this only on change
	writeEditorCache()

	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		// may have already written to output?
		writeError(w, err)
		return
	}
}

func addonHandlerID(w http.ResponseWriter, r *http.Request) {
	// Get addon id from /addon/12345
	addonID, err := strconv.Atoi(r.URL.Path[7:])
	if err != nil {
		writeError(w, err)
		return
	}

	data, err := requestAddonData(addonID)
	if err != nil {
		writeError(w, err)
		return
	}

	// Update cache
	// TODO: do this only on change
	writeEditorCache()

	err = json.NewEncoder(w).Encode(data)
	if err != nil {
		// may have already written to output?
		writeError(w, err)
		return
	}
}

func main() {
	port := flag.Int("port", 8080, "The port that the HTTP server listens on")
	ip := flag.String("ip", "127.0.0.1", "The ip that the HTTP server listens on")
	nocache := flag.Bool("nocache", false, "Don't store cached mod listings or modpack folders")
	flag.Parse()

	staticFilesBox = packr.NewBox("./static")
	blankPackBox = packr.NewBox("./blankPack")
	disableCacheStore = *nocache

	loadedModpack := loadEditorCache()
	if loadedModpack != nil {
		modpackMutex.Lock()
		modpack = *loadedModpack
		modpackMutex.Unlock()
	}

	fmt.Println("Welcome to modpack-editor!")
	fmt.Printf("Listening on port %d, accessible at http://%s:%d/\n", *port, *ip, *port)
	fmt.Println("Press CTRL+C to exit.")

	http.Handle("/", http.FileServer(staticFilesBox))
	http.HandleFunc("/ajax/", ajaxHandler)
	http.HandleFunc("/addon/", addonHandlerSlug)
	http.HandleFunc("/addonSlug/", addonHandlerID)
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
