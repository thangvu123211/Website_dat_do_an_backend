package controllers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/config"
	"github.com/vpa/quanlynhahang-backend/dto"
	"github.com/vpa/quanlynhahang-backend/internal/websocket"
	"github.com/vpa/quanlynhahang-backend/models"
	"github.com/vpa/quanlynhahang-backend/utils"
)

type DatBanController struct {
	Hub *websocket.Hub
}

func NewDatBanController(hub *websocket.Hub) *DatBanController {
	return &DatBanController{Hub: hub}
}

type WS_DatBan struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

func (h *ChatHandler) CreateDatBan(c *gin.Context) {
	var input models.DatBan
	userID, _ := c.Get("user_id")

	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 🔥 CHECK BÀN ĐÃ CÓ NGƯỜI ĐẶT CHƯA (CÙNG NGÀY + GIỜ)
	var count int64
	config.DB.Model(&models.DatBan{}).
		Where(`
			ma_ban_an = ?
			AND ngay = ?
			AND gio = ?
			AND trang_thai IN ?
		`,
			input.MaBanAn,
			input.Ngay,
			input.Gio,
			[]string{"dang_xu_ly", "da_xac_nhan"},
		).
		Count(&count)

	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Bàn đã có người đặt vào khung giờ này",
		})
		return
	}

	// ================== GIỮ NGUYÊN CODE CỦA BẠN ==================
	datban := models.DatBan{
		TenKhachHang: input.TenKhachHang,
		Email:        input.Email,
		SDT:          input.SDT,
		GhiChu:       input.GhiChu,
		MaBanAn:      input.MaBanAn,
		Ngay:         input.Ngay,
		Gio:          input.Gio,
		TrangThai:    "dang_xu_ly",
		MaNguoiDung:  userID.(uint),
	}

	if err := config.DB.Create(&datban).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo đặt bàn"})
		return
	}

	h.syncDatBanEmbedding(datban.Ngay, datban.MaBanAn)

	config.DB.
		Preload("BanAn").
		First(&datban, datban.MaDatBan)

	// 🔥 broadcast realtime
	go func(db models.DatBan) {

		h.pushDatBan("dat_ban_created", datban)

	}(datban)

	// gửi mail (giữ nguyên)
	go func(db models.DatBan) {
		var ban models.BanAn
		config.DB.First(&ban, db.MaBanAn)

		if err := utils.SendMailDatBan(db.Email, utils.DatBanMailInfo{
			TenKhachHang: db.TenKhachHang,
			MaDatBan:     db.MaDatBan,
			Ngay:         db.Ngay,
			Gio:          db.Gio,
			TenBan:       ban.TenBan,
			Email:        db.Email,
			GhiChu:       db.GhiChu,
		}); err != nil {
			log.Println("Send mail dat ban error:", err)
		}
	}(datban)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Đặt bàn thành công",
		"data":    datban,
	})

	h.Hub.Broadcast(dto.WSMessage{
		Type: "khung_gio_updated",
		Payload: gin.H{
			"ma_ban_an":  datban.MaBanAn,
			"ngay":       datban.Ngay,
			"gio":        datban.Gio,
			"trang_thai": datban.TrangThai,
		},
	})

	h.Hub.Broadcast(dto.WSMessage{
		Type:    "new_dat_ban",
		Payload: datban,
	})
}

func (h *ChatHandler) GetAllDatBan(c *gin.Context) {
	var datbans []models.DatBan

	if err := config.DB.Find(&datbans).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy danh sách đặt bàn",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": datbans,
	})

	h.Hub.Broadcast(dto.WSMessage{
		Type:    "dat_ban_refresh_list",
		Payload: nil,
	})
}

func GetDatBanByID(c *gin.Context) {
	id := c.Param("id")
	var datban models.DatBan

	if err := config.DB.Preload("NhanVienXacNhan").First(&datban, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy đặt bàn",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": datban,
	})
}

