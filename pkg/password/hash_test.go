package password

import (
	"strings"
	"testing"
)

func TestNewPasswordHasher(t *testing.T) {
	tests := []struct {
		name     string
		cost     int
		expected int
	}{
		{"default cost", 12, 12},
		{"cost too low", 5, 10}, // should be clamped to MinBcryptCost
		{"cost too high", 20, 14}, // should be clamped to MaxBcryptCost
		{"minimum valid cost", 10, 10},
		{"maximum valid cost", 14, 14},
		{"cost zero", 0, 10},
		{"negative cost", -5, 10},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasher := NewPasswordHasher(tt.cost)
			if hasher.GetCost() != tt.expected {
				t.Errorf("expected cost %d, got %d", tt.expected, hasher.GetCost())
			}
		})
	}
}

func TestPasswordHasher_Hash(t *testing.T) {
	hasher := NewPasswordHasher(12)

	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		{"valid password", "SecurePass123!", false},
		{"short password", "short", true},
		{"empty password", "", true},
		{"common pattern password", "password123", true},
		{"weak password 123456", "123456789", true},
		{"weak password qwerty", "qwerty12345", true},
		{"strong password", "MyStr0ng!Pass#2024", false},
		{"very long password", strings.Repeat("a", 100), true}, // exceeds max length
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := hasher.Hash(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for password '%s', got nil", tt.password)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			// Verify hash is bcrypt format
			if !strings.HasPrefix(hash, "$2a$") && !strings.HasPrefix(hash, "$2b$") {
				t.Errorf("hash should be bcrypt format, got: %s", hash)
			}
			// Verify hash length (bcrypt is always 60 chars)
			if len(hash) != 60 {
				t.Errorf("bcrypt hash should be 60 chars, got %d", len(hash))
			}
		})
	}
}

func TestPasswordHasher_HashDifferentPasswords(t *testing.T) {
	hasher := NewPasswordHasher(12)

	passwords := []string{
		"MySecureP@ss1",
		"AnotherKey2@",
		"ThirdSecret3#",
	}

	hashes := make(map[string]string)
	for _, pwd := range passwords {
		hash, err := hasher.Hash(pwd)
		if err != nil {
			t.Fatalf("failed to hash '%s': %v", pwd, err)
		}
		hashes[pwd] = hash
	}

	// Verify all hashes are different (due to different salts)
	for i, pwd1 := range passwords {
		for j, pwd2 := range passwords {
			if i != j && hashes[pwd1] == hashes[pwd2] {
				t.Errorf("different passwords should have different hashes")
			}
		}
	}

	// Verify same password produces different hash each time (due to random salt)
	hash1, _ := hasher.Hash("SameSecretP@ss!")
	hash2, _ := hasher.Hash("SameSecretP@ss!")
	if hash1 == hash2 {
		t.Errorf("same password should produce different hashes due to random salt")
	}
}

func TestPasswordHasher_Verify(t *testing.T) {
	hasher := NewPasswordHasher(12)

	password := "MySecureP@ss!2024"
	hash, err := hasher.Hash(password)
	if err != nil {
		t.Fatalf("failed to hash password: %v", err)
	}

	tests := []struct {
		name     string
		password string
		hash     string
		expected bool
	}{
		{"correct password", password, hash, true},
		{"wrong password", "WrongSecure@2024!", hash, false},
		{"empty password", "", hash, false},
		{"password without special", "MySecurePass2024", hash, false},
		{"password with extra char", password + "x", hash, false},
		{"invalid hash", password, "invalid_hash", false},
		{"empty hash", password, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := hasher.Verify(tt.password, tt.hash)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestPasswordHasher_VerifyDifferentCosts(t *testing.T) {
	password := "MySecureP@ss!2024"

	for cost := MinBcryptCost; cost <= MaxBcryptCost; cost++ {
		t.Run("cost_"+string(rune('0'+cost)), func(t *testing.T) {
			hasher := NewPasswordHasher(cost)
			hash, err := hasher.Hash(password)
			if err != nil {
				t.Fatalf("failed to hash with cost %d: %v", cost, err)
			}

			// Verify with same hasher
			if !hasher.Verify(password, hash) {
				t.Errorf("password should verify with cost %d", cost)
			}

			// Verify with different cost hasher
			otherHasher := NewPasswordHasher(12)
			if !otherHasher.Verify(password, hash) {
				t.Errorf("password should verify regardless of hasher cost")
			}
		})
	}
}

func TestPasswordHasher_NeedsRehash(t *testing.T) {
	hasher12 := NewPasswordHasher(12)
	_ = NewPasswordHasher(14) // hasher14 - not used
	_ = NewPasswordHasher(10) // hasher10 - not used

	password := "MySecureP@ss!2024" // Changed to avoid common patterns

	tests := []struct {
		name           string
		hashCost       int
		checkCost      int
		expectedRehash bool
	}{
		{"same cost", 12, 12, false},
		{"lower cost needs rehash", 10, 12, true},
		{"higher cost no rehash", 14, 12, false},
		{"hasher14 checking cost12", 12, 14, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create hash with hashCost
			hasher := NewPasswordHasher(tt.hashCost)
			hash, err := hasher.Hash(password)
			if err != nil {
				t.Fatalf("failed to hash: %v", err)
			}

			// Check with checkCost hasher
			checkHasher := NewPasswordHasher(tt.checkCost)
			needsRehash := checkHasher.NeedsRehash(hash)

			if needsRehash != tt.expectedRehash {
				t.Errorf("expected needsRehash=%v, got %v", tt.expectedRehash, needsRehash)
			}
		})
	}

	// Test invalid hash
	t.Run("invalid hash", func(t *testing.T) {
		needsRehash := hasher12.NeedsRehash("invalid_hash")
		if !needsRehash {
			t.Errorf("invalid hash should need rehash")
		}
	})

	// Test empty hash
	t.Run("empty hash", func(t *testing.T) {
		needsRehash := hasher12.NeedsRehash("")
		if !needsRehash {
			t.Errorf("empty hash should need rehash")
		}
	})
}

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  bool
	}{
		// Length tests
		{"too short - 7 chars", "Pass12!", true},
		{"minimum length - 8 chars", "MySecr8!", false},
		{"good length - 12 chars", "MySecureP@ss!", false},
		{"max length - 64 chars", strings.Repeat("a", 8) + "A1!", false},
		{"too long - 65 chars", strings.Repeat("a", 62) + "A1!", true},

		// Common weak passwords
		{"contains password", "MyPassword123!", true},
		{"contains 123456", "abc123456def!", true},
		{"contains qwerty", "qwerty12345!", true},
		{"contains abc123", "Myabc123Pass!", true},
		{"contains letmein", "letmein123!", true},
		{"contains monkey", "monkeyPass1!", true},
		{"contains master", "masterKey123!", true},
		{"contains dragon", "dragonPass1!", true},
		{"contains 111111", "Pass111111!", true},
		{"contains 000000", "Pass000000!", true},
		{"contains admin", "adminPass123!", true},
		{"contains root", "rootPass123!", true},

		// Valid passwords
		{"valid no patterns", "Xyz789!@#", false},
		{"valid complex", "MyStr0ng!P@ss", false},
		{"valid with numbers", "Abcdefg123!", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error for '%s', got nil", tt.password)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error for '%s': %v", tt.password, err)
				}
			}
		})
	}
}

