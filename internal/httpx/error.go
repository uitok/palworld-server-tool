package httpx

import "github.com/gin-gonic/gin"

type ErrorResponse struct {
	Error     string `json:"error"`
	ErrorCode string `json:"error_code,omitempty"`
	Details   any    `json:"details,omitempty"`
	Errors    int    `json:"errors,omitempty"`
}

func WriteError(c *gin.Context, status int, message string, code string, details any, errorsCount int) {
	resp := ErrorResponse{
		Error:     message,
		ErrorCode: code,
		Details:   details,
		Errors:    errorsCount,
	}
	c.AbortWithStatusJSON(status, resp)
}
