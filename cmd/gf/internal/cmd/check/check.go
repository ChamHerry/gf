// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// Package check implements the project convention checking engine
// for the GoFrame CLI tool. It provides a rule-based engine that scans
// a GoFrame project directory and reports any violations of framework
// conventions, including directory structure, API definitions, controller
// patterns, layer dependencies, and generated file protections.
package check

import (
	"context"
	"fmt"
)

// EngineOptions configures the behavior of the check engine.
type EngineOptions struct {
	// Strict escalates warning-level violations to error-level.
	Strict bool
	// SkipRules is a list of rule IDs to skip during checking.
	SkipRules []string
}

// Engine orchestrates rule execution against a scanned project.
type Engine struct {
	project *Project
	options EngineOptions
	rules   []Rule
	skipSet map[string]bool
}

// NewEngine creates a new check engine for the given project root.
// It scans the project directory and prepares the engine for rule registration.
func NewEngine(rootPath string, options EngineOptions) (*Engine, error) {
	project, err := ScanProject(rootPath)
	if err != nil {
		return nil, fmt.Errorf("scan project failed: %w", err)
	}

	skipSet := make(map[string]bool)
	for _, id := range options.SkipRules {
		skipSet[id] = true
	}

	return &Engine{
		project: project,
		options: options,
		skipSet: skipSet,
	}, nil
}

// RegisterRule adds a rule to the engine for execution.
func (e *Engine) RegisterRule(rule Rule) {
	e.rules = append(e.rules, rule)
}

// Run executes all registered rules and returns a report of violations.
// Rules whose IDs appear in the SkipRules option are not executed.
// In strict mode, warning-level violations are escalated to error-level.
func (e *Engine) Run(ctx context.Context) *Report {
	var violations []*Violation

	for _, rule := range e.rules {
		// Check for context cancellation between rules.
		if ctx.Err() != nil {
			break
		}
		// Skip rules in the skip list.
		if e.skipSet[rule.ID()] {
			continue
		}
		// Execute the rule and collect violations.
		ruleViolations := rule.Run(ctx, e.project)
		// In strict mode, escalate warning violations to error.
		if e.options.Strict {
			for _, v := range ruleViolations {
				if v.Severity == SeverityWarning {
					v.Severity = SeverityError
				}
			}
		}
		violations = append(violations, ruleViolations...)
	}

	return NewReport(e.project.RootPath, violations)
}

// Project returns the scanned project associated with this engine.
func (e *Engine) Project() *Project {
	return e.project
}