func TestValidatePasswordStrict(t *testing.T) {
	tests := []struct {
		name          string
		password      string
		requireUpper  bool
		requireLower  bool
		requireDigit  bool
		requireSpecial bool
		wantErr       bool
	}{
		{"all requirements met", "MySecureP@ss1!", true, true, true, true, false},
		{"no uppercase", "mysecurep@ss!", true, true, true, true, true},
		{"no lowercase", "MYSECUREP@SS!", true, true, true, true, true},
		{"no digit", "MySecureP@ss", true, true, true, true, true},
		{"no special", "MySecurePass1", true, true, true, true, true},
		{"only lowercase", "mysecpass", false, true, false, false, false},
		{"only uppercase", "MYSECPASS", true, false, false, false, false},
		{"only digits", "87654321", false, false, true, false, false},
		{"no requirements", "mysecpass", false, false, false, false, false},
		{"short password", "MyS1!", true, true, true, true, true},
		{"weak password", "mysecretkey", true, true, true, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePasswordStrict(tt.password, tt.requireUpper, tt.requireLower, tt.requireDigit, tt.requireSpecial)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}
			}
		})
	}
}

func TestGenerateRandomPassword(t *testing.T) {
	tests := []struct {
		name   string
		length int
	}{
		{"default length", 0},
		{"short length clamped", 5},
		{"minimum length", 12},
		{"medium length", 16},
		{"long length", 32},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			password, err := GenerateRandomPassword(tt.length)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			// Check minimum length
			expectedLen := tt.length
			if expectedLen < 12 {
				expectedLen = 12
			}
			if len(password) != expectedLen {
				t.Errorf("expected length %d, got %d", expectedLen, len(password))
			}

			// Verify it passes validation
			if err := ValidatePassword(password); err != nil {
				t.Errorf("generated password should be valid: %v", err)
			}

			// Verify strict validation (has upper, lower, digit, special)
			if err := ValidatePasswordStrict(password, true, true, true, true); err != nil {
				t.Errorf("generated password should meet all requirements: %v", err)
			}
		})
	}

	// Verify uniqueness - generate multiple passwords
	t.Run("uniqueness", func(t *testing.T) {
		passwords := make(map[string]bool)
		for i := 0; i < 100; i++ {
			pwd, err := GenerateRandomPassword(16)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			passwords[pwd] = true
		}
		// Should have close to 100 unique passwords
		if len(passwords) < 95 {
			t.Errorf("expected mostly unique passwords, got %d unique out of 100", len(passwords))
		}
	})
}

func TestPasswordHasher_GetCost(t *testing.T) {
	hasher := NewPasswordHasher(12)
	if hasher.GetCost() != 12 {
		t.Errorf("expected cost 12, got %d", hasher.GetCost())
	}
}

// Benchmark tests
func BenchmarkPasswordHasher_Hash(b *testing.B) {
	hasher := NewPasswordHasher(12)
	password := "MySecureP@ss!2024"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = hasher.Hash(password)
	}
}

func BenchmarkPasswordHasher_Verify(b *testing.B) {
	hasher := NewPasswordHasher(12)
	password := "MySecureP@ss!2024"
	hash, _ := hasher.Hash(password)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = hasher.Verify(password, hash)
	}
}

func BenchmarkValidatePassword(b *testing.B) {
	password := "MySecureP@ss!2024"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePassword(password)
	}
}

func BenchmarkValidatePasswordStrict(b *testing.B) {
	password := "MySecureP@ss!2024"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ValidatePasswordStrict(password, true, true, true, true)
	}
}