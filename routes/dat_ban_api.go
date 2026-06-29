package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
	"github.com/vpa/quanlynhahang-backend/middleware"
)

func DatBanRoutes(r *gin.Engine, h *controllers.ChatHandler) {
	//ctrl := controllers.NewDatBanController(hub)
	datban := r.Group("/dat-ban")
	{
		// Khách
		datban.POST("", middleware.AuthMiddleware(), h.CreateDatBan)                                    // tạo đặt bàn //ok
		datban.GET("", middleware.AuthMiddleware(), middleware.RoleMiddleware("admin"), h.GetAllDatBan) // danh sách //ok
		datban.GET("/:id", middleware.AuthMiddleware(), controllers.GetDatBanByID)                                // chi tiết
		datban.PUT("/:id", middleware.AuthMiddleware(), h.UpdateDatBan)                                 // sửa thông tin
		datban.DELETE("/:id", middleware.AuthMiddleware(), h.DeleteDatBan)
		datban.GET("/da-dat", h.GetBanAnDaDat)     
		datban.GET("/khung-gio", h.GetKhungGioBan)                             //ok

		// Nhân viên
		datban.PUT("/:id/xac-nhan", middleware.AuthMiddleware(), h.XacNhanDatBan)
		datban.GET("/lay-danh-sach-dat-ban-cua-nguoi-dung", middleware.AuthMiddleware(), h.GetDatBanCuaNguoiDung)
		datban.PUT("/huy-dat-ban/:id", middleware.AuthMiddleware(), h.HuyDatBan)
	}
}
