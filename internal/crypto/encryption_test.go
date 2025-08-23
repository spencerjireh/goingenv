package crypto

import (
	"bytes"
	"strings"
	"testing"

	"goingenv/pkg/types"
)

func TestService_EncryptDecrypt(t *testing.T) {
	service := NewService()

	tests := []struct {
		name     string
		data     []byte
		password string
	}{
		{
			name:     "Simple text",
			data:     []byte("Hello, World!"),
			password: "testpassword123",
		},
		{
			name:     "Small data",
			data:     []byte("small"),
			password: "password",
		},
		{
			name:     "Binary data",
			data:     []byte{0x00, 0x01, 0x02, 0xFF, 0xFE},
			password: "binarytest",
		},
		{
			name:     "Large data",
			data:     bytes.Repeat([]byte("A"), 10000),
			password: "largedata",
		},
		{
			name:     "Environment file content",
			data:     []byte("DATABASE_URL=postgres://localhost/test\nAPI_KEY=secret123\nDEBUG=true"),
			password: "envfile",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test encryption
			encrypted, err := service.Encrypt(tt.data, tt.password)
			if err != nil {
				t.Fatalf("Encrypt failed: %v", err)
			}

			// Encrypted data should be different from original (unless empty)
			if len(tt.data) > 0 && bytes.Equal(tt.data, encrypted) {
				t.Error("Encrypted data is identical to original data")
			}

			// Encrypted data should be longer due to salt, nonce, and auth tag
			if len(encrypted) <= len(tt.data) {
				t.Error("Encrypted data should be longer than original")
			}

			// Test decryption
			decrypted, err := service.Decrypt(encrypted, tt.password)
			if err != nil {
				t.Fatalf("Decrypt failed: %v", err)
			}

			// Decrypted data should match original
			if !bytes.Equal(tt.data, decrypted) {
				t.Errorf("Decrypted data doesn't match original\nOriginal: %v\nDecrypted: %v", tt.data, decrypted)
			}
		})
	}
}

func TestService_EncryptErrors(t *testing.T) {
	service := NewService()

	tests := []struct {
		name     string
		data     []byte
		password string
		wantErr  bool
	}{
		{
			name:     "Empty password",
			data:     []byte("test"),
			password: "",
			wantErr:  true,
		},
		{
			name:     "Valid password with minimal data",
			data:     []byte("x"),
			password: "validpassword123",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Encrypt(tt.data, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Encrypt() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				if cryptoErr, ok := err.(*types.CryptoError); !ok {
					t.Errorf("Expected CryptoError, got %T", err)
				} else if cryptoErr.Operation != "encrypt" {
					t.Errorf("Expected operation 'encrypt', got %s", cryptoErr.Operation)
				}
			}
		})
	}
}

func TestService_DecryptErrors(t *testing.T) {
	service := NewService()

	tests := []struct {
		name     string
		data     []byte
		password string
		wantErr  bool
	}{
		{
			name:     "Invalid data - too short",
			data:     []byte("short"),
			password: "testpassword123",
			wantErr:  true,
		},
		{
			name:     "Empty password",
			data:     make([]byte, SaltSize+NonceSize+10),
			password: "",
			wantErr:  true,
		},
		{
			name:     "Wrong password",
			data:     mustEncrypt([]byte("test"), "correct"),
			password: "wrong",
			wantErr:  true,
		},
		{
			name:     "Corrupted data",
			data:     corruptData(mustEncrypt([]byte("test"), "correct")),
			password: "correct",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Decrypt(tt.data, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("Decrypt() error = %v, wantErr %v", err, tt.wantErr)
			}

			if err != nil {
				if cryptoErr, ok := err.(*types.CryptoError); !ok {
					t.Errorf("Expected CryptoError, got %T", err)
				} else if cryptoErr.Operation != "decrypt" {
					t.Errorf("Expected operation 'decrypt', got %s", cryptoErr.Operation)
				}
			}
		})
	}
}

