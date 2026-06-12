// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package cmd

import (
	"context"
	"fmt"

	"github.com/gogf/gf/v2/errors/gerror"
	"github.com/gogf/gf/v2/frame/g"
	"github.com/gogf/gf/v2/os/gfile"
	"github.com/gogf/gf/v2/text/gstr"

	"github.com/gogf/gf/cmd/gf/v2/internal/cmd/check"
	"github.com/gogf/gf/cmd/gf/v2/internal/utility/mlog"
)

var (
	Check = cCheck{}
)

// cCheck defines the `gf check` command that validates whether a project
// complies with GoFrame directory structure, API definition, controller
// pattern, and layer dependency conventions.
type cCheck struct {
	g.Meta `name:"check" brief:"check project for GoFrame convention compliance" usage:"gf check" `
}

// cCheckInput defines the input parameters for the check command.
type cCheckInput struct {
	g.Meta     `name:"check"`
	Path       string `name:"path"   short:"p" brief:"directory path to check, default is current working directory"`
	Strict     bool   `name:"strict" short:"s" brief:"strict mode, escalate warnings to errors" orphan:"true"`
	SkipRules  string `name:"skip"            brief:"comma-separated rule IDs to skip (e.g., DIR,API)"`
	OutputJSON bool   `name:"json"            brief:"output results in JSON format" orphan:"true"`
}

// cCheckOutput defines the output of the check command (currently empty,
// results are printed directly via mlog).
type cCheckOutput struct{}

// Index executes the check command: scans the project, runs all registered
// rules, prints the report, and returns an error if any violations are found.
func (c cCheck) Index(ctx context.Context, in cCheckInput) (out *cCheckOutput, err error) {
	if in.Path == "" {
		in.Path = gfile.Pwd()
	}

	// Parse skip rules from comma-separated string.
	var skipRules []string
	if in.SkipRules != "" {
		skipRules = gstr.SplitAndTrim(in.SkipRules, ",")
	}

	// Create the check engine with the specified options.
	engine, err := check.NewEngine(in.Path, check.EngineOptions{
		Strict:    in.Strict,
		SkipRules: skipRules,
	})
	if err != nil {
		return nil, gerror.Newf("failed to scan project: %v", err)
	}

	// Register all built-in rules.
	engine.RegisterRule(check.NewDirectoryRule())
	engine.RegisterRule(check.NewAPIRule())
	engine.RegisterRule(check.NewControllerRule())
	engine.RegisterRule(check.NewLayerRule())
	engine.RegisterRule(check.NewModuleRule())
	engine.RegisterRule(check.NewConfigRule())
	engine.RegisterRule(check.NewGeneratedFileRule())
	engine.RegisterRule(check.NewDAORule())
	engine.RegisterRule(check.NewResponseRule())

	// Run all registered rules and collect violations.
	report := engine.Run(ctx)

	// Format and output the results.
	var output string
	if in.OutputJSON {
		output, err = check.FormatJSON(report)
		if err != nil {
			return nil, gerror.Wrap(err, "failed to format report as JSON")
		}
	} else {
		output = check.FormatText(report)
	}
	mlog.Print(output)

	// Return a plain error (no stack trace) if there are error-level violations.
	// The full report has already been printed above; this only signals exit code 1.
	if report.HasErrors() {
		return nil, fmt.Errorf(
			"check failed: %d error(s), %d warning(s), %d info(s)",
			report.ErrorCount, report.WarningCount, report.InfoCount,
		)
	}

	return
}
