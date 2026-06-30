package controllers

import (
	"encoding/json"
	"fmt"
	"strings"

	//"log"
	"net/http"
	//"strconv"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/config"

	//"github.com/vpa/quanlynhahang-backend/dto"
	"github.com/vpa/quanlynhahang-backend/internal/websocket"
	"github.com/vpa/quanlynhahang-backend/models"
)

type BanAnController struct {
	Hub *websocket.Hub
}

func NewBanAnController(hub *websocket.Hub) *BanAnController {
	return &BanAnController{Hub: hub}
}

func (h *ChatHandler) CreateBanAn(c *gin.Context) {
	var ban models.BanAn

	if err := c.ShouldBind(&ban); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu form không hợp lệ: " + err.Error()})
		return
	}

	if err := config.DB.Create(&ban).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo bàn ăn: " + err.Error()})
		return
	}

	config.DB.Save(&ban)

	file, err := c.FormFile("image")
	if err == nil && file != nil {
		src, err := file.Open()
		if err == nil {
			defer src.Close()

			uploadResult, err := config.CLD.Upload.Upload(c, src, uploader.UploadParams{
				Folder: "banan",
			})

			if err == nil {
				img := models.HinhAnh{
					OwnerID:   ban.MaBanAn,
					OwnerType: "ban_an",
					Url:       uploadResult.SecureURL,
				}
				config.DB.Create(&img)
			}
		}
	}

	config.DB.Preload("AnhBan").First(&ban, ban.MaBanAn)

	// ======================
	// EMBEDDING (0/1 STATUS)
	// ======================
	document := buildBanAnEmbeddingDocument(ban)

	embedding, err := h.llm.Embed(c.Request.Context(), document)
	if err == nil && len(embedding) > 0 {

		metaJSON, _ := json.Marshal(map[string]any{
			"type":       "ban_an",
			"id":         ban.MaBanAn,
			"ten_ban":    ban.TenBan,
			"trang_thai": ban.TrangThai,
		})

		embeddingID := fmt.Sprintf("ban_an_%d", ban.MaBanAn)

		config.DB.Exec(`
			INSERT INTO menu_embeddings (id, document, metadata, embedding)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (id) DO UPDATE SET
				document = EXCLUDED.document,
				metadata = EXCLUDED.metadata,
				embedding = EXCLUDED.embedding
		`,
			embeddingID,
			document,
			string(metaJSON),
			vectorToString(embedding),
		)
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tạo bàn ăn + embedding thành công",
		"data":    ban,
	})
}

// Lấy tất cả bàn ăn kèm ảnh
func GetAllBanAn(c *gin.Context) {
	var dsBanAn []models.BanAn

	// ✅ Preload ảnh bàn (quan hệ polymorphic)
	if err := config.DB.Preload("AnhBan").Find(&dsBanAn).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể lấy danh sách bàn ăn: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Lấy danh sách bàn ăn thành công",
		"data":    dsBanAn,
	})
}

func GetBanAnByID(c *gin.Context) {
	id := c.Param("id")

	var banan models.BanAn

	// 🔥 Query đúng: WHERE id = ? + Preload ảnh
	if err := config.DB.Preload("AnhBan").First(&banan, "ma_ban = ?", id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy bàn ăn với ID " + id,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Lấy thông tin bàn ăn thành công",
		"data":    banan,
	})
}

// ✅ Cập nhật thông tin bàn ăn
func (h *ChatHandler) UpdateBanAn(c *gin.Context) {
	id := c.Param("id")

	var ban models.BanAn

	if err := config.DB.First(&ban, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy bàn ăn"})
		return
	}

	var input models.BanAn
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ban.TenBan = input.TenBan
	ban.SoChoNgoi = input.SoChoNgoi
	ban.TrangThai = input.TrangThai // 0 = hết bàn, 1 = còn bàn

	if err := config.DB.Save(&ban).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật bàn ăn"})
		return
	}

	file, err := c.FormFile("image")
	if err == nil && file != nil {
		src, err := file.Open()
		if err == nil {
			defer src.Close()

			uploadResult, err := config.CLD.Upload.Upload(c, src, uploader.UploadParams{
				Folder: "banan",
			})

			if err == nil {
				config.DB.Where("owner_id = ? AND owner_type = ?", ban.MaBanAn, "ban_an").
					Delete(&models.HinhAnh{})

				config.DB.Create(&models.HinhAnh{
					OwnerID:   ban.MaBanAn,
					OwnerType: "ban_an",
					Url:       uploadResult.SecureURL,
				})
			}
		}
	}

	config.DB.Preload("AnhBan").First(&ban, ban.MaBanAn)

	// ======================
	// EMBEDDING (0/1 STATUS)
	// ======================
	document := buildBanAnEmbeddingDocument(ban)

	embedding, err := h.llm.Embed(c.Request.Context(), document)
	if err == nil && len(embedding) > 0 {

		metaJSON, _ := json.Marshal(map[string]any{
			"type":       "ban_an",
			"id":         ban.MaBanAn,
			"ten_ban":    ban.TenBan,
			"trang_thai": ban.TrangThai,
		})

		embeddingID := fmt.Sprintf("ban_an_%d", ban.MaBanAn)

		config.DB.Exec(`
			UPDATE menu_embeddings
			SET document = $1,
			    metadata = $2,
			    embedding = $3
			WHERE id = $4
		`,
			document,
			string(metaJSON),
			vectorToString(embedding),
			embeddingID,
		)
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật bàn ăn + embedding thành công",
		"data":    ban,
	})
}

// ✅ Xóa bàn ăn
func (h *ChatHandler) DeleteBanAn(c *gin.Context) {
	id := c.Param("id")

	var ban models.BanAn

	if err := config.DB.First(&ban, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "Không tìm thấy bàn ăn"})
		return
	}

	// =====================
	// DELETE EMBEDDING
	// =====================
	config.DB.Exec(`
		DELETE FROM menu_embeddings
		WHERE id = $1
	`, fmt.Sprintf("ban_an_%d", ban.MaBanAn))

	// =====================
	// DELETE IMAGE
	// =====================
	config.DB.Where(
		"owner_id = ? AND owner_type = ?",
		ban.MaBanAn,
		"ban_an",
	).Delete(&models.HinhAnh{})

	// =====================
	// DELETE DB
	// =====================
	config.DB.Delete(&ban)

	c.JSON(200, gin.H{
		"message": "Xóa bàn ăn + embedding thành công",
	})
}

func buildBanAnEmbeddingDocument(ban models.BanAn) string {
	allHours := []string{
		"10:00", "11:00", "12:00", "13:00", "14:00", "15:00",
		"16:00", "17:00", "18:00", "19:00", "20:00",
		"21:00", "22:00", "23:00",
	}

	statusText := "CÒN BÀN"
	if ban.TrangThai == 0 {
		statusText = "HẾT BÀN"
	}

	doc := fmt.Sprintf(
		`Bàn ăn: %s
Số chỗ: %d
Trạng thái: %s

KHUNG GIỜ ĐẶT BÀN (áp dụng cho tất cả bàn):
%s`,
		ban.TenBan,
		ban.SoChoNgoi,
		statusText,
		strings.Join(allHours, ", "),
	)

	return doc
}
