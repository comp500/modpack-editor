package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	if cachedMods[addonID].Available {
		if time.Since(cachedMods[addonID].LastQueried) < 48*time.Hour {
			return cachedMods[addonID], nil
		}
	}

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
	cachedMods[addonID] = data
	return data, nil
}
