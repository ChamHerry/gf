// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// check_result.go defines the Violation and Report types, along with
// text and JSON formatters for presenting check results.

package check

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

// Violation represents a single rule violation found during project checking.
type Violation struct {
	// RuleID is the identifier of the rule that produced this violation (e.g., "DIR-001").
	RuleID string `json:"rule_id"`
	// Severity indicates the severity level of this violation.
	Severity Severity `json:"severity"`
	// Message describes what was found wrong.
	Message string `json:"message"`
	// FilePath is the relative path of the file where the violation was found.
	FilePath string `json:"file_path,omitempty"`
	// Line is the line number where the violation was found (0 if not applicable).
	Line int `json:"line,omitempty"`
	// Suggestion provides a recommended fix for the violation.
	Suggestion string `json:"suggestion,omitempty"`
}

// Report holds the aggregated results of all rule checks for a project.
type Report struct {
	// ProjectPath is the root path of the checked project.
	ProjectPath string `json:"project_path"`
	// Violations is the list of all violations found, sorted by severity.
	Violations []*Violation `json:"violations"`
	// ErrorCount is the number of error-level violations.
	ErrorCount int `json:"error_count"`
	// WarningCount is the number of warning-level violations.
	WarningCount int `json:"warning_count"`
	// InfoCount is the number of info-level violations.
	InfoCount int `json:"info_count"`
}

// NewReport creates a Report from a project path and a list of violations.
// It calculates severity counts and sorts violations by severity (error first).
func NewReport(projectPath string, violations []*Violation) *Report {
	report := &Report{
		ProjectPath: projectPath,
		Violations:  violations,
	}
	for _, v := range violations {
		switch v.Severity {
		case SeverityError:
			report.ErrorCount++
		case SeverityWarning:
			report.WarningCount++
		case SeverityInfo:
			report.InfoCount++
		}
	}
	sortViolations(report.Violations)
	return report
}

// HasErrors returns true if the report contains any error-level violations.
func (r *Report) HasErrors() bool {
	return r.ErrorCount > 0
}

// FormatText formats the report as a human-readable text string.
func FormatText(report *Report) string {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("Project: %s\n\n", report.ProjectPath))

	if len(report.Violations) == 0 {
		buf.WriteString("No violations found. Project complies with GoFrame conventions.\n")
		return buf.String()
	}

	for _, v := range report.Violations {
		location := v.FilePath
		if v.Line > 0 {
			location = fmt.Sprintf("%s:%d", v.FilePath, v.Line)
		}
		if location == "" {
			location = "(project root)"
		}
		buf.WriteString(fmt.Sprintf("[%s] %s: %s\n", strings.ToUpper(string(v.Severity)), v.RuleID, v.Message))
		buf.WriteString(fmt.Sprintf("  Location: %s\n", location))
		if v.Suggestion != "" {
			buf.WriteString(fmt.Sprintf("  Suggestion: %s\n", v.Suggestion))
		}
		buf.WriteString("\n")
	}

	buf.WriteString(fmt.Sprintf("Summary: %d error(s), %d warning(s), %d info(s)\n",
		report.ErrorCount, report.WarningCount, report.InfoCount))

	return buf.String()
}

// FormatJSON formats the report as a JSON string.
func FormatJSON(report *Report) (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// sortViolations sorts violations by severity descending (error first), then by file path.
func sortViolations(violations []*Violation) {
	sort.Slice(violations, func(i, j int) bool {
		si, sj := severityOrder[violations[i].Severity], severityOrder[violations[j].Severity]
		if si != sj {
			return si > sj
		}
		return violations[i].FilePath < violations[j].FilePath
	})
}
