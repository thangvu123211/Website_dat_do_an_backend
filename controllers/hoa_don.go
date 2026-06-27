package controllers

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jung-kurt/gofpdf"
	"github.com/vpa/quanlynhahang-backend/config"
	"github.com/vpa/quanlynhahang-backend/dto"
	"github.com/vpa/quanlynhahang-backend/internal/websocket"
	"github.com/vpa/quanlynhahang-backend/models"
	"github.com/vpa/quanlynhahang-backend/utils"
	"gorm.io/gorm"
)

type HoaDonController struct {
	Hub *websocket.Hub
}

func NewHoaDonController(hub *websocket.Hub) *HoaDonController {
	return &HoaDonController{
		Hub: hub,
	}
}

type OptionDatInput struct {
	MaOptionItem uint `json:"ma_option_item"`
}

type MonDatInput struct {
	MaMonAn uint   `json:"ma_mon_an"`
	SoLuong int    `json:"so_luong"`
	GhiChu  string `json:"ghi_chu"`

	Options []OptionDatInput `json:"options"`
}

type DatDoAnInput struct {
	HoTen       string `json:"ho_ten"`
	SDT         string `json:"sdt"`
	DiaChi      string `json:"dia_chi"`
	GhiChu      string `json:"ghi_chu"`
	CodeGiamGia string `json:"code_giam_gia"`

	MonAns []MonDatInput `json:"mon_ans"`
}

