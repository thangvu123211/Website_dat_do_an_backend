package controllers

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/config"
	"github.com/vpa/quanlynhahang-backend/models"
)

// API tạo nhóm option


type UpdateNhomOptionRequest struct {
	TenNhom        string `json:"ten_nhom"`
	BatBuoc        bool   `json:"bat_buoc"`
	ChonNhieu      bool   `json:"chon_nhieu"`
	SoLuongToiDa   int    `json:"so_luong_toi_da"`
	SoLuongToiThieu int   `json:"so_luong_toi_thieu"`
}

func (h *ChatHandler) CreateNhomOption(c *gin.Context) {
	var input models.NhomOption

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	if input.MaMonAn == 0 || input.TenNhom == "" {
		c.JSON(400, gin.H{"error": "Thiếu dữ liệu"})
		return
	}

	// check món ăn
	var monan models.MonAn
	if err := config.DB.First(&monan, input.MaMonAn).Error; err != nil {
		c.JSON(404, gin.H{"error": "Món ăn không tồn tại"})
		return
	}

	input.TrangThai = 1

	// =====================
	// SAVE DB
	// =====================
	if err := config.DB.Create(&input).Error; err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}

	// =====================
	// DOCUMENT (GIỐNG MONAN)
	// =====================
	document := fmt.Sprintf(
		"Nhóm option: %s\nThuộc món: %s\nBắt buộc: %v\nChọn nhiều: %v",
		input.TenNhom,
		monan.TenMonAn,
		input.BatBuoc,
		input.ChonNhieu,
	)

	// =====================
	// EMBEDDING
	// =====================
	embedding, err := h.llm.Embed(c.Request.Context(), document)
	if err != nil {
		log.Println("embed error:", err)
	}

	// =====================
	// METADATA
	// =====================
	metaJSON, _ := json.Marshal(map[string]any{
		"type":        "nhom_option",
		"id":          input.MaNhomOption,
		"ma_mon_an":   input.MaMonAn,
		"ten_nhom":    input.TenNhom,
	})

	// =====================
	// INSERT VECTOR
	// =====================
	if len(embedding) > 0 {

		embeddingID := fmt.Sprintf("nhom_option_%d", input.MaNhomOption)

		config.DB.Exec(`
			INSERT INTO menu_embeddings (id, document, metadata, embedding)
			VALUES ($1, $2, $3, $4)
		`,
			embeddingID,
			document,
			string(metaJSON),
			vectorToString(embedding),
		)
	}

	c.JSON(201, gin.H{
		"message": "Tạo nhóm option + embedding thành công",
		"data":    input,
	})
}

func GetAllNhomOption(c *gin.Context) {

	var nhoms []models.NhomOption

	err := config.DB.
		Preload("OptionItems").
		Find(&nhoms).Error

	if err != nil {
		c.JSON(500, gin.H{
			"error": "Không thể lấy danh sách nhóm option",
		})
		return
	}

	c.JSON(200, gin.H{
		"data": nhoms,
	})
}

func GetNhomOptionByID(c *gin.Context) {

	id := c.Param("id")

	var nhom models.NhomOption

	err := config.DB.
		Preload("OptionItems").
		First(&nhom, id).Error

	if err != nil {
		c.JSON(404, gin.H{
			"error": "Không tìm thấy nhóm option",
		})
		return
	}

	c.JSON(200, gin.H{
		"data": nhom,
	})
}

func (h *ChatHandler) UpdateNhomOption(c *gin.Context) {
	id := c.Param("id")

	var nhom models.NhomOption

	if err := config.DB.First(&nhom, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "Không tìm thấy nhóm option"})
		return
	}

	var input UpdateNhomOptionRequest
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	// update DB
	nhom.TenNhom = input.TenNhom
	nhom.BatBuoc = input.BatBuoc
	nhom.ChonNhieu = input.ChonNhieu

	config.DB.Save(&nhom)

	// get monan
	var monan models.MonAn
	config.DB.First(&monan, nhom.MaMonAn)

	// =====================
	// DOCUMENT
	// =====================
	document := fmt.Sprintf(
		"Nhóm option: %s\nMón: %s\nBắt buộc: %v\nChọn nhiều: %v",
		nhom.TenNhom,
		monan.TenMonAn,
		nhom.BatBuoc,
		nhom.ChonNhieu,
	)

	// =====================
	// EMBEDDING
	// =====================
	embedding, _ := h.llm.Embed(c.Request.Context(), document)

	metaJSON, _ := json.Marshal(map[string]any{
		"type":        "nhom_option",
		"id":          nhom.MaNhomOption,
		"ma_mon_an":   nhom.MaMonAn,
	})

	// =====================
	// UPDATE VECTOR
	// =====================
	embeddingID := fmt.Sprintf("nhom_option_%d", nhom.MaNhomOption)

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

	c.JSON(200, gin.H{
		"message": "Update nhóm option + embedding thành công",
		"data":    nhom,
	})
}

