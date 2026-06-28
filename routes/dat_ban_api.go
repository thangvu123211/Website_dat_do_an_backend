package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
	"github.com/vpa/quanlynhahang-backend/internal/websocket"
	"github.com/vpa/quanlynhahang-backend/middleware"
)

func DatBanRoutes(r *gin.Engine, hub *websocket.Hub) {
	ctrl := controllers.NewDatBanController(hub)
	datban := r.Group("/dat-ban")
	{
		// Khách
		datban.POST("", middleware.AuthMiddleware(), ctrl.CreateDatBan)                                    // tạo đặt bàn //ok
		datban.GET("", middleware.AuthMiddleware(), middleware.RoleMiddleware("admin"), ctrl.GetAllDatBan) // danh sách //ok
		datban.GET("/:id", middleware.AuthMiddleware(), controllers.GetDatBanByID)                                // chi tiết
		datban.PUT("/:id", middleware.AuthMiddleware(), ctrl.UpdateDatBan)                                 // sửa thông tin
		datban.DELETE("/:id", middleware.AuthMiddleware(), ctrl.DeleteDatBan)
		datban.GET("/da-dat", ctrl.GetBanAnDaDat)     
		datban.GET("/khung-gio", ctrl.GetKhungGioBan)                             //ok

		// Nhân viên
		datban.PUT("/:id/xac-nhan", middleware.AuthMiddleware(), ctrl.XacNhanDatBan)
		datban.GET("/lay-danh-sach-dat-ban-cua-nguoi-dung", middleware.AuthMiddleware(), ctrl.GetDatBanCuaNguoiDung)
		datban.PUT("/huy-dat-ban/:id", middleware.AuthMiddleware(), ctrl.HuyDatBan)
	}
}
