// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// rules_module.go checks the go.mod file for required module configuration,
// including module name, GoFrame dependency, and minimum Go version.

package check

import (
	"context"
	"strings"
)

// ModuleRule checks go.mod for required dependencies and configuration.
type ModuleRule struct {
	BaseRule
}

// NewModuleRule creates a new ModuleRule.
func NewModuleRule() *ModuleRule {
	return &ModuleRule{
		BaseRule: BaseRule{
			RuleID:          "MODULE",
			RuleName:        "Module Configuration",
			RuleDescription: "Checks go.mod for required module configuration and dependencies",
			RuleSeverity:    SeverityError,
		},
	}
}

// Run executes module configuration checks against the project.
func (r *ModuleRule) Run(_ context.Context, project *Project) []*Violation {
	var violations []*Violation

	goModContent := project.ReadFile("go.mod")
	if goModContent == "" {
		violations = append(violations, &Violation{
			RuleID:     "MOD-001",
			Severity:   SeverityError,
			Message:    "go.mod file not found or empty",
			Suggestion: "Run 'go mod init <module-name>' to create go.mod",
		})
		return violations
	}

	// MOD-002: Check module name exists.
	if project.ModuleName == "" {
		violations = append(violations, &Violation{
			RuleID:     "MOD-002",
			Severity:   SeverityError,
			Message:    "Module name not found in go.mod",
			Suggestion: "Ensure go.mod contains a 'module <name>' directive",
		})
	}

	// MOD-003: Check dependency on github.com/gogf/gf/v2.
	if !r.hasGoFrameDependency(goModContent) {
		violations = append(violations, &Violation{
			RuleID:     "MOD-003",
			Severity:   SeverityError,
			Message:    "go.mod does not depend on github.com/gogf/gf/v2",
			Suggestion: "Run 'go get github.com/gogf/gf/v2' to add the GoFrame dependency",
		})
	}

	// MOD-004: Check minimum Go version (>= 1.23).
	goVersion := r.extractGoVersion(goModContent)
	if goVersion != "" && !r.isGoVersionValid(goVersion) {
		violations = append(violations, &Violation{
			RuleID:     "MOD-004",
			Severity:   SeverityWarning,
			Message:    "Go version " + goVersion + " in go.mod is below recommended 1.23",
			Suggestion: "Update the 'go' directive in go.mod to 'go 1.23' or higher",
		})
	}

	return violations
}

// hasGoFrameDependency checks if go.mod content contains a direct dependency on gf/v2.
func (r *ModuleRule) hasGoFrameDependency(content string) bool {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.Contains(line, "github.com/gogf/gf/v2") && !strings.Contains(line, "// indirect") {
			return true
		}
	}
	return false
}

// extractGoVersion extracts the Go version from go.mod content.
func (r *ModuleRule) extractGoVersion(content string) string {
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "go ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "go "))
		}
	}
	return ""
}

// isGoVersionValid checks if the Go version meets the minimum requirement (>= 1.23).
func (r *ModuleRule) isGoVersionValid(version string) bool {
	// Simple check: version should start with "1.2" and the minor should be >= 23.
	// Handles formats like "1.23", "1.23.0", "1.24", etc.
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return true // can't parse, don't report
	}
	major := parts[0]
	minor := parts[1]
	if major != "1" {
		return true // not Go 1.x, can't compare
	}
	// Remove any suffix from minor (e.g., "23rc1").
	for i, ch := range minor {
		if ch < '0' || ch > '9' {
			minor = minor[:i]
			break
		}
	}
	minorNum := 0
	for _, ch := range minor {
		minorNum = minorNum*10 + int(ch-'0')
	}
	return minorNum >= 23
}
