package testutils

import (
	"os"
	"path/filepath"
	"testing"
)

// PatternTestCase defines a test case for file pattern matching
type PatternTestCase struct {
	Name           string            // Test case name
	Description    string            // Description of what's being tested
	Files          map[string]string // path -> content (files to create)
	ShouldMatch    []string          // files that should be detected
	ShouldNotMatch []string          // files that should NOT be detected
}

// GetStandardPatternCases returns test cases for standard .env patterns
func GetStandardPatternCases() []PatternTestCase {
	return []PatternTestCase{
		{
			Name:        "BasicEnvFile",
			Description: "Standard .env file at root",
			Files: map[string]string{
				".env": "DATABASE_URL=postgres://localhost/test",
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "EnvLocal",
			Description: ".env.local file for local overrides",
			Files: map[string]string{
				".env.local": "DEBUG=true",
			},
			ShouldMatch:    []string{".env.local"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "EnvDevelopment",
			Description: ".env.development for dev environment",
			Files: map[string]string{
				".env.development": "NODE_ENV=development",
			},
			ShouldMatch:    []string{".env.development"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "EnvProduction",
			Description: ".env.production for prod environment",
			Files: map[string]string{
				".env.production": "NODE_ENV=production",
			},
			ShouldMatch:    []string{".env.production"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "EnvStaging",
			Description: ".env.staging for staging environment",
			Files: map[string]string{
				".env.staging": "NODE_ENV=staging",
			},
			ShouldMatch:    []string{".env.staging"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "EnvTest",
			Description: ".env.test for test environment",
			Files: map[string]string{
				".env.test": "TEST_DB=memory",
			},
			ShouldMatch:    []string{".env.test"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "MultipleEnvFiles",
			Description: "Multiple standard .env files",
			Files: map[string]string{
				".env":             "BASE=value",
				".env.local":       "LOCAL=value",
				".env.development": "DEV=value",
				".env.production":  "PROD=value",
			},
			ShouldMatch:    []string{".env", ".env.local", ".env.development", ".env.production"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "EnvDevelopmentLocal",
			Description: ".env.development.local compound suffix",
			Files: map[string]string{
				".env.development.local": "LOCAL_DEV=true",
			},
			ShouldMatch:    []string{".env.development.local"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "EnvInSubdirectory",
			Description: ".env files in subdirectories",
			Files: map[string]string{
				"config/.env":       "CONFIG_VAR=value",
				"deploy/.env.prod":  "DEPLOY_PROD=true",
				"scripts/.env.test": "SCRIPT_TEST=value",
			},
			ShouldMatch:    []string{"config/.env", "deploy/.env.prod", "scripts/.env.test"},
			ShouldNotMatch: []string{},
		},
	}
}

// GetFalsePositiveCases returns test cases for files that should NOT match
func GetFalsePositiveCases() []PatternTestCase {
	return []PatternTestCase{
		{
			Name:        "NotEnvFile",
			Description: "File named 'not.env' should not match",
			Files: map[string]string{
				"not.env": "FAKE=value",
			},
			ShouldMatch:    []string{},
			ShouldNotMatch: []string{"not.env"},
		},
		{
			Name:        "JustEnv",
			Description: "File named 'env' without dot should not match",
			Files: map[string]string{
				"env": "NO_MATCH=value",
			},
			ShouldMatch:    []string{},
			ShouldNotMatch: []string{"env"},
		},
		{
			Name:        "DotEnvironment",
			Description: ".environment is not a standard pattern",
			Files: map[string]string{
				".environment": "NOT_ENV=value",
			},
			ShouldMatch:    []string{},
			ShouldNotMatch: []string{".environment"},
		},
		{
			Name:        "Envrc",
			Description: ".envrc (direnv) is not a standard .env file",
			Files: map[string]string{
				".envrc": "export VAR=value",
			},
			ShouldMatch:    []string{},
			ShouldNotMatch: []string{".envrc"},
		},
		{
			Name:        "MyEnv",
			Description: "myenv file should not match",
			Files: map[string]string{
				"myenv": "MY_VAR=value",
			},
			ShouldMatch:    []string{},
			ShouldNotMatch: []string{"myenv"},
		},
		{
			Name:        "UppercaseENV",
			Description: ".ENV (uppercase) should not match (regex is case-sensitive)",
			Files: map[string]string{
				".ENV": "UPPER=value",
			},
			ShouldMatch:    []string{},
			ShouldNotMatch: []string{".ENV"},
		},
		{
			Name:        "MixedCaseEnv",
			Description: ".Env (mixed case) should not match",
			Files: map[string]string{
				".Env": "MIXED=value",
			},
			ShouldMatch:    []string{},
			ShouldNotMatch: []string{".Env"},
		},
	}
}

// GetExcludedDirCases returns test cases for excluded directories
func GetExcludedDirCases() []PatternTestCase {
	return []PatternTestCase{
		{
			Name:        "NodeModulesExcluded",
			Description: ".env in node_modules should be excluded",
			Files: map[string]string{
				".env":                "ROOT=value",
				"node_modules/.env":   "EXCLUDED=value",
				"node_modules/a/.env": "ALSO_EXCLUDED=value",
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{"node_modules/.env", "node_modules/a/.env"},
		},
		{
			Name:        "GitDirExcluded",
			Description: ".env in .git should be excluded",
			Files: map[string]string{
				".env":      "ROOT=value",
				".git/.env": "GIT_EXCLUDED=value",
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{".git/.env"},
		},
		{
			Name:        "VendorExcluded",
			Description: ".env in vendor should be excluded",
			Files: map[string]string{
				".env":        "ROOT=value",
				"vendor/.env": "VENDOR_EXCLUDED=value",
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{"vendor/.env"},
		},
		{
			Name:        "MultipleExcludedDirs",
			Description: "Multiple excluded directories in one test",
			Files: map[string]string{
				".env":                         "ROOT=value",
				"config/.env":                  "CONFIG=value",
				"node_modules/.env":            "NM_EXCLUDED=value",
				"node_modules/package/.env":    "NM_PKG_EXCLUDED=value",
				".git/.env":                    "GIT_EXCLUDED=value",
				"vendor/.env":                  "VENDOR_EXCLUDED=value",
				"vendor/github.com/pkg/.env":   "VENDOR_DEEP_EXCLUDED=value",
			},
			ShouldMatch:    []string{".env", "config/.env"},
			ShouldNotMatch: []string{"node_modules/.env", ".git/.env", "vendor/.env"},
		},
	}
}

// GetDepthLimitCases returns test cases for depth limiting
func GetDepthLimitCases() []PatternTestCase {
	return []PatternTestCase{
		{
			Name:        "Depth1",
			Description: "Files at depth 1 (root)",
			Files: map[string]string{
				".env": "DEPTH0=value",
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "Depth2",
			Description: "Files at depth 2",
			Files: map[string]string{
				".env":   "DEPTH0=value",
				"a/.env": "DEPTH1=value",
			},
			ShouldMatch:    []string{".env", "a/.env"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "Depth3",
			Description: "Files at depth 3",
			Files: map[string]string{
				".env":     "DEPTH0=value",
				"a/.env":   "DEPTH1=value",
				"a/b/.env": "DEPTH2=value",
			},
			ShouldMatch:    []string{".env", "a/.env", "a/b/.env"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "BeyondDefaultDepth",
			Description: "Files beyond default depth (3) should be skipped",
			Files: map[string]string{
				".env":         "DEPTH0=value",
				"a/.env":       "DEPTH1=value",
				"a/b/.env":     "DEPTH2=value",
				"a/b/c/.env":   "DEPTH3=value",
				"a/b/c/d/.env": "DEPTH4=value", // Beyond depth 3
			},
			ShouldMatch:    []string{".env", "a/.env", "a/b/.env", "a/b/c/.env"},
			ShouldNotMatch: []string{"a/b/c/d/.env"},
		},
		{
			Name:        "VeryDeepNesting",
			Description: "Very deep nesting (5+ levels)",
			Files: map[string]string{
				".env":                 "ROOT=value",
				"a/b/c/d/e/.env":       "VERY_DEEP=value",
				"a/b/c/d/e/f/.env":     "EVEN_DEEPER=value",
				"a/b/c/d/e/f/g/h/.env": "WAY_TOO_DEEP=value",
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{"a/b/c/d/e/.env", "a/b/c/d/e/f/.env", "a/b/c/d/e/f/g/h/.env"},
		},
	}
}

// GetEdgeCaseCases returns test cases for edge cases
func GetEdgeCaseCases() []PatternTestCase {
	return []PatternTestCase{
		{
			Name:        "EmptyEnvFile",
			Description: "Empty .env file (0 bytes) should be included",
			Files: map[string]string{
				".env": "", // Empty file
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "SpecialCharsInName",
			Description: ".env files with special characters in suffix",
			Files: map[string]string{
				".env-backup":    "BACKUP=value",
				".env_old":       "OLD=value",
				".env.backup":    "BACKUP2=value",
				".env.2024-01":   "DATED=value",
			},
			ShouldMatch:    []string{".env-backup", ".env_old", ".env.backup", ".env.2024-01"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "UnicodeInSuffix",
			Description: ".env files with unicode in suffix",
			Files: map[string]string{
				".env.produccion":   "ES_PROD=value", // Spanish
				".env.entwicklung":  "DE_DEV=value",  // German
				".env.production":   "EN_PROD=value",
			},
			ShouldMatch:    []string{".env.produccion", ".env.entwicklung", ".env.production"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "LongSuffix",
			Description: ".env files with long suffix names",
			Files: map[string]string{
				".env.development.local.backup.old": "LONG=value",
				".env.production.secrets.encrypted": "VERY_LONG=value",
			},
			ShouldMatch:    []string{".env.development.local.backup.old", ".env.production.secrets.encrypted"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "WhitespaceContent",
			Description: ".env file with only whitespace",
			Files: map[string]string{
				".env": "   \n\t\n  ",
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{},
		},
		{
			Name:        "MixedValidAndInvalid",
			Description: "Mix of valid and invalid files",
			Files: map[string]string{
				".env":             "VALID=value",
				".env.local":       "VALID_LOCAL=value",
				"not.env":          "INVALID=value",
				"env":              "INVALID2=value",
				".environment":     "INVALID3=value",
				"config/.env":      "VALID_CONFIG=value",
				"config/env.json":  `{"invalid": true}`,
			},
			ShouldMatch:    []string{".env", ".env.local", "config/.env"},
			ShouldNotMatch: []string{"not.env", "env", ".environment", "config/env.json"},
		},
	}
}

// GetSymlinkCases returns test cases for symlink handling
func GetSymlinkCases() []PatternTestCase {
	return []PatternTestCase{
		{
			Name:        "SymlinkToEnvFile",
			Description: "Symlink pointing to .env file should be skipped",
			Files: map[string]string{
				".env.real": "REAL=value",
				// Symlink ".env" -> ".env.real" created separately
			},
			ShouldMatch:    []string{".env.real"},
			ShouldNotMatch: []string{".env"}, // symlink should be skipped
		},
		{
			Name:        "SymlinkToDirectory",
			Description: "Symlink pointing to directory should be skipped",
			Files: map[string]string{
				"real_config/.env": "CONFIG=value",
				// Symlink "config" -> "real_config" created separately
			},
			ShouldMatch:    []string{"real_config/.env"},
			ShouldNotMatch: []string{"config/.env"}, // symlink dir should be skipped
		},
	}
}

// GetBinaryDataCases returns test cases for binary data handling
func GetBinaryDataCases() []PatternTestCase {
	return []PatternTestCase{
		{
			Name:        "BinaryEnvFile",
			Description: ".env file with binary data should be skipped with warning",
			Files: map[string]string{
				".env":        "VALID=value",
				".env.binary": string([]byte{0x00, 0x01, 0x02, 0xFF, 0xFE}), // Binary content
			},
			ShouldMatch:    []string{".env"},
			ShouldNotMatch: []string{".env.binary"},
		},
	}
}

// CreatePatternTestDir creates a temporary directory with the specified files
func CreatePatternTestDir(t *testing.T, files map[string]string) string {
	t.Helper()

	tmpDir, err := os.MkdirTemp("", "goingenv-pattern-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}

	for path, content := range files {
		fullPath := filepath.Join(tmpDir, path)
		dir := filepath.Dir(fullPath)

		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatalf("Failed to create directory %s: %v", dir, err)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create file %s: %v", fullPath, err)
		}
	}

	return tmpDir
}

// CreateSymlink creates a symlink for testing
func CreateSymlink(t *testing.T, target, linkPath string) {
	t.Helper()

	// Ensure parent directory exists
	dir := filepath.Dir(linkPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.Symlink(target, linkPath); err != nil {
		t.Fatalf("Failed to create symlink %s -> %s: %v", linkPath, target, err)
	}
}

// CreateEmptyFile creates an empty file for testing
func CreateEmptyFile(t *testing.T, path string) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	file, err := os.Create(path)
	if err != nil {
		t.Fatalf("Failed to create empty file %s: %v", path, err)
	}
	file.Close()
}

// CreateBinaryFile creates a file with binary content for testing
func CreateBinaryFile(t *testing.T, path string, content []byte) {
	t.Helper()

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatalf("Failed to create directory %s: %v", dir, err)
	}

	if err := os.WriteFile(path, content, 0644); err != nil {
		t.Fatalf("Failed to create binary file %s: %v", path, err)
	}
}

// GetAllPatternCases returns all pattern test cases combined
func GetAllPatternCases() []PatternTestCase {
	var all []PatternTestCase
	all = append(all, GetStandardPatternCases()...)
	all = append(all, GetFalsePositiveCases()...)
	all = append(all, GetExcludedDirCases()...)
	all = append(all, GetEdgeCaseCases()...)
	return all
}

// SetupPatternTestCase creates a temp directory with the test case files
// Returns the directory path and a cleanup function
func SetupPatternTestCase(t *testing.T, tc PatternTestCase) (string, func()) {
	t.Helper()

	tmpDir := CreatePatternTestDir(t, tc.Files)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// SetupPatternTestCaseWithInit creates a temp directory with test files and initializes goingenv
func SetupPatternTestCaseWithInit(t *testing.T, tc PatternTestCase) (string, func()) {
	t.Helper()

	tmpDir := CreatePatternTestDir(t, tc.Files)
	CreateTempGoingEnvDir(t, tmpDir)

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}

// TestFixtures holds common test data
type TestFixtures struct {
	Password        string
	WrongPassword   string
	SampleEnvFiles  map[string]string
	SampleConfig    string
}

// GetTestFixtures returns common test fixtures
func GetTestFixtures() TestFixtures {
	return TestFixtures{
		Password:      "test-password-123",
		WrongPassword: "wrong-password-456",
		SampleEnvFiles: map[string]string{
			".env":             "DATABASE_URL=postgres://localhost/test\nAPI_KEY=secret123",
			".env.local":       "DEBUG=true\nLOG_LEVEL=debug",
			".env.development": "NODE_ENV=development\nAPI_URL=http://localhost:3000",
			".env.production":  "NODE_ENV=production\nAPI_URL=https://api.example.com",
		},
		SampleConfig: `{
			"default_depth": 3,
			"env_patterns": ["\\.env$", "\\.env\\..*$"],
			"exclude_patterns": ["node_modules/", "\\.git/", "vendor/"],
			"max_file_size": 10485760
		}`,
	}
}

// CreateSampleProject creates a sample project structure for testing
func CreateSampleProject(t *testing.T) (string, func()) {
	t.Helper()

	fixtures := GetTestFixtures()
	tmpDir := CreatePatternTestDir(t, fixtures.SampleEnvFiles)

	// Add some non-env files
	WriteTestFile(t, filepath.Join(tmpDir, "package.json"), `{"name": "test", "version": "1.0.0"}`)
	WriteTestFile(t, filepath.Join(tmpDir, "README.md"), "# Test Project")
	WriteTestFile(t, filepath.Join(tmpDir, "src/index.js"), "console.log('hello');")

	// Add excluded directories with env files
	WriteTestFile(t, filepath.Join(tmpDir, "node_modules/.env"), "NM_EXCLUDED=true")
	WriteTestFile(t, filepath.Join(tmpDir, "vendor/.env"), "VENDOR_EXCLUDED=true")

	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return tmpDir, cleanup
}
