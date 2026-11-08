package services

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math/big"
	"time"
)

const (
	otpLength      = 6
	otpTTL         = 10 * time.Minute
	otpMaxAttempts = 5
	otpHashLabel   = "email-otp"
)

// OTPHasher hashes and compares email verification codes.
type OTPHasher struct {
	secret []byte
}

func newOTPHasher(secret []byte) *OTPHasher {
	return &OTPHasher{secret: secret}
}

// Hash returns the HMAC-SHA256 hash of code.
func (h *OTPHasher) Hash(code string) string {
	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(otpHashLabel))
	mac.Write([]byte(":"))
	mac.Write([]byte(code))
	return hex.EncodeToString(mac.Sum(nil))
}

// Compare reports whether code matches hash in constant time.
func (h *OTPHasher) Compare(hash, code string) bool {
	expected, err := hex.DecodeString(hash)
	if err != nil {
		return false
	}

	mac := hmac.New(sha256.New, h.secret)
	mac.Write([]byte(otpHashLabel))
	mac.Write([]byte(":"))
	mac.Write([]byte(code))
	return hmac.Equal(expected, mac.Sum(nil))
}

func generateOTP() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate otp: %w", err)
	}
	return fmt.Sprintf("%0*d", otpLength, n.Int64()), nil
}
