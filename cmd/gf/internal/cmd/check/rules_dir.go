// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// rules_dir.go checks that the project follows the standard GoFrame
// directory structure for both single-repo and mono-repo layouts.

package check

import "context"

// dirCheck defines a single directory or file existence check.
type dirCheck struct {
	id       string
	severity Severity
	path     string
	isDir    bool
	message  string
	hint     string
}

// singleRepoDirChecks lists all directory structure checks for a single-repo project.
var singleRepoDirChecks = []dirCheck{
	{"DIR-001", SeverityError, "main.go", false, "main.go not found in project root", "Create main.go as the application entry point"},
	{"DIR-002", SeverityError, "go.mod", false, "go.mod not found in project root", "Run 'go mod init <module-name>' to create the module file"},
	{"DIR-003", SeverityError, "api", true, "api/ directory not found", "Create api/ directory for API request/response definitions"},
	{"DIR-004", SeverityError, "internal", true, "internal/ directory not found", "Create internal/ directory for private application packages"},
	{"DIR-005", SeverityWarning, "manifest", true, "manifest/ directory not found", "Create manifest/ for deployment and configuration files"},
	{"DIR-006", SeverityWarning, "manifest/config", true, "manifest/config/ not found", "Create manifest/config/ for runtime configuration"},
	{"DIR-007", SeverityInfo, "hack/config.yaml", false, "hack/config.yaml not found", "Create hack/config.yaml for GoFrame CLI tool configuration"},
	{"DIR-008", SeverityInfo, "resource", true, "resource/ directory not found", "Create resource/ for static assets (templates, public files)"},
	{"DIR-009", SeverityError, "internal/cmd", true, "internal/cmd/ not found", "Create internal/cmd/ for command entry definitions"},
	{"DIR-010", SeverityError, "internal/controller", true, "internal/controller/ not found", "Create internal/controller/ for HTTP controller implementations"},
	{"DIR-011", SeverityWarning, "internal/service", true, "internal/service/ not found", "Create internal/service/ for service interface definitions"},
	{"DIR-012", SeverityWarning, "internal/dao", true, "internal/dao/ not found", "Create internal/dao/ for data access objects"},
	{"DIR-013", SeverityWarning, "internal/model", true, "internal/model/ not found", "Create internal/model/ for data model definitions"},
	{"DIR-014", SeverityInfo, "internal/consts", true, "internal/consts/ not found", "Create internal/consts/ for application constants"},
}

// DirectoryRule checks that the project follows the standard GoFrame directory structure.
type DirectoryRule struct {
	BaseRule
}

// NewDirectoryRule creates a new DirectoryRule.
func NewDirectoryRule() *DirectoryRule {
	return &DirectoryRule{
		BaseRule: BaseRule{
			RuleID:          "DIR",
			RuleName:        "Directory Structure",
			RuleDescription: "Checks that the project follows the standard GoFrame directory structure",
			RuleSeverity:    SeverityError,
		},
	}
}

// Run executes directory structure checks against the project.
func (r *DirectoryRule) Run(_ context.Context, project *Project) []*Violation {
	if project.IsMono {
		return r.checkMonoRepo(project)
	}
	return r.checkSingleRepo(project)
}

// checkSingleRepo runs directory checks for a single-repository project.
func (r *DirectoryRule) checkSingleRepo(project *Project) []*Violation {
	var violations []*Violation

	for _, dc := range singleRepoDirChecks {
		var exists bool
		if dc.isDir {
			exists = project.HasDir(dc.path)
		} else {
			exists = project.FileExists(dc.path)
		}
		if !exists {
			violations = append(violations, &Violation{
				RuleID:     dc.id,
				Severity:   dc.severity,
				Message:    dc.message,
				Suggestion: dc.hint,
			})
		}
	}

	// DIR-016: Check that api/ has versioned subdirectories (v1, v2, etc.).
	violations = append(violations, r.checkAPIVersionDirs(project)...)

	return violations
}

