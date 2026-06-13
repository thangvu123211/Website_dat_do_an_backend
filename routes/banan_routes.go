package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
)

func BanAnRoutes(r *gin.Engine) {
	banan := r.Group("/banan")
	{
		banan.POST("/create", controllers.CreateBanAn)
		banan.GET("/layTatCa", controllers.GetAllBanAn)
		banan.GET("/layRaThongTinBanan/:id", controllers.GetBanAnByID)

		banan.PATCH("/update/:id", controllers.UpdateBanAn)
		banan.DELETE("/delete/:id", controllers.DeleteBanAn)
	}
}
