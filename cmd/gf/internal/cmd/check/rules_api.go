// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// rules_api.go checks API definition files under api/ directories,
// verifying that request/response structs follow GoFrame naming conventions
// and that g.Meta struct tags contain required attributes (path, method).

package check

import (
	"context"
	"go/ast"
	"reflect"
	"strings"
)

// APIRule checks API definition files for naming and tag conventions.
type APIRule struct {
	BaseRule
}

// NewAPIRule creates a new APIRule.
func NewAPIRule() *APIRule {
	return &APIRule{
		BaseRule: BaseRule{
			RuleID:          "API",
			RuleName:        "API Definitions",
			RuleDescription: "Checks API request/response struct naming and g.Meta tag conventions",
			RuleSeverity:    SeverityError,
		},
	}
}

// Run executes API definition checks against the project.
func (r *APIRule) Run(_ context.Context, project *Project) []*Violation {
	var violations []*Violation

	for _, baseDir := range getProjectBaseDirs(project) {
		apiDir := joinPath(baseDir, "api")
		files := project.FilesInDirRecursive(apiDir)
		for _, f := range files {
			if f.AST == nil {
				continue
			}
			violations = append(violations, r.checkFile(f)...)
		}
	}

	return violations
}

// checkFile inspects a single API definition file for convention violations.
func (r *APIRule) checkFile(f *GoFile) []*Violation {
	var violations []*Violation

	ast.Inspect(f.AST, func(n ast.Node) bool {
		typeSpec, ok := n.(*ast.TypeSpec)
		if !ok {
			return true
		}

		structType, ok := typeSpec.Type.(*ast.StructType)
		if !ok {
			return true
		}

		structName := typeSpec.Name.Name
		line := f.FileSet.Position(typeSpec.Pos()).Line

		// Check Req structs for g.Meta tag conventions.
		if strings.HasSuffix(structName, "Req") {
			violations = append(violations, r.checkReqStruct(f.Path, line, structName, structType)...)
		}

		return true
	})

	return violations
}

// checkReqStruct validates that a Req struct has proper g.Meta tags.
func (r *APIRule) checkReqStruct(filePath string, line int, structName string, structType *ast.StructType) []*Violation {
	var violations []*Violation

	found, tagStr := findMetaField(structType)

	// CODE-META-001: Req struct must embed g.Meta.
	if !found {
		violations = append(violations, &Violation{
			RuleID:     "CODE-META-001",
			Severity:   SeverityError,
			Message:    structName + " does not embed g.Meta",
			FilePath:   filePath,
			Line:       line,
			Suggestion: "Add 'g.Meta' as an embedded field with path, method, and tags tags",
		})
		return violations
	}

	// Parse the g.Meta struct tag.
	tag := reflect.StructTag(tagStr)

	// CODE-META-002: g.Meta must have path tag.
	if tag.Get("path") == "" {
		violations = append(violations, &Violation{
			RuleID:     "CODE-META-002",
			Severity:   SeverityError,
			Message:    structName + " g.Meta is missing 'path' tag",
			FilePath:   filePath,
			Line:       line,
			Suggestion: "Add path tag to g.Meta, e.g.: g.Meta `path:\"/hello\"`",
		})
	}

	// CODE-META-003: g.Meta must have method tag.
	if tag.Get("method") == "" {
		violations = append(violations, &Violation{
			RuleID:     "CODE-META-003",
			Severity:   SeverityError,
			Message:    structName + " g.Meta is missing 'method' tag",
			FilePath:   filePath,
			Line:       line,
			Suggestion: "Add method tag to g.Meta, e.g.: g.Meta `method:\"get\"`",
		})
	}

	// CODE-META-004: g.Meta should have tags tag (warning).
	if tag.Get("tags") == "" {
		violations = append(violations, &Violation{
			RuleID:     "CODE-META-004",
			Severity:   SeverityWarning,
			Message:    structName + " g.Meta is missing 'tags' tag",
			FilePath:   filePath,
			Line:       line,
			Suggestion: "Add tags tag for API grouping, e.g.: g.Meta `tags:\"Hello\"`",
		})
	}

	// CODE-META-005: g.Meta should have summary tag (info).
	if tag.Get("summary") == "" {
		violations = append(violations, &Violation{
			RuleID:     "CODE-META-005",
			Severity:   SeverityInfo,
			Message:    structName + " g.Meta is missing 'summary' tag",
			FilePath:   filePath,
			Line:       line,
			Suggestion: "Add summary tag for API documentation, e.g.: g.Meta `summary:\"Description of the API\"`",
		})
	}

	return violations
}

// findMetaField searches for an embedded g.Meta field in a struct type
// and returns whether it was found and its raw tag value.
func findMetaField(structType *ast.StructType) (found bool, tag string) {
	if structType.Fields == nil {
		return false, ""
	}
	for _, field := range structType.Fields.List {
		// g.Meta is an embedded field of type *ast.SelectorExpr (g.Meta).
		selExpr, ok := field.Type.(*ast.SelectorExpr)
		if !ok {
			continue
		}
		ident, ok := selExpr.X.(*ast.Ident)
		if !ok {
			continue
		}
		if ident.Name == "g" && selExpr.Sel.Name == "Meta" {
			found = true
			if field.Tag != nil {
				tag = strings.Trim(field.Tag.Value, "`")
			}
			return
		}
	}
	return false, ""
}

// isAllDigits checks if a string consists entirely of digit characters.
func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, ch := range s {
		if ch < '0' || ch > '9' {
			return false
		}
	}
	return true
}
