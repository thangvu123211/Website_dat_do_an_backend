package middleware

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/vpa/quanlynhahang-backend/utils"
	//"github.com/vpa/quanlynhahang-backend/utils"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Thiếu hoặc sai định dạng Authorization header"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// ✅ Giải mã token bằng secret
		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			// Kiểm tra thuật toán ký
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("phương thức ký không hợp lệ")
			}
			return utils.JWTSecret(), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token không hợp lệ hoặc đã hết hạn"})
			c.Abort()
			return
		}

		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Không thể đọc thông tin token"})
			c.Abort()
			return
		}

		// ✅ Lấy các giá trị từ claims một cách an toàn (không panic)
		var userID uint
		if id, ok := claims["id"].(float64); ok {
			userID = uint(id)
		}

		email, _ := claims["email"].(string)
		role, _ := claims["role"].(string)

		// ✅ Lưu thông tin vào context để các controller dùng lại
		c.Set("user_id", userID)
		c.Set("email", email)
		c.Set("role", role)

		c.Next()
	}
}