func (h *ChatHandler) DeleteNhomOption(c *gin.Context) {
	id := c.Param("id")

	var nhom models.NhomOption
	if err := config.DB.First(&nhom, id).Error; err != nil {
		c.JSON(404, gin.H{"error": "Không tìm thấy nhóm option"})
		return
	}

	// 1. lấy items
	var items []models.OptionItem
	config.DB.Where("ma_nhom_option = ?", id).Find(&items)

	// 2. xóa embedding option item
	for _, it := range items {
		config.DB.Exec(`DELETE FROM menu_embeddings WHERE id = $1`,
			fmt.Sprintf("option_item_%d", it.MaOptionItem))
	}

	// 3. xóa items
	config.DB.Where("ma_nhom_option = ?", id).Delete(&models.OptionItem{})

	// 4. xóa embedding nhóm
	config.DB.Exec(`DELETE FROM menu_embeddings WHERE id = $1`,
		fmt.Sprintf("nhom_option_%d", nhom.MaNhomOption))

	// 5. xóa DB
	config.DB.Delete(&nhom)

	c.JSON(200, gin.H{
		"message": "Xóa nhóm option + embedding thành công",
	})
}
// API tạo option item

func (h *ChatHandler) CreateOptionItem(c *gin.Context) {
	var input models.OptionItem

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	var nhom models.NhomOption
	if err := config.DB.First(&nhom, input.MaNhomOption).Error; err != nil {
		c.JSON(404, gin.H{"error": "Nhóm option không tồn tại"})
		return
	}

	input.TrangThai = 1

	config.DB.Create(&input)

	// =====================
	// DOCUMENT
	// =====================
	document := fmt.Sprintf(
		"Option: %s\nThuộc nhóm: %s\nGiá thêm: %.0f",
		input.TenOption,
		nhom.TenNhom,
		input.GiaThem,
	)

	// =====================
	// EMBEDDING
	// =====================
	embedding, _ := h.llm.Embed(c.Request.Context(), document)

	metaJSON, _ := json.Marshal(map[string]any{
		"type":           "option_item",
		"id":             input.MaOptionItem,
		"ma_nhom_option": input.MaNhomOption,
	})

	// =====================
	// INSERT VECTOR
	// =====================
	config.DB.Exec(`
		INSERT INTO menu_embeddings (id, document, metadata, embedding)
		VALUES ($1,$2,$3,$4)
	`,
		fmt.Sprintf("option_item_%d", input.MaOptionItem),
		document,
		string(metaJSON),
		vectorToString(embedding),
	)

	c.JSON(201, gin.H{
		"message": "Tạo option item + embedding thành công",
		"data":    input,
	})
}

func GetAllOptionItem(c *gin.Context) {

	var items []models.OptionItem

	if err := config.DB.Find(&items).Error; err != nil {
		c.JSON(500, gin.H{
			"error": "Không thể lấy option item",
		})
		return
	}

	c.JSON(200, gin.H{
		"data": items,
	})
}

func GetOptionItemByID(c *gin.Context) {

	id := c.Param("id")

	var item models.OptionItem

	if err := config.DB.First(&item, id).Error; err != nil {
		c.JSON(404, gin.H{
			"error": "Không tìm thấy option item",
		})
		return
	}

	c.JSON(200, gin.H{
		"data": item,
	})
}

func (h *ChatHandler) UpdateOptionItem(c *gin.Context) {
	id := c.Param("id")

	var item models.OptionItem
	config.DB.First(&item, id)

	var input models.OptionItem
	c.ShouldBindJSON(&input)

	item.TenOption = input.TenOption
	item.GiaThem = input.GiaThem

	config.DB.Save(&item)

	var nhom models.NhomOption
	config.DB.First(&nhom, item.MaNhomOption)

	document := fmt.Sprintf(
		"Option: %s\nNhóm: %s\nGiá thêm: %.0f",
		item.TenOption,
		nhom.TenNhom,
		item.GiaThem,
	)

	embedding, _ := h.llm.Embed(c.Request.Context(), document)

	config.DB.Exec(`
		UPDATE menu_embeddings
		SET document = $1,
		    metadata = $2,
		    embedding = $3
		WHERE id = $4
	`,
		document,
		string(`{"type":"option_item","id":`+id+`}`),
		vectorToString(embedding),
		fmt.Sprintf("option_item_%s", id),
	)

	c.JSON(200, gin.H{"message": "update thành công"})
}

func (h *ChatHandler) DeleteOptionItem(c *gin.Context) {
	id := c.Param("id")

	config.DB.Exec(`DELETE FROM menu_embeddings WHERE id = $1`,
		fmt.Sprintf("option_item_%s", id),
	)

	config.DB.Delete(&models.OptionItem{}, id)

	c.JSON(200, gin.H{"message": "xóa thành công"})
}

