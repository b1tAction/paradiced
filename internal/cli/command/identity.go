// Package command provides CLI commands for paradiced CLI.
package command

import (
	"crypto/rand"
	"fmt"
	"strings"
	"unicode"
)

func resolveUserID(name string) (string, error) {
	cleanName := sanitizeName(name)
	if cleanName == "" {
		return "", fmt.Errorf("--name is required")
	}

	suffix, err := randomHex(4) // 8 hex chars
	if err != nil {
		return "", fmt.Errorf("failed to generate user-id: %w", err)
	}

	return fmt.Sprintf("%s-%s", cleanName, suffix), nil
}

func sanitizeName(name string) string {
	raw := strings.TrimSpace(strings.ToLower(name))
	if raw == "" {
		return ""
	}

	var b strings.Builder
	lastDash := false
	for _, r := range raw {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || unicode.IsSpace(r):
			if b.Len() > 0 && !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}

	result := strings.Trim(b.String(), "-")
	if result == "" {
		return "player"
	}
	return result
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", buf), nil
}
