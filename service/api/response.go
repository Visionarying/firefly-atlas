package api

import "github.com/gin-gonic/gin"

// Response is the unified envelope used by every API.
type Response struct {
	Code int         `json:"code"`
	Data interface{} `json:"data"`
	Msg  string      `json:"msg"`
}

func OK(c *gin.Context, data interface{}) {
	c.JSON(200, Response{Code: 0, Data: data, Msg: "success"})
}

func Fail(c *gin.Context, msg string) {
	c.JSON(200, Response{Code: -1, Data: nil, Msg: msg})
}

func FailCode(c *gin.Context, code int, msg string) {
	c.JSON(200, Response{Code: code, Data: nil, Msg: msg})
}
