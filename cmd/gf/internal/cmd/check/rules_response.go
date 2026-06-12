// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// you can obtain one at https://github.com/gogf/gf.

// rules_response.go checks that the project registers a unified HTTP
// response middleware, either the built-in ghttp.MiddlewareHandlerResponse
// or a custom implementation whose name matches "Middleware.*Response".
// This encourages consistent API response structures ({code, message, data})
// across the entire project.

package check

import (
	"context"
	"regexp"
)

// responseMiddlewarePattern matches common response middleware names including
// the built-in MiddlewareHandlerResponse and custom implementations like
// ResponseHandler, MiddlewareResponse, etc.
var responseMiddlewarePattern = regexp.MustCompile(`(?i)(Middleware\w*Response|ResponseHandler)`)

// ResponseRule checks that the project uses a unified response middleware.
type ResponseRule struct {
	BaseRule
}

// NewResponseRule creates a new ResponseRule.
func NewResponseRule() *ResponseRule {
	return &ResponseRule{
		BaseRule: BaseRule{
			RuleID:          "RESP",
			RuleName:        "Response Middleware",
			RuleDescription: "Checks that the project registers a unified HTTP response middleware",
			RuleSeverity:    SeverityInfo,
		},
	}
}

// Run executes the response middleware check against the project.
func (r *ResponseRule) Run(_ context.Context, project *Project) []*Violation {
	var violations []*Violation

	for _, baseDir := range getProjectBaseDirs(project) {
		if r.hasResponseMiddleware(project, baseDir) {
			return violations
		}
	}

	// No response middleware found in any base directory.
	violations = append(violations, &Violation{
		RuleID:     "CODE-RESP-001",
		Severity:   SeverityInfo,
		Message:    "No unified response middleware detected (e.g., ghttp.MiddlewareHandlerResponse)",
		Suggestion: "Register ghttp.MiddlewareHandlerResponse or a custom response middleware for consistent API response format ({code, message, data})",
	})

	return violations
}

// hasResponseMiddleware checks whether the project registers a response
// middleware by scanning Go source files under the controller and cmd
// directories for known middleware name patterns.
func (r *ResponseRule) hasResponseMiddleware(project *Project, baseDir string) bool {
	// Directories where middleware registration typically occurs.
	checkDirs := []string{
		joinPath(baseDir, "internal/cmd"),
		joinPath(baseDir, "internal/controller"),
		joinPath(baseDir, "internal/logic/middleware"),
		joinPath(baseDir, "internal/service"),
	}

	for _, dir := range checkDirs {
		files := project.FilesInDirRecursive(dir)
		for _, f := range files {
			if f.Content == "" {
				continue
			}
			// The regex matches both the built-in MiddlewareHandlerResponse
			// and custom patterns like ResponseHandler, MiddlewareResponse.
			if responseMiddlewarePattern.MatchString(f.Content) {
				return true
			}
		}
	}

	return false
}
