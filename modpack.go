package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gobuffalo/packr"
)

// Modpack is a modpack being edited by modpack-editor
type Modpack struct {
	Folder        string
	CurseManifest CurseManifest
}

// CurseManifest is a curse manifest.json file
type CurseManifest struct {
	Minecraft struct {
		Version    string `json:"version"`
		ModLoaders []struct {
			ID      string `json:"id"`
			Primary bool   `json:"primary"`
		} `json:"modLoaders"`
	} `json:"minecraft"`
	ManifestType    string `json:"manifestType"`
	ManifestVersion int    `json:"manifestVersion"`
	Name            string `json:"name"`
	Version         string `json:"version"`
	Author          string `json:"author"`
	ProjectID       int    `json:"projectID"`
	Files           []struct {
		ProjectID int  `json:"projectID"`
		FileID    int  `json:"fileID"`
		Required  bool `json:"required"`
	} `json:"files"`
	Overrides string `json:"overrides"`
}

func (m *Modpack) loadConfigFiles() error {
	manifest, err := ioutil.ReadFile(filepath.Join(m.Folder, "manifest.json"))
	if err != nil {
		return err
	}
	err = json.Unmarshal(manifest, &m.CurseManifest)
	if err != nil {
		return err
	}
	return nil
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
