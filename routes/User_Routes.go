package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/controllers"
	"github.com/vpa/quanlynhahang-backend/middleware"
)

func UserRoutes(r *gin.Engine) {

	User := r.Group("/user")
	{
		User.GET("/so-tien-da-mua", middleware.AuthMiddleware(), middleware.RoleMiddleware("user"), controllers.GetTongTienDaMua)
		User.GET("/tong-hoa-don-da-thanh-toan-va-huy", middleware.AuthMiddleware(), middleware.RoleMiddleware("user"), controllers.TongSoHoaDonDaHuyVaThanhToan)
		User.GET("/tong-hoa-don-da-giao", middleware.AuthMiddleware(), middleware.RoleMiddleware("user"), controllers.TongSoDonHangDaGiao)
		User.GET("/tong-tien-theo-ngay", middleware.AuthMiddleware(), middleware.RoleMiddleware("user"), controllers.TongTienDaThanhToanTheoNgay)
	}
}