func (ctrl *HoaDonController) DatDoAn(c *gin.Context) {

	var input DatDoAnInput

	if err := c.ShouldBindJSON(&input); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Dữ liệu không hợp lệ",
		})
		return
	}

	// lấy user từ middleware
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

	// validate input
	if input.HoTen == "" ||
		input.SDT == "" ||
		input.DiaChi == "" {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Thiếu thông tin khách hàng",
		})
		return
	}

	if len(input.MonAns) == 0 {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Chưa có món ăn",
		})
		return
	}

	tx := config.DB.Begin()

	// rollback nếu panic
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	var tongTienServer float64

	// tạo hóa đơn
	hoaDon := models.HoaDon{
		MaNguoiDung:        maNguoiDung,
		HoTen:              input.HoTen,
		SDT:                input.SDT,
		DiaChi:             input.DiaChi,
		GhiChu:             input.GhiChu,
		Ngay:               time.Now(),
		TrangThai:          "cho_xac_nhan",
		TrangThaiThanhToan: "chua_thanh_toan",
	}

	if err := tx.Create(&hoaDon).Error; err != nil {

		tx.Rollback()

		log.Println("CREATE HOADON ERROR:", err)

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể tạo hóa đơn",
		})
		return
	}

	// thêm món ăn
	for _, item := range input.MonAns {

		if item.SoLuong <= 0 {
			continue
		}

		var monAn models.MonAn

		if err := tx.
			First(&monAn, "ma_mon_an = ?", item.MaMonAn).Error; err != nil {

			tx.Rollback()

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Món ăn không tồn tại",
			})
			return
		}

		// thanhTien := monAn.GiaTien * float64(item.SoLuong)

		optionTotal := 0.0
		log.Println("OPTIONS:", item.Options)

		for _, op := range item.Options {

			var optionItem models.OptionItem

			if err := tx.
				First(&optionItem, "ma_option_item = ?", op.MaOptionItem).
				Error; err != nil {

				tx.Rollback()

				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Option không tồn tại",
				})
				return
			}

			optionTotal += optionItem.GiaThem
		}

		// giá 1 phần
		donGiaSauOption := monAn.GiaBan + optionTotal

		// thành tiền
		thanhTien := donGiaSauOption * float64(item.SoLuong)

		tongTienServer += thanhTien

		chiTiet := models.ChiTietHoaDon{
			MaHoaDon:  hoaDon.MaHoaDon,
			MaMonAn:   item.MaMonAn,
			SoLuong:   item.SoLuong,
			DonGia:    monAn.GiaBan,
			ThanhTien: thanhTien,
		}

		if err := tx.Create(&chiTiet).Error; err != nil {

			tx.Rollback()

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể thêm món ăn",
			})
			return
		}

		for _, op := range item.Options {

			var optionItem models.OptionItem

			if err := tx.First(&optionItem, "ma_option_item = ?", op.MaOptionItem).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Option không tồn tại",
				})
				return
			}

			ctOption := models.ChiTietHoaDonOption{
				MaChiTiet:    chiTiet.MaChiTiet,
				MaOptionItem: optionItem.MaOptionItem,
				TenOption:    optionItem.TenOption,
				GiaThem:      optionItem.GiaThem,
			}

			if err := tx.Create(&ctOption).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Không thể lưu option",
				})
				return
			}
		}
	}

	// =========================
	// xử lý mã giảm giá
	// =========================

	var tienGiam float64
	var giamGia models.GiamGia

	if input.CodeGiamGia != "" {

		err := tx.
			Where("code = ?", input.CodeGiamGia).
			First(&giamGia).Error

		if err != nil {

			tx.Rollback()

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Mã giảm giá không tồn tại",
			})
			return
		}

		// check active
		if !giamGia.IsActive {

			tx.Rollback()

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Mã giảm giá đã bị khóa",
			})
			return
		}

		now := time.Now()

		// check thời gian
		if now.Before(giamGia.NgayBatDau) ||
			now.After(giamGia.NgayKetThuc) {

			tx.Rollback()

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Mã giảm giá đã hết hạn",
			})
			return
		}

		// check đơn tối thiểu
		if tongTienServer < giamGia.DonToiThieu {

			tx.Rollback()

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Chưa đủ giá trị đơn tối thiểu",
			})
			return
		}

		// check giới hạn sử dụng
		if giamGia.GioiHanSuDung != nil &&
			giamGia.SoLanDaDung >= *giamGia.GioiHanSuDung {

			tx.Rollback()

			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Mã giảm giá đã hết lượt sử dụng",
			})
			return
		}

		// tính giảm giá
		switch giamGia.LoaiGiamGia {

		case "percent":

			tienGiam =
				tongTienServer *
					giamGia.GiaTriGiam / 100
			log.Println("TONG:", tongTienServer)
			log.Println("GIAM %:", giamGia.GiaTriGiam)
			log.Println("TIEN GIAM:", tienGiam)

		case "fixed":

			tienGiam = giamGia.GiaTriGiam
		}

		// tránh âm tiền
		if tienGiam > tongTienServer {
			tienGiam = tongTienServer
		}

		// gắn voucher vào hóa đơn
		hoaDon.GiamGiaID = &giamGia.ID

		// tăng số lần sử dụng
		if err := tx.Model(&giamGia).
			Update(
				"so_lan_da_dung",
				gorm.Expr("so_lan_da_dung + ?", 1),
			).Error; err != nil {

			tx.Rollback()

			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể cập nhật mã giảm giá",
			})
			return
		}
	}

	// tổng cuối
	tongCuoi := tongTienServer - tienGiam

	updateData := map[string]interface{}{
		"tam_tinh":    tongTienServer,
		"tien_giam":   tienGiam,
		"tong_tien":   tongCuoi,
		"giam_gia_id": hoaDon.GiamGiaID,
	}

	if err := tx.
		Model(&hoaDon).
		Updates(updateData).Error; err != nil {

		tx.Rollback()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể cập nhật hóa đơn",
		})
		return
	}

	// commit
	if err := tx.Commit().Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lưu hóa đơn",
		})
		return
	}

	content := fmt.Sprintf("HD%07d", hoaDon.MaHoaDon)

	// Tạo qr động chuyển khoản từ sepay có webhook gửi về serve
	qrURL := utils.GenerateSePayQR(
		"0123456789", // STK
		"MBBank",
		int(tongCuoi),
		content,
	)

	// lấy kết quả cuối
	var result models.HoaDon

	if err := config.DB.
		Preload("GiamGia").
		Preload("ChiTietHoaDons").
		Preload("ChiTietHoaDons.MonAn").
		First(&result, "ma_hoa_don = ?", hoaDon.MaHoaDon).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy hóa đơn",
		})
		return
	}

	// ✉️ gửi mail xác nhận — chạy nền, không block response
	go func() {
		// lấy email từ DB theo maNguoiDung
		var nguoiDung models.NguoiDung
		if err := config.DB.First(&nguoiDung, maNguoiDung).Error; err != nil {
			log.Printf("SendMail: không lấy được email user %d: %v", maNguoiDung, err)
			return
		}

		err := utils.SendMailSauKhiDatDoAn(nguoiDung.Email, utils.DatDoAnMailInfo{
			TenKhachHang: result.HoTen,
			MaDon:        fmt.Sprintf("%d", result.MaHoaDon),
			NgayGio:      result.Ngay.Format("02/01/2006 lúc 15:04"),
			DiaChi:       result.DiaChi,
			SoMonAn:      len(result.ChiTietHoaDons),
			TamTinh:      result.TamTinh,
			TienGiam:     result.TienGiam,
			TongCuoi:     result.TongTien,
			GhiChu:       result.GhiChu,
		})
		if err != nil {
			log.Printf("SendMail: lỗi gửi mail đơn #%d: %v", result.MaHoaDon, err)
		}
	}()

	// realtime cho admin
	ctrl.Hub.Broadcast(dto.WSMessage{
		Type:    "new_hoa_don",
		Payload: result,
	})

	// realtime cho user
	ctrl.Hub.Broadcast(dto.WSMessage{
		Type:    "new_hoa_don_user",
		Payload: result,
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Đặt đồ ăn thành công",
		"data":    result,
		"qr_url":  qrURL,
	})
}

