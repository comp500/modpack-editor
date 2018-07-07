package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

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

// ModInfo is a partial mod information struct used for listing mods
type ModInfo struct {
	Name         string
	IconURL      string
	ErrorMessage error
	// TODO: clientonly/serveronly
	// TODO: Required?
	// TODO: dates, rating, download counts?
	// TODO: categories?
	// TODO: author(s)?
	Summary    string
	WebsiteURL string
}

func (m *Modpack) getModInfoList() (map[int]ModInfo, error) {
	info := make(map[int]ModInfo)
	var wg sync.WaitGroup
	// Mutex for the ModInfo map
	var mutex = &sync.RWMutex{}

	for _, v := range m.CurseManifest.Files {
		// Increment the WaitGroup counter.
		wg.Add(1)

		go func(projectID int) {
			// Decrement the counter when the goroutine completes.
			defer wg.Done()

			data, err := requestAddonData(projectID)
			if err != nil {
				mutex.Lock()
				info[projectID] = ModInfo{
					ErrorMessage: err,
				}
				mutex.Unlock()
			}

			var iconURL string
			// Loop through attachments, set iconURL to one which is set to true
			// Replace size in url (256/256) to 62/62
			for _, v := range data.Attachments {
				if !v.Default {
					continue
				}
				// TODO: move replacement to JS?
				iconURL = strings.Replace(v.ThumbnailURL, "256/256", "62/62", 1)
				// Curseforge does this for gifs for some reason...
				// TODO: Find out when it does this somehow.
				// Informational Accessories doesn't do this, but AE2 and Hwyla and Clumps do.
				iconURL = strings.Replace(iconURL, ".gif", "_animated.gif", 1)
			}

			mutex.Lock()
			info[projectID] = ModInfo{
				Name:       data.Name,
				IconURL:    iconURL,
				Summary:    data.Summary,
				WebsiteURL: data.WebsiteURL,
			}
			mutex.Unlock()
		}(v.ProjectID)
	}

	// Wait for all HTTP fetches to complete.
	wg.Wait()

	return info, nil
}

func handleGetModInfoList(w http.ResponseWriter) {
	if modpack.Folder == "" { // Empty modpack
		json.NewEncoder(w).Encode(make(map[int]ModInfo))
	} else {
		info, err := modpack.getModInfoList()
		if err != nil {
			writeError(w, err)
			return
		}

		// Send the mod info list to the client
		json.NewEncoder(w).Encode(&info)
	}
}
