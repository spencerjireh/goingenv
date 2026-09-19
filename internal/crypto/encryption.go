// Package crypto implements the goingenv archive encryption: a small plaintext
// header, then AES-256-GCM over the payload with a key derived from the
// password by Argon2id.
//
// Blob layout (format v1, big-endian):
//
//	offset size field
//	0      4    magic "GENV"
//	4      1    format version (1)
//	5      1    KDF id (1 = Argon2id)
//	6      4    Argon2 time
//	10     4    Argon2 memory in KiB
//	14     1    Argon2 threads
//	15     1    key mode (0 = password; reserved for per-recipient wrapping)
//	16     32   salt
//	48     12   GCM nonce
//	60     ..   ciphertext followed by the 16-byte GCM tag
//
// Bytes 0..59 are the GCM additional data, so any change to the header, salt
// or nonce fails authentication.
package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"

	"goingenv/pkg/types"

	"golang.org/x/crypto/argon2"
)

const (
	// Magic opens every blob written in format v1 or later.
	Magic = "GENV"
	// FormatVersion is the blob format this build writes.
	FormatVersion uint8 = 1
	// KDFArgon2id is the only key derivation function defined so far.
	KDFArgon2id uint8 = 1
	// KeyModePassword derives the key from a shared password.
	KeyModePassword uint8 = 0

	// HeaderSize is the fixed header before the salt.
	HeaderSize = 16
	// SaltSize is the size of the salt in bytes.
	SaltSize = 32
	// NonceSize is the size of the GCM nonce in bytes.
	NonceSize = 12
	// KeySize is the size of the AES-256 key in bytes.
	KeySize = 32
	// PrefixSize is everything before the ciphertext; it is the GCM AAD.
	PrefixSize = HeaderSize + SaltSize + NonceSize
	// TagSize is the GCM authentication tag appended to the ciphertext.
	TagSize = 16
	// MinBlobSize is the smallest well-formed blob: prefix plus a tag.
	MinBlobSize = PrefixSize + TagSize

	// DefaultArgonTime, DefaultArgonMemoryKiB and DefaultArgonThreads are the
	// RFC 9106 second recommended option (t=3, m=64 MiB, p=4).
	DefaultArgonTime      uint32 = 3
	DefaultArgonMemoryKiB uint32 = 64 * 1024
	DefaultArgonThreads   uint8  = 4

	// Caps applied before deriving a key. The KDF runs before the tag can be
	// checked, so without them a hostile header could demand unbounded work.
	maxArgonTime      uint32 = 32
	maxArgonMemoryKiB uint32 = 1 << 20
	maxArgonThreads   uint8  = 32
)

// Params are the Argon2id cost parameters stored in the header.
type Params struct {
	Time      uint32
	MemoryKiB uint32
	Threads   uint8
}

// DefaultParams returns the parameters this build writes.
func DefaultParams() Params {
	return Params{Time: DefaultArgonTime, MemoryKiB: DefaultArgonMemoryKiB, Threads: DefaultArgonThreads}
}

func (p Params) validate() error {
	switch {
	case p.Time == 0 || p.Time > maxArgonTime:
		return fmt.Errorf("unsupported KDF parameters: time %d", p.Time)
	case p.Threads == 0 || p.Threads > maxArgonThreads:
		return fmt.Errorf("unsupported KDF parameters: threads %d", p.Threads)
	case p.MemoryKiB > maxArgonMemoryKiB || p.MemoryKiB < 8*uint32(p.Threads):
		return fmt.Errorf("unsupported KDF parameters: memory %d KiB", p.MemoryKiB)
	}
	return nil
}

// Header is the plaintext part of a blob.
type Header struct {
	Version uint8
	KDF     uint8
	KeyMode uint8
	Params  Params
}

func encodeHeader(h Header) []byte {
	out := make([]byte, HeaderSize)
	copy(out[0:4], Magic)
	out[4] = h.Version
	out[5] = h.KDF
	binary.BigEndian.PutUint32(out[6:10], h.Params.Time)
	binary.BigEndian.PutUint32(out[10:14], h.Params.MemoryKiB)
	out[14] = h.Params.Threads
	out[15] = h.KeyMode
	return out
}

// parseHeader decodes the fixed header. A blob without the magic is reported
// as ErrLegacyArchive; a blob with the magic but unknown version, KDF or key
// mode is reported as newer than this build.
func parseHeader(data []byte) (Header, error) {
	if len(data) < len(Magic) || string(data[0:4]) != Magic {
		return Header{}, types.ErrLegacyArchive
	}
	if len(data) < HeaderSize {
		return Header{}, errors.New("invalid encrypted data: truncated header")
	}
	h := Header{
		Version: data[4],
		KDF:     data[5],
		KeyMode: data[15],
		Params: Params{
			Time:      binary.BigEndian.Uint32(data[6:10]),
			MemoryKiB: binary.BigEndian.Uint32(data[10:14]),
			Threads:   data[14],
		},
	}
	switch {
	case h.Version != FormatVersion:
		return h, fmt.Errorf("archive format version %d is newer than this goingenv", h.Version)
	case h.KDF != KDFArgon2id:
		return h, fmt.Errorf("archive uses KDF %d, which this goingenv does not support", h.KDF)
	case h.KeyMode != KeyModePassword:
		return h, fmt.Errorf("archive uses key mode %d, which this goingenv does not support", h.KeyMode)
	}
	return h, nil
}