func (ctrl *HoaDonController) XoaHoaDon(c *gin.Context) {

	id := c.Param("id")

	var hoaDon models.HoaDon

	// kiểm tra hóa đơn tồn tại
	if err := config.DB.
		First(&hoaDon, "ma_hoa_don = ?", id).Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Hóa đơn không tồn tại",
		})
		return
	}

	tx := config.DB.Begin()
	tx.Where("ma_hoa_don = ?", id).Delete(&models.ChiTietHoaDonOption{})
	// xóa chi tiết hóa đơn trước
	if err := tx.
		Where("ma_hoa_don = ?", id).
		Delete(&models.ChiTietHoaDon{}).Error; err != nil {

		tx.Rollback()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể xóa chi tiết hóa đơn",
		})
		return
	}

	// xóa hóa đơn
	if err := tx.
		Delete(&hoaDon).Error; err != nil {

		tx.Rollback()

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể xóa hóa đơn",
		})
		return
	}

	tx.Commit()

	c.JSON(http.StatusOK, gin.H{
		"message": "Xóa hóa đơn thành công",
	})
}

func (ctrl *HoaDonController) GetHoaDons(c *gin.Context) {

	var hoaDons []models.HoaDon

	if err := config.DB.
		Preload("Shipper").
		Preload("ChiTietHoaDons").
		Preload("ChiTietHoaDons.MonAn").
		Preload("ChiTietHoaDons.Options").
		Preload("ChiTietHoaDons.Options.OptionItem").
		Preload("ChiTietHoaDons.Options.OptionItem.NhomOption").
		Order("ma_hoa_don DESC").
		Find(&hoaDons).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy danh sách hóa đơn",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": hoaDons,
	})
}

func (ctrl *HoaDonController) GetHoaDonsToday(c *gin.Context) {

	var hoaDons []models.HoaDon

	// Lấy thời gian hiện tại theo VN (nếu cần)
	loc, _ := time.LoadLocation("Asia/Ho_Chi_Minh")
	now := time.Now().In(loc)

	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	endOfDay := startOfDay.Add(24*time.Hour - time.Nanosecond)

	if err := config.DB.
		Preload("Shipper").
		Preload("ChiTietHoaDons").
		Preload("ChiTietHoaDons.MonAn").
		Preload("ChiTietHoaDons.Options").
		Preload("ChiTietHoaDons.Options.OptionItem").
		Preload("ChiTietHoaDons.Options.OptionItem.NhomOption").
		Where("ngay BETWEEN ? AND ?", startOfDay, endOfDay).
		Order("ma_hoa_don DESC").
		Find(&hoaDons).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy danh sách hóa đơn hôm nay",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": hoaDons,
	})
}

func (ctrl *HoaDonController) GetHoaDonByShipper(c *gin.Context) {

	shipperIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Chưa đăng nhập",
		})
		return
	}

	shipperID := shipperIDAny.(uint)

	var hoaDons []models.HoaDon

	if err := config.DB.
		Where("ma_shipper = ? AND trang_thai = ?", shipperID, "da_giao_shipper").
		Preload("Shipper").
		Preload("Shipper").
		Preload("ChiTietHoaDons").
		Preload("ChiTietHoaDons.MonAn").
		Preload("ChiTietHoaDons.Options").
		Preload("ChiTietHoaDons.Options.OptionItem").
		Preload("ChiTietHoaDons.Options.OptionItem.NhomOption").
		Order("ma_hoa_don DESC").
		Find(&hoaDons).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": hoaDons,
	})
}

