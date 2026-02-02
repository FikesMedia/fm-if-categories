package main

import (
	"archive/tar"
	"bufio"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

const (
	ut1URL = "https://dsi.ut-capitole.fr/blacklists/download/all.tar.gz"
	blpAPI = "https://api.github.com/repos/blocklistproject/Lists/contents/"
	fmAPI  = "https://api.github.com/repos/FikesMedia/fm-if-categories/contents/CustomList"
)

var categoryMerger = map[string]string{
	"publicite": "ads",
	"drogue":    "drugs",
	"doh":       "DNS_Over_HTTPS",
	"gaming":    "games",
	"X":         "twitter",
}

var exclusions = map[string]bool{
	"agressif.txt":                 true,
	"arjel.txt":                    true,
	"child.txt":                    true,
	"list_blanche.txt":             true,
	"list_bu.txt":                  true,
	"tricheur.txt":                 true,
	"tricheur_pix.txt":             true,
	"update.txt":                   true,
	"reaffected.txt":               true,
	"associations_religieuses.txt": true,
	"sect.txt":                     true,
	"exceptions_liste_bu.txt":      true,
	"examen_pix.txt":               true,
	"everything.txt":               true,
}

type GitHubContent struct {
	Name        string `json:"name"`
	DownloadURL string `json:"download_url"`
	Type        string `json:"type"`
}

func main() {
	fmt.Println("🚀 Initializing CIPA Master Pipeline...")

	dirs := []string{"Temp/ut1", "Temp/blp", "Temp/fm", "master_export"}
	for _, d := range dirs {
		os.RemoveAll(d)
		os.MkdirAll(d, 0755)
	}

	fetchUT1()
	fetchGitHub(blpAPI, "Temp/blp")
	fetchGitHub(fmAPI, "Temp/fm")

	totalCount := mergeAndClean()

	fmt.Println("\n--- FINAL REPORT ---")
	fmt.Printf("✨ Success! Master lists available in './master_export'\n")
	fmt.Printf("🛡️  Total unique domains protected: %d\n", totalCount)
	fmt.Println("--------------------")
}

// --- Fetching Logic ---

func fetchUT1() {
	fmt.Println("📥 Streaming UT1 Archive...")
	resp, err := http.Get(ut1URL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	gr, _ := gzip.NewReader(resp.Body)
	tr := tar.NewReader(gr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if strings.HasSuffix(header.Name, "/domains") {
			rawName := strings.Split(header.Name, "/")[1]
			fileName := strings.ToLower(rawName + ".txt")
			if exclusions[fileName] {
				continue
			}
			out, _ := os.Create(filepath.Join("Temp/ut1", fileName))
			io.Copy(out, tr)
			out.Close()
		}
	}
}

func fetchGitHub(apiURL, targetDir string) {
	fmt.Printf("📥 Fetching GitHub Repo: %s\n", targetDir)
	resp, err := http.Get(apiURL)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var files []GitHubContent
	json.NewDecoder(resp.Body).Decode(&files)

	for _, file := range files {
		fileName := strings.ToLower(file.Name)
		if file.Type == "file" && strings.HasSuffix(fileName, ".txt") {
			if exclusions[fileName] {
				continue
			}
			fResp, err := http.Get(file.DownloadURL)
			if err != nil {
				continue
			}
			out, _ := os.Create(filepath.Join(targetDir, fileName))
			io.Copy(out, fResp.Body)
			out.Close()
			fResp.Body.Close()
		}
	}
}

// --- Merging & Stats Logic ---

func mergeAndClean() int {
	fmt.Println("🔄 Merging and Normalizing...")

	masterMap := make(map[string]map[string]bool)
	sources := []string{"Temp/ut1", "Temp/blp", "Temp/fm"}

	for _, srcDir := range sources {
		files, _ := os.ReadDir(srcDir)
		for _, f := range files {
			fileName := strings.ToLower(f.Name())
			if exclusions[fileName] {
				continue
			}

			rawCategory := strings.TrimSuffix(fileName, ".txt")
			targetCategory := rawCategory
			if mapped, exists := categoryMerger[rawCategory]; exists {
				targetCategory = mapped
			}

			if masterMap[targetCategory] == nil {
				masterMap[targetCategory] = make(map[string]bool)
			}

			path := filepath.Join(srcDir, f.Name())
			file, _ := os.Open(path)
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				line := strings.TrimSpace(scanner.Text())
				if line == "" || strings.HasPrefix(line, "#") {
					continue
				}
				fields := strings.Fields(line)
				domain := strings.ToLower(fields[len(fields)-1])
				masterMap[targetCategory][domain] = true
			}
			file.Close()
		}
	}

	globalUniqueDomains := 0
	for cat, domains := range masterMap {
		title := strings.ReplaceAll(cat, "_", " ")
		title = strings.ReplaceAll(title, "-", " ")
		title = strings.Title(title)

		outPath := filepath.Join("master_export", cat+".txt")
		outFile, _ := os.Create(outPath)

		outFile.WriteString(fmt.Sprintf("# %s\n", title))
		for dom := range domains {
			outFile.WriteString(fmt.Sprintf("0.0.0.0 %s\n", dom))
			globalUniqueDomains++ // Increment for every line written
		}
		outFile.Close()
	}

	return globalUniqueDomains
}
