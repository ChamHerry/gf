// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// check_project.go defines the Project scanner that walks a GoFrame project
// directory, collects Go source files, parses their AST, and caches the
// results for use by check rules.

package check

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"

	"github.com/gogf/gf/v2/os/gfile"
)

// GoFile holds parsed information about a single Go source file.
type GoFile struct {
	// Path is the file path relative to the project root.
	Path string
	// AbsPath is the absolute file path.
	AbsPath string
	// Content is the raw file content.
	Content string
	// FileSet is the token.FileSet used for parsing.
	FileSet *token.FileSet
	// AST is the parsed abstract syntax tree.
	AST *ast.File
	// ParseErr holds any error encountered during parsing.
	ParseErr error
	// PackageName is the Go package name declared in the file.
	PackageName string
}

// Project holds scanned information about a GoFrame project.
type Project struct {
	// RootPath is the absolute path to the project root directory.
	RootPath string
	// ModuleName is the Go module name from go.mod.
	ModuleName string
	// IsMono indicates whether the project is a mono-repo (has an app/ directory).
	IsMono bool
	// GoFiles maps relative file paths to parsed GoFile objects.
	GoFiles map[string]*GoFile
	// Dirs is a set of relative directory paths that exist in the project.
	Dirs map[string]bool
}

// directoriesToSkip lists directory names that should not be scanned.
var directoriesToSkip = map[string]bool{
	"vendor":       true,
	"testdata":     true,
	".git":         true,
	"node_modules": true,
	".idea":        true,
	".vscode":      true,
	"tmp":          true,
	"temp":         true,
}

// ScanProject scans a project directory and returns a populated Project.
// It collects all .go files (excluding test files), parses their AST,
// extracts the module name from go.mod, and detects mono-repo structure.
func ScanProject(rootPath string) (*Project, error) {
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return nil, err
	}

	project := &Project{
		RootPath: absRoot,
		GoFiles:  make(map[string]*GoFile),
		Dirs:     make(map[string]bool),
	}

	// Walk the directory tree to collect Go files and directories.
	err = filepath.WalkDir(absRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, relErr := filepath.Rel(absRoot, path)
		if relErr != nil {
			relPath = path
		}
		relPath = filepath.ToSlash(relPath)

		if d.IsDir() {
			// Skip excluded directories.
			if directoriesToSkip[d.Name()] {
				return filepath.SkipDir
			}
			// Record the directory.
			if relPath != "." {
				project.Dirs[relPath] = true
			}
			return nil
		}

		// Only process .go files, skip test files.
		if !strings.HasSuffix(d.Name(), ".go") || strings.HasSuffix(d.Name(), "_test.go") {
			return nil
		}

		goFile := parseGoFile(path, relPath)
		project.GoFiles[relPath] = goFile

		return nil
	})
	if err != nil {
		return nil, err
	}

	// Extract module name from go.mod.
	project.ModuleName = parseModuleName(absRoot)

	// Detect mono-repo (has an app/ directory at the root).
	project.IsMono = project.HasDir("app")

	return project, nil
}

// parseGoFile reads and parses a Go source file, returning a GoFile.
func parseGoFile(absPath, relPath string) *GoFile {
	content := gfile.GetContents(absPath)
	fileSet := token.NewFileSet()
	astFile, parseErr := parser.ParseFile(fileSet, absPath, content, parser.ParseComments)

	goFile := &GoFile{
		Path:     relPath,
		AbsPath:  absPath,
		Content:  content,
		FileSet:  fileSet,
		AST:      astFile,
		ParseErr: parseErr,
	}
	if astFile != nil && astFile.Name != nil {
		goFile.PackageName = astFile.Name.Name
	}
	return goFile
}

// parseModuleName reads go.mod and extracts the module path.
func parseModuleName(projectRoot string) string {
	goModPath := filepath.Join(projectRoot, "go.mod")
	content := gfile.GetContents(goModPath)
	if content == "" {
		return ""
	}
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

// HasDir checks whether a directory exists relative to the project root.
func (p *Project) HasDir(relPath string) bool {
	return p.Dirs[filepath.ToSlash(relPath)]
}

// HasFile checks whether a Go file exists at the given relative path.
func (p *Project) HasFile(relPath string) bool {
	return p.GoFiles[filepath.ToSlash(relPath)] != nil
}

// FileExists checks whether any file exists at the given relative path on disk.
func (p *Project) FileExists(relPath string) bool {
	return gfile.Exists(filepath.Join(p.RootPath, relPath))
}

// ReadFile reads and returns the content of a file by its relative path.
// Returns an empty string if the file does not exist.
func (p *Project) ReadFile(relPath string) string {
	return gfile.GetContents(filepath.Join(p.RootPath, relPath))
}

// FilesInDir returns all Go files directly in the given directory (non-recursive).
// relDir is relative to the project root (e.g., "api/v1").
func (p *Project) FilesInDir(relDir string) []*GoFile {
	relDir = filepath.ToSlash(relDir)
	if relDir == "" {
		relDir = "."
	}
	var result []*GoFile
	for path, f := range p.GoFiles {
		if filepath.ToSlash(filepath.Dir(path)) == relDir {
			result = append(result, f)
		}
	}
	return result
}

// FilesInDirRecursive returns all Go files under the given directory (recursive).
// relDir is relative to the project root (e.g., "internal/controller").
func (p *Project) FilesInDirRecursive(relDir string) []*GoFile {
	relDir = filepath.ToSlash(relDir)
	if relDir == "" || relDir == "." {
		result := make([]*GoFile, 0, len(p.GoFiles))
		for _, f := range p.GoFiles {
			result = append(result, f)
		}
		return result
	}
	prefix := relDir + "/"
	var result []*GoFile
	for path, f := range p.GoFiles {
		if strings.HasPrefix(path, prefix) {
			result = append(result, f)
		}
	}
	return result
}
