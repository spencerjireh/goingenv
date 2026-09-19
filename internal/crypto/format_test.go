package crypto

import (
	"crypto/rand"
	"errors"
	"strings"
	"testing"
	"time"

	"goingenv/pkg/types"
)

func TestFormat_HeaderRoundTrip(t *testing.T) {
	blob := mustEncrypt([]byte("payload"), "pw")

	h, err := Inspect(blob)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if h.Version != FormatVersion || h.KDF != KDFArgon2id || h.KeyMode != KeyModePassword {
		t.Errorf("header = %+v, want version %d kdf %d mode %d", h, FormatVersion, KDFArgon2id, KeyModePassword)
	}
	if h.Params != testParams {
		t.Errorf("params = %+v, want %+v", h.Params, testParams)
	}
	if string(blob[:4]) != Magic {
		t.Errorf("blob does not start with magic: %q", blob[:4])
	}
	if len(blob) != PrefixSize+len("payload")+TagSize {
		t.Errorf("blob length = %d, want %d", len(blob), PrefixSize+len("payload")+TagSize)
	}
}

func TestFormat_DefaultServiceWritesDefaultParams(t *testing.T) {
	blob, err := NewService().Encrypt([]byte("x"), "pw")
	if err != nil {
		t.Fatalf("Encrypt: %v", err)
	}
	h, err := Inspect(blob)
	if err != nil {
		t.Fatalf("Inspect: %v", err)
	}
	if h.Params != DefaultParams() {
		t.Errorf("params = %+v, want %+v", h.Params, DefaultParams())
	}
}

// Every byte before the ciphertext is additional data: flipping any of them
// must fail authentication, not silently change the parameters.
func TestFormat_PrefixIsAuthenticated(t *testing.T) {
	service := newTestService()
	blob := mustEncrypt([]byte("payload"), "pw")

	for i := 0; i < PrefixSize; i++ {
		tampered := append([]byte(nil), blob...)
		tampered[i] ^= 0x01
		if _, err := service.Decrypt(tampered, "pw"); err == nil {
			t.Errorf("byte %d: tampering was not detected", i)
		}
	}
}

func TestFormat_LegacyBlobIsRejectedWithRemedy(t *testing.T) {
	// A v1.6 blob is 32 random salt bytes followed by nonce and ciphertext:
	// no magic, so it is indistinguishable from random data.
	legacy := make([]byte, 120)
	if _, err := rand.Read(legacy); err != nil {
		t.Fatal(err)
	}
	if string(legacy[:4]) == Magic {
		legacy[0] ^= 0xFF
	}

	_, err := newTestService().Decrypt(legacy, "pw")
	if !errors.Is(err, types.ErrLegacyArchive) {
		t.Fatalf("err = %v, want ErrLegacyArchive", err)
	}
	if !strings.Contains(err.Error(), "v1.6.0") {
		t.Errorf("error does not name the last version that reads the old format: %v", err)
	}

	if _, err := Inspect(legacy); !errors.Is(err, types.ErrLegacyArchive) {
		t.Errorf("Inspect err = %v, want ErrLegacyArchive", err)
	}
}

// An empty or truncated file has no magic either, but sending the user to
// v1.6.0 for it would fail there too: it must read as too short.
func TestFormat_ShortBlobWithoutMagicIsNotLegacy(t *testing.T) {
	short := make([]byte, legacyMinBlobSize-1)
	if _, err := rand.Read(short); err != nil {
		t.Fatal(err)
	}
	if string(short[:4]) == Magic {
		short[0] ^= 0xFF
	}
	for name, blob := range map[string][]byte{"nil": nil, "empty": {}, "short": short} {
		_, err := newTestService().Decrypt(blob, "pw")
		if err == nil || errors.Is(err, types.ErrLegacyArchive) {
			t.Errorf("%s: Decrypt err = %v, want too short and not ErrLegacyArchive", name, err)
		}
		if _, err := Inspect(blob); err == nil || errors.Is(err, types.ErrLegacyArchive) {
			t.Errorf("%s: Inspect err = %v, want too short and not ErrLegacyArchive", name, err)
		}
	}

	// The boundary itself is the smallest 1.x blob and keeps the remedy.
	boundary := make([]byte, legacyMinBlobSize)
	if _, err := rand.Read(boundary); err != nil {
		t.Fatal(err)
	}
	if string(boundary[:4]) == Magic {
		boundary[0] ^= 0xFF
	}
	if _, err := Inspect(boundary); !errors.Is(err, types.ErrLegacyArchive) {
		t.Errorf("boundary: err = %v, want ErrLegacyArchive", err)
	}
}

func TestFormat_UnknownVersionKDFOrModeIsNotLegacy(t *testing.T) {
	cases := []struct {
		name   string
		offset int
		value  byte
	}{
		{"future version", 4, 2},
		{"unknown kdf", 5, 9},
		{"unknown key mode", 15, 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			blob := mustEncrypt([]byte("payload"), "pw")
			blob[tc.offset] = tc.value
			_, err := newTestService().Decrypt(blob, "pw")
			if err == nil {
				t.Fatal("expected an error")
			}
			if errors.Is(err, types.ErrLegacyArchive) {
				t.Errorf("reported as legacy, want a newer-format error: %v", err)
			}
			if errors.Is(err, types.ErrDecryptFailed) {
				t.Errorf("reported as wrong password, want a format error: %v", err)
			}
		})
	}
}

// Parameters outside the caps must be refused before any key derivation
// runs, otherwise a hostile header could demand gigabytes of work.
func TestFormat_OversizedParamsFailFast(t *testing.T) {
	blob := mustEncrypt([]byte("payload"), "pw")
	// memory = 0xFFFFFFFF KiB (~4 TiB)
	blob[10], blob[11], blob[12], blob[13] = 0xFF, 0xFF, 0xFF, 0xFF

	start := time.Now()
	_, err := newTestService().Decrypt(blob, "pw")
	if err == nil {
		t.Fatal("expected an error")
	}
	if !strings.Contains(err.Error(), "unsupported KDF parameters") {
		t.Errorf("err = %v", err)
	}
	if time.Since(start) > time.Second {
		t.Errorf("took %v; the cap should reject before deriving", time.Since(start))
	}
}

func TestFormat_TruncatedBlobIsRejected(t *testing.T) {
	blob := mustEncrypt([]byte("payload"), "pw")
	for _, n := range []int{3, 4, 15, 16, PrefixSize, MinBlobSize - 1} {
		if _, err := newTestService().Decrypt(blob[:n], "pw"); err == nil {
			t.Errorf("len %d: expected an error", n)
		}
	}
}

func TestFormat_WrongPasswordIsErrDecryptFailed(t *testing.T) {
	blob := mustEncrypt([]byte("payload"), "pw")
	_, err := newTestService().Decrypt(blob, "nope")
	if !errors.Is(err, types.ErrDecryptFailed) {
		t.Errorf("err = %v, want ErrDecryptFailed", err)
	}
}

func TestFormat_EncryptRejectsBadParams(t *testing.T) {
	_, err := NewServiceWithParams(Params{Time: 0, MemoryKiB: 8, Threads: 1}).Encrypt([]byte("x"), "pw")
	if err == nil {
		t.Fatal("expected an error")
	}
}
