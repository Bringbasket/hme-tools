// 数学图形验证码（SVG 内嵌文本，无需字体文件；docs/04 §4 验证码接口）
package common

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math/big"
)

type MathCaptcha struct {
	Expression   string // 如 "3 + 5 = ?"
	Answer       string // 如 "8"
	ImageDataURI string // data:image/svg+xml;base64,...
}

func NewMathCaptcha() (*MathCaptcha, error) {
	a, err := rand.Int(rand.Reader, big.NewInt(10))
	if err != nil {
		return nil, err
	}
	b, err := rand.Int(rand.Reader, big.NewInt(10))
	if err != nil {
		return nil, err
	}
	expr := fmt.Sprintf("%d + %d = ?", a.Int64(), b.Int64())
	answer := fmt.Sprintf("%d", a.Int64()+b.Int64())
	svg := fmt.Sprintf(
		`<svg xmlns="http://www.w3.org/2000/svg" width="150" height="50"><rect width="150" height="50" rx="7" fill="#f0fdfa"/><text x="16" y="34" font-size="24" font-family="Consolas,monospace" font-weight="bold" fill="#0f766e">%s</text></svg>`,
		expr,
	)
	return &MathCaptcha{
		Expression:   expr,
		Answer:       answer,
		ImageDataURI: "data:image/svg+xml;base64," + base64.StdEncoding.EncodeToString([]byte(svg)),
	}, nil
}
