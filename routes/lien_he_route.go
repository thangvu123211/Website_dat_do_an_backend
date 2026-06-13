package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
)

func LienHeRoutes(r *gin.Engine, contactHandler *controllers.ContactHandler) {
	lienhe := r.Group("/lien-he")
	{
		lienhe.POST("/create", contactHandler.GuiLienHe)
		lienhe.GET("", controllers.AdminGetAllLienHe)
		lienhe.DELETE("/:id", controllers.DeleteLienHe)
	}
}
