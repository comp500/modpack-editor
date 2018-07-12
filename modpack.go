package main

import (
	"encoding/json"
	"errors"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
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
	Summary    string
	WebsiteURL string
	Slug       string
	OnClient   bool
	OnServer   bool
}

func (m *Modpack) getModInfoList() {
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

			mutex.Lock()
			info[projectID] = ModInfo{
				Name:       data.Name,
				IconURL:    iconURL,
				Summary:    data.Summary,
				WebsiteURL: data.WebsiteURL,
				Slug:       data.Slug,
				OnClient:   true,
				OnServer:   onServer,
			}
			mutex.Unlock()
		}(v.ProjectID)
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

			re := regexp.MustCompile("https://minecraft.curseforge.com/projects/([\\w\\-]+)/")
			matches := re.FindSubmatch([]byte(projectURL))
			if len(matches) < 2 {
				// TODO: where is this output?
				return
			}
			slug := string(matches[1])

			data, err := requestAddonDataFromSlug(slug)
			if err != nil {
				mutex.Lock()
				// TODO: where is this output?
				/*info[] = ModInfo{
					ErrorMessage: err,
				}*/
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
			info[data.ID] = ModInfo{
				Name:       data.Name,
				IconURL:    iconURL,
				Summary:    data.Summary,
				WebsiteURL: data.WebsiteURL,
				Slug:       data.Slug,
				OnClient:   false,
				OnServer:   true,
			}
			mutex.Unlock()
		}(v.URL)
	}

	// Wait for all HTTP fetches to complete.
	wg.Wait()

	// Update cache
	writeEditorCache()

	m.Mods = info
}
