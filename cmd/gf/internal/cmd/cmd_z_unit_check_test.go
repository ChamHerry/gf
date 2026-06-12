// Copyright GoFrame gf Author(https://goframe.org). All Rights Reserved.
//
// This Source Code Form is subject to the terms of the MIT License.
// If a copy of the MIT was not distributed with this file,
// You can obtain one at https://github.com/gogf/gf.

package cmd

import (
	"context"
	"regexp"
	"testing"

	"github.com/gogf/gf/v2/test/gtest"
	"github.com/gogf/gf/v2/text/gstr"

	"github.com/gogf/gf/cmd/gf/v2/internal/cmd/check"
)

// registerAllCheckRules registers all built-in rules on the engine,
// mirroring the registration in cCheck.Index.
func registerAllCheckRules(engine *check.Engine) {
	engine.RegisterRule(check.NewDirectoryRule())
	engine.RegisterRule(check.NewAPIRule())
	engine.RegisterRule(check.NewControllerRule())
	engine.RegisterRule(check.NewLayerRule())
	engine.RegisterRule(check.NewModuleRule())
	engine.RegisterRule(check.NewConfigRule())
	engine.RegisterRule(check.NewGeneratedFileRule())
	engine.RegisterRule(check.NewDAORule())
}

// Test_Check_ValidProject runs all rules against the valid test project
// and verifies that no error-level or warning-level violations are produced.
func Test_Check_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		registerAllCheckRules(engine)

		report := engine.Run(context.Background())

		// The valid project should have zero errors and zero warnings.
		t.Assert(report.ErrorCount, 0)
		t.Assert(report.WarningCount, 0)
		// Info-level suggestions are acceptable (e.g., missing resource/ directory).
		t.AssertGT(report.InfoCount, 0)
	})
}

// Test_Check_InvalidProject runs all rules against the invalid test project
// and verifies that expected violations are produced.
func Test_Check_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		registerAllCheckRules(engine)

		report := engine.Run(context.Background())

		// The invalid project should have multiple error-level violations.
		t.AssertGT(report.ErrorCount, 0)
		t.AssertGT(report.WarningCount, 0)

		// Collect all rule IDs that produced violations.
		ruleIDs := make(map[string]bool)
		for _, v := range report.Violations {
			ruleIDs[v.RuleID] = true
		}

		// Verify key violations are detected.
		t.Assert(ruleIDs["CODE-META-001"], true)  // Missing g.Meta in Req struct
		t.Assert(ruleIDs["CODE-CTRL-001"], true)  // No ControllerV{N} struct
		t.Assert(ruleIDs["LAYER-002"], true)      // Controller imports dao
		t.Assert(ruleIDs["MOD-003"], true)        // No gf/v2 dependency
	})
}

// Test_Check_StrictMode verifies that warning-level violations are escalated
// to error-level when strict mode is enabled.
func Test_Check_StrictMode(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		// Run without strict mode to get baseline counts.
		engineNormal, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		registerAllCheckRules(engineNormal)
		reportNormal := engineNormal.Run(context.Background())

		// Run with strict mode.
		engineStrict, err := check.NewEngine(projectPath, check.EngineOptions{
			Strict: true,
		})
		t.AssertNil(err)
		registerAllCheckRules(engineStrict)
		reportStrict := engineStrict.Run(context.Background())

		// Strict mode should have zero warnings (they become errors).
		t.Assert(reportStrict.WarningCount, 0)

		// Strict mode should have more errors than normal mode.
		t.AssertGT(reportStrict.ErrorCount, reportNormal.ErrorCount)

		// Total violation count should remain the same.
		var totalNormal = reportNormal.ErrorCount + reportNormal.WarningCount + reportNormal.InfoCount
		var totalStrict = reportStrict.ErrorCount + reportStrict.WarningCount + reportStrict.InfoCount
		t.Assert(totalStrict, totalNormal)
	})
}

