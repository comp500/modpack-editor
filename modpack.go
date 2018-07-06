package main

import (
	"encoding/json"
	"io/ioutil"
	"path/filepath"
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