func (ctrl *HoaDonController) GetHoaDonByID(c *gin.Context) {

	id := c.Param("id")

	// lấy user_id từ token
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Chưa đăng nhập",
		})
		return
	}

	userID := userIDAny.(uint)

	var hoaDon models.HoaDon

	if err := config.DB.
		Preload("ChiTietHoaDons").
		Preload("ChiTietHoaDons.Options").
		First(&hoaDon, "ma_hoa_don = ? AND ma_nguoi_dung = ?", id, userID).Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Bạn không có quyền xem hóa đơn này",
		})
		return
	}
	log.Println("userID:", userID)
	log.Println("hoaDonID:", id)
	// QR mặc định rỗng
	qrURL := ""

	// Nếu chưa thanh toán thì tạo QR
	if hoaDon.TrangThaiThanhToan != "da_thanh_toan" {

		qrURL = utils.GenerateSePayQR(
			"123456789", // số tài khoản
			"MB",        // mã ngân hàng
			int(hoaDon.TongTien),
			fmt.Sprintf("HD%d", hoaDon.MaHoaDon),
		)
	}

	// realtime
	ctrl.Hub.Broadcast(dto.WSMessage{
		Type: "xem_hoa_don_da_dat",
		Payload: gin.H{
			"hoa_don": hoaDon,
			"qr_url":  qrURL,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"data":   hoaDon,
		"qr_url": qrURL,
	})
}

func (ctrl *HoaDonController) UpdateTrangThaiHoaDon(c *gin.Context) {

	id := c.Param("id")

	var input struct {
		TrangThai string `json:"trang_thai"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Dữ liệu không hợp lệ",
		})
		return
	}

	// tìm hóa đơn
	var hoaDon models.HoaDon

	if err := config.DB.First(&hoaDon, "ma_hoa_don = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy hóa đơn",
		})
		return
	}

	// update thẳng trạng thái (KHÔNG validate gì hết)
	if err := config.DB.
		Model(&hoaDon).
		Update("trang_thai", input.TrangThai).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể cập nhật trạng thái",
		})
		return
	}

	// reload lại data mới
	config.DB.First(&hoaDon, "ma_hoa_don = ?", id)

	// broadcast realtime admin
	ctrl.Hub.Broadcast(dto.WSMessage{
		Type:    "update_trang_thai_hoa_don",
		Payload: hoaDon,
	})

	// broadcast realtime user
	ctrl.Hub.Broadcast(dto.WSMessage{
		Type: "update_trang_thai_hoa_don_user",
		Payload: gin.H{
			"ma_hoa_don": hoaDon.MaHoaDon,
			"trang_thai": hoaDon.TrangThai,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật trạng thái thành công",
		"hoa_don": hoaDon,
	})
}

func (ctrl *HoaDonController) HuyHoaDon(c *gin.Context) {

	id := c.Param("id")

	var hoaDon models.HoaDon

	if err := config.DB.
		First(&hoaDon, "ma_hoa_don = ?", id).Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy hóa đơn",
		})
		return
	}

	if hoaDon.TrangThai == "da_giao" {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Không thể hủy hóa đơn đã giao",
		})
		return
	}

	if err := config.DB.Model(&hoaDon).
		Update("trang_thai", "da_huy").Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể hủy hóa đơn",
		})
		return
	}

	config.DB.First(&hoaDon, "ma_hoa_don = ?", id)

	ctrl.Hub.Broadcast(dto.WSMessage{
		Type:    "cancel_hoa_don",
		Payload: hoaDon,
	})

	ctrl.Hub.Broadcast(dto.WSMessage{
		Type: "cancel_hoa_don_user",
		Payload: gin.H{
			"ma_hoa_don": hoaDon.MaHoaDon,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Hủy hóa đơn thành công",
	})
}

func (ctrl *HoaDonController) GetHoaDonByTrangThai(c *gin.Context) {

	trangThai := c.Query("trang_thai")

	var hoaDons []models.HoaDon

	if err := config.DB.
		Where("trang_thai = ?", trangThai).
		Preload("Shipper").
		Preload("ChiTietHoaDons").
		Preload("ChiTietHoaDons.Options").
		Find(&hoaDons).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy hóa đơn",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": hoaDons,
	})
}

func (ctrl *HoaDonController) UpdateHoaDon(c *gin.Context) {

	id := c.Param("id")

	var input struct {
		HoTen  string `json:"ho_ten"`
		SDT    string `json:"sdt"`
		DiaChi string `json:"dia_chi"`
		GhiChu string `json:"ghi_chu"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Dữ liệu không hợp lệ",
		})
		return
	}

	var hoaDon models.HoaDon

	if err := config.DB.
		First(&hoaDon, "ma_hoa_don = ?", id).Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy hóa đơn",
		})
		return
	}

	if err := config.DB.Model(&hoaDon).Updates(models.HoaDon{
		HoTen:  input.HoTen,
		SDT:    input.SDT,
		DiaChi: input.DiaChi,
		GhiChu: input.GhiChu,
	}).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể cập nhật hóa đơn",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật hóa đơn thành công",
	})
}

