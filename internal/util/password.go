package util

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// These constants may be tuned to match your security requirements
	passwordSaltBytes    = 16
	passwordHashBytes    = 32
	passwordIterations   = 1_000_000 // OWASP recommendation as of 2023 is 600,000 for PBKDF2-SHA256
	passwordAlgorithmKey = "pbkdf2-sha256"
)

// ErrInvalidHashFormat indicates that the hash string is not in the expected format.
var ErrInvalidHashFormat = errors.New("invalid hash format")

// ErrIncompatibleAlgorithm indicates that the algorithm used for hashing is not supported.
var ErrIncompatibleAlgorithm = errors.New("incompatible algorithm")

// HashPassword creates a PBKDF2 hash of the password.
// The returned string is in the format "algorithm:iterations:salt:hash".
func HashPassword(password string) (string, error) {
	if password == "" {
		return "", errors.New("password cannot be empty")
	}

	salt := make([]byte, passwordSaltBytes)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := pbkdf2.Key([]byte(password), salt, passwordIterations, passwordHashBytes, sha256.New)

	// Encode salt and hash to base64
	b64Salt := base64.RawStdEncoding.EncodeToString(salt)
	b64Hash := base64.RawStdEncoding.EncodeToString(hash)

	// Format: algorithm:iterations:salt:hash
	return fmt.Sprintf("%s:%d:%s:%s", passwordAlgorithmKey, passwordIterations, b64Salt, b64Hash), nil
}

// CheckPasswordHash verifies a password against a stored PBKDF2 hash.
// The storedHash is expected to be in the format "algorithm:iterations:salt:hash".
func CheckPasswordHash(password, storedHash string) (bool, error) {
	if password == "" || storedHash == "" {
		return false, errors.New("password and stored hash cannot be empty")
	}

	parts := strings.Split(storedHash, ":")
	if len(parts) != 4 {
		return false, ErrInvalidHashFormat
	}

	algorithm := parts[0]
	if algorithm != passwordAlgorithmKey {
		return false, ErrIncompatibleAlgorithm
	}

	iterations, err := parseInt(parts[1])
	if err != nil {
		return false, fmt.Errorf("failed to parse iterations: %w", err)
	}

	salt, err := base64.RawStdEncoding.DecodeString(parts[2])
	if err != nil {
		return false, fmt.Errorf("failed to decode salt: %w", err)
	}

	hash, err := base64.RawStdEncoding.DecodeString(parts[3])
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	// Verify the password
	comparisonHash := pbkdf2.Key([]byte(password), salt, iterations, len(hash), sha256.New)

	// Constant time comparison to prevent timing attacks
	if subtle.ConstantTimeCompare(hash, comparisonHash) == 1 {
		return true, nil
	}

	return false, nil
}

// Helper function to parse int, as strconv.Atoi is not used directly to avoid import cycle if this moves.
func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscan(s, &n)
	return n, err
}
