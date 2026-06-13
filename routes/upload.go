package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
)

func UploadRoutes(route *gin.Engine) {
	route.POST("/upload", controllers.UploadHandler)
	route.GET("/images", controllers.GetImage)
	//route.GET("/images", controllers.GetProductImage)

}
