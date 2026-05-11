package password

import (
	"crypto/rand"
	"fmt"
	"math/big"
	"strings"
	"unicode"

	"github.com/zhaojiewen/open-station/pkg/errors"
	"golang.org/x/crypto/bcrypt"
)

const (
	// bcrypt cost范围: 4-31
	// cost=10: ~100ms
	// cost=12: ~400ms (推荐)
	// cost=14: ~1.5s (高安全场景)
	DefaultBcryptCost = 12
	MaxBcryptCost     = 14
	MinBcryptCost     = 10
)

// PasswordHasher 密码哈希服务
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher 创建密码哈希服务
func NewPasswordHasher(cost int) *PasswordHasher {
	if cost < MinBcryptCost {
		cost = MinBcryptCost
	}
	if cost > MaxBcryptCost {
		cost = MaxBcryptCost
	}
	return &PasswordHasher{cost: cost}
}

// Hash 创建密码哈希
func (h *PasswordHasher) Hash(password string) (string, error) {
	// 1. 验证密码复杂度
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	// 2. bcrypt哈希（自动包含salt）
	hash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		h.cost,
	)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// Verify 验证密码
func (h *PasswordHasher) Verify(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// NeedsRehash 检查哈希强度（用于迁移旧密码）
func (h *PasswordHasher) NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return true // 无效哈希，需要重新生成
	}
	return cost < h.cost // cost低于当前标准，需要升级
}

// GetCost 获取当前bcrypt cost
func (h *PasswordHasher) GetCost() int {
	return h.cost
}

// ValidatePassword 验证密码复杂度
func ValidatePassword(password string) error {
	// 长度检查
	if len(password) < 8 {
		return errors.ErrPasswordTooShort
	}
	if len(password) > 64 {
		return errors.ErrPasswordTooLong
	}

	// 检查是否包含常见弱密码模式
	lowerPassword := strings.ToLower(password)
	commonPatterns := []string{
		"password", "123456", "qwerty", "abc123",
		"letmein", "monkey", "master", "dragon",
		"111111", "000000", "admin", "root",
	}
	for _, pattern := range commonPatterns {
		if strings.Contains(lowerPassword, pattern) {
			return errors.ErrPasswordTooWeak
		}
	}

	return nil
}

// ValidatePasswordStrict 严格验证密码复杂度（可选）
func ValidatePasswordStrict(password string, requireUpper, requireLower, requireDigit, requireSpecial bool) error {
	// 基础验证
	if err := ValidatePassword(password); err != nil {
		return err
	}

	var (
		hasUpper   bool
		hasLower   bool
		hasDigit   bool
		hasSpecial bool
	)

	for _, r := range password {
		if unicode.IsUpper(r) {
			hasUpper = true
		}
		if unicode.IsLower(r) {
			hasLower = true
		}
		if unicode.IsDigit(r) {
			hasDigit = true
		}
		if unicode.IsPunct(r) || unicode.IsSymbol(r) {
			hasSpecial = true
		}
	}

	if requireUpper && !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if requireLower && !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if requireDigit && !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if requireSpecial && !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// GenerateRandomPassword 生成随机密码（用于重置等场景）
func GenerateRandomPassword(length int) (string, error) {
	if length < 12 {
		length = 12
	}

	// 包含大小写字母、数字、特殊字符
	const (
		uppercase = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
		lowercase = "abcdefghijklmnopqrstuvwxyz"
		digits    = "0123456789"
		special   = "!@#$%^&*()-_=+[]{}|;:,.<>?"
		all       = uppercase + lowercase + digits + special
	)

	// 确保每种字符至少有一个
	password := make([]byte, length)

	// 至少一个大写
	password[0] = uppercase[randomIndex(len(uppercase))]
	// 至少一个小写
	password[1] = lowercase[randomIndex(len(lowercase))]
	// 至少一个数字
	password[2] = digits[randomIndex(len(digits))]
	// 至少一个特殊字符
	password[3] = special[randomIndex(len(special))]

	// 剩余位置随机填充
	for i := 4; i < length; i++ {
		password[i] = all[randomIndex(len(all))]
	}

	// 打乱顺序
	for i := range password {
		j := randomIndex(length)
		password[i], password[j] = password[j], password[i]
	}

	return string(password), nil
}

// randomIndex uses crypto/rand to generate unbiased random index
func randomIndex(max int) int {
	if max <= 0 {
		return 0
	}
	// Calculate number of bytes needed
	n := 1
	bigMax := big.NewInt(int64(max))
	if bigMax.BitLen() > 8 {
		n = (bigMax.BitLen() + 7) / 8
	}

	b := make([]byte, n)
	for {
		_, _ = rand.Read(b)
		// Convert bytes to big.Int
		r := new(big.Int).SetBytes(b)
		// Use modulo to get index in range
		r.Mod(r, bigMax)
		if r.Int64() < int64(max) {
			return int(r.Int64())
		}
		// Retry if value is out of range (rare case)
	}
}