package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
	"github.com/vpa/quanlynhahang-backend/internal/websocket"
)

func BanAnRoutes(r *gin.Engine, hub *websocket.Hub , h *controllers.ChatHandler) {
	//ctrl := controllers.NewDatBanController(hub)
	banan := r.Group("/banan")
	{
		banan.POST("/create", h.CreateBanAn)
		banan.GET("/layTatCa", controllers.GetAllBanAn)
		banan.GET("/layRaThongTinBanan/:id", controllers.GetBanAnByID)

		banan.PATCH("/update/:id", h.UpdateBanAn)
		banan.DELETE("/delete/:id", h.DeleteBanAn)
	}
}
