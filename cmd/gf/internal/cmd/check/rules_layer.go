// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// rules_layer.go checks layer dependency rules to ensure that GoFrame's
// recommended architecture layers are respected: controllers should not
// import DAO packages directly, and API definitions should not reference
// internal model packages.

package check

import (
	"context"
	"strings"
)

// LayerRule checks layer dependency rules across project packages.
type LayerRule struct {
	BaseRule
}

// NewLayerRule creates a new LayerRule.
func NewLayerRule() *LayerRule {
	return &LayerRule{
		BaseRule: BaseRule{
			RuleID:          "LAYER",
			RuleName:        "Layer Dependencies",
			RuleDescription: "Checks that controller files do not import dao packages and API files do not import internal packages",
			RuleSeverity:    SeverityError,
		},
	}
}

// Run executes layer dependency checks against the project.
func (r *LayerRule) Run(_ context.Context, project *Project) []*Violation {
	var violations []*Violation

	for _, baseDir := range getProjectBaseDirs(project) {
		// Check controller files for forbidden imports.
		ctrlDir := joinPath(baseDir, "internal/controller")
		violations = append(violations, r.checkControllerImports(project, ctrlDir)...)

		// Check API files for forbidden imports.
		apiDir := joinPath(baseDir, "api")
		violations = append(violations, r.checkAPIImports(project, apiDir)...)
	}

	return violations
}

// checkControllerImports checks that controller files do not import
// dao or model/do packages directly.
func (r *LayerRule) checkControllerImports(project *Project, ctrlDir string) []*Violation {
	var violations []*Violation

	files := project.FilesInDirRecursive(ctrlDir)
	for _, f := range files {
		if f.AST == nil {
			continue
		}
		for _, imp := range f.AST.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			line := 0
			if imp.Path != nil {
				line = f.FileSet.Position(imp.Path.Pos()).Line
			}

			// LAYER-002: Controller must not import dao package.
			if strings.Contains(path, "/internal/dao") {
				violations = append(violations, &Violation{
					RuleID:     "LAYER-002",
					Severity:   SeverityError,
					Message:    "Controller file imports dao package directly: " + path,
					FilePath:   f.Path,
					Line:       line,
					Suggestion: "Controllers should call service interfaces instead of accessing dao directly. Use dependency injection via service registration.",
				})
			}

			// LAYER-005: Controller must not import model/do package.
			if strings.Contains(path, "/internal/model/do") {
				violations = append(violations, &Violation{
					RuleID:     "LAYER-005",
					Severity:   SeverityError,
					Message:    "Controller file imports model/do package: " + path,
					FilePath:   f.Path,
					Line:       line,
					Suggestion: "Controllers should not import model/do; use the Req/Res API types or service interfaces instead.",
				})
			}
		}
	}

	return violations
}

// checkAPIImports checks that API definition files do not import
// internal packages (they should only define request/response types).
func (r *LayerRule) checkAPIImports(project *Project, apiDir string) []*Violation {
	var violations []*Violation

	files := project.FilesInDirRecursive(apiDir)
	for _, f := range files {
		if f.AST == nil {
			continue
		}
		for _, imp := range f.AST.Imports {
			path := strings.Trim(imp.Path.Value, `"`)
			line := 0
			if imp.Path != nil {
				line = f.FileSet.Position(imp.Path.Pos()).Line
			}

			// LAYER-008: API files should not import internal packages.
			// API definitions should be self-contained with only standard/gf imports.
			if strings.Contains(path, "/internal/model") || strings.Contains(path, "/internal/service") {
				violations = append(violations, &Violation{
					RuleID:     "LAYER-008",
					Severity:   SeverityWarning,
					Message:    "API definition file imports internal package: " + path,
					FilePath:   f.Path,
					Line:       line,
					Suggestion: "API definition files should only contain request/response struct types and should not import internal packages.",
				})
			}
		}
	}

	return violations
}
