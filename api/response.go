package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/zaigie/palworld-server-tool/internal/config"
	"github.com/zaigie/palworld-server-tool/internal/httpx"
	"github.com/zaigie/palworld-server-tool/internal/tool"
)

func writeError(c *gin.Context, status int, message string, code string, details any, errorsCount int) {
	httpx.WriteError(c, status, message, code, details, errorsCount)
}

func writeBadRequest(c *gin.Context, message string) {
	writeError(c, http.StatusBadRequest, message, "bad_request", nil, 0)
}

func writeBadRequestCode(c *gin.Context, message string, code string) {
	writeError(c, http.StatusBadRequest, message, code, nil, 0)
}

func writeBadRequestErr(c *gin.Context, err error) {
	if err == nil {
		writeBadRequest(c, "bad request")
		return
	}

	var validationErr *config.ValidationError
	if errors.As(err, &validationErr) {
		writeBadRequestDetails(c, validationErr.Error(), "invalid_configuration", validationErr.Issues, len(validationErr.Issues))
		return
	}

	if code := tool.SaveOperationErrorCode(err); code != "" {
		writeBadRequestDetails(c, err.Error(), code, tool.SaveOperationErrorDetails(err), 1)
		return
	}

	writeBadRequest(c, err.Error())
}

func writeBadRequestDetails(c *gin.Context, message string, code string, details any, errorsCount int) {
	writeError(c, http.StatusBadRequest, message, code, details, errorsCount)
}

func writeNotFound(c *gin.Context, message string) {
	writeError(c, http.StatusNotFound, message, "not_found", nil, 0)
}

func writeUnauthorized(c *gin.Context, message string) {
	writeError(c, http.StatusUnauthorized, message, "unauthorized", nil, 0)
}

func writeSuccess(c *gin.Context) {
	c.JSON(http.StatusOK, SuccessResponse{Success: true})
}

func writeSuccessMessage(c *gin.Context, message string) {
	c.JSON(http.StatusOK, gin.H{"success": true, "message": message})
}

func writeInternalError(c *gin.Context, message string) {
	writeError(c, http.StatusInternalServerError, message, "internal_error", nil, 0)
}

func writeInternalErrorErr(c *gin.Context, err error) {
	if err == nil {
		writeInternalError(c, "internal server error")
		return
	}
	writeInternalError(c, err.Error())
}