func (h *ChatHandler) UpdateDatBan(c *gin.Context) {
	id := c.Param("id")

	var datban models.DatBan
	if err := config.DB.First(&datban, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đặt bàn"})
		return
	}

	var input struct {
		TenKhachHang string `json:"ten_khach_hang"`
		SDT          string `json:"sdt"`
		GhiChu       string `json:"ghi_chu"`
		Ngay         string `json:"ngay"`
		Gio          string `json:"gio"`
		TrangThai    string `json:"trang_thai"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// ✅ update
	if err := config.DB.Model(&datban).Updates(input).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật"})
		return
	}

	h.syncDatBanEmbedding(datban.Ngay, datban.MaBanAn)

	// ✅ reload + preload
	if err := config.DB.
		Preload("BanAn").
		First(&datban, id).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{"error": "Lỗi load dữ liệu"})
		return
	}

	// ======================
	// 🔥 REALTIME
	// ======================

	// admin nhận tất cả
	h.Hub.Broadcast(dto.WSMessage{
		Type:    "dat_ban_updated",
		Role:    "admin",
		Payload: datban,
	})

	// user liên quan nhận đơn của mình
	h.Hub.Broadcast(dto.WSMessage{
		Type: "dat_ban_updated_user",
		Payload: gin.H{
			"id":         datban.MaDatBan,
			"trang_thai": datban.TrangThai,
		},
	})

	h.Hub.Broadcast(dto.WSMessage{
		Type: "khung_gio_updated",
		Payload: gin.H{
			"ma_ban_an":  datban.MaBanAn,
			"ngay":       datban.Ngay,
			"gio":        datban.Gio,
			"trang_thai": datban.TrangThai,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật đặt bàn thành công",
		"data":    datban,
	})
}

func (h *ChatHandler) XacNhanDatBan(c *gin.Context) {
	id := c.Param("id")

	var datban models.DatBan
	if err := config.DB.First(&datban, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy đặt bàn"})
		return
	}

	// tránh xác nhận lại
	if datban.TrangThai == "da_xac_nhan" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Đặt bàn đã được xác nhận"})
		return
	}

	// update trạng thái
	if err := config.DB.Model(&datban).Updates(map[string]interface{}{
		"trang_thai": "da_xac_nhan",
	}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể xác nhận đặt bàn"})
		return
	}

	h.syncDatBanEmbedding(datban.Ngay, datban.MaBanAn)

	// load lại thông tin bàn (optional)
	var ban models.BanAn
	config.DB.First(&ban, datban.MaBanAn)

	// gửi email async
	go func(db models.DatBan, tenBan string) {
		err := utils.SendMailDatBanXacNhan(db.Email, utils.DatBanXacNhanMailInfo{
			TenKhachHang: db.TenKhachHang,
			MaDatBan:     db.MaDatBan,
			Ngay:         db.Ngay,
			Gio:          db.Gio,
			TenBan:       tenBan,
			Email:        db.Email,
			GhiChu:       db.GhiChu,
		})

		if err != nil {
			log.Println("Send mail xác nhận đặt bàn lỗi:", err)
		}
	}(datban, ban.TenBan)

	c.JSON(http.StatusOK, gin.H{
		"message": "Xác nhận đặt bàn thành công",
	})

	config.DB.First(&datban, id)
	h.pushDatBan("dat_ban_confirmed", datban)

	h.Hub.Broadcast(dto.WSMessage{
		Type: "khung_gio_updated",
		Payload: gin.H{
			"ma_ban_an":  datban.MaBanAn,
			"ngay":       datban.Ngay,
			"gio":        datban.Gio,
			"trang_thai": datban.TrangThai,
		},
	})
}

func (h *ChatHandler) DeleteDatBan(c *gin.Context) {
	id := c.Param("id")
	var datban models.DatBan

	if err := config.DB.First(&datban, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy đặt bàn",
		})
		return
	}

	config.DB.Delete(&datban)

	h.Hub.Broadcast(dto.WSMessage{
		Type: "dat_ban_deleted",
		Payload: map[string]interface{}{
			"ma_dat_ban": id,
		},
	})

	c.JSON(http.StatusOK, gin.H{
		"message": "Xóa đặt bàn thành công",
	})
}
func (h *ChatHandler) GetDatBanCuaNguoiDung(c *gin.Context) {
	userIDRaw, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Không xác thực người dùng",
		})
		return
	}

	userID := userIDRaw.(uint) // hoặc uint64 / int tùy bạn lưu

	var datbans []models.DatBan

	if err := config.DB.
		Where("ma_nguoi_dung = ?", userID).
		Find(&datbans).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể lấy danh sách đặt bàn",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": datbans,
	})
}
func (h *ChatHandler) HuyDatBan(c *gin.Context) {
	id := c.Param("id")

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Không xác thực người dùng",
		})
		return
	}

	var datban models.DatBan
	if err := config.DB.First(&datban, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy đặt bàn",
		})
		return
	}

	// chỉ chủ đặt bàn mới được hủy
	role := c.GetString("role") // hoặc lấy từ JWT middleware

	if datban.MaNguoiDung != userID.(uint) && role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Bạn không có quyền hủy đặt bàn này",
		})
		return
	}

	// không cho hủy nếu đã xác nhận

	// ❌ user thường không được hủy nếu đã xác nhận
	if datban.TrangThai == "da_xac_nhan" && role != "admin" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Đặt bàn đã được xác nhận, không thể hủy",
		})
		return
	}

	// update trạng thái hủy
	if err := config.DB.Model(&datban).Updates(map[string]interface{}{
		"trang_thai": "da_huy",
	}).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể hủy đặt bàn",
		})
		return
	}

	h.syncDatBanEmbedding(datban.Ngay, datban.MaBanAn)

	c.JSON(http.StatusOK, gin.H{
		"message": "Hủy đặt bàn thành công",
	})

	h.Hub.Broadcast(dto.WSMessage{
		Type:    "dat_ban_cancelled",
		Role:    "admin",
		Payload: datban,
	})

	h.Hub.Broadcast(dto.WSMessage{
		Type: "admin_cancel_booking",
		Payload: gin.H{
			"id":      datban.MaDatBan,
			"message": "Đặt bàn của bạn đã bị admin hủy",
		},
	})

	h.Hub.Broadcast(dto.WSMessage{
		Type: "khung_gio_updated",
		Payload: gin.H{
			"ma_ban_an":  datban.MaBanAn,
			"ngay":       datban.Ngay,
			"gio":        datban.Gio,
			"trang_thai": datban.TrangThai,
		},
	})

}

func (h *ChatHandler) GetBanAnDaDat(c *gin.Context) {
	ngay := c.Query("ngay")
	gio := c.Query("gio")

	if ngay == "" || gio == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Thiếu ngày hoặc giờ",
		})
		return
	}

	var datbans []models.DatBan

	config.DB.
		Where(`
			ngay = ?
			AND gio = ?
			AND trang_thai IN ?
		`,
			ngay,
			gio,
			[]string{"dang_xu_ly", "da_xac_nhan"},
		).
		Find(&datbans)

	var maBan []uint
	for _, d := range datbans {
		maBan = append(maBan, d.MaBanAn)
	}

	c.JSON(http.StatusOK, gin.H{
		"data": maBan,
	})

}

func (h *ChatHandler) GetKhungGioBan(c *gin.Context) {
	ngay := c.Query("ngay")
	maBanStr := c.Query("ma_ban")

	if ngay == "" || maBanStr == "" {
		c.JSON(400, gin.H{"error": "Thiếu ngày hoặc mã bàn"})
		return
	}

	maBan, err := strconv.Atoi(maBanStr)
	if err != nil {
		c.JSON(400, gin.H{"error": "Mã bàn không hợp lệ"})
		return
	}

	allHours := []string{
		"10:00", "11:00",
		"12:00", "13:00", "14:00", "15:00", "16:00", "17:00",
		"18:00", "19:00", "20:00", "21:00", "22:00", "23:00",
	}

	var raw []string

	config.DB.
		Model(&models.DatBan{}).
		Where(`
			ma_ban_an = ?
			AND ngay = ?
			AND trang_thai IN ?
		`, maBan, ngay, []string{"dang_xu_ly", "da_xac_nhan"}).
		Pluck("gio", &raw)

	// ===== normalize tất cả giờ trong DB =====
	busy := make(map[string]bool)

	for _, g := range raw {
		n := normalizeHour(g)
		busy[n] = true
	}

	var result []gin.H
	for _, g := range allHours {
		result = append(result, gin.H{
			"gio":    g,
			"da_dat": busy[g],
		})
	}

	c.JSON(200, gin.H{"data": result})

}

func normalizeHour(g string) string {
	t, err := time.Parse("15:04:05", g)
	if err == nil {
		return t.Format("15:04")
	}

	t2, err := time.Parse("15:04", g)
	if err == nil {
		return t2.Format("15:04")
	}

	// fallback
	if len(g) >= 5 {
		return g[:5]
	}

	if len(g) == 2 {
		return g + ":00"
	}

	return g
}

func (h *ChatHandler) pushDatBan(event string, db models.DatBan) {
	h.Hub.Broadcast(dto.WSMessage{
		Type: event,
		Payload: map[string]interface{}{
			"id":             db.MaDatBan,
			"sdt":            db.SDT,
			"ten_khach_hang": db.TenKhachHang,
			"email":          db.Email,
			"ghi_chu":        db.GhiChu,
			"ma_ban_an":      db.MaBanAn,
			"ngay":           db.Ngay,
			"gio":            db.Gio,
			"trang_thai":     db.TrangThai,
			"ma_nguoi_dung":  db.MaNguoiDung,
		},
	})
}
func buildDatBanEmbedding(ngay string, maBan uint) string {
	var datbans []models.DatBan

	config.DB.
		Where(`
			ma_ban_an = ?
			AND ngay = ?
			AND trang_thai IN ?
		`, maBan, ngay, []string{"dang_xu_ly", "da_xac_nhan"}).
		Find(&datbans)

	allHours := []string{
		"10:00","11:00","12:00","13:00","14:00","15:00",
		"16:00","17:00","18:00","19:00","20:00",
		"21:00","22:00","23:00",
	}

	// 👉 CASE QUAN TRỌNG: KHÔNG CÓ DATA
	if len(datbans) == 0 {
		return fmt.Sprintf(
			"Bàn ăn %d ngày %s: TẤT CẢ khung giờ đều CÒN TRỐNG (chưa có ai đặt)",
			maBan,
			ngay,
		)
	}

	busy := make(map[string]bool)
	for _, d := range datbans {
		busy[normalizeHour(d.Gio)] = true
	}

	type Slot struct {
		Gio   string `json:"gio"`
		Trang int    `json:"trang"`
	}

	var slots []Slot

	for _, h := range allHours {
		trang := 1
		if busy[h] {
			trang = 0
		}
		slots = append(slots, Slot{Gio: h, Trang: trang})
	}

	return fmt.Sprintf(
		"Bàn ăn %d ngày %s trạng thái giờ (1=còn, 0=hết): %v",
		maBan,
		ngay,
		slots,
	)
}

func (h *ChatHandler) syncDatBanEmbedding(ngay string, maBan uint) {

	document := buildDatBanEmbedding(ngay, maBan)

	embedding, err := h.llm.Embed(context.Background(), document)
	if err != nil || len(embedding) == 0 {
		return
	}

	metaJSON, _ := json.Marshal(map[string]any{
		"type":   "dat_ban",
		"ma_ban": maBan,
		"ngay":   ngay,
	})

	id := fmt.Sprintf("datban_%d_%s", maBan, ngay)

	config.DB.Exec(`
		INSERT INTO menu_embeddings (id, document, metadata, embedding)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (id) DO UPDATE SET
			document = EXCLUDED.document,
			metadata = EXCLUDED.metadata,
			embedding = EXCLUDED.embedding
	`,
		id,
		document,
		string(metaJSON),
		vectorToString(embedding),
	)
}
