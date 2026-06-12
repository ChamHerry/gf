// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// rules_dao.go checks DAO (Data Access Object) file patterns in projects
// that use gf gen dao. It verifies that DAO files follow the expected
// naming and structure conventions when they exist.

package check

import (
	"context"
	"go/ast"
	"strings"
)

// DAORule checks DAO file naming and structure conventions.
type DAORule struct {
	BaseRule
}

// NewDAORule creates a new DAORule.
func NewDAORule() *DAORule {
	return &DAORule{
		BaseRule: BaseRule{
			RuleID:          "DAO",
			RuleName:        "DAO Patterns",
			RuleDescription: "Checks DAO file naming and structure conventions for gf gen dao generated files",
			RuleSeverity:    SeverityWarning,
		},
	}
}

// Run executes DAO pattern checks against the project.
func (r *DAORule) Run(ctx context.Context, project *Project) []*Violation {
	var violations []*Violation

	for _, baseDir := range getProjectBaseDirs(project) {
		daoDir := joinPath(baseDir, "internal/dao")
		files := project.FilesInDirRecursive(daoDir)

		for _, f := range files {
			if f.AST == nil {
				continue
			}
			// Only check files directly in the dao/ directory (not internal/ subdirectory).
			dir := pathDir(f.Path)
			if dir != daoDir {
				continue
			}
			violations = append(violations, r.checkDAOFile(f)...)
		}
	}

	return violations
}

// checkDAOFile inspects a single DAO file for naming conventions.
func (r *DAORule) checkDAOFile(f *GoFile) []*Violation {
	var violations []*Violation

	// Check that the file defines at least one DAO struct or variable
	// matching the file name pattern (e.g., user.go should define UserDao).
	expectedSuffix := "Dao"
	fileBaseName := pathBase(f.Path)
	// Remove .go extension and _internal suffix.
	fileBaseName = strings.TrimSuffix(fileBaseName, ".go")

	// Only check top-level type declarations (not local types defined inside
	// function bodies, which are implementation details, not DAO structs).
	for _, decl := range f.AST.Decls {
		genDecl, ok := decl.(*ast.GenDecl)
		if !ok {
			continue
		}
		for _, spec := range genDecl.Specs {
			typeSpec, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			// Check that DAO struct names end with "Dao".
			if _, ok := typeSpec.Type.(*ast.StructType); ok {
				typeName := typeSpec.Name.Name
				if !strings.HasSuffix(typeName, expectedSuffix) {
					line := f.FileSet.Position(typeSpec.Pos()).Line
					violations = append(violations, &Violation{
						RuleID:     "CODE-DAO-001",
						Severity:   SeverityWarning,
						Message:    "DAO struct '" + typeName + "' should end with 'Dao' suffix",
						FilePath:   f.Path,
						Line:       line,
						Suggestion: "Rename the struct to '" + typeName + "Dao' to follow the DAO naming convention",
					})
				}
			}
		}
	}

	return violations
}

// pathDir returns the directory portion of a slash-separated path.
func pathDir(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return ""
	}
	return path[:idx]
}

// pathBase returns the last component of a slash-separated path.
func pathBase(path string) string {
	idx := strings.LastIndex(path, "/")
	if idx < 0 {
		return path
	}
	return path[idx+1:]
}
