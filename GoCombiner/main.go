package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 3 {
		fmt.Println("Usage: go run main.go <dir1> <dir2>")
		return
	}

	dir1 := os.Args[1]
	dir2 := os.Args[2]
	outDir := "CombinedList"

	// Create the output directory
	if err := os.MkdirAll(outDir, 0755); err != nil {
		fmt.Printf("Error creating output directory: %v\n", err)
		return
	}

	// Map to track all unique filenames across both directories (lowercase)
	allFiles := make(map[string]bool)

	fillFileMap(dir1, allFiles)
	fillFileMap(dir2, allFiles)

	for fileName := range allFiles {
		mergeFiles(fileName, dir1, dir2, outDir)
	}

	fmt.Printf("\nProcessing complete. Files merged into ./%s\n", outDir)
}

func fillFileMap(dir string, fileMap map[string]bool) {
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		if !f.IsDir() && strings.HasSuffix(strings.ToLower(f.Name()), ".txt") {
			fileMap[strings.ToLower(f.Name())] = true
		}
	}
}

func mergeFiles(fileName, dir1, dir2, outDir string) {
	uniqueLines := make(map[string]struct{})

	// Process file from first directory
	processFile(filepath.Join(dir1, fileName), uniqueLines)
	// Process file from second directory
	processFile(filepath.Join(dir2, fileName), uniqueLines)

	if len(uniqueLines) == 0 {
		return
	}

	// Write unique lines to the new file
	outFile, err := os.Create(filepath.Join(outDir, fileName))
	if err != nil {
		fmt.Printf("Error creating file %s: %v\n", fileName, err)
		return
	}
	defer outFile.Close()

	writer := bufio.NewWriter(outFile)
	for line := range uniqueLines {
		fmt.Fprintln(writer, line)
	}
	writer.Flush()
	fmt.Printf("Merged: %s\n", fileName)
}

func processFile(path string, lines map[string]struct{}) {
	file, err := os.Open(path)
	if err != nil {
		return // Skip if file doesn't exist in this specific directory
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip empty lines and basic comments if you want a clean merge
		if line != "" {
			lines[line] = struct{}{}
		}
	}
}