// Test_Check_SkipRules verifies that rules in the SkipRules list are not executed.
func Test_Check_SkipRules(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		// Run without skipping to get baseline.
		engineFull, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		registerAllCheckRules(engineFull)
		reportFull := engineFull.Run(context.Background())

		// Run with DIR and MODULE rules skipped.
		engineSkipped, err := check.NewEngine(projectPath, check.EngineOptions{
			SkipRules: []string{"DIR", "MODULE"},
		})
		t.AssertNil(err)
		registerAllCheckRules(engineSkipped)
		reportSkipped := engineSkipped.Run(context.Background())

		// Skipped report should have fewer violations.
		t.AssertLT(
			len(reportSkipped.Violations),
			len(reportFull.Violations),
		)

		// Verify that no DIR-* or MOD-* violations appear in the skipped report.
		for _, v := range reportSkipped.Violations {
			rulePrefix := v.RuleID
			if len(rulePrefix) >= 3 {
				rulePrefix = rulePrefix[:3]
			}
			t.AssertNE(rulePrefix, "DIR")
			t.AssertNE(rulePrefix, "MOD")
		}
	})
}

// Test_Check_ReportFormatText verifies that text formatting produces readable output.
func Test_Check_ReportFormatText(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		registerAllCheckRules(engine)
		report := engine.Run(context.Background())

		text := check.FormatText(report)
		t.AssertNE(text, "")
		// Text output should contain the project path.
		t.Assert(gstr.Contains(text, "Project:"), true)
		// Should contain summary line.
		t.Assert(gstr.Contains(text, "Summary:"), true)
	})
}

// Test_Check_ReportFormatText_NoViolations verifies text output for a clean project.
func Test_Check_ReportFormatText_NoViolations(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		report := check.NewReport("/fake/path", nil)
		text := check.FormatText(report)
		t.AssertNE(text, "")
		t.Assert(gstr.Contains(text, "No violations found"), true)
	})
}

// Test_Check_ReportFormatJSON verifies that JSON formatting produces valid JSON.
func Test_Check_ReportFormatJSON(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		registerAllCheckRules(engine)
		report := engine.Run(context.Background())

		jsonStr, err := check.FormatJSON(report)
		t.AssertNil(err)
		t.AssertNE(jsonStr, "")
		// JSON should contain expected fields.
		t.Assert(gstr.Contains(jsonStr, `"project_path"`), true)
		t.Assert(gstr.Contains(jsonStr, `"violations"`), true)
		t.Assert(gstr.Contains(jsonStr, `"error_count"`), true)
		t.Assert(gstr.Contains(jsonStr, `"warning_count"`), true)
		t.Assert(gstr.Contains(jsonStr, `"info_count"`), true)
	})
}

// Test_Check_ProjectScan verifies the project scanner correctly identifies
// directories, Go files, and module name.
func Test_Check_ProjectScan(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		project, err := check.ScanProject(projectPath)
		t.AssertNil(err)

		// Module name should be extracted from go.mod.
		t.Assert(project.ModuleName, "github.com/test/valid-project")

		// Should not be detected as mono-repo.
		t.Assert(project.IsMono, false)

		// Key directories should be present.
		t.Assert(project.HasDir("api"), true)
		t.Assert(project.HasDir("api/hello/v1"), true)
		t.Assert(project.HasDir("internal/cmd"), true)
		t.Assert(project.HasDir("internal/controller/hello"), true)
		t.Assert(project.HasDir("internal/service"), true)

		// Key files should be present.
		t.Assert(project.HasFile("main.go"), true)
		t.Assert(project.HasFile("api/hello/v1/hello.go"), true)
		t.Assert(project.HasFile("internal/controller/hello/hello_v1_hello.go"), true)

		// Non-existent dir/file should return false.
		t.Assert(project.HasDir("nonexistent"), false)
		t.Assert(project.HasFile("nonexistent.go"), false)
	})
}

// Test_Check_ProjectScan_MonoRepo verifies that mono-repo detection works.
func Test_Check_ProjectScan_MonoRepo(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// The valid-project does not have app/ dir, so it's not mono.
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		project, err := check.ScanProject(projectPath)
		t.AssertNil(err)
		t.Assert(project.IsMono, false)
	})
}

