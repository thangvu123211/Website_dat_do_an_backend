package controllers

import (
	//"fmt"
	//"log"
	//"math"
	"net/http"
	//"strconv"
	//"time"

	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/config"
	//"github.com/vpa/quanlynhahang-backend/dto"
	//"github.com/vpa/quanlynhahang-backend/internal/websocket"
	"github.com/vpa/quanlynhahang-backend/models"
	//"github.com/vpa/quanlynhahang-backend/utils"
	//"gorm.io/gorm"
	//"github.com/xuri/excelize/v2"
)

func GetTongTienDaMua(c *gin.Context) {

	// lấy user_id từ middleware
	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Vui lòng đăng nhập",
		})
		return
	}

	maNguoiDung, ok := maNguoiDungAny.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "user_id không hợp lệ",
		})
		return
	}

	var tongTien float64

	err := config.DB.
		Model(&models.HoaDon{}).
		Where("ma_nguoi_dung = ?", maNguoiDung).
		Where("trang_thai_thanh_toan = ?", "da_thanh_toan").
		Select("COALESCE(SUM(tong_tien), 0)").
		Scan(&tongTien).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy tổng tiền",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tong_tien_da_mua": tongTien,
	})
}

func TongSoHoaDonDaHuyVaThanhToan(c *gin.Context) {

	// lấy user_id từ middleware
	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Vui lòng đăng nhập",
		})
		return
	}

	maNguoiDung, ok := maNguoiDungAny.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "user_id không hợp lệ",
		})
		return
	}

	var tongHoaDonDaThanhToan int64 = 0
	var tongHoaDonDaHuy int64 = 0

	// ✅ Tổng số hóa đơn đã thanh toán
	err := config.DB.
		Model(&models.HoaDon{}).
		Where("ma_nguoi_dung = ?", maNguoiDung).
		Where("trang_thai_thanh_toan = ?", "da_thanh_toan").
		Count(&tongHoaDonDaThanhToan).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy số hóa đơn đã thanh toán",
		})
		return
	}

	// ✅ Tổng số hóa đơn đã hủy
	err = config.DB.
		Model(&models.HoaDon{}).
		Where("ma_nguoi_dung = ?", maNguoiDung).
		Where("trang_thai = ?", "da_huy").
		Where("trang_thai_thanh_toan = ?", "da_huy").
		Count(&tongHoaDonDaHuy).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy số hóa đơn đã hủy",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tong_hoa_don_da_thanh_toan": tongHoaDonDaThanhToan,
		"tong_hoa_don_da_huy":       tongHoaDonDaHuy,
	})
}

func TongSoDonHangDaGiao(c *gin.Context) {

	// lấy user_id từ middleware
	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Vui lòng đăng nhập",
		})
		return
	}

	maNguoiDung, ok := maNguoiDungAny.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "user_id không hợp lệ",
		})
		return
	}

	var tongDonHangDaGiao int64 = 0

	err := config.DB.
		Model(&models.HoaDon{}).
		Where("ma_nguoi_dung = ?", maNguoiDung).
		Where("trang_thai = ?", "da_giao").
		Where("trang_thai_thanh_toan = ?", "da_thanh_toan").
		Count(&tongDonHangDaGiao).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy tổng số đơn hàng đã giao",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tong_don_hang_da_giao": tongDonHangDaGiao,
	})
}

func TongTienDaThanhToanTheoNgay(c *gin.Context) {

	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Vui lòng đăng nhập"})
		return
	}

	maNguoiDung, ok := maNguoiDungAny.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "user_id không hợp lệ"})
		return
	}

	type KetQua struct {
		Ngay     string  `json:"ngay"`
		TongTien float64 `json:"tong_tien"`
	}

	var ketQua []KetQua

	err := config.DB.
		Model(&models.HoaDon{}).
		Select(`
			DATE(ngay) AS ngay,
			COALESCE(SUM(tong_tien), 0) AS tong_tien
		`).
		Where("ma_nguoi_dung = ?", maNguoiDung).
		Where("trang_thai = ?", "da_giao").
		Where("trang_thai_thanh_toan = ?", "da_thanh_toan").
		Group("DATE(ngay)").
		Order("DATE(ngay) ASC").
		Scan(&ketQua).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy thống kê theo ngày",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": ketQua,
	})
}