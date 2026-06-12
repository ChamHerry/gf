// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// rules_controller.go checks controller implementation files under
// internal/controller/ directories, verifying that controllers follow
// the GoFrame ControllerV{N} naming pattern, have proper factory functions,
// and use structured parameters instead of raw *ghttp.Request.

package check

import (
	"context"
	"go/ast"
	"strings"
)

// ControllerRule checks controller files for naming and structure conventions.
type ControllerRule struct {
	BaseRule
}

// NewControllerRule creates a new ControllerRule.
func NewControllerRule() *ControllerRule {
	return &ControllerRule{
		BaseRule: BaseRule{
			RuleID:          "CTRL",
			RuleName:        "Controller Conventions",
			RuleDescription: "Checks controller naming (ControllerV{N}), factory functions (NewV{N}), and parameter patterns",
			RuleSeverity:    SeverityError,
		},
	}
}

// Run executes controller checks against the project.
func (r *ControllerRule) Run(ctx context.Context, project *Project) []*Violation {
	var violations []*Violation

	for _, baseDir := range getProjectBaseDirs(project) {
		ctrlDir := joinPath(baseDir, "internal/controller")
		files := project.FilesInDirRecursive(ctrlDir)
		if len(files) == 0 {
			continue
		}

		hasControllerV := false
		hasNewV := false

		for _, f := range files {
			if f.AST == nil {
				continue
			}
			fileViolations, fileHasCtrl, fileHasNew := r.checkFile(f)
			violations = append(violations, fileViolations...)
			if fileHasCtrl {
				hasControllerV = true
			}
			if fileHasNew {
				hasNewV = true
			}
		}

		// CODE-CTRL-001: At least one ControllerV{N} struct should exist.
		if !hasControllerV {
			violations = append(violations, &Violation{
				RuleID:     "CODE-CTRL-001",
				Severity:   SeverityError,
				Message:    "No ControllerV{N} struct found in " + ctrlDir,
				Suggestion: "Define a controller struct named ControllerV1 (or ControllerV2, etc.) matching the API version",
			})
		}

		// CODE-CTRL-002: At least one NewV{N} factory function should exist.
		if !hasNewV && hasControllerV {
			violations = append(violations, &Violation{
				RuleID:     "CODE-CTRL-002",
				Severity:   SeverityError,
				Message:    "No NewV{N}() factory function found in " + ctrlDir,
				Suggestion: "Define a factory function like 'func NewV1() IHelloV1 { return &ControllerV1{} }'",
			})
		}
	}

	return violations
}

// checkFile inspects a single controller file and returns violations,
// along with flags indicating whether ControllerV{N} and NewV{N} were found.
func (r *ControllerRule) checkFile(f *GoFile) (violations []*Violation, hasControllerV bool, hasNewV bool) {
	ast.Inspect(f.AST, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.TypeSpec:
			// Check for ControllerV{N} struct naming.
			if _, ok := node.Type.(*ast.StructType); ok {
				if isControllerVStruct(node.Name.Name) {
					hasControllerV = true
				}
			}

		case *ast.FuncDecl:
			// Check for NewV{N} factory function.
			if node.Recv == nil && isNewVFunc(node.Name.Name) {
				hasNewV = true
			}

			// CODE-STRUCT-001: Check that methods on ControllerV structs
			// do not use *ghttp.Request as a parameter.
			if node.Recv != nil && node.Type.Params != nil {
				if hasGHTTPRequest(node) {
					line := f.FileSet.Position(node.Pos()).Line
					violations = append(violations, &Violation{
						RuleID:     "CODE-STRUCT-001",
						Severity:   SeverityWarning,
						Message:    "Controller method uses *ghttp.Request parameter; use structured Req/Res parameters instead",
						FilePath:   f.Path,
						Line:       line,
						Suggestion: "Use GoFrame's structured API parameters (ctx, *v1.XxxReq) instead of *ghttp.Request",
					})
				}
			}
		}
		return true
	})
	return
}

// isControllerVStruct checks if a struct name matches the ControllerV{N} pattern.
func isControllerVStruct(name string) bool {
	if !strings.HasPrefix(name, "ControllerV") {
		return false
	}
	suffix := strings.TrimPrefix(name, "ControllerV")
	return isAllDigits(suffix)
}

// isNewVFunc checks if a function name matches the NewV{N} pattern.
func isNewVFunc(name string) bool {
	if !strings.HasPrefix(name, "NewV") {
		return false
	}
	suffix := strings.TrimPrefix(name, "NewV")
	return isAllDigits(suffix)
}

// hasGHTTPRequest checks if a function declaration has a *ghttp.Request parameter.
func hasGHTTPRequest(funcDecl *ast.FuncDecl) bool {
	if funcDecl.Type.Params == nil {
		return false
	}
	for _, param := range funcDecl.Type.Params.List {
		// Check for *ghttp.Request type.
		starExpr, ok := param.Type.(*ast.StarExpr)
		if !ok {
			continue
		}
		selExpr, ok := starExpr.X.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "ghttp" && selExpr.Sel.Name == "Request" {
			return true
		}
	}
	return false
}
