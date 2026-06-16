package controllers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/config"
	"github.com/vpa/quanlynhahang-backend/models"
)

func TongTienShipperDaGiaoHomNay(c *gin.Context) {

	// 🔐 lấy user_id từ middleware
	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Vui lòng đăng nhập",
		})
		return
	}

	maShipper, ok := maNguoiDungAny.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "user_id không hợp lệ",
		})
		return
	}

	// 🕒 Lấy ngày hôm nay (00:00 → 23:59)
	today := time.Now().Format("2006-01-02")

	var tongTien float64 = 0

	err := config.DB.
		Model(&models.HoaDon{}).
		Select("COALESCE(SUM(tong_tien), 0)").
		Where("ma_shipper = ?", maShipper).
		Where("trang_thai = ?", "da_giao").
		Where("trang_thai_thanh_toan = ?", "da_thanh_toan").
		Where("DATE(ngay) = ?", today).
		Scan(&tongTien).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy tổng tiền shipper hôm nay",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tong_tien_giao_hom_nay": tongTien,
	})
}

func TongTienShipperDaGiao(c *gin.Context) {

	// 🔐 lấy user_id từ middleware
	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Vui lòng đăng nhập",
		})
		return
	}

	maShipper, ok := maNguoiDungAny.(uint)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "user_id không hợp lệ",
		})
		return
	}


	var tongTien float64 = 0

	err := config.DB.
		Model(&models.HoaDon{}).
		Select("COALESCE(SUM(tong_tien), 0)").
		Where("ma_shipper = ?", maShipper).
		Where("trang_thai = ?", "da_giao").
		Where("trang_thai_thanh_toan = ?", "da_thanh_toan").
		Scan(&tongTien).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy tổng tiền shipper hôm nay",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tong_tien_giao": tongTien,
	})
}
func TongSoDonHangDaGiaoHomNay(c *gin.Context) {

	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Vui lòng đăng nhập",
		})
		return
	}

	maShipper := maNguoiDungAny.(uint)

	today := time.Now().Format("2006-01-02")

	var tongDon int64 = 0

	err := config.DB.
		Model(&models.HoaDon{}).
		Where("ma_shipper = ?", maShipper).
		Where("trang_thai = ?", "da_giao").
		Where("trang_thai_thanh_toan = ?", "da_thanh_toan").
		Where("DATE(ngay) = ?", today).
		Count(&tongDon).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy số đơn hôm nay",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tong_don_hom_nay": tongDon,
	})
}

func TongTatCaDonHangDaGiao(c *gin.Context) {

	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Vui lòng đăng nhập",
		})
		return
	}

	maShipper := maNguoiDungAny.(uint)

	var tongDon int64 = 0

	err := config.DB.
		Model(&models.HoaDon{}).
		Where("ma_shipper = ?", maShipper).
		Where("trang_thai = ?", "da_giao").
		Where("trang_thai_thanh_toan = ?", "da_thanh_toan").
		Count(&tongDon).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy tổng số đơn đã giao",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"tong_tat_ca_don": tongDon,
	})
}
func ShipperTongTienDaGiaoTheoNgay(c *gin.Context) {

	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Vui lòng đăng nhập"})
		return
	}

	maShipper, ok := maNguoiDungAny.(uint)
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
		Where("ma_shipper = ?", maShipper).
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

	if ketQua == nil {
		ketQua = []KetQua{}
	}

	c.JSON(http.StatusOK, gin.H{
		"data": ketQua,
	})
}

