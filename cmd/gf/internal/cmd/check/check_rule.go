// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// check_rule.go defines the Rule interface, Severity type, and BaseRule
// struct that serve as the foundation for all convention check rules.

package check

import "context"

// Severity represents the severity level of a rule violation.
type Severity string

const (
	// SeverityError indicates a critical violation that breaks GoFrame conventions.
	SeverityError Severity = "error"
	// SeverityWarning indicates a potential issue that may cause problems.
	SeverityWarning Severity = "warning"
	// SeverityInfo indicates a suggestion or informational note.
	SeverityInfo Severity = "info"
)

// severityOrder maps severity levels to numeric values for comparison.
var severityOrder = map[Severity]int{
	SeverityError:   3,
	SeverityWarning: 2,
	SeverityInfo:    1,
}

// AtLeast returns true if this severity is greater than or equal to the other.
func (s Severity) AtLeast(other Severity) bool {
	return severityOrder[s] >= severityOrder[other]
}

// Rule defines the interface that all convention check rules must implement.
// Each rule inspects the project and reports any violations found.
type Rule interface {
	// ID returns the unique identifier of the rule category (e.g., "DIR", "API").
	ID() string
	// Name returns a human-readable name of the rule.
	Name() string
	// Description returns a brief description of what the rule checks.
	Description() string
	// Severity returns the default severity level of this rule's violations.
	Severity() Severity
	// Run executes the rule against the project and returns any violations found.
	Run(ctx context.Context, project *Project) []*Violation
}

// BaseRule provides a base implementation of the Rule interface methods.
// Concrete rule structs embed BaseRule and only need to implement the Run method.
type BaseRule struct {
	RuleID          string
	RuleName        string
	RuleDescription string
	RuleSeverity    Severity
}

// ID returns the unique identifier of the rule.
func (b *BaseRule) ID() string { return b.RuleID }

// Name returns the human-readable name of the rule.
func (b *BaseRule) Name() string { return b.RuleName }

// Description returns a brief description of what the rule checks.
func (b *BaseRule) Description() string { return b.RuleDescription }

// Severity returns the default severity level of this rule's violations.
func (b *BaseRule) Severity() Severity { return b.RuleSeverity }
