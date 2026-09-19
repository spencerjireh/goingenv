package config

import (
	"strings"
	"testing"
)

func TestValidateEnvName(t *testing.T) {
	cases := []struct {
		name    string
		wantErr bool
	}{
		{"", false},
		{"prod", false},
		{"prod-eu", false},
		{"prod_eu", false},
		{"1staging", false},
		{"Prod", true},
		{"-prod", true},
		{"prod eu", true},
		{"prod/eu", true},
		{"archive", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := ValidateEnvName(tc.name)
			if (err != nil) != tc.wantErr {
				t.Errorf("ValidateEnvName(%q) = %v, wantErr %v", tc.name, err, tc.wantErr)
			}
		})
	}
}

func TestArchivePaths(t *testing.T) {
	if got := ArchivePrefix(""); got != "archive-" {
		t.Errorf("ArchivePrefix(\"\") = %q", got)
	}
	if got := ArchivePrefix("prod"); got != "prod-" {
		t.Errorf("ArchivePrefix(prod) = %q", got)
	}
	p := GetArchivePath("prod")
	if !strings.HasPrefix(p, ".goingenv/prod-") || !strings.HasSuffix(p, ".enc") {
		t.Errorf("GetArchivePath(prod) = %q", p)
	}
	if d := GetDefaultArchivePath(); !strings.HasPrefix(d, ".goingenv/archive-") {
		t.Errorf("GetDefaultArchivePath() = %q", d)
	}
}
