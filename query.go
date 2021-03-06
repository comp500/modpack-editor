package main

import (
	"bytes"
	"compress/gzip"
	"encoding/gob"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sync"
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
	DownloadCount float64    `json:"downloadCount"`
	Rating        int        `json:"rating"`
	InstallCount  int        `json:"installCount"`
	LatestFiles   []FileData `json:"latestFiles"`
	Categories    []struct {
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

// FileData is deserialised JSON from the curse.nikky.moe API
// Can be provided in a seperate request or in LatestFiles
type FileData struct {
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
}

func requestAddonData(addonID int) (AddonData, error) {
	// Use a cached mod, if it's available and up to date
	mainCache.cachedModsMutex.RLock()
	if mainCache.CachedMods[addonID].Available {
		if time.Since(mainCache.CachedMods[addonID].LastQueried) < 48*time.Hour {
			defer mainCache.cachedModsMutex.RUnlock()
			return mainCache.CachedMods[addonID], nil
		}
	}
	mainCache.cachedModsMutex.RUnlock()

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
		return data, err
	}

	// Add files to file cache
	mainCache.cachedFilesMutex.Lock()
	for _, v := range data.LatestFiles {
		if _, ok := mainCache.CachedFiles[v.ID]; !ok {
			mainCache.CachedFiles[v.ID] = v
		}
	}
	mainCache.cachedFilesMutex.Unlock()

	// Add to cache
	data.LastQueried = time.Now()
	mainCache.cachedModsMutex.Lock()
	mainCache.CachedMods[addonID] = data
	mainCache.cachedModsMutex.Unlock()
	return data, nil
}

func requestFileData(addonID, fileID int) (FileData, error) {
	// Use a cached file, if it's available and up to date
	mainCache.cachedFilesMutex.RLock()
	if mainCache.CachedFiles[fileID].Available {
		defer mainCache.cachedFilesMutex.RUnlock()
		return mainCache.CachedFiles[fileID], nil
	}
	mainCache.cachedFilesMutex.RUnlock()

	// Uses the curse.nikky.moe api
	var data FileData
	client := &http.Client{}

	req, err := http.NewRequest("GET", fmt.Sprintf("https://curse.nikky.moe/api/addon/%d/file/%d", addonID, fileID), nil)
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
		return data, err
	}

	// Add to cache
	mainCache.cachedFilesMutex.Lock()
	mainCache.CachedFiles[fileID] = data
	mainCache.cachedFilesMutex.Unlock()
	return data, nil
}

// AddonSlugRequest is sent to the CurseProxy GraphQL api to get the id from a slug
type AddonSlugRequest struct {
	Query     string `json:"query"`
	Variables struct {
		Slug string `json:"slug"`
	} `json:"variables"`
}

// AddonSlugResponse is received from the CurseProxy GraphQL api to get the id from a slug
type AddonSlugResponse struct {
	Data struct {
		Addons []struct {
			ID int `json:"id"`
		} `json:"addons"`
	} `json:"data"`
	Exception  string   `json:"exception"`
	Message    string   `json:"message"`
	Stacktrace []string `json:"stacktrace"`
}

func requestAddonDataFromSlug(slug string) (AddonData, error) {
	// Use a cached slug id, if it exists
	mainCache.cachedSlugIDsMutex.RLock()
	if id, ok := mainCache.CachedSlugIDs[slug]; ok {
		mainCache.cachedSlugIDsMutex.RUnlock()
		return requestAddonData(id)
	}
	mainCache.cachedSlugIDsMutex.RUnlock()

	var data AddonData

	request := AddonSlugRequest{
		Query: `
		query getIDFromSlug($slug: String) {
			{
				addons(slug: $slug) {
					id
				}
			}
		}
		`,
	}
	request.Variables.Slug = slug

	// Uses the curse.nikky.moe api
	var response AddonSlugResponse
	client := &http.Client{}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return data, err
	}

	req, err := http.NewRequest("POST", "https://curse.nikky.moe/graphql", bytes.NewBuffer(requestBytes))
	if err != nil {
		return data, err
	}

	req.Header.Set("User-Agent", "comp500/modpack-editor client")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return data, err
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil && err != io.EOF {
		return data, err
	}

	if len(response.Exception) > 0 || len(response.Message) > 0 {
		return data, fmt.Errorf("Error requesting id for slug: %s", response.Message)
	}

	if len(response.Data.Addons) < 1 {
		return data, errors.New("Addon not found")
	}

	data, err = requestAddonData(response.Data.Addons[0].ID)
	if err != nil {
		return data, err
	}
	// If the request succeeded, cache the ID
	mainCache.cachedSlugIDsMutex.Lock()
	mainCache.CachedSlugIDs[slug] = response.Data.Addons[0].ID
	mainCache.cachedSlugIDsMutex.Unlock()
	return data, err
}

