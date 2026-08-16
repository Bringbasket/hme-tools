// 密码哈希（docs/07 §7：bcrypt cost ≥ 10）
package common

import "golang.org/x/crypto/bcrypt"

const BcryptCost = 10

func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), BcryptCost)
	return string(b), err
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
