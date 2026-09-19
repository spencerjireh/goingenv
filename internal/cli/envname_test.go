package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"goingenv/pkg/types"
)

func TestArchivesForEnv(t *testing.T) {
	archives := []string{
		".goingenv/archive-20260101-000000.enc",
		".goingenv/prod-20260102-000000.enc",
		".goingenv/archive-20260103-000000.enc",
		".goingenv/prod-eu-20260104-000000.enc",
		".goingenv/prod-20260105-000000.enc",
		".goingenv/custom.enc",
		".goingenv/my-archive-20260106-000000.enc",
	}
	cases := []struct {
		env  string
		want []string
	}{
		{"", []string{".goingenv/archive-20260103-000000.enc", ".goingenv/archive-20260101-000000.enc"}},
		{"prod", []string{".goingenv/prod-20260105-000000.enc", ".goingenv/prod-20260102-000000.enc"}},
		{"prod-eu", []string{".goingenv/prod-eu-20260104-000000.enc"}},
		{"staging", nil},
	}
	for _, tc := range cases {
		t.Run(tc.env, func(t *testing.T) {
			if got := archivesForEnv(archives, tc.env); !reflect.DeepEqual(got, tc.want) {
				t.Errorf("archivesForEnv(%q) = %v, want %v", tc.env, got, tc.want)
			}
		})
	}
}

func TestPickArchive(t *testing.T) {
	app := &types.App{Archiver: &types.MockArchiver{
		GetAvailableArchivesFunc: func(string) ([]string, error) {
			return []string{
				".goingenv/archive-20260101-000000.enc",
				".goingenv/prod-20260102-000000.enc",
				".goingenv/zzz-custom.enc",
			}, nil
		},
	}}

	got, err := pickArchive(app, "", "")
	if err != nil || got != ".goingenv/archive-20260101-000000.enc" {
		t.Errorf("unnamed: got %q, %v", got, err)
	}
	got, err = pickArchive(app, "", "prod")
	if err != nil || got != ".goingenv/prod-20260102-000000.enc" {
		t.Errorf("prod: got %q, %v", got, err)
	}
	if _, stagingErr := pickArchive(app, "", "staging"); stagingErr == nil {
		t.Error("staging: expected an error")
	}
	got, err = pickArchive(app, "/explicit/path.enc", "prod")
	if err != nil || got != "/explicit/path.enc" {
		t.Errorf("explicit: got %q, %v", got, err)
	}
}

func TestResolveArchiveArg(t *testing.T) {
	dir := t.TempDir()
	t.Chdir(dir)
	if err := os.MkdirAll(".goingenv", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(".goingenv", "a.enc"), []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := resolveArchiveArg("a.enc"); got != filepath.Join(".goingenv", "a.enc") {
		t.Errorf("bare name: got %q", got)
	}
	if got := resolveArchiveArg("missing.enc"); got != "missing.enc" {
		t.Errorf("missing: got %q", got)
	}
}

func TestDescribeDecryptError(t *testing.T) {
	legacy := &types.ArchiveError{Operation: "list", Err: fmt.Errorf("wrap: %w", &types.CryptoError{Operation: "decrypt", Err: types.ErrLegacyArchive})}
	if got := describeDecryptError(legacy); !errors.Is(got, types.ErrLegacyArchive) || got.Error() != types.ErrLegacyArchive.Error() {
		t.Errorf("legacy: got %v", got)
	}
	other := &types.ArchiveError{Operation: "list", Err: types.ErrDecryptFailed}
	if got := describeDecryptError(other); !errors.Is(got, types.ErrDecryptFailed) {
		t.Errorf("other: got %v", got)
	}
}

func TestPasswordOptionsFor(t *testing.T) {
	opts := passwordOptionsFor(PassOpts{}, "", true)
	if !reflect.DeepEqual(opts.FallbackEnvs, []string{"GOINGENV_PASSWORD"}) || !opts.Confirm {
		t.Errorf("unnamed: %+v", opts)
	}
	opts = passwordOptionsFor(PassOpts{PassEnv: "X"}, "prod-eu", true)
	if !reflect.DeepEqual(opts.FallbackEnvs, []string{"GOINGENV_PASSWORD_PROD_EU", "GOINGENV_PASSWORD"}) || opts.PasswordEnv != "X" {
		t.Errorf("prod-eu: %+v", opts)
	}
	opts = passwordOptionsFor(PassOpts{PassStdin: true}, "", true)
	if !opts.Stdin || opts.Confirm {
		t.Errorf("stdin must disable confirm: %+v", opts)
	}
}