func TestService_ValidatePassword(t *testing.T) {
	service := NewService()
	data := []byte("test data for password validation")
	password := "correct password 123"

	// Encrypt data first
	encrypted, err := service.Encrypt(data, password)
	if err != nil {
		t.Fatalf("Failed to encrypt test data: %v", err)
	}

	tests := []struct {
		name     string
		data     []byte
		password string
		wantErr  bool
	}{
		{
			name:     "Correct password",
			data:     encrypted,
			password: password,
			wantErr:  false,
		},
		{
			name:     "Wrong password",
			data:     encrypted,
			password: "wrong password",
			wantErr:  true,
		},
		{
			name:     "Invalid data format",
			data:     []byte("invalid"),
			password: password,
			wantErr:  true,
		},
		{
			name:     "Empty password",
			data:     encrypted,
			password: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.ValidatePassword(tt.data, tt.password)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePassword() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerateSecurePassword(t *testing.T) {
	tests := []struct {
		name    string
		length  int
		wantErr bool
	}{
		{"Valid length", 12, false},
		{"Minimum length", 8, false},
		{"Long password", 64, false},
		{"Too short", 7, true},
		{"Zero length", 0, true},
		{"Negative length", -5, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateSecurePassword(tt.length)
			if (err != nil) != tt.wantErr {
				t.Errorf("GenerateSecurePassword() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(password) != tt.length {
					t.Errorf("Password length = %d, want %d", len(password), tt.length)
				}

				// Check that password contains only allowed characters
				allowedChars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
				for _, char := range password {
					if !strings.ContainsRune(allowedChars, char) {
						t.Errorf("Password contains invalid character: %c", char)
					}
				}

				// For short passwords, complexity might be limited by randomness
				// Just check that password contains valid characters
				if tt.length >= 12 {
					// Only check complexity for longer passwords
					hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
					hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
					hasDigit := strings.ContainsAny(password, "0123456789")
					hasSpecial := strings.ContainsAny(password, "!@#$%^&*")

					complexityCount := 0
					if hasLower {
						complexityCount++
					}
					if hasUpper {
						complexityCount++
					}
					if hasDigit {
						complexityCount++
					}
					if hasSpecial {
						complexityCount++
					}

					if complexityCount < 2 {
						t.Error("Generated long password should contain at least 2 character types")
					}
				}
			}
		})
	}
}

func TestPasswordStrength(t *testing.T) {
	tests := []struct {
		name     string
		password string
		minScore int // Minimum acceptable strength score
	}{
		{"Very weak", "123", 1},
		{"Weak", "password", 2},
		{"Medium", "Password123", 3},
		{"Strong", "MyStr0ng!Pass", 4},
		{"Very strong", "MyVeryStr0ng!P@ssw0rd", 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := calculatePasswordStrength(tt.password)
			if score < tt.minScore {
				t.Errorf("Password strength score = %d, want at least %d", score, tt.minScore)
			}
		})
	}
}

func TestService_EncryptionConsistency(t *testing.T) {
	service := NewService()
	data := []byte("consistency test data")
	password := "consistent password"

	// Encrypt the same data multiple times
	var encrypted [][]byte
	for i := 0; i < 5; i++ {
		enc, err := service.Encrypt(data, password)
		if err != nil {
			t.Fatalf("Encryption %d failed: %v", i, err)
		}
		encrypted = append(encrypted, enc)
	}

	// Each encryption should produce different results (due to random salt/nonce)
	for i := 0; i < len(encrypted); i++ {
		for j := i + 1; j < len(encrypted); j++ {
			if bytes.Equal(encrypted[i], encrypted[j]) {
				t.Errorf("Encryption %d and %d produced identical results", i, j)
			}
		}
	}

	// But all should decrypt to the same original data
	for i, enc := range encrypted {
		decrypted, err := service.Decrypt(enc, password)
		if err != nil {
			t.Errorf("Decryption %d failed: %v", i, err)
		}
		if !bytes.Equal(data, decrypted) {
			t.Errorf("Decryption %d produced wrong result", i)
		}
	}
}

func BenchmarkEncrypt(b *testing.B) {
	service := NewService()
	data := []byte("benchmark test data for encryption performance")
	password := "benchmarkpassword123"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.Encrypt(data, password)
		if err != nil {
			b.Fatalf("Encryption failed: %v", err)
		}
	}
}

func BenchmarkDecrypt(b *testing.B) {
	service := NewService()
	data := []byte("benchmark test data for decryption performance")
	password := "benchmarkpassword123"

	encrypted, err := service.Encrypt(data, password)
	if err != nil {
		b.Fatalf("Setup encryption failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := service.Decrypt(encrypted, password)
		if err != nil {
			b.Fatalf("Decryption failed: %v", err)
		}
	}
}

// Helper functions for tests
func mustEncrypt(data []byte, password string) []byte {
	service := NewService()
	encrypted, err := service.Encrypt(data, password)
	if err != nil {
		panic(err)
	}
	return encrypted
}

func corruptData(data []byte) []byte {
	if len(data) == 0 {
		return data
	}
	corrupted := make([]byte, len(data))
	copy(corrupted, data)
	// Corrupt the last byte
	corrupted[len(corrupted)-1] ^= 0xFF
	return corrupted
}

func calculatePasswordStrength(password string) int {
	score := 0

	// Length scoring
	if len(password) >= 8 {
		score++
	}
	if len(password) >= 12 {
		score++
	}

	// Character type scoring
	hasLower := strings.ContainsAny(password, "abcdefghijklmnopqrstuvwxyz")
	hasUpper := strings.ContainsAny(password, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
	hasDigit := strings.ContainsAny(password, "0123456789")
	hasSpecial := strings.ContainsAny(password, "!@#$%^&*()_+-=[]{}|;:,.<>?")

	if hasLower {
		score++
	}
	if hasUpper {
		score++
	}
	if hasDigit {
		score++
	}
	if hasSpecial {
		score++
	}

	return score
}
