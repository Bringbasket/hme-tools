// 业务错误模型（docs/04 §6：业务码与 HTTP 状态一致）
package common

import (
	"errors"
	"net/http"
)

type BizError struct {
	Code int
	Msg  string
}

func (e *BizError) Error() string { return e.Msg }

func NewBizError(code int, msg string) *BizError {
	return &BizError{Code: code, Msg: msg}
}

// ToResponse 将任意 error 映射为统一响应包（docs/07 §5）
func ToResponse(err error) Response {
	var biz *BizError
	if errors.As(err, &biz) {
		return Fail(biz.Code, biz.Msg)
	}
	return Fail(http.StatusInternalServerError, "系统繁忙，请稍后重试")
}
