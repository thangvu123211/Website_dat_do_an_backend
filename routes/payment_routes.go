package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
)

func Payment(r *gin.Engine) {
	r.POST("/qr", controllers.GetVietQR)
}