// Test_Check_ProjectFilesInDir verifies the FilesInDir and FilesInDirRecursive methods.
func Test_Check_ProjectFilesInDir(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		project, err := check.ScanProject(projectPath)
		t.AssertNil(err)

		// FilesInDir: api/hello/v1 should have exactly one .go file.
		v1Files := project.FilesInDir("api/hello/v1")
		t.Assert(len(v1Files), 1)
		t.Assert(v1Files[0].Path, "api/hello/v1/hello.go")

		// FilesInDirRecursive: api should find files recursively.
		allApiFiles := project.FilesInDirRecursive("api")
		t.AssertGT(len(allApiFiles), 1)

		// FilesInDirRecursive: internal/controller should find controller files.
		ctrlFiles := project.FilesInDirRecursive("internal/controller")
		t.AssertGT(len(ctrlFiles), 1)
	})
}

// Test_Check_RuleIDs verifies that each rule has the correct category ID.
func Test_Check_RuleIDs(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var testCases = []struct {
			rule check.Rule
			id   string
		}{
			{check.NewDirectoryRule(), "DIR"},
			{check.NewAPIRule(), "API"},
			{check.NewControllerRule(), "CTRL"},
			{check.NewLayerRule(), "LAYER"},
			{check.NewModuleRule(), "MODULE"},
			{check.NewConfigRule(), "CONFIG"},
			{check.NewGeneratedFileRule(), "GEN"},
			{check.NewDAORule(), "DAO"},
		}

		for _, tc := range testCases {
			t.Assert(tc.rule.ID(), tc.id)
			t.AssertNE(tc.rule.Name(), "")
			t.AssertNE(tc.rule.Description(), "")
		}
	})
}

// Test_Check_DirectoryRule_ValidProject tests the directory rule in isolation
// against the valid project.
func Test_Check_DirectoryRule_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewDirectoryRule())

		report := engine.Run(context.Background())
		// Valid project has all required dirs; only resource/ is missing (info).
		t.AssertEQ(report.ErrorCount, 0)
		t.AssertEQ(report.WarningCount, 0)
	})
}

// Test_Check_DirectoryRule_InvalidProject tests the directory rule in isolation
// against the invalid project.
func Test_Check_DirectoryRule_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewDirectoryRule())

		report := engine.Run(context.Background())
		// Invalid project is missing several required directories.
		t.AssertGT(report.WarningCount, 0)
	})
}

// Test_Check_APIRule_ValidProject tests the API rule against the valid project.
func Test_Check_APIRule_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewAPIRule())

		report := engine.Run(context.Background())
		// Valid project has proper g.Meta tags.
		t.AssertEQ(len(report.Violations), 0)
	})
}

// Test_Check_APIRule_InvalidProject tests the API rule against the invalid project.
func Test_Check_APIRule_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewAPIRule())

		report := engine.Run(context.Background())
		// Invalid project's HelloReq has no g.Meta.
		t.AssertGT(report.ErrorCount, 0)

		var found bool
		for _, v := range report.Violations {
			if v.RuleID == "CODE-META-001" {
				found = true
			}
		}
		t.Assert(found, true)
	})
}

// Test_Check_ControllerRule_ValidProject tests the controller rule against the valid project.
func Test_Check_ControllerRule_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewControllerRule())

		report := engine.Run(context.Background())
		// Valid project has ControllerV1 and NewV1.
		t.AssertEQ(len(report.Violations), 0)
	})
}

// Test_Check_ControllerRule_InvalidProject tests the controller rule against the invalid project.
func Test_Check_ControllerRule_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewControllerRule())

		report := engine.Run(context.Background())
		// Invalid project has no ControllerV{N} struct.
		t.AssertGT(report.ErrorCount, 0)
	})
}

// Test_Check_LayerRule_InvalidProject tests the layer rule against the invalid project.
func Test_Check_LayerRule_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewLayerRule())

		report := engine.Run(context.Background())
		// Invalid project controller imports dao directly.
		t.AssertGT(report.ErrorCount, 0)

		var foundLayerViolation bool
		for _, v := range report.Violations {
			if v.RuleID == "LAYER-002" {
				foundLayerViolation = true
			}
		}
		t.Assert(foundLayerViolation, true)
	})
}

