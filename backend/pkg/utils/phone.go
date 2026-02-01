package utils

import (
	"errors"
	"regexp"
)

// pkg/utils/phone.go

func ValidatePhone(phone string) (string, error) {
	if len(phone) == 0 {
		return "", errors.New("phone is required")
	}

	// Must start with +
	if phone[0] != '+' {
		return "", errors.New("invalid phone format, must start with +")
	}

	// Get country code and number
	// Strip leading 0 from the number part after country code
	// e.g. +9720501234567 → +972501234567
	re := regexp.MustCompile(`^(\+\d{1,3})(0?)(\d{9,12})$`)
	matches := re.FindStringSubmatch(phone)
	if matches == nil {
		return "", errors.New("invalid phone format")
	}

	// matches[1] = "+972"
	// matches[2] = "0" (or empty)  ← we drop this
	// matches[3] = "501234567"
	cleaned := matches[1] + matches[3]

	// Final E.164 validation
	e164 := regexp.MustCompile(`^\+[1-9]\d{9,14}$`)
	if !e164.MatchString(cleaned) {
		return "", errors.New("invalid phone number")
	}

	return cleaned, nil
}