func (ctrl *HoaDonController) GetHoaDonByNguoiDung(c *gin.Context) {

	maNguoiDungAny, exists := c.Get("user_id")

	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Chưa đăng nhập",
		})
		return
	}

	maNguoiDung := maNguoiDungAny.(uint)

	var hoaDons []models.HoaDon

	if err := config.DB.
		Where("ma_nguoi_dung = ?", maNguoiDung).
		Preload("ChiTietHoaDons").
		Preload("Shipper").
		Preload("Shipper.AnhNhanVien").
		Preload("ChiTietHoaDons.Options.OptionItem.NhomOption").
		Preload("ChiTietHoaDons.MonAn").
		Preload("ChiTietHoaDons.Options").
		Preload("ChiTietHoaDons.Options.OptionItem").
		Order("ma_hoa_don DESC").
		Find(&hoaDons).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	ctrl.Hub.Broadcast(dto.WSMessage{
		Type: "xem_tat_ca_hoa_don_da_dat",
		Payload: gin.H{
			"hoa_don": hoaDons,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"data": hoaDons,
	})
}

func (ctrl *HoaDonController) HuyHoaDonNguoiDung(c *gin.Context) {

	id := c.Param("id")

	// lấy user_id từ token
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Chưa đăng nhập",
		})
		return
	}
	userID := userIDAny.(uint)

	var hoaDon models.HoaDon

	// 🔒 chỉ chủ hóa đơn mới hủy được
	if err := config.DB.
		First(&hoaDon, "ma_hoa_don = ? AND ma_nguoi_dung = ?", id, userID).
		Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy hóa đơn",
		})
		return
	}

	// ❌ đã giao thì cấm hủy
	if hoaDon.TrangThai == "da_giao" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Hóa đơn đã giao, không thể hủy",
		})
		return
	}

	// ❌ đã thanh toán thì chặn (chưa làm hoàn tiền)
	if hoaDon.TrangThaiThanhToan == "da_thanh_toan" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Hóa đơn đã thanh toán, vui lòng liên hệ hỗ trợ",
		})
		return
	}

	// ✅ cập nhật trạng thái
	if err := config.DB.Model(&hoaDon).Updates(map[string]interface{}{
		"trang_thai":            "da_huy",
		"trang_thai_thanh_toan": "da_huy",
	}).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể hủy hóa đơn",
		})
		return
	}

	// reload
	config.DB.First(&hoaDon, hoaDon.MaHoaDon)

	// 🔥 realtime admin
	ctrl.Hub.Broadcast(dto.WSMessage{
		Type:    "hoa_don_bi_huy",
		Payload: hoaDon,
	})

	// 🔥 realtime user
	ctrl.Hub.Broadcast(dto.WSMessage{
		Type: "hoa_don_bi_huy_user",
		Payload: gin.H{
			"ma_hoa_don":            hoaDon.MaHoaDon,
			"trang_thai":            "da_huy",
			"trang_thai_thanh_toan": "da_huy",
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Hủy hóa đơn thành công",
	})
}

func (ctrl *HoaDonController) GetHoaDonChoThanhToan(c *gin.Context) {

	// 🔐 lấy user từ middleware (GIỐNG CÁC HÀM KHÁC)
	userIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Chưa đăng nhập",
		})
		return
	}

	userID := userIDAny.(uint)

	var hoaDons []models.HoaDon

	err := config.DB.
		Where("ma_nguoi_dung = ? AND trang_thai_thanh_toan = ?", userID, "chua_thanh_toan").
		Order("ma_hoa_don DESC").
		Find(&hoaDons).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không lấy được hóa đơn chờ thanh toán",
		})
		return
	}

	// ✅ QUAN TRỌNG: đồng bộ format với FE
	c.JSON(http.StatusOK, gin.H{
		"data": hoaDons,
	})
}

