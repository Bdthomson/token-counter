package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gitignore "github.com/sabhiram/go-gitignore"
	"github.com/tiktoken-go/tokenizer"
)

// FileTokenInfo stores token count information for a file
type FileTokenInfo struct {
	Path       string
	TokenCount int
}

// DirTokenInfo stores token count information for a directory
type DirTokenInfo struct {
	Path       string
	TokenCount int
	Files      []*FileTokenInfo
}

// RepoTokenInfo stores token count information for the entire repository
type RepoTokenInfo struct {
	Path       string
	TokenCount int
	Dirs       map[string]*DirTokenInfo
}

// CommandOptions stores the command-line options
type CommandOptions struct {
	Path            string
	Model           string
	RespectGitignore bool
	ShowFiles       bool
	MinTokens       int
	SortByTokens    bool
	IgnoreHidden    bool
	IsSingleFile    bool  // Indicates if the path is a single file rather than a directory
}

// CountTokensInFile counts the number of tokens in a single file
func CountTokensInFile(path string, modelName string) (int, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return 0, err
	}

	// Use the specified model or default to cl100k_base
	enc, err := tokenizer.Get(tokenizer.Encoding(modelName))
	if err != nil {
		return 0, err
	}

	tokens, _, err := enc.Encode(string(data))
	return len(tokens), err
}

// ProcessRepository walks through the repository and counts tokens
func ProcessRepository(rootPath string, options *CommandOptions) (*RepoTokenInfo, error) {
	repo := &RepoTokenInfo{
		Path: rootPath,
		Dirs: make(map[string]*DirTokenInfo),
	}

	// Load .gitignore if needed
	var ignorer *gitignore.GitIgnore
	var err error
	if options.RespectGitignore {
		gitignorePath := filepath.Join(rootPath, ".gitignore")
		if _, statErr := os.Stat(gitignorePath); statErr == nil {
			ignorer, err = gitignore.CompileIgnoreFile(gitignorePath)
			if err != nil {
				fmt.Printf("Warning: Error loading .gitignore file: %v\n", err)
			}
		}
	}

	err = filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path for gitignore matching
		relPath, err := filepath.Rel(rootPath, path)
		if err != nil {
			relPath = path
		}

		// Skip hidden files and directories if specified
		if options.IgnoreHidden && strings.HasPrefix(filepath.Base(path), ".") {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Check if the file is ignored by .gitignore
		if ignorer != nil && ignorer.MatchesPath(relPath) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip directories themselves (we'll count files inside them)
		if info.IsDir() {
			return nil
		}

		// Skip binary files and certain extensions
		ext := strings.ToLower(filepath.Ext(path))
		if shouldSkipFile(path, ext, info) {
			return nil
		}

		// Count tokens in the file
		tokenCount, err := CountTokensInFile(path, options.Model)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", path, err)
			return nil
		}

		// Skip files with fewer tokens than the minimum if specified
		if options.MinTokens > 0 && tokenCount < options.MinTokens {
			return nil
		}

		// Get directory path
		dirPath := filepath.Dir(path)
		
		// Create or update directory info
		dirInfo, exists := repo.Dirs[dirPath]
		if !exists {
			dirInfo = &DirTokenInfo{
				Path:  dirPath,
				Files: []*FileTokenInfo{},
			}
			repo.Dirs[dirPath] = dirInfo
		}

		// Add file info to directory
		fileInfo := &FileTokenInfo{
			Path:       path,
			TokenCount: tokenCount,
		}
		dirInfo.Files = append(dirInfo.Files, fileInfo)
		dirInfo.TokenCount += tokenCount
		
		// Add to repository total
		repo.TokenCount += tokenCount

		return nil
	})

	return repo, err
}

// ProcessSingleFile counts tokens in a single file
func ProcessSingleFile(filePath string, options *CommandOptions) (*RepoTokenInfo, error) {
	// Check if file exists
	fileInfo, err := os.Stat(filePath)
	if (err != nil) {
		return nil, fmt.Errorf("error accessing file: %v", err)
	}
	
	// Make sure it's not a directory
	if fileInfo.IsDir() {
		return nil, fmt.Errorf("%s is a directory, not a file", filePath)
	}
	
	// Check if we should skip this file
	ext := strings.ToLower(filepath.Ext(filePath))
	if shouldSkipFile(filePath, ext, fileInfo) {
		return nil, fmt.Errorf("skipping binary or unsupported file type: %s", filePath)
	}
	
	// Count tokens in the file
	tokenCount, err := CountTokensInFile(filePath, options.Model)
	if err != nil {
		return nil, fmt.Errorf("error processing file: %v", err)
	}
	
	// Skip if fewer tokens than minimum
	if options.MinTokens > 0 && tokenCount < options.MinTokens {
		return nil, fmt.Errorf("file has fewer tokens (%d) than minimum (%d)", tokenCount, options.MinTokens)
	}
	
	// Create repo info structure with just this file
	dirPath := filepath.Dir(filePath)
	
	repo := &RepoTokenInfo{
		Path:       filePath,
		TokenCount: tokenCount,
		Dirs:       make(map[string]*DirTokenInfo),
	}
	
	// Add directory info
	dirInfo := &DirTokenInfo{
		Path:       dirPath,
		TokenCount: tokenCount,
		Files:      []*FileTokenInfo{},
	}
	repo.Dirs[dirPath] = dirInfo
	
	// Add file info
	fileTokenInfo := &FileTokenInfo{
		Path:       filePath,
		TokenCount: tokenCount,
	}
	dirInfo.Files = append(dirInfo.Files, fileTokenInfo)
	
	return repo, nil
}

