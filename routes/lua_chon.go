package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
)

func OptionRoutes(r *gin.Engine,h *controllers.ChatHandler) {
	option := r.Group("/option")
	{
		// nhóm option
		option.POST("/nhom", h.CreateNhomOption)
		option.GET("/nhom", controllers.GetAllNhomOption)
		option.GET("/nhom/:id", controllers.GetNhomOptionByID)
		option.PUT("/nhom/:id", h.UpdateNhomOption)
		option.DELETE("/nhom/:id", h.DeleteNhomOption)

		// option item
		option.POST("/item", h.CreateOptionItem)
		option.GET("/item", controllers.GetAllOptionItem)
		option.GET("/item/:id", controllers.GetOptionItemByID)
		option.PUT("/item/:id", h.UpdateOptionItem)
		option.DELETE("/item/:id", h.DeleteOptionItem)
	}
}
