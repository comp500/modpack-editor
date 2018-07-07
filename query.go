package main

import (
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

// AddonData is deserialised JSON from the curse.nikky.moe API
type AddonData struct {
	ID      int    `json:"id"`
	Name    string `json:"name"`
	Authors []struct {
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"authors"`
	Attachments []struct {
		ID           int    `json:"id"`
		ProjectID    int    `json:"projectId"`
		Description  string `json:"description"`
		ThumbnailURL string `json:"thumbnailUrl"`
		Title        string `json:"title"`
		URL          string `json:"url"`
		Default      bool   `json:"default"`
	} `json:"attachments"`
	WebsiteURL    string `json:"webSiteURL"`
	GameID        int    `json:"gameId"`
	Summary       string `json:"summary"`
	DefaultFileID int    `json:"defaultFileId"`
	CommentCount  int    `json:"commentCount"`
	// Should be float format because for some reason the field is given using e
	DownloadCount float64 `json:"downloadCount"`
	Rating        int     `json:"rating"`
	InstallCount  int     `json:"installCount"`
	LatestFiles   []struct {
		ID              int    `json:"id"`
		FileName        string `json:"fileName"`
		FileNameOnDisk  string `json:"fileNameOnDisk"`
		FileDate        int64  `json:"fileDate"`
		ReleaseType     string `json:"releaseType"`
		FileStatus      string `json:"fileStatus"`
		DownloadURL     string `json:"downloadURL"`
		AlternateFileID int    `json:"alternateFileId"`
		Dependencies    []struct {
			AddonID int    `json:"addOnId"`
			Type    string `json:"type"`
		} `json:"dependencies"`
		Modules []struct {
			Fingerprint int64  `json:"fingerprint"`
			Foldername  string `json:"foldername"`
		} `json:"modules"`
		PackageFingerprint int64    `json:"packageFingerprint"`
		GameVersion        []string `json:"gameVersion"`
		Alternate          bool     `json:"alternate"`
		Available          bool     `json:"available"`
	} `json:"latestFiles"`
	Categories []struct {
		ID   int    `json:"id"`
		Name string `json:"name"`
		URL  string `json:"url"`
	} `json:"categories"`
	PrimaryAuthorName        string `json:"primaryAuthorName"`
	ExternalURL              string `json:"externalUrl"`
	Status                   string `json:"status"`
	Stage                    string `json:"stage"`
	DonationURL              string `json:"donationUrl"`
	PrimaryCategoryName      string `json:"primaryCategoryName"`
	PrimaryCategoryAvatarURL string `json:"primaryCategoryAvatarUrl"`
	Likes                    int    `json:"likes"`
	CategorySection          struct {
		ID                      int    `json:"id"`
		GameID                  int    `json:"gameID"`
		Name                    string `json:"name"`
		PackageType             string `json:"packageType"`
		Path                    string `json:"path"`
		InitialInclusionPattern string `json:"initialInclusionPattern"`
		ExtraIncludePattern     string `json:"extraIncludePattern"`
	} `json:"categorySection"`
	PackageType            string `json:"packageType"`
	AvatarURL              string `json:"avatarUrl"`
	Slug                   string `json:"slug"`
	ClientURL              string `json:"clientUrl"`
	GameVersionLatestFiles []struct {
		GameVersion     string `json:"gameVersion"`
		ProjectFileID   int    `json:"projectFileID"`
		ProjectFileName string `json:"projectFileName"`
		FileType        string `json:"fileType"`
	} `json:"gameVersionLatestFiles"`
	PopularityScore    float64 `json:"popularityScore"`
	GamePopularityRank int     `json:"gamePopularityRank"`
	FullDescription    string  `json:"fullDescription"`
	GameName           string  `json:"gameName"`
	PortalName         string  `json:"portalName"`
	SectionName        string  `json:"sectionName"`
	DateModified       int64   `json:"dateModified"`
	DateCreated        int64   `json:"dateCreated"`
	DateReleased       int64   `json:"dateReleased"`
	CategoryList       string  `json:"categoryList"`
	Available          bool    `json:"available"`

	LastQueried time.Time
}

func requestAddonData(addonID int) (AddonData, error) {
	// Use a cached mod, if it's available and up to date
	cachedModsMutex.RLock()
	if cachedMods[addonID].Available {
		if time.Since(cachedMods[addonID].LastQueried) < 48*time.Hour {
			defer cachedModsMutex.RUnlock()
			return cachedMods[addonID], nil
		}
	}
	cachedModsMutex.RUnlock()

	// Uses the curse.nikky.moe api
	var data AddonData
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://curse.nikky.moe/api/addon/%d", addonID), nil)
	if err != nil {
		return data, err
	}

	req.Header.Set("User-Agent", "comp500/modpack-editor client")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return data, err
	}

	err = json.NewDecoder(resp.Body).Decode(&data)
	if err != nil && err != io.EOF {
		fmt.Println(err)
		return data, err
	}

	// Add to cache
	data.LastQueried = time.Now()
	cachedModsMutex.Lock()
	cachedMods[addonID] = data
	cachedModsMutex.Unlock()
	return data, nil
}

// ModpackEditorCache is saved and loaded from disk
type ModpackEditorCache struct {
	CachedMods        map[int]AddonData
	LastOpenedModpack string
}

func loadEditorCache() {
	file, err := os.Open("modpackEditorCache.bin")
	if err == nil {
		defer file.Close()
		var modpackEditorCache ModpackEditorCache
		zr, err := gzip.NewReader(file)
		if err != nil {
			log.Print("Error loading from cache:")
			log.Print(err)
			cachedMods = make(map[int]AddonData)
			return
		}
		err = gob.NewDecoder(zr).Decode(&modpackEditorCache)
		if err != nil && err != io.EOF {
			log.Print("Error loading from cache:")
			log.Print(err)
			cachedMods = make(map[int]AddonData)
			return
		}

		cachedMods = modpackEditorCache.CachedMods
		if len(modpackEditorCache.LastOpenedModpack) > 0 && modpack.Folder == "" {
			folderAbsolute, err := filepath.Abs(modpackEditorCache.LastOpenedModpack)
			if err != nil {
				log.Print("Error loading modpack from cached folder:")
				log.Print(err)
				return
			}

			modpack = Modpack{Folder: folderAbsolute}
			err = modpack.loadConfigFiles()
			if err != nil {
				log.Print("Error loading modpack from cached folder:")
				log.Print(err)
			}
		}
	} else if os.IsNotExist(err) {
		cachedMods = make(map[int]AddonData)
	} else {
		log.Print("Error loading from cache:")
		log.Print(err)
	}
}

func writeEditorCache() {
	cachedModsMutex.RLock()
	defer cachedModsMutex.RUnlock()

	file, err := os.Create("modpackEditorCache.bin")
	if err != nil {
		log.Print("Error writing to cache:")
		log.Print(err)
		return
	}
	defer file.Close()

	modpackEditorCache := ModpackEditorCache{
		CachedMods:        cachedMods,
		LastOpenedModpack: modpack.Folder,
	}
	zw := gzip.NewWriter(file)
	defer zw.Close()
	err = gob.NewEncoder(zw).Encode(&modpackEditorCache)
	if err != nil {
		log.Print("Error writing to cache:")
		log.Print(err)
	}
}
