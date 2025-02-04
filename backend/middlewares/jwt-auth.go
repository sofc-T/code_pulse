package middlewares

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/dgrijalva/jwt-go"
	core "github.com/sofc-t/code_pulse/delivery/core"
)

// AuthorizeJWT validates the token from the http request, returning a 401 if it's not valid
func AuthorizeJWT() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString := ""
		receivemodelsken := c.Query("token")
		if receivemodelsken != "" {
			tokenString = receivemodelsken
		} else {
			const BEARER_SCHEMA = "Bearer "
			authHeader := c.GetHeader("Authorization")
			if authHeader == "" || len(authHeader) < len(BEARER_SCHEMA) {
				c.AbortWithStatus(http.StatusUnauthorized)
			}
			tokenString = authHeader[len(BEARER_SCHEMA):]
		}

		token, err := core.NewJWTService().ValidateToken(tokenString)
		print(token)
		print(err)
		if err != nil || token == nil {
			c.AbortWithStatus(http.StatusUnauthorized)
			return 
		}
		if token.Valid {
			claims := token.Claims.(jwt.MapClaims)
			log.Println("Claims[Name]: ", claims["name"])
			log.Println("Claims[Admin]: ", claims["admin"])
			log.Println("Claims[Issuer]: ", claims["iss"])
			log.Println("Claims[IssuedAt]: ", claims["iat"])
			log.Println("Claims[ExpiresAt]: ", claims["exp"])
		} else {
			log.Println(err)
			c.AbortWithStatus(http.StatusUnauthorized)
		}
	}
}
