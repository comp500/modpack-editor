package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"

	"github.com/gobuffalo/packr"
	yaml "gopkg.in/yaml.v2"
)

// Modpack is a modpack being edited by modpack-editor
type Modpack struct {
	Folder            string
	CurseManifest     CurseManifest
	ServerSetupConfig ServerSetupConfig
	Mods              map[int]ModInfo
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

// ServerSetupConfig is a ServerStarter server-setup-config.yaml
type ServerSetupConfig struct {
	Specver int `yaml:"_specver"`
	Modpack struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	} `yaml:"modpack"`
	Install struct {
		McVersion         string `yaml:"mcVersion"`
		ForgeVersion      string `yaml:"forgeVersion"`
		ForgeInstallerURL string `yaml:"forgeInstallerUrl"`
		ModpackURL        string `yaml:"modpackUrl"`
		ModpackFormat     string `yaml:"modpackFormat"`
		FormatSpecific    struct {
			IgnoreProject []int `yaml:"ignoreProject"`
		} `yaml:"formatSpecific"`
		BaseInstallPath string   `yaml:"baseInstallPath"`
		IgnoreFiles     []string `yaml:"ignoreFiles"`
		AdditionalFiles []struct {
			URL         string `yaml:"url"`
			Destination string `yaml:"destination"`
		} `yaml:"additionalFiles"`
		LocalFiles []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		} `yaml:"localFiles"`
		CheckFolder        bool   `yaml:"checkFolder"`
		InstallForge       bool   `yaml:"installForge"`
		SpongeBootstrapper string `yaml:"spongeBootstrapper"`
	} `yaml:"install"`
	Launch struct {
		SpongeFix    bool     `yaml:"spongefix"`
		CheckOffline bool     `yaml:"checkOffline"`
		MaxRAM       string   `yaml:"maxRam"`
		AutoRestart  bool     `yaml:"autoRestart"`
		CrashLimit   int      `yaml:"crashLimit"`
		CrashTimer   string   `yaml:"crashTimer"`
		PreJavaArgs  string   `yaml:"preJavaArgs"`
		JavaArgs     []string `yaml:"javaArgs"`
	} `yaml:"launch"`
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

	config, err := ioutil.ReadFile(filepath.Join(m.Folder, "server-setup-config.yaml"))
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(config, &m.ServerSetupConfig)
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

	modpackMutex.Lock()
	defer modpackMutex.Unlock()

	modpack = Modpack{Folder: folderAbsolute}
	err = modpack.loadConfigFiles()
	if err != nil {
		writeError(w, err)
		// Clear value
		modpack = Modpack{}
		return
	}
	// Update mod list
	modpack.getModInfoList()

	// Update cache
	writeEditorCache()

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

	modpackMutex.Lock()
	defer modpackMutex.Unlock()

	modpack = Modpack{Folder: folderAbsolute}
	err = modpack.loadConfigFiles()
	if err != nil {
		writeError(w, err)
		// Clear value
		modpack = Modpack{}
		return
	}
	// Create empty modlist
	modpack.Mods = make(map[int]ModInfo)

	// Update cache
	writeEditorCache()

	// Send the modpack to the client
	json.NewEncoder(w).Encode(struct {
		Modpack Modpack
	}{modpack})
}

func getCurrentPackDetails(w io.Writer) {
	modpackMutex.RLock()
	defer modpackMutex.RUnlock()

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
	// TODO: Required?
	// TODO: dates, rating, download counts?
	// TODO: categories?
	// TODO: author(s)?
	Summary      string
	WebsiteURL   string
	Slug         string
	OnClient     bool
	OnServer     bool
	FileID       int
	Dependencies []struct {
		AddonID int    `json:"addOnId"`
		Type    string `json:"type"`
	}
	Dependants []struct {
		AddonID int    `json:"addOnId"`
		Type    string `json:"type"`
	}
}

