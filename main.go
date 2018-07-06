package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"

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

func loadModpackFolder(w http.ResponseWriter, folder string) {
	folderAbsolute, err := filepath.Abs(folder)
	if err != nil {
		writeError(w, err)
		return
	}

	modpack = Modpack{Folder: folderAbsolute}
	err = modpack.loadConfigFiles()
	if err != nil {
		writeError(w, err)
		return
	}

	// Send the modpack to the client
	json.NewEncoder(w).Encode(struct {
		Modpack Modpack
	}{modpack})
}

func createModpackFolder(w http.ResponseWriter, folder string) {
	folderAbsolute, err := filepath.Abs(folder)
	if err != nil {
		writeError(w, err)
		return
	}

	// If pack exists, stop
	if stat, err := os.Stat(folderAbsolute); err == nil && stat.IsDir() {
		writeError(w, errors.New("Pack already exists"))
		return
	}

	// Make pack folder
	err = os.MkdirAll(folderAbsolute, os.ModePerm)
	if err != nil {
		writeError(w, err)
		return
	}

	// Copy all the files to the new folder
	err = blankPackBox.Walk(func(fileName string, file packr.File) error {
		out, err := os.Create(filepath.Join(folderAbsolute, fileName))
		if err != nil {
			return err
		}
		defer out.Close()

		_, err = io.Copy(out, file)
		if err != nil {
			return err
		}
		return out.Close()
	})
	if err != nil {
		writeError(w, err)
		return
	}

	modpack = Modpack{Folder: folderAbsolute}
	err = modpack.loadConfigFiles()
	if err != nil {
		writeError(w, err)
		return
	}

	// Send the modpack to the client
	json.NewEncoder(w).Encode(struct {
		Modpack Modpack
	}{modpack})
}

func getCurrentPackDetails(w io.Writer) {
	if modpack.Folder == "" { // Empty modpack
		json.NewEncoder(w).Encode(struct {
			Modpack []byte
		}{nil})
	} else {
		// Send the modpack to the client
		json.NewEncoder(w).Encode(struct {
			Modpack Modpack
		}{modpack})
	}
}

func writeError(w http.ResponseWriter, e error) {
	w.WriteHeader(400)
	json.NewEncoder(w).Encode(struct {
		ErrorMessage string
	}{e.Error()})
}