// Test_Check_ModuleRule_ValidProject tests the module rule against the valid project.
func Test_Check_ModuleRule_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewModuleRule())

		report := engine.Run(context.Background())
		// Valid project has gf/v2 dependency and go 1.23.
		t.AssertEQ(len(report.Violations), 0)
	})
}

// Test_Check_ModuleRule_InvalidProject tests the module rule against the invalid project.
func Test_Check_ModuleRule_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewModuleRule())

		report := engine.Run(context.Background())
		// Invalid project has no gf/v2 and go version < 1.23.
		t.AssertGT(report.ErrorCount, 0)
	})
}

// Test_Check_ConfigRule_ValidProject tests the config rule against the valid project.
func Test_Check_ConfigRule_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewConfigRule())

		report := engine.Run(context.Background())
		// Valid project has both config files.
		t.AssertEQ(len(report.Violations), 0)
	})
}

// Test_Check_ConfigRule_InvalidProject tests the config rule against the invalid project.
func Test_Check_ConfigRule_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewConfigRule())

		report := engine.Run(context.Background())
		// Invalid project is missing both config files.
		t.AssertGT(len(report.Violations), 0)
	})
}

// Test_Check_EngineProjectAccessor verifies the Engine.Project() accessor.
func Test_Check_EngineProjectAccessor(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)

		project := engine.Project()
		t.AssertNE(project, nil)
		t.Assert(project.ModuleName, "github.com/test/valid-project")
	})
}

// Test_Check_ReportHasErrors verifies the Report.HasErrors() method.
func Test_Check_ReportHasErrors(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// Report with no violations.
		cleanReport := check.NewReport("/fake", nil)
		t.Assert(cleanReport.HasErrors(), false)

		// Report with an error violation.
		errorReport := check.NewReport("/fake", []*check.Violation{
			{RuleID: "TEST", Severity: check.SeverityError, Message: "test error"},
		})
		t.Assert(errorReport.HasErrors(), true)

		// Report with only a warning violation.
		warningReport := check.NewReport("/fake", []*check.Violation{
			{RuleID: "TEST", Severity: check.SeverityWarning, Message: "test warning"},
		})
		t.Assert(warningReport.HasErrors(), false)
	})
}

// Test_Check_SeverityMethods tests Severity.AtLeast() comparison.
func Test_Check_SeverityMethods(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		t.Assert(check.SeverityError.AtLeast(check.SeverityWarning), true)
		t.Assert(check.SeverityError.AtLeast(check.SeverityError), true)
		t.Assert(check.SeverityWarning.AtLeast(check.SeverityError), false)
		t.Assert(check.SeverityInfo.AtLeast(check.SeverityWarning), false)
		t.Assert(check.SeverityWarning.AtLeast(check.SeverityInfo), true)
	})
}

// Test_Check_RuleSeverity tests that each rule returns its expected default severity.
func Test_Check_RuleSeverity(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		t.Assert(check.NewDirectoryRule().Severity(), check.SeverityError)
		t.Assert(check.NewAPIRule().Severity(), check.SeverityError)
		t.Assert(check.NewControllerRule().Severity(), check.SeverityError)
		t.Assert(check.NewLayerRule().Severity(), check.SeverityError)
		t.Assert(check.NewModuleRule().Severity(), check.SeverityError)
		t.Assert(check.NewConfigRule().Severity(), check.SeverityWarning)
		t.Assert(check.NewGeneratedFileRule().Severity(), check.SeverityWarning)
		t.Assert(check.NewDAORule().Severity(), check.SeverityWarning)
	})
}

// Test_Check_FilesInDirRecursiveRoot tests FilesInDirRecursive with root ".".
func Test_Check_FilesInDirRecursiveRoot(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		project, err := check.ScanProject(projectPath)
		t.AssertNil(err)

		// Root dir should return all Go files.
		allFiles := project.FilesInDirRecursive(".")
		t.AssertGT(len(allFiles), 3)

		// Empty string should also return all files.
		allFiles2 := project.FilesInDirRecursive("")
		t.Assert(len(allFiles2), len(allFiles))
	})
}

