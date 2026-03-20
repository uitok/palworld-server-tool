package auth

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"github.com/zaigie/palworld-server-tool/internal/httpx"
)

var prefixBearer = "Bearer "
var prefixJWT = "JWT "

func getSecretKey() []byte {
	return []byte(viper.GetString("web.password"))
}

func GetSecretKey() []byte {
	return getSecretKey()
}

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")

		var tokenString string
		if strings.HasPrefix(authHeader, prefixBearer) {
			tokenString = strings.TrimPrefix(authHeader, prefixBearer)
		} else if strings.HasPrefix(authHeader, prefixJWT) {
			tokenString = strings.TrimPrefix(authHeader, prefixJWT)
		} else {
			httpx.WriteError(c, http.StatusUnauthorized, "unauthorized - token missing", "auth_token_missing", nil, 0)
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return getSecretKey(), nil
		})

		if err != nil {
			httpx.WriteError(c, http.StatusUnauthorized, "unauthorized - invalid token", "auth_token_invalid", nil, 0)
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("claims", claims)
		} else {
			httpx.WriteError(c, http.StatusUnauthorized, "unauthorized - invalid claims", "auth_token_claims_invalid", nil, 0)
			c.Abort()
			return
		}

		c.Next()
	}
}

func OptionalJWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set("loggedIn", false)
		authHeader := c.GetHeader("Authorization")
		if authHeader != "" {
			var tokenString string
			if strings.HasPrefix(authHeader, prefixBearer) {
				tokenString = strings.TrimPrefix(authHeader, prefixBearer)
			} else if strings.HasPrefix(authHeader, prefixJWT) {
				tokenString = strings.TrimPrefix(authHeader, prefixJWT)
			}
			if tokenString != "" {
				token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
					return getSecretKey(), nil
				})
				if err == nil && token != nil {
					if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
						c.Set("claims", claims)
						c.Set("loggedIn", true)
					}
				}
			}
		}
		c.Next()
	}
}

func GenerateToken() (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour * 24).Unix(),
	})
	tokenString, err := token.SignedString(getSecretKey())
	if err != nil {
		return "", err
	}
	return tokenString, nil
}
