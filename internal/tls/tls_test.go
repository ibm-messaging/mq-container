package tls

import "testing"

func TestGeneratePassword(t *testing.T) {
	previousPasswords := map[string]bool{}
	for range 100 {
		newPass := generateRandomPassword()
		for _, ch := range newPass.String() {
			switch {
			case ch >= 'A' && ch <= 'Z':
			case ch >= 'a' && ch <= 'z':
			case ch >= '0' && ch <= '9':
			default:
				t.Fatalf("New password generated has invalid character ('%c' found in password '%s')", ch, newPass.String())
			}
		}
		if previousPasswords[newPass.String()] {
			t.Fatalf("Duplicate random password generated ('%s')", newPass.String())
		}
		previousPasswords[newPass.String()] = true
	}
}