func requestFileDataFromSlug(slug string, fileID int) (FileData, error) {
	// Use a cached slug id, if it exists
	mainCache.cachedSlugIDsMutex.RLock()
	if id, ok := mainCache.CachedSlugIDs[slug]; ok {
		mainCache.cachedSlugIDsMutex.RUnlock()
		return requestFileData(id, fileID)
	}
	mainCache.cachedSlugIDsMutex.RUnlock()

	var data FileData

	request := AddonSlugRequest{
		Query: `
		query getIDFromSlug($slug: String) {
			{
				addons(slug: $slug) {
					id
				}
			}
		}
		`,
	}
	request.Variables.Slug = slug

	// Uses the curse.nikky.moe api
	var response AddonSlugResponse
	client := &http.Client{}

	requestBytes, err := json.Marshal(request)
	if err != nil {
		return data, err
	}

	req, err := http.NewRequest("POST", "https://curse.nikky.moe/graphql", bytes.NewBuffer(requestBytes))
	if err != nil {
		return data, err
	}

	req.Header.Set("User-Agent", "comp500/modpack-editor client")
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return data, err
	}

	err = json.NewDecoder(resp.Body).Decode(&response)
	if err != nil && err != io.EOF {
		return data, err
	}

	if len(response.Exception) > 0 || len(response.Message) > 0 {
		return data, fmt.Errorf("Error requesting id for slug: %s", response.Message)
	}

	if len(response.Data.Addons) < 1 {
		return data, errors.New("Addon not found")
	}

	data, err = requestFileData(response.Data.Addons[0].ID, fileID)
	if err != nil {
		return data, err
	}
	// If the request succeeded, cache the ID
	mainCache.cachedSlugIDsMutex.Lock()
	mainCache.CachedSlugIDs[slug] = response.Data.Addons[0].ID
	mainCache.cachedSlugIDsMutex.Unlock()
	return data, err
}

var mainCache ModpackEditorCache

// ModpackEditorCache is saved and loaded from disk
type ModpackEditorCache struct {
	CachedMods         map[int]AddonData
	cachedModsMutex    sync.RWMutex
	CachedSlugIDs      map[string]int
	cachedSlugIDsMutex sync.RWMutex
	CachedFiles        map[int]FileData
	cachedFilesMutex   sync.RWMutex
	LastOpenedModpack  string
	CacheVersion       int
}

// NewModpackEditorCache initialises the maps in ModpackEditorCache
func NewModpackEditorCache() *ModpackEditorCache {
	cache := ModpackEditorCache{
		CachedMods:    make(map[int]AddonData),
		CachedSlugIDs: make(map[string]int),
		CachedFiles:   make(map[int]FileData),
		CacheVersion:  CurrentCacheVersion,
	}
	return &cache
}

// CurrentCacheVersion is the version of the editor cache file being used. Older caches are ignored.
const CurrentCacheVersion = 3

func loadEditorCache() *Modpack {
	if disableCacheStore {
		mainCache = *NewModpackEditorCache()
		return nil
	}

	file, err := os.Open("modpackEditorCache.bin")
	if err == nil {
		defer file.Close()
		var newModpackEditorCache ModpackEditorCache
		zr, err := gzip.NewReader(file)
		if err != nil {
			log.Print("Error loading from cache:")
			log.Print(err)
			mainCache = *NewModpackEditorCache()
			return nil
		}
		err = gob.NewDecoder(zr).Decode(&newModpackEditorCache)
		if err != nil && err != io.EOF {
			log.Print("Error loading from cache:")
			log.Print(err)
			mainCache = *NewModpackEditorCache()
			return nil
		}

		if newModpackEditorCache.CacheVersion < CurrentCacheVersion {
			log.Print("Cache is too old, discarding")
			mainCache = *NewModpackEditorCache()
			return nil
		}

		// Can't assign directly as it contains mutexes
		mainCache = ModpackEditorCache{
			CachedMods:        newModpackEditorCache.CachedMods,
			CachedSlugIDs:     newModpackEditorCache.CachedSlugIDs,
			CachedFiles:       newModpackEditorCache.CachedFiles,
			LastOpenedModpack: newModpackEditorCache.LastOpenedModpack,
			CacheVersion:      CurrentCacheVersion,
		}
		if newModpackEditorCache.CachedMods == nil {
			mainCache.CachedMods = make(map[int]AddonData)
		}
		if newModpackEditorCache.CachedSlugIDs == nil {
			mainCache.CachedSlugIDs = make(map[string]int)
		}
		if newModpackEditorCache.CachedFiles == nil {
			mainCache.CachedFiles = make(map[int]FileData)
		}

		// Load existing modpack
		if len(mainCache.LastOpenedModpack) > 0 {
			folderAbsolute, err := filepath.Abs(mainCache.LastOpenedModpack)
			if err != nil {
				log.Print("Error loading modpack from cached folder:")
				log.Print(err)
				return nil
			}

			newModpack := &Modpack{Folder: folderAbsolute}
			err = newModpack.loadConfigFiles()
			if err != nil {
				log.Print("Error loading modpack from cached folder:")
				log.Print(err)
				// Clear value
				newModpack = nil
			}
			// Update mod list
			newModpack.getModInfoList()

			return newModpack
		}
	} else if os.IsNotExist(err) {
		mainCache = *NewModpackEditorCache()
	} else {
		log.Print("Error loading from cache:")
		log.Print(err)
	}

	return nil
}

func writeEditorCache() {
	if disableCacheStore {
		return
	}

	// Ensure mutexes are correct for gob to encode the maps
	mainCache.cachedModsMutex.RLock()
	defer mainCache.cachedModsMutex.RUnlock()
	mainCache.cachedSlugIDsMutex.RLock()
	defer mainCache.cachedSlugIDsMutex.RUnlock()
	mainCache.cachedFilesMutex.RLock()
	defer mainCache.cachedFilesMutex.RUnlock()

	// Update lastOpenedModpack
	mainCache.LastOpenedModpack = modpack.Folder

	file, err := os.Create("modpackEditorCache.bin")
	if err != nil {
		log.Print("Error writing to cache:")
		log.Print(err)
		return
	}
	defer file.Close()

	zw := gzip.NewWriter(file)
	defer zw.Close()
	err = gob.NewEncoder(zw).Encode(&mainCache)
	if err != nil {
		log.Print("Error writing to cache:")
		log.Print(err)
	}
}