// Inspect reads the header of a blob without a password.
func Inspect(data []byte) (Header, error) {
	return parseHeader(data)
}

// Service implements the Cryptor interface
type Service struct {
	params Params
}

// NewService creates a crypto service that writes DefaultParams.
func NewService() *Service {
	return &Service{params: DefaultParams()}
}

// NewServiceWithParams creates a crypto service that writes the given
// parameters. Tests use small values to keep the suite fast; the parameters a
// blob was written with are always read back from its header.
func NewServiceWithParams(p Params) *Service {
	return &Service{params: p}
}

func deriveKey(password string, salt []byte, p Params) []byte {
	return argon2.IDKey([]byte(password), salt, p.Time, p.MemoryKiB, p.Threads, KeySize)
}

func newGCM(key []byte) (cipher.AEAD, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	return gcm, nil
}

// Encrypt encrypts data using AES-256-GCM with an Argon2id-derived key.
func (s *Service) Encrypt(data []byte, password string) ([]byte, error) {
	if len(data) == 0 {
		return nil, &types.CryptoError{Operation: "encrypt", Err: fmt.Errorf("data cannot be empty")}
	}
	if password == "" {
		return nil, &types.CryptoError{Operation: "encrypt", Err: fmt.Errorf("password cannot be empty")}
	}
	if err := s.params.validate(); err != nil {
		return nil, &types.CryptoError{Operation: "encrypt", Err: err}
	}

	prefix := make([]byte, PrefixSize)
	copy(prefix, encodeHeader(Header{Version: FormatVersion, KDF: KDFArgon2id, KeyMode: KeyModePassword, Params: s.params}))
	if _, err := rand.Read(prefix[HeaderSize:]); err != nil {
		return nil, &types.CryptoError{Operation: "encrypt", Err: fmt.Errorf("failed to generate salt and nonce: %w", err)}
	}
	salt := prefix[HeaderSize : HeaderSize+SaltSize]
	nonce := prefix[HeaderSize+SaltSize : PrefixSize]

	gcm, err := newGCM(deriveKey(password, salt, s.params))
	if err != nil {
		return nil, &types.CryptoError{Operation: "encrypt", Err: err}
	}

	// Seal appends to prefix, so the result is header || salt || nonce || ciphertext.
	return gcm.Seal(prefix, nonce, data, prefix), nil
}

// Decrypt decrypts a blob written by Encrypt, reading the KDF parameters from
// its header.
func (s *Service) Decrypt(data []byte, password string) ([]byte, error) {
	h, err := parseHeader(data)
	if err != nil {
		return nil, &types.CryptoError{Operation: "decrypt", Err: err}
	}
	if len(data) < MinBlobSize {
		return nil, &types.CryptoError{Operation: "decrypt", Err: fmt.Errorf("invalid encrypted data: too short")}
	}
	if password == "" {
		return nil, &types.CryptoError{Operation: "decrypt", Err: fmt.Errorf("password cannot be empty")}
	}
	if paramErr := h.Params.validate(); paramErr != nil {
		return nil, &types.CryptoError{Operation: "decrypt", Err: paramErr}
	}

	prefix := data[:PrefixSize]
	salt := prefix[HeaderSize : HeaderSize+SaltSize]
	nonce := prefix[HeaderSize+SaltSize : PrefixSize]

	gcm, err := newGCM(deriveKey(password, salt, h.Params))
	if err != nil {
		return nil, &types.CryptoError{Operation: "decrypt", Err: err}
	}

	plaintext, err := gcm.Open(nil, nonce, data[PrefixSize:], prefix)
	if err != nil {
		return nil, &types.CryptoError{Operation: "decrypt", Err: types.ErrDecryptFailed}
	}
	return plaintext, nil
}

// ValidatePassword validates if a password can decrypt the given data
func (s *Service) ValidatePassword(data []byte, password string) error {
	_, err := s.Decrypt(data, password)
	if err != nil {
		var cryptoErr *types.CryptoError
		if errors.As(err, &cryptoErr) {
			return &types.CryptoError{Operation: "validate", Err: cryptoErr.Err}
		}
		return &types.CryptoError{Operation: "validate", Err: err}
	}
	return nil
}

// GenerateSecurePassword generates a cryptographically secure random password
func GenerateSecurePassword(length int) (string, error) {
	if length < 8 {
		return "", fmt.Errorf("password length must be at least 8 characters")
	}

	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	password := make([]byte, length)

	for i := range password {
		randomByte := make([]byte, 1)
		if _, err := rand.Read(randomByte); err != nil {
			return "", fmt.Errorf("failed to generate random password: %w", err)
		}
		password[i] = charset[int(randomByte[0])%len(charset)]
	}

	return string(password), nil
}

// EstimateDecryptionTime estimates the time needed to decrypt data (for UI progress)
func EstimateDecryptionTime(dataSize int64) int {
	// Very rough estimate: ~1MB per second for decryption
	const decryptionSpeed = 1024 * 1024 // bytes per second

	estimatedSeconds := int(dataSize / decryptionSpeed)
	if estimatedSeconds < 1 {
		estimatedSeconds = 1
	}

	return estimatedSeconds
}
