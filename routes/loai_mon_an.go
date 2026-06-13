package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
)

func LoaiMonAnRoutes(r *gin.Engine) {
	loai := r.Group("/loaimonan")
	{
		loai.POST("/create", controllers.CreateLoaiMonAn)
		loai.GET("/all", controllers.GetAllLoaiMonAn)
		loai.GET("/:id", controllers.GetLoaiMonAnByID)
		loai.PATCH("/update/:id", controllers.UpdateLoaiMonAn)
		loai.DELETE("/delete/:id", controllers.DeleteLoaiMonAn)
	}
}