func (ctrl *HoaDonController) SoDonTheoNgay(c *gin.Context) {
	type Result struct {
		Ngay  string `json:"ngay"`
		SoDon int    `json:"so_don"`
	}

	var result []Result

	err := config.DB.Raw(`
		SELECT 
			TO_CHAR(DATE(h.ngay), 'YYYY-MM-DD') AS ngay,
			COUNT(*) AS so_don
		FROM hoa_dons h
		WHERE h.trang_thai = 'da_giao'
		  AND h.trang_thai_thanh_toan = 'da_thanh_toan'
		GROUP BY DATE(h.ngay)
		ORDER BY DATE(h.ngay)
	`).Scan(&result).Error

	if err != nil {
		c.JSON(500, gin.H{
			"message": "Lỗi lấy số đơn theo ngày",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, result)
}

func (ctrl *HoaDonController) DonHangDaGiaoHomNay(c *gin.Context) {
	type Result struct {
		MaHD         uint    `json:"ma_hd" gorm:"column:ma_hd"`
		TenKhachHang string  `json:"ten_khach_hang" gorm:"column:ten_khach_hang"`
		ThanhTien    float64 `json:"thanh_tien" gorm:"column:thanh_tien"`
		TrangThai    string  `json:"trang_thai" gorm:"column:trang_thai"`
	}

	var result []Result

	err := config.DB.Raw(`
        SELECT 
            h.ma_hoa_don AS ma_hd,
            h.ho_ten AS ten_khach_hang,
            h.tong_tien AS thanh_tien,
            h.trang_thai
        FROM hoa_dons h
        WHERE 
            h.trang_thai = 'da_giao'
            AND DATE(h.ngay) = CURRENT_DATE
        ORDER BY h.ma_hoa_don DESC
    `).Scan(&result).Error

	if err != nil {
		c.JSON(500, gin.H{
			"message": "Lỗi lấy đơn hàng đã giao hôm nay",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, result)
}

func (ctrl *HoaDonController) GetALLHoaDonByShipper(c *gin.Context) {

	shipperIDAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Chưa đăng nhập",
		})
		return
	}

	shipperID := shipperIDAny.(uint)

	var hoaDons []models.HoaDon

	if err := config.DB.
		Where("ma_shipper = ?", shipperID).
		Preload("ChiTietHoaDons").
		Preload("Shipper").
		Preload("Shipper.AnhNhanVien").
		Preload("ChiTietHoaDons.Options.OptionItem.NhomOption").
		Preload("ChiTietHoaDons.MonAn").
		Preload("ChiTietHoaDons.Options").
		Preload("ChiTietHoaDons.Options.OptionItem").
		Order("ma_hoa_don DESC").
		Find(&hoaDons).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	// realtime (optional nếu bạn cần)
	ctrl.Hub.Broadcast(dto.WSMessage{
		Type: "shipper_xem_hoa_don",
		Payload: gin.H{
			"hoa_don": hoaDons,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"data": hoaDons,
	})
}
func (ctrl *HoaDonController) ExportHoaDonPDF(c *gin.Context) {

	maHD := c.Param("mahd")
	if maHD == "" {
		c.JSON(400, gin.H{"error": "Thiếu mã hóa đơn"})
		return
	}

	var hoaDon models.HoaDon

	err := config.DB.
		Preload("ChiTietHoaDons").
		Preload("ChiTietHoaDons.MonAn").
		Preload("ChiTietHoaDons.Options.OptionItem").
		First(&hoaDon, "ma_hoa_don = ?", maHD).Error

	if err != nil {
		c.JSON(404, gin.H{"error": "Không tìm thấy hóa đơn"})
		return
	}

	// ======================
	// INIT PDF
	// ======================
	pdf := gofpdf.New("P", "mm", "A4", "")

	fontPath := "./fonts/DejaVuSans.ttf"
	pdf.AddUTF8Font("DejaVu", "", fontPath)
	pdf.AddUTF8Font("DejaVu", "B", fontPath)

	pdf.SetMargins(15, 15, 15) // Tăng margin lên 15mm cho thoáng, cân đối bản in
	pdf.AddPage()

	// KHÔNG DÙNG KHUNG NGOÀI CỨNG NHẮC ĐỂ GIỐNG HÓA ĐƠN THỰC TẾ

	// ======================
	// HEADER CÔNG TY (Thiết kế lại gọn gàng, tinh tế)
	// ======================
	pdf.SetFont("DejaVu", "B", 12) // Giảm xíu cho đỡ thô
	pdf.SetTextColor(44, 62, 80)   // Màu xanh đen thanh lịch thay vì đen xì
	pdf.CellFormat(0, 6, "CÔNG TY TNHH FOOD HUB", "", 1, "C", false, 0, "")

	pdf.SetFont("DejaVu", "", 9)
	pdf.SetTextColor(127, 140, 141) // Màu xám nhẹ cho thông tin phụ
	pdf.CellFormat(0, 5, "Địa chỉ: 123 Nguyễn Văn A, TP.HCM", "", 1, "C", false, 0, "")
	pdf.CellFormat(0, 5, "Hotline: 0933 924 075", "", 1, "C", false, 0, "")
	
	// Đường phân cách mảnh dưới Header
	pdf.SetDrawColor(220, 220, 220)
	pdf.SetLineWidth(0.3)
	pdf.Line(15, 36, 195, 36)
	pdf.Ln(6)

	// ======================
	// TITLE
	// ======================
	pdf.SetFont("DejaVu", "B", 16)
	pdf.SetTextColor(44, 62, 80)
	pdf.CellFormat(0, 8, "HÓA ĐƠN BÁN LẺ", "", 1, "C", false, 0, "")
	pdf.Ln(4)

	// ======================
	// THÔNG TIN KHÁCH HÀNG & HÓA ĐƠN (Chia 2 cột cân đối)
	// ======================
	pdf.SetFont("DejaVu", "", 10)
	pdf.SetTextColor(60, 60, 60)

	// Dòng 1: Khách hàng & Ngày lập
	pdf.CellFormat(100, 6, "Khách hàng: "+hoaDon.HoTen, "", 0, "L", false, 0, "")
	pdf.CellFormat(80, 6, "Ngày: "+hoaDon.Ngay.Format("02-01-2006 15:04"), "", 1, "R", false, 0, "")

	// Dòng 2: Số điện thoại & Mã hóa đơn (bổ sung hiển thị mã HD cho chuyên nghiệp)
	pdf.CellFormat(100, 6, "Số điện thoại: "+hoaDon.SDT, "", 0, "L", false, 0, "")
	pdf.CellFormat(80, 6, "Mã HD: "+maHD, "", 1, "R", false, 0, "")

	// Dòng 3: Địa chỉ (nếu có)
	if hoaDon.DiaChi != "" {
		pdf.CellFormat(0, 6, "Địa chỉ: "+hoaDon.DiaChi, "", 1, "L", false, 0, "")
	}

	pdf.Ln(6)

	// ======================
	// TABLE HEADER (Giao diện phẳng - Flat Design)
	// ======================
	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetTextColor(44, 62, 80)
	
	// Vẽ đường kẻ trên của header bảng
	pdf.SetDrawColor(180, 180, 180)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(1)

	// Độ rộng các cột tổng = 180mm (vừa khít với giấy A4 margin 15mm)
	wSTT := 12.0
	wTen := 83.0
	wSL := 15.0
	wGia := 32.0
	wTong := 38.0

	pdf.CellFormat(wSTT, 8, "STT", "", 0, "C", false, 0, "")
	pdf.CellFormat(wTen, 8, "TÊN MÓN ĂN / HÀNG HÓA", "", 0, "L", false, 0, "")
	pdf.CellFormat(wSL, 8, "SL", "", 0, "C", false, 0, "")
	pdf.CellFormat(wGia, 8, "ĐƠN GIÁ", "", 0, "R", false, 0, "")
	pdf.CellFormat(wTong, 8, "THÀNH TIỀN", "", 1, "R", false, 0, "")

	// Vẽ đường kẻ dưới của header bảng
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(2)

	// ======================
	// ITEMS (Dòng dữ liệu món ăn)
	// ======================
	pdf.SetFont("DejaVu", "", 10)
	pdf.SetTextColor(50, 50, 50)

	stt := 1
	for _, ct := range hoaDon.ChiTietHoaDons {
		thanhTien := ct.ThanhTien

		// MAIN ROW (Món chính)
		pdf.SetFont("DejaVu", "B", 10) // Tên món in đậm nhẹ cho nổi bật
		pdf.CellFormat(wSTT, 7, fmt.Sprintf("%d", stt), "", 0, "C", false, 0, "")
		pdf.CellFormat(wTen, 7, ct.MonAn.TenMonAn, "", 0, "L", false, 0, "")
		pdf.CellFormat(wSL, 7, fmt.Sprintf("%d", ct.SoLuong), "", 0, "C", false, 0, "")
		pdf.CellFormat(wGia, 7, formatMoneyVN(ct.DonGia), "", 0, "R", false, 0, "")
		pdf.CellFormat(wTong, 7, formatMoneyVN(thanhTien), "", 1, "R", false, 0, "")

		stt++

		// OPTIONS (Món phụ/Topping - Chữ nhỏ hơn, in nghiêng nhẹ hoặc lùi lề)
		pdf.SetFont("DejaVu", "", 9)
		pdf.SetTextColor(100, 100, 100) // Màu chữ nhạt hơn món chính

		for _, op := range ct.Options {
			name := op.TenOption
			if name == "" {
				name = op.OptionItem.TenOption
			}

			pdf.CellFormat(wSTT, 6, "", "", 0, "C", false, 0, "")
			pdf.CellFormat(wTen, 6, "  + "+name, "", 0, "L", false, 0, "")
			pdf.CellFormat(wSL, 6, "", "", 0, "C", false, 0, "")
			pdf.CellFormat(wGia, 6, "+ "+formatMoneyVN(op.GiaThem), "", 0, "R", false, 0, "")
			pdf.CellFormat(wTong, 6, "", "", 1, "R", false, 0, "")
		}
		
		// Trả lại định dạng cũ cho item tiếp theo
		pdf.SetFont("DejaVu", "", 10)
		pdf.SetTextColor(50, 50, 50)
		pdf.Ln(1) 
	}

	// Kẻ đường chấm chấm hoặc nét mảnh kết thúc danh sách món
	pdf.SetDrawColor(200, 200, 200)
	pdf.Line(15, pdf.GetY(), 195, pdf.GetY())
	pdf.Ln(4)

	// ======================
	// PHÍ + GIẢM + TỔNG (Căn lề phải chuẩn hóa)
	// ======================
	wLabel := wSTT + wTen + wSL + wGia // Tổng độ rộng phần text nhãn bên trái

	pdf.SetFont("DejaVu", "", 10)
	pdf.CellFormat(wLabel, 6, "Tạm tính:", "", 0, "R", false, 0, "")
	pdf.CellFormat(wTong, 6, formatMoneyVN(hoaDon.TamTinh), "", 1, "R", false, 0, "")

	pdf.CellFormat(wLabel, 6, "Giảm giá:", "", 0, "R", false, 0, "")
	pdf.CellFormat(wTong, 6, "- "+formatMoneyVN(hoaDon.TienGiam), "", 1, "R", false, 0, "")

	pdf.Ln(2)
	// Đường kẻ dày phân tách phần Tổng tiền thanh toán
	pdf.SetDrawColor(44, 62, 80)
	pdf.SetLineWidth(0.5)
	pdf.Line(110, pdf.GetY(), 195, pdf.GetY()) 
	pdf.Ln(2)

	pdf.SetFont("DejaVu", "B", 12)
	pdf.SetTextColor(192, 57, 43) // Màu đỏ đậm tinh tế cho Tổng Tiền đem lại cảm giác chuyên nghiệp
	pdf.CellFormat(wLabel, 8, "TỔNG THANH TOÁN:", "", 0, "R", false, 0, "")
	pdf.CellFormat(wTong, 8, formatMoneyVN(hoaDon.TongTien), "", 1, "R", false, 0, "")

	// ======================
	// FOOTER (Ký tên & Lời cảm ơn)
	// ======================
	pdf.Ln(12)
	pdf.SetFont("DejaVu", "", 10)
	pdf.SetTextColor(60, 60, 60)

	// Chia 2 bên chữ ký cân đối
	pdf.CellFormat(90, 5, "Khách hàng", "", 0, "C", false, 0, "")
	pdf.CellFormat(90, 5, "Người lập hóa đơn", "", 1, "C", false, 0, "")
	
	pdf.SetFont("DejaVu", "", 9)
	pdf.SetTextColor(140, 140, 140)
	pdf.CellFormat(90, 5, "(Ký, ghi rõ họ tên)", "", 0, "C", false, 0, "")
	pdf.CellFormat(90, 5, "(Ký, ghi rõ họ tên)", "", 1, "C", false, 0, "")

	// Khoảng trống giả lập chỗ ký tên
	pdf.Ln(15) 

	pdf.SetFont("DejaVu", "B", 10)
	pdf.SetTextColor(44, 62, 80)
	pdf.CellFormat(0, 8, "Cảm ơn quý khách - Hẹn gặp lại!", "", 1, "C", false, 0, "")

	// ======================
	// OUTPUT
	// ======================
	fileName := fmt.Sprintf("hoa_don_%s.pdf", maHD)

	c.Header("Content-Type", "application/pdf")
	c.Header("Content-Disposition", "attachment; filename="+fileName)

	err = pdf.Output(c.Writer)
	if err != nil {
		log.Println("PDF ERROR:", err)
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
}

func formatMoneyVN(amount float64) string {
	return formatNumber(amount) + " đ"
}
func formatNumber(n float64) string {
	s := fmt.Sprintf("%.0f", n)
	nStr := ""
	count := 0

	for i := len(s) - 1; i >= 0; i-- {
		count++
		nStr = string(s[i]) + nStr
		if count%3 == 0 && i != 0 {
			nStr = "," + nStr
		}
	}
	return nStr
}
