// 统一响应包与分页（docs/04-API接口规范 §2/§3）
package common

import "net/http"

type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data"`
}

func OK(data interface{}) Response {
	return Response{Code: http.StatusOK, Msg: "操作成功", Data: data}
}

func Fail(code int, msg string) Response {
	return Response{Code: code, Msg: msg, Data: nil}
}

type PageResult[T any] struct {
	List     []T   `json:"list"`
	Total    int64 `json:"total"`
	Page     int   `json:"page"`
	PageSize int   `json:"pageSize"`
}
