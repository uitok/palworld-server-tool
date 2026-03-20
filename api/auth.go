package api

import (
	"net/http"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/zaigie/palworld-server-tool/internal/auth"
)

type LoginInfo struct {
	Password string `json:"password"`
}

// loginHandler godoc
// @Summary		Login
// @Description	Login
// @Tags			Auth
// @Accept			json
// @Produce		json
// @Param			login_info	body		LoginInfo	true	"Login Info"
// @Success		200			{object}	SuccessResponse
// @Failure		400			{object}	ErrorResponse
// @Failure		401			{object}	ErrorResponse
// @Router			/api/login [post]
func loginHandler(c *gin.Context) {
	var loginInfo LoginInfo
	if err := c.ShouldBindJSON(&loginInfo); err != nil {
		writeBadRequestErr(c, err)
		return
	}
	correctPassword := viper.GetString("web.password")
	if loginInfo.Password != correctPassword {
		writeError(c, http.StatusUnauthorized, "incorrect password", "auth_failed", nil, 0)
		return
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(auth.GetSecretKey())
	if err != nil {
		writeBadRequestCode(c, "could not generate token", "token_generation_failed")
		return
	}
	c.JSON(http.StatusOK, gin.H{"token": tokenString})
}