func (m *Modpack) getModInfoList() {
	info := make(map[int]ModInfo)
	var wg sync.WaitGroup
	// Mutex for the ModInfo map
	var mutex = &sync.RWMutex{}

	for _, v := range m.CurseManifest.Files {
		// Increment the WaitGroup counter.
		wg.Add(1)

		go func(projectID, fileID int) {
			// Decrement the counter when the goroutine completes.
			defer wg.Done()

			data, err := requestAddonData(projectID)
			if err != nil {
				mutex.Lock()
				info[projectID] = ModInfo{
					ErrorMessage: err,
				}
				mutex.Unlock()
				return
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

			onServer := true
			for _, v := range m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject {
				if v == projectID {
					onServer = false
					break
				}
			}

			fileInfo, err := requestFileData(projectID, fileID)
			if err != nil {
				mutex.Lock()
				info[projectID] = ModInfo{
					ErrorMessage: err,
				}
				mutex.Unlock()
				return
			}

			mutex.Lock()
			info[projectID] = ModInfo{
				Name:         data.Name,
				IconURL:      iconURL,
				Summary:      data.Summary,
				WebsiteURL:   data.WebsiteURL,
				Slug:         data.Slug,
				OnClient:     true,
				OnServer:     onServer,
				FileID:       fileID,
				Dependencies: fileInfo.Dependencies,
			}
			mutex.Unlock()
		}(v.ProjectID, v.FileID)
	}

	for _, v := range m.ServerSetupConfig.Install.AdditionalFiles {
		// Ignore non-curseforge download links
		if !strings.HasPrefix(v.URL, "https://minecraft.curseforge.com/projects/") {
			continue
		}

		// Increment the WaitGroup counter.
		wg.Add(1)

		go func(projectURL string) {
			// Decrement the counter when the goroutine completes.
			defer wg.Done()

			re := regexp.MustCompile("https://minecraft.curseforge.com/projects/([\\w\\-]+)/files/(\\d+)/")
			matches := re.FindSubmatch([]byte(projectURL))
			if len(matches) < 3 {
				// TODO: where is this output?
				return
			}
			slug := string(matches[1])
			fileID, err := strconv.Atoi(string(matches[2]))
			if err != nil {
				return
			}

			data, err := requestAddonDataFromSlug(slug)
			if err != nil {
				/*mutex.Lock()
				// TODO: where is this output?
				info[] = ModInfo{
					ErrorMessage: err,
				}
				mutex.Unlock()*/
				return
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

			fileInfo, err := requestFileData(data.ID, fileID)
			if err != nil {
				mutex.Lock()
				info[data.ID] = ModInfo{
					ErrorMessage: err,
				}
				mutex.Unlock()
				return
			}

			mutex.Lock()
			info[data.ID] = ModInfo{
				Name:         data.Name,
				IconURL:      iconURL,
				Summary:      data.Summary,
				WebsiteURL:   data.WebsiteURL,
				Slug:         data.Slug,
				OnClient:     false,
				OnServer:     true,
				FileID:       fileID,
				Dependencies: fileInfo.Dependencies,
			}
			mutex.Unlock()
		}(v.URL)
	}

	// Wait for all HTTP fetches to complete.
	wg.Wait()

	// Update cache
	writeEditorCache()

	// After modInfos are populated, calculate dependants
	for depProjectID, v := range info {
		for _, dep := range v.Dependencies {
			if p, ok := info[dep.AddonID]; ok {
				if p.Dependants == nil {
					p.Dependants = make([]struct {
						AddonID int    `json:"addOnId"`
						Type    string `json:"type"`
					}, 1)
					p.Dependants[0] = struct {
						AddonID int    `json:"addOnId"`
						Type    string `json:"type"`
					}{depProjectID, dep.Type}
				} else {
					p.Dependants = append(info[dep.AddonID].Dependants, struct {
						AddonID int    `json:"addOnId"`
						Type    string `json:"type"`
					}{depProjectID, dep.Type})
				}
				info[dep.AddonID] = p
			}
		}
	}

	m.Mods = info
}

func (m *Modpack) syncCurseKeyMap(shouldExist bool, projectID, fileID int, curseKeyMap map[int]int) {
	// Does the project exist in the manifest?
	if key, ok := curseKeyMap[projectID]; ok {
		if shouldExist {
			// Modify
			if m.CurseManifest.Files[key].FileID != fileID {
				fmt.Printf("old %d new %d\n", m.CurseManifest.Files[key].FileID, fileID)
				fmt.Println("Updated curse id")
				m.CurseManifest.Files[key].FileID = fileID
			}
		} else {
			// Delete (from SliceTricks)
			m.CurseManifest.Files = append(m.CurseManifest.Files[:key], m.CurseManifest.Files[key+1:]...)
			fmt.Println("Deleted curse id")
			// Shift every key down
			for i, v := range curseKeyMap {
				if v > key {
					curseKeyMap[i] = v - 1
				}
			}
		}
	} else if shouldExist {
		// Add
		m.CurseManifest.Files = append(m.CurseManifest.Files, struct {
			ProjectID int  `json:"projectID"`
			FileID    int  `json:"fileID"`
			Required  bool `json:"required"`
		}{projectID, fileID, true})
		fmt.Println("Added curse id")
	}
	// If !exists and !shouldExist, ignore
	// Delete from keyMap
	delete(curseKeyMap, projectID)
}

func (m *Modpack) syncIgnoreProjectKeyMap(shouldExist bool, projectID int, ignoreProjectKeyMap map[int]int) {
	// Does the project exist in the manifest?
	if key, ok := ignoreProjectKeyMap[projectID]; ok {
		if shouldExist {
			// Nothing to do here
		} else {
			// Delete (from SliceTricks)
			m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject = append(m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject[:key], m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject[key+1:]...)
			fmt.Println("Deleted ignore")
			// Shift every key down
			for i, v := range ignoreProjectKeyMap {
				if v > key {
					ignoreProjectKeyMap[i] = v - 1
				}
			}
		}
	} else if shouldExist {
		// Add
		m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject = append(m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject, projectID)
		fmt.Println("Added ignore")
	}
	// If !exists and !shouldExist, ignore
	// Delete from keyMap
	delete(ignoreProjectKeyMap, projectID)
}

func (m *Modpack) syncAdditionalFilesSlugMap(shouldExist bool, slug string, fileID int, additionalFilesSlugMap map[string]int) {
	// Does the project exist in the manifest?
	if key, ok := additionalFilesSlugMap[slug]; ok {
		if shouldExist {
			// Modify
			re := regexp.MustCompile("https://minecraft.curseforge.com/projects/([\\w\\-]+)/files/(\\d+)/")
			matches := re.FindSubmatch([]byte(m.ServerSetupConfig.Install.AdditionalFiles[key].URL))
			if len(matches) < 3 {
				// TODO: throw an error?
				return
			}
			oldFileID, err := strconv.Atoi(string(matches[2]))
			if err != nil {
				return
			}

			if oldFileID != fileID {
				fmt.Println("modified additonalFile")

				downloadURL := fmt.Sprintf("https://minecraft.curseforge.com/projects/%s/files/%d/download", slug, fileID)
				fileInfo, err := requestFileDataFromSlug(slug, fileID)
				if err != nil {
					return
				}
				destination := fmt.Sprintf("mods/%s", fileInfo.FileNameOnDisk)

				m.ServerSetupConfig.Install.AdditionalFiles[key] = struct {
					URL         string `yaml:"url"`
					Destination string `yaml:"destination"`
				}{downloadURL, destination}
			}
		} else {
			// Delete (from SliceTricks)
			m.ServerSetupConfig.Install.AdditionalFiles = append(m.ServerSetupConfig.Install.AdditionalFiles[:key], m.ServerSetupConfig.Install.AdditionalFiles[key+1:]...)
			fmt.Println("Deleted additional file")
			// Shift every key down
			for i, v := range additionalFilesSlugMap {
				if v > key {
					additionalFilesSlugMap[i] = v - 1
				}
			}
		}
	} else if shouldExist {
		// Add
		fmt.Println("added additonalFile")

		downloadURL := fmt.Sprintf("https://minecraft.curseforge.com/projects/%s/files/%d/download", slug, fileID)
		fileInfo, err := requestFileDataFromSlug(slug, fileID)
		if err != nil {
			return
		}
		destination := fmt.Sprintf("mods/%s", fileInfo.FileNameOnDisk)

		m.ServerSetupConfig.Install.AdditionalFiles = append(m.ServerSetupConfig.Install.AdditionalFiles, struct {
			URL         string `yaml:"url"`
			Destination string `yaml:"destination"`
		}{downloadURL, destination})
	}
	// If !exists and !shouldExist, ignore
	// Delete from slugMap
	delete(additionalFilesSlugMap, slug)
}

// This function is painful.
func (m *Modpack) updateModLists() error {
	// Update the CurseManifest and ServerSetupConfig with the changes in m.Mods
	// while preserving the order of each array

	// I did this because diffs.

	// keyMaps are used to map the project IDs to array indexes
	curseKeyMap := make(map[int]int)
	for i, v := range m.CurseManifest.Files {
		curseKeyMap[v.ProjectID] = i
	}
	ignoreProjectKeyMap := make(map[int]int)
	for i, v := range m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject {
		ignoreProjectKeyMap[v] = i
	}
	// additionalFilesSlugMap is used for additionalFiles
	additionalFilesSlugMap := make(map[string]int)
	for i, v := range m.ServerSetupConfig.Install.AdditionalFiles {
		if !strings.HasPrefix(v.URL, "https://minecraft.curseforge.com/projects/") {
			continue
		}

		re := regexp.MustCompile("https://minecraft.curseforge.com/projects/([\\w\\-]+)/")
		matches := re.FindSubmatch([]byte(v.URL))
		if len(matches) < 2 {
			return fmt.Errorf("Could not match slug from project URL: %s", v.URL)
		}
		slug := string(matches[1])

		additionalFilesSlugMap[slug] = i
	}

	for projectID, v := range m.Mods {
		if v.OnClient {
			// Must be in curseKeyMap
			m.syncCurseKeyMap(true, projectID, v.FileID, curseKeyMap)
			// Must not be in additionalFilesSlugMap
			m.syncAdditionalFilesSlugMap(false, v.Slug, v.FileID, additionalFilesSlugMap)
			if v.OnServer {
				// Must not be in ignoreProjectKeyMap
				m.syncIgnoreProjectKeyMap(false, projectID, ignoreProjectKeyMap)
			} else {
				// Must be in ignoreProjectKeyMap
				m.syncIgnoreProjectKeyMap(true, projectID, ignoreProjectKeyMap)
			}
		} else if v.OnServer {
			// Must not be in curseKeyMap
			m.syncCurseKeyMap(false, projectID, v.FileID, curseKeyMap)
			// Must not be in ignoreProjectKeyMap
			m.syncIgnoreProjectKeyMap(false, projectID, ignoreProjectKeyMap)
			// Must be in additionalFilesSlugMap
			m.syncAdditionalFilesSlugMap(true, v.Slug, v.FileID, additionalFilesSlugMap)
		} else {
			return fmt.Errorf("Mod is not on server or client: %d", projectID)
		}
	}

	for _, v := range curseKeyMap {
		// Delete (from SliceTricks)
		fmt.Println("delete from curse")
		m.CurseManifest.Files = append(m.CurseManifest.Files[:v], m.CurseManifest.Files[v+1:]...)
	}

	for _, v := range ignoreProjectKeyMap {
		// Delete (from SliceTricks)
		fmt.Println("delete from ignore")
		m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject = append(m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject[:v], m.ServerSetupConfig.Install.FormatSpecific.IgnoreProject[v+1:]...)
	}

	for _, v := range additionalFilesSlugMap {
		// Delete (from SliceTricks)
		fmt.Println("delete from additional")
		m.ServerSetupConfig.Install.AdditionalFiles = append(m.ServerSetupConfig.Install.AdditionalFiles[:v], m.ServerSetupConfig.Install.AdditionalFiles[v+1:]...)
	}

	return nil
}

func (m *Modpack) saveConfigFiles() error {
	manifest, err := json.Marshal(&m.CurseManifest)
	if err != nil {
		return err
	}

	var manifestBuffer bytes.Buffer
	json.Indent(&manifestBuffer, manifest, "", "  ")

	f, err := os.Create(filepath.Join(m.Folder, "manifest.json"))
	if err != nil {
		return err
	}
	_, err = manifestBuffer.WriteTo(f)
	if err != nil {
		return err
	}
	err = f.Close()
	if err != nil {
		return err
	}

	config, err := yaml.Marshal(&m.ServerSetupConfig)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(m.Folder, "server-setup-config.yaml"), config, 0664)
	if err != nil {
		return err
	}
	return nil
}

func saveModpack(w http.ResponseWriter, newPack Modpack) {
	modpackMutex.Lock()
	defer modpackMutex.Unlock()
	modpack = newPack

	err := modpack.updateModLists()
	if err != nil {
		writeError(w, err)
		return
	}

	err = modpack.saveConfigFiles()
	if err != nil {
		writeError(w, err)
		return
	}

	json.NewEncoder(w).Encode(struct{}{})
}