// Test_Check_GenRule_ValidProject tests the generated file rule against valid-project.
// The valid-project has DO NOT EDIT headers in api/hello/hello.go and controller hello_new.go.
func Test_Check_GenRule_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewGeneratedFileRule())

		report := engine.Run(context.Background())
		// Valid project generated files have DO NOT EDIT headers.
		t.AssertEQ(report.ErrorCount, 0)
	})
}

// Test_Check_GenRule_InvalidProject tests the generated file rule against invalid-project.
func Test_Check_GenRule_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewGeneratedFileRule())

		report := engine.Run(context.Background())
		// Invalid project has no generated files, so no violations.
		t.AssertEQ(report.ErrorCount, 0)
	})
}

// Test_CHECK_DAORule_ValidProject tests the DAO rule against valid-project.
func Test_Check_DAORule_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewDAORule())

		report := engine.Run(context.Background())
		// Valid project has DAO files with proper naming; no violations.
		// The outer dao file also has a local struct type inside a function
		// body, which must NOT trigger CODE-DAO-001.
		t.AssertEQ(len(report.Violations), 0)
	})
}

// Test_Check_ReadFile tests the Project.ReadFile method.
func Test_Check_ReadFile(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		project, err := check.ScanProject(projectPath)
		t.AssertNil(err)

		// go.mod should be readable.
		content := project.ReadFile("go.mod")
		t.Assert(gstr.Contains(content, "module github.com/test/valid-project"), true)

		// Non-existent file should return empty string.
		missing := project.ReadFile("nonexistent.txt")
		t.Assert(missing, "")
	})
}

// Test_Check_ScanProject_NonExistent tests scanning a non-existent directory.
func Test_Check_ScanProject_NonExistent(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		_, err := check.ScanProject("/nonexistent/path/that/does/not/exist")
		t.AssertNE(err, nil)
	})
}

// Test_Check_NewEngine_NonExistent tests engine creation with invalid path.
func Test_Check_NewEngine_NonExistent(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		_, err := check.NewEngine("/nonexistent/path/that/does/not/exist", check.EngineOptions{})
		t.AssertNE(err, nil)
	})
}

// Test_Check_MonoRepo_Detection tests that mono-repo structure is detected.
func Test_Check_MonoRepo_Detection(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "mono-project")
		)

		project, err := check.ScanProject(projectPath)
		t.AssertNil(err)

		// Should be detected as mono-repo (has app/ directory).
		t.Assert(project.IsMono, true)
		t.Assert(project.HasDir("app"), true)
		t.Assert(project.HasDir("app/user"), true)
		t.Assert(project.HasDir("app/user/internal"), true)
	})
}

// Test_Check_DirectoryRule_MonoRepo tests the directory rule on a mono-repo project.
func Test_Check_DirectoryRule_MonoRepo(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "mono-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewDirectoryRule())

		report := engine.Run(context.Background())
		// Mono-repo with app/user/ containing main.go and internal/ should
		// produce violations only for missing optional dirs (none are required
		// beyond main.go and internal/ in the mono-repo check path).
		// This test exercises the checkMonoRepo code path without asserting
		// specific violations.
		t.AssertNE(report, nil)
	})
}

// Test_Check_DAORule_WithDAOFiles tests the DAO rule against a project with actual DAO files.
func Test_Check_DAORule_WithDAOFiles(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewDAORule())

		report := engine.Run(context.Background())
		// Valid project has proper DAO files with DO NOT EDIT headers.
		t.AssertEQ(report.ErrorCount, 0)
	})
}

// Test_Check_GenRule_WithGeneratedFiles tests the generated file rule with actual generated files.
func Test_Check_GenRule_WithGeneratedFiles(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewGeneratedFileRule())

		report := engine.Run(context.Background())
		// Valid project has model/do, model/entity, dao/internal files with DO NOT EDIT.
		// Outer dao files use "auto-generated" header and are NOT checked.
		t.AssertEQ(report.ErrorCount, 0)
	})
}