// checkAPIVersionDirs checks that api/ contains module directories with version subdirs.
func (r *DirectoryRule) checkAPIVersionDirs(project *Project) []*Violation {
	var violations []*Violation

	// Find all directories under api/ that match api/{module}/ pattern.
	moduleDirs := make(map[string]bool)
	for dir := range project.Dirs {
		if hasPrefix(dir, "api/") {
			parts := splitPath(dir)
			if len(parts) >= 2 {
				moduleDirs[parts[1]] = true
			}
		}
	}

	// For each module dir, check if it has at least one v{n} subdir.
	for modDir := range moduleDirs {
		hasVersion := false
		for dir := range project.Dirs {
			if hasPrefix(dir, "api/"+modDir+"/v") {
				hasVersion = true
				break
			}
		}
		if !hasVersion {
			violations = append(violations, &Violation{
				RuleID:     "DIR-016",
				Severity:   SeverityError,
				Message:    "api/" + modDir + "/ has no versioned subdirectory (e.g., v1/)",
				Suggestion: "Create api/" + modDir + "/v1/ for versioned API definitions",
			})
		}
	}

	return violations
}

// checkMonoRepo runs directory checks for a mono-repository project.
// Each app under app/ is checked as a mini single-repo.
func (r *DirectoryRule) checkMonoRepo(project *Project) []*Violation {
	var violations []*Violation

	// Find all app subdirectories (app/{service-name}).
	appDirs := make(map[string]bool)
	for dir := range project.Dirs {
		if hasPrefix(dir, "app/") {
			parts := splitPath(dir)
			if len(parts) >= 2 {
				appDirs[parts[1]] = true
			}
		}
	}

	if len(appDirs) == 0 {
		violations = append(violations, &Violation{
			RuleID:     "DIR-018",
			Severity:   SeverityWarning,
			Message:    "Mono-repo detected (app/ exists) but no service directories found under app/",
			Suggestion: "Create service directories under app/ (e.g., app/my-service/)",
		})
		return violations
	}

	// Check each app for essential files.
	for appDir := range appDirs {
		appPrefix := "app/" + appDir
		// DIR-019: Each app should have main.go (for executable services).
		if !project.FileExists(appPrefix + "/main.go") {
			// Check if it's a library-only service (no main.go is OK for shared libs).
			hasGoFiles := false
			for path := range project.GoFiles {
				if hasPrefix(path, appPrefix+"/") {
					hasGoFiles = true
					break
				}
			}
			if hasGoFiles {
				violations = append(violations, &Violation{
					RuleID:     "DIR-019",
					Severity:   SeverityInfo,
					Message:    appPrefix + "/main.go not found",
					Suggestion: "If this is an executable service, create main.go as the entry point",
				})
			}
		}
		// Check each app has internal/ directory.
		if !project.HasDir(appPrefix + "/internal") {
			violations = append(violations, &Violation{
				RuleID:     "DIR-004",
				Severity:   SeverityError,
				Message:    appPrefix + "/internal/ directory not found",
				Suggestion: "Create internal/ directory for private packages",
			})
		}
	}

	return violations
}

// getProjectBaseDirs returns the base directories to check for a project.
// For single-repo, it returns [""] (meaning paths are relative to root).
// For mono-repo, it returns ["app/{service1}", "app/{service2}", ...].
func getProjectBaseDirs(project *Project) []string {
	if project.IsMono {
		var dirs []string
		for dir := range project.Dirs {
			if hasPrefix(dir, "app/") {
				parts := splitPath(dir)
				if len(parts) >= 2 {
					dirs = append(dirs, "app/"+parts[1])
				}
			}
		}
		return dirs
	}
	return []string{""}
}

// joinPath joins two path components with a slash separator.
func joinPath(base, sub string) string {
	if base == "" {
		return sub
	}
	return base + "/" + sub
}

// hasPrefix is a helper to check if a string has a given prefix.
func hasPrefix(s, prefix string) bool {
	return len(s) >= len(prefix) && s[:len(prefix)] == prefix
}

// splitPath splits a slash-separated path into its components.
func splitPath(path string) []string {
	var parts []string
	current := ""
	for _, ch := range path {
		if ch == '/' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}
