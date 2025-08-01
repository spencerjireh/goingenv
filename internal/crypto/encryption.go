package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"

	"goingenv/pkg/types"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// SaltSize is the size of the salt in bytes
	SaltSize = 32
	// NonceSize is the size of the nonce in bytes
	NonceSize = 12
	// KeySize is the size of the encryption key in bytes
	KeySize = 32
	// PBKDF2Iterations is the number of iterations for PBKDF2
	PBKDF2Iterations = 100000
)

// Service implements the Cryptor interface
type Service struct{}

// NewService creates a new crypto service
func NewService() *Service {
	return &Service{}
}

// Encrypt encrypts data using AES-256-GCM with PBKDF2 key derivation
func (s *Service) Encrypt(data []byte, password string) ([]byte, error) {
	if len(data) == 0 {
		return nil, &types.CryptoError{
			Operation: "encrypt",
			Err:       fmt.Errorf("data cannot be empty"),
		}
	}

	if password == "" {
		return nil, &types.CryptoError{
			Operation: "encrypt",
			Err:       fmt.Errorf("password cannot be empty"),
		}
	}

	// Generate random salt
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, &types.CryptoError{
			Operation: "encrypt",
			Err:       fmt.Errorf("failed to generate salt: %w", err),
		}
	}

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, &types.CryptoError{
			Operation: "encrypt",
			Err:       fmt.Errorf("failed to create cipher: %w", err),
		}
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, &types.CryptoError{
			Operation: "encrypt",
			Err:       fmt.Errorf("failed to create GCM: %w", err),
		}
	}

	// Generate random nonce
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, &types.CryptoError{
			Operation: "encrypt",
			Err:       fmt.Errorf("failed to generate nonce: %w", err),
		}
	}

	// Encrypt data
	ciphertext := gcm.Seal(nil, nonce, data, nil)

	// Combine salt + nonce + ciphertext
	result := make([]byte, SaltSize+NonceSize+len(ciphertext))
	copy(result[:SaltSize], salt)
	copy(result[SaltSize:SaltSize+NonceSize], nonce)
	copy(result[SaltSize+NonceSize:], ciphertext)

	return result, nil
}

// Decrypt decrypts data using AES-256-GCM with PBKDF2 key derivation
func (s *Service) Decrypt(data []byte, password string) ([]byte, error) {
	if len(data) < SaltSize+NonceSize {
		return nil, &types.CryptoError{
			Operation: "decrypt",
			Err:       fmt.Errorf("invalid encrypted data: too short"),
		}
	}

	if password == "" {
		return nil, &types.CryptoError{
			Operation: "decrypt",
			Err:       fmt.Errorf("password cannot be empty"),
		}
	}

	// Extract salt, nonce, and ciphertext
	salt := data[:SaltSize]
	nonce := data[SaltSize : SaltSize+NonceSize]
	ciphertext := data[SaltSize+NonceSize:]

	// Derive key using PBKDF2
	key := pbkdf2.Key([]byte(password), salt, PBKDF2Iterations, KeySize, sha256.New)

	// Create AES cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, &types.CryptoError{
			Operation: "decrypt",
			Err:       fmt.Errorf("failed to create cipher: %w", err),
		}
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, &types.CryptoError{
			Operation: "decrypt",
			Err:       fmt.Errorf("failed to create GCM: %w", err),
		}
	}

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, &types.CryptoError{
			Operation: "decrypt",
			Err:       fmt.Errorf("decryption failed: invalid password or corrupted data"),
		}
	}

	return plaintext, nil
}

// ValidatePassword validates if a password can decrypt the given data
func (s *Service) ValidatePassword(data []byte, password string) error {
	_, err := s.Decrypt(data, password)
	if err != nil {
		if cryptoErr, ok := err.(*types.CryptoError); ok {
			return &types.CryptoError{
				Operation: "validate",
				Err:       cryptoErr.Err,
			}
		}
		return &types.CryptoError{
			Operation: "validate",
			Err:       err,
		}
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