// Test_Check_GenRule_OuterDaoNotFlagged verifies that outer DAO wrapper files
// (which use "auto-generated" header, not "DO NOT EDIT") are not flagged by GEN-003.
// Only internal/dao/internal/ files should be checked for DO NOT EDIT.
func Test_Check_GenRule_OuterDaoNotFlagged(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewGeneratedFileRule())

		report := engine.Run(context.Background())

		// No GEN-003 violations — the outer dao/hello.go has "auto-generated"
		// but NOT "DO NOT EDIT", and should not be flagged.
		for _, v := range report.Violations {
			t.AssertNE(v.RuleID, "GEN-003")
		}
	})
}

// Test_Check_DAORule_LocalTypeNotFlagged verifies that struct types defined
// inside function bodies are not flagged by CODE-DAO-001. Only top-level
// type declarations should be checked for DAO naming conventions.
func Test_Check_DAORule_LocalTypeNotFlagged(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewDAORule())

		report := engine.Run(context.Background())

		// The outer dao/hello.go defines a local `type Result struct` inside
		// CountByStatus(). This must NOT trigger CODE-DAO-001.
		t.AssertEQ(len(report.Violations), 0)
	})
}

// Test_Check_CmdErrorNoStackTrace verifies that the command returns a plain
// error (without stack trace) when violations are found. The error should
// signal exit code 1 for CI without noisy gerror stack output.
func Test_Check_CmdErrorNoStackTrace(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		c := cCheck{}
		projectPath := gtest.DataPath("check", "invalid-project")

		_, err := c.Index(context.Background(), cCheckInput{
			Path: projectPath,
		})
		// Should return an error (for exit code 1).
		t.AssertNE(err, nil)
		// Should NOT contain stack trace markers from gerror.
		t.Assert(gstr.Contains(err.Error(), "github.com/gogf"), false)
		t.Assert(gstr.Contains(err.Error(), ".go:"), false)
	})
}

// Test_Check_ResponseRule_ValidProject tests that the response middleware rule
// passes when the project uses ghttp.MiddlewareHandlerResponse.
func Test_Check_ResponseRule_ValidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "valid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewResponseRule())

		report := engine.Run(context.Background())
		// Valid project registers ghttp.MiddlewareHandlerResponse.
		t.AssertEQ(len(report.Violations), 0)
	})
}

// Test_Check_ResponseRule_InvalidProject tests that the response middleware rule
// reports CODE-RESP-001 when no response middleware is found.
func Test_Check_ResponseRule_InvalidProject(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		var (
			projectPath = gtest.DataPath("check", "invalid-project")
		)

		engine, err := check.NewEngine(projectPath, check.EngineOptions{})
		t.AssertNil(err)
		engine.RegisterRule(check.NewResponseRule())

		report := engine.Run(context.Background())
		// Invalid project has no response middleware.
		t.AssertEQ(len(report.Violations), 1)
		t.AssertEQ(report.Violations[0].RuleID, "CODE-RESP-001")
	})
}

// Test_Check_ResponseRule_CustomMiddleware tests that custom middleware
// names matching the pattern (e.g., ResponseHandler) are also detected.
func Test_Check_ResponseRule_CustomMiddleware(t *testing.T) {
	gtest.C(t, func(t *gtest.T) {
		// The regex pattern should match common custom response middleware names.
		pattern := regexp.MustCompile(`(?i)(Middleware\w*Response|ResponseHandler)`)
		t.AssertEQ(pattern.MatchString("ResponseHandler"), true)
		t.AssertEQ(pattern.MatchString("MiddlewareHandlerResponse"), true)
		t.AssertEQ(pattern.MatchString("MiddlewareResponse"), true)
		// Should not match unrelated middleware names.
		t.AssertEQ(pattern.MatchString("MiddlewareCORS"), false)
		t.AssertEQ(pattern.MatchString("MiddlewareAuth"), false)
	})
}