// shouldSkipFile determines if a file should be skipped based on extension or other criteria
func shouldSkipFile(path string, ext string, info os.FileInfo) bool {
	// Skip files without an extension if they're executables (like the token-counter binary)
	if ext == "" && info.Mode()&0111 != 0 {
		return true
	}

	// Skip the binary with the same name as the module
	if filepath.Base(path) == "token-counter" {
		return true
	}
	
	// List of binary or non-text file extensions to skip
	skipExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, 
		".pdf": true, ".zip": true, ".tar": true, ".gz": true,
		".exe": true, ".dll": true, ".so": true, ".dylib": true,
		".bin": true, ".obj": true, ".o": true,
	}
	
	return skipExts[ext]
}

// PrintResults prints the token counting results
func PrintResults(repo *RepoTokenInfo, options *CommandOptions) {
	fmt.Printf("Token Count Summary for: %s\n", repo.Path)
	
	// Special handling for single file
	if options.IsSingleFile {
		fmt.Printf("Total tokens: %d\n", repo.TokenCount)
		return
	}
	
	fmt.Printf("Total tokens in repository: %d\n\n", repo.TokenCount)
	
	// Sort directories by token count (highest first)
	type DirEntry struct {
		Path  string
		Info  *DirTokenInfo
	}
	
	var dirs []DirEntry
	for path, info := range repo.Dirs {
		dirs = append(dirs, DirEntry{path, info})
	}
	
	sort.Slice(dirs, func(i, j int) bool {
		return dirs[i].Info.TokenCount > dirs[j].Info.TokenCount
	})
	
	// Print directory summaries
	fmt.Println("Directories (sorted by token count):")
	fmt.Println("----------------------------------")
	for _, entry := range dirs {
		dirInfo := entry.Info
		fmt.Printf("%s: %d tokens\n", dirInfo.Path, dirInfo.TokenCount)
		
		// Only print file details if requested
		if options.ShowFiles {
			// Sort files within directory
			sort.Slice(dirInfo.Files, func(i, j int) bool {
				return dirInfo.Files[i].TokenCount > dirInfo.Files[j].TokenCount
			})
			
			// Print file details
			for _, fileInfo := range dirInfo.Files {
				relativePath, _ := filepath.Rel(repo.Path, fileInfo.Path)
				fmt.Printf("  |- %s: %d tokens\n", relativePath, fileInfo.TokenCount)
			}
		}
		fmt.Println()
	}
}

func main() {
	options := &CommandOptions{}

	// Define command line flags
	flag.StringVar(&options.Path, "path", "", "Path to the directory or file to analyze (defaults to current directory if not provided)")
	flag.StringVar(&options.Model, "model", string(tokenizer.Cl100kBase), "Token counting model to use (e.g., cl100k_base for GPT-4)")
	flag.BoolVar(&options.RespectGitignore, "gitignore", true, "Whether to respect .gitignore rules")
	flag.BoolVar(&options.ShowFiles, "files", true, "Whether to show individual file details")
	flag.IntVar(&options.MinTokens, "min", 0, "Minimum token count for a file to be included")
	flag.BoolVar(&options.IgnoreHidden, "no-hidden", true, "Whether to ignore hidden files and directories (starting with .)")
	flag.BoolVar(&options.IsSingleFile, "file", false, "Treat the path as a single file rather than a directory")
	
	// Parse command line flags
	flag.Parse()
	
	// If no path is provided via flags, check positional args or use current directory
	if options.Path == "" {
		if flag.NArg() > 0 {
			options.Path = flag.Arg(0)
		} else {
			var err error
			options.Path, err = os.Getwd()
			if err != nil {
				fmt.Printf("Error getting current directory: %v\n", err)
				os.Exit(1)
			}
		}
	}
	
	// Check if path is a file
	if !options.IsSingleFile {
		fileInfo, err := os.Stat(options.Path)
		if err == nil && !fileInfo.IsDir() {
			options.IsSingleFile = true
		}
	}

	var repo *RepoTokenInfo
	var err error
	
	// Process a single file or a repository based on the options
	if options.IsSingleFile {
		fmt.Printf("Processing single file: %s\n", options.Path)
		repo, err = ProcessSingleFile(options.Path, options)
		if err != nil {
			fmt.Printf("Error processing file: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("Processing directory: %s\n", options.Path)
		if options.RespectGitignore {
			fmt.Println("Respecting .gitignore rules if present")
		}
		repo, err = ProcessRepository(options.Path, options)
		if err != nil {
			fmt.Printf("Error processing repository: %v\n", err)
			os.Exit(1)
		}
	}
	
	// Print results
	PrintResults(repo, options)
}