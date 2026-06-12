// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

// rules_config.go checks for the existence and basic structure of
// configuration files required by GoFrame projects.

package check

import "context"

// ConfigRule checks configuration file existence and basic structure.
type ConfigRule struct {
	BaseRule
}

// NewConfigRule creates a new ConfigRule.
func NewConfigRule() *ConfigRule {
	return &ConfigRule{
		BaseRule: BaseRule{
			RuleID:          "CONFIG",
			RuleName:        "Configuration Files",
			RuleDescription: "Checks for required configuration files (hack/config.yaml, manifest/config/config.yaml)",
			RuleSeverity:    SeverityWarning,
		},
	}
}

// Run executes configuration checks against the project.
func (r *ConfigRule) Run(ctx context.Context, project *Project) []*Violation {
	var violations []*Violation

	// CFG-001: hack/config.yaml for CLI tool configuration.
	if !project.FileExists("hack/config.yaml") {
		violations = append(violations, &Violation{
			RuleID:     "CFG-001",
			Severity:   SeverityInfo,
			Message:    "hack/config.yaml not found",
			Suggestion: "Create hack/config.yaml for GoFrame CLI tool configuration (e.g., gen.dao, gen.ctrl settings)",
		})
	}

	// CFG-002: manifest/config/config.yaml for runtime configuration.
	if !project.FileExists("manifest/config/config.yaml") {
		violations = append(violations, &Violation{
			RuleID:     "CFG-002",
			Severity:   SeverityWarning,
			Message:    "manifest/config/config.yaml not found",
			Suggestion: "Create manifest/config/config.yaml for application runtime configuration (server, database, logger)",
		})
	}

	return violations
}
