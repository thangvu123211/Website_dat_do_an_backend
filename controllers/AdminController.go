package controllers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"

	//"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/config"
	"github.com/xuri/excelize/v2"

	//"github.com/vpa/quanlynhahang-backend/dto"
	//"github.com/vpa/quanlynhahang-backend/internal/websocket"
	"github.com/vpa/quanlynhahang-backend/models"
	//"github.com/vpa/quanlynhahang-backend/utils"
	//"gorm.io/gorm"
	//"github.com/xuri/excelize/v2"
)

type MonAnBanChayDTO struct {
	MaMonAn  uint   `json:"ma_mon_an"`
	TenMonAn string `json:"ten_mon_an"`
	SoLuong  int64  `json:"so_luong"`
}

type TopMonBanChayDTO struct {
	MaMonAn  uint   `json:"ma_mon_an"`
	TenMonAn string `json:"ten_mon_an"`
	TongBan  int    `json:"tong_ban"`
}

type DoanhThuDTO struct {
	Ngay              string  `json:"ngay,omitempty"`
	Thang             int     `json:"thang,omitempty"`
	Nam               int     `json:"nam,omitempty"`
	DoanhThu          float64 `json:"doanh_thu"`
	SoDon             int64   `json:"so_don"`
	DoanhThuTrungBinh float64 `json:"doanh_thu_trung_binh"`
}

func GetDoanhThuTheoNgay(c *gin.Context) {

	ngay := c.Query("ngay")
	if ngay == "" {
		ngay = time.Now().Format("2006-01-02")
	}

	start := ngay + " 00:00:00"
	end := ngay + " 23:59:59"

	var result DoanhThuDTO

	err := config.DB.
		Model(&models.HoaDon{}).
		Select(`
			COALESCE(SUM(tong_tien), 0) AS doanh_thu,
			COUNT(*) AS so_don
		`).
		Where(`
			ngay BETWEEN ? AND ?
			AND trang_thai = ?
			AND trang_thai_thanh_toan = ?
		`, start, end, "da_giao", "da_thanh_toan").
		Scan(&result).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể tính doanh thu",
		})
		return
	}

	result.Ngay = ngay

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}

func GetDoanhThuTheoThang(c *gin.Context) {

	thang, _ := strconv.Atoi(c.DefaultQuery("thang", fmt.Sprint(int(time.Now().Month()))))
	nam, _ := strconv.Atoi(c.DefaultQuery("nam", fmt.Sprint(time.Now().Year())))

	var result DoanhThuDTO

	err := config.DB.
		Model(&models.HoaDon{}).
		Select(`
			COALESCE(SUM(tong_tien), 0) AS doanh_thu,
			COUNT(ma_hoa_don) AS so_don
		`).
		Where(`
			EXTRACT(MONTH FROM ngay) = ?
			AND EXTRACT(YEAR FROM ngay) = ?
			AND trang_thai = 'da_giao'
			AND trang_thai_thanh_toan = 'da_thanh_toan'
		`, thang, nam).
		Scan(&result).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể tính doanh thu tháng"})
		return
	}

	result.Thang = thang
	result.Nam = nam

	c.JSON(200, gin.H{"data": result})
}

func GetDoanhThuTheoNam(c *gin.Context) {

	nam, _ := strconv.Atoi(c.DefaultQuery("nam", fmt.Sprint(time.Now().Year())))

	var result DoanhThuDTO

	err := config.DB.
		Model(&models.HoaDon{}).
		Select(`
			COALESCE(SUM(tong_tien), 0) AS doanh_thu,
			COUNT(ma_hoa_don) AS so_don
		`).
		Where(`
			EXTRACT(YEAR FROM ngay) = ?
			AND trang_thai = 'da_giao'
			AND trang_thai_thanh_toan = 'da_thanh_toan'
		`, nam).
		Scan(&result).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể tính doanh thu năm"})
		return
	}

	result.Nam = nam
	c.JSON(200, gin.H{"data": result})
}
func ExportDoanhThuNgay(c *gin.Context) {

	ngay := c.Query("ngay")
	if ngay == "" {
		ngay = time.Now().Format("2006-01-02")
	}

	var hoaDons []models.HoaDon

	config.DB.
		Preload("ChiTietHoaDons.MonAn").
		Preload("ChiTietHoaDons.Options.OptionItem").
		Where("DATE(ngay)=? AND trang_thai=? AND trang_thai_thanh_toan=?",
			ngay, "da_giao", "da_thanh_toan").
		Find(&hoaDons)

	f := excelize.NewFile()
	sheet := "DoanhThu"
	f.NewSheet(sheet)

	// ======================
	// KHAI BÁO STYLES (ĐẸP & CHUẨN MOBILE)
	// ======================

	// Định dạng số tiền Việt Nam Đồng chuẩn Excel (Ví dụ: 150.000)
	// Việc dùng NumFmt giúp Mobile tự căn phải cực đẹp, không bao giờ bị chồng chữ
	vndFormat := "#,##0"

	// Style cho Tiêu đề lớn
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 16, Color: "2C3E50"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Style cho Header của bảng
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"34495E"}, Pattern: 1}, // Màu xám xanh đậm sang trọng
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	// Style cho dòng Hóa Đơn (Thông tin chính)
	hdStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "2C3E50", Size: 10},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"F2F4F4"}, Pattern: 1}, // Nền xám nhạt phân biệt rõ các hóa đơn
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "BDC3C7", Style: 1},
		},
	})
	
	// Style riêng cho các ô Tiền tệ của dòng Hóa Đơn
	hdMoneyStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "2C3E50", Size: 10},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"F2F4F4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &vndFormat,
		Border: []excelize.Border{
			{Type: "bottom", Color: "BDC3C7", Style: 1},
		},
	})

	// Style cho dòng Món Ăn Chính
	itemStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "333333"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	
	itemMoneyStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Size: 10, Color: "333333"},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &vndFormat,
	})

	// Style cho dòng Topping / Options (In nghiêng chữ nhỏ, màu nhạt)
	optionStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 9, Color: "7F8C8D"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	
	optionMoneyStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Italic: true, Size: 9, Color: "7F8C8D"},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &vndFormat,
	})

	// Style cho dòng Tổng kết cuối trang
	grandTotalStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "C0392B", Size: 11}, // Màu đỏ nổi bật chuyên nghiệp
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})
	
	grandTotalMoneyStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "C0392B", Size: 12},
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &vndFormat,
	})

	// ======================
	// TITLE
	// ======================
	f.SetCellValue(sheet, "A1", "BÁO CÁO DOANH THU NGÀY "+ngay)
	f.MergeCell(sheet, "A1", "F1")
	f.SetCellStyle(sheet, "A1", "F1", titleStyle)
	f.SetRowHeight(sheet, 1, 35) // Tăng chiều cao tiêu đề cho thoáng

	row := 3
	var grandTotal float64

	// ======================
	// HEADER TABLE
	// ======================
	headers := []string{"MÃ HÓA ĐƠN", "HỌ TÊN KHÁCH HÀNG", "SỐ ĐIỆN THOẠI", "TẠM TÍNH", "GIẢM GIÁ", "TỔNG TIỀN"}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheet, row, 26)

	row++

	// ======================
	// DATA PROCESSING
	// ======================
	for _, hd := range hoaDons {

		// Đổ dữ liệu dòng hóa đơn tổng (Giao diện giữ nguyên giá trị gốc float/int để Excel format tốt nhất)
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), hd.MaHoaDon)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), hd.HoTen)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), hd.SDT)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), hd.TamTinh)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), hd.TienGiam)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), hd.TongTien)

		// Set style dòng hóa đơn chính
		for col := 1; col <= 3; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellStyle(sheet, cell, cell, hdStyle)
		}
		for col := 4; col <= 6; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellStyle(sheet, cell, cell, hdMoneyStyle)
		}
		f.SetRowHeight(sheet, row, 22)
		row++

		// ======================
		// CHI TIẾT MÓN ĂN
		// ======================
		for _, ct := range hd.ChiTietHoaDons {

			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "  • "+ct.MonAn.TenMonAn) // Dùng dấu chấm tròn tinh tế thay chữ "Món:"
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), fmt.Sprintf("SL: %d", ct.SoLuong))
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), ct.ThanhTien)

			for col := 1; col <= 5; col++ {
				cell, _ := excelize.CoordinatesToCellName(col, row)
				f.SetCellStyle(sheet, cell, cell, itemStyle)
			}
			f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), itemMoneyStyle)
			
			f.SetRowHeight(sheet, row, 20)
			row++

			// OPTIONS / TOPPING
			for _, op := range ct.Options {
				name := op.TenOption
				if name == "" {
					name = op.OptionItem.TenOption
				}

				f.SetCellValue(sheet, fmt.Sprintf("B%d", row), "      + "+name) // Thụt lề sâu hơn bằng khoảng trắng cố định
				f.SetCellValue(sheet, fmt.Sprintf("F%d", row), op.GiaThem)

				for col := 1; col <= 5; col++ {
					cell, _ := excelize.CoordinatesToCellName(col, row)
					f.SetCellStyle(sheet, cell, cell, optionStyle)
				}
				f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), optionMoneyStyle)

				f.SetRowHeight(sheet, row, 18)
				row++
			}
		}

		grandTotal += hd.TongTien
		row++ // Tạo khoảng trống dòng nhẹ giữa các hóa đơn cho dễ nhìn
	}

	// ======================
	// GRAND TOTAL (TỔNG DOANH THU)
	// ======================
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), "TỔNG DOANH THU:")
	f.SetCellValue(sheet, fmt.Sprintf("F%d", row), grandTotal)

	f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), grandTotalStyle)
	f.SetCellStyle(sheet, fmt.Sprintf("F%d", row), fmt.Sprintf("F%d", row), grandTotalMoneyStyle)
	f.SetRowHeight(sheet, row, 28)

	// ======================
	// CẤU HÌNH ĐỘ RỘNG CỘT AN TOÀN CHO MOBILE
	// ======================
	f.SetColWidth(sheet, "A", "A", 15) // Mã HD rộng rãi
	f.SetColWidth(sheet, "B", "B", 35) // Tên món/Khách hàng thoải mái không bị lấp chữ
	f.SetColWidth(sheet, "C", "C", 16) // Số điện thoại / Số lượng
	f.SetColWidth(sheet, "D", "F", 18) // Các cột tiền tệ đủ rộng để không bị lỗi "###"

	f.DeleteSheet("Sheet1")

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=doanh_thu.xlsx")

	_ = f.Write(c.Writer)
}

func GetDanhSachNgayDoanhThu(c *gin.Context) {

	var days []string

	err := config.DB.
		Model(&models.HoaDon{}).
		Select("DISTINCT CAST(ngay AS DATE)").
		Where("trang_thai = ? AND trang_thai_thanh_toan = ?", "da_giao", "da_thanh_toan").
		Order("ngay DESC").
		Pluck("ngay", &days).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "fail"})
		return
	}

	c.JSON(200, gin.H{
		"data": days,
	})
}
func GetDanhSachThangDoanhThu(c *gin.Context) {

	var months []string

	err := config.DB.
		Model(&models.HoaDon{}).
		Select("DISTINCT TO_CHAR(ngay, 'YYYY-MM') as month").
		Where("trang_thai = ? AND trang_thai_thanh_toan = ?", "da_giao", "da_thanh_toan").
		Order("month DESC").
		Pluck("month", &months).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "fail"})
		return
	}

	c.JSON(200, gin.H{
		"data": months,
	})
}
func GetDanhSachNamDoanhThu(c *gin.Context) {

	var years []string

	err := config.DB.
		Model(&models.HoaDon{}).
		Select("DISTINCT EXTRACT(YEAR FROM ngay)::text as year").
		Where("trang_thai = ? AND trang_thai_thanh_toan = ?", "da_giao", "da_thanh_toan").
		Order("year DESC").
		Pluck("year", &years).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "fail"})
		return
	}

	c.JSON(200, gin.H{
		"data": years,
	})
}

func TopMonBanChay(c *gin.Context) {
	type Result struct {
		TenMonAn string `json:"ten_mon_an"`
		TongBan  int    `json:"tong_ban"`
	}

	var result []Result

	err := config.DB.Raw(`
		SELECT 
			m.ten_mon_an,
			SUM(ct.so_luong) AS tong_ban
		FROM chi_tiet_hoa_dons ct
		JOIN mon_ans m ON m.ma_mon_an = ct.ma_mon_an
		JOIN hoa_dons h ON h.ma_hoa_don = ct.ma_hoa_don
		WHERE h.trang_thai = 'da_giao'
		  AND h.trang_thai_thanh_toan = 'da_thanh_toan'
		GROUP BY m.ten_mon_an
		ORDER BY tong_ban DESC
		LIMIT 9
	`).Scan(&result).Error

	if err != nil {
		c.JSON(500, gin.H{
			"message": "Lỗi lấy top món bán chạy",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(200, result)
}

func GetTiLeHoanThanhHomNay(c *gin.Context) {
	today := time.Now().Format("2006-01-02")

	var tongDon int64
	var donHoanThanh int64

	config.DB.Model(&models.HoaDon{}).
		Where("CAST(ngay AS DATE) = ?", today).
		Count(&tongDon)

	config.DB.Model(&models.HoaDon{}).
		Where(`
			CAST(ngay AS DATE) = ?
			AND trang_thai = 'da_giao'
			AND trang_thai_thanh_toan = 'da_thanh_toan'
		`, today).
		Count(&donHoanThanh)

	tiLe := 0.0
	if tongDon > 0 {
		tiLe = float64(donHoanThanh) / float64(tongDon) * 100
	}

	c.JSON(200, gin.H{
		"data": gin.H{
			"tong_don":   tongDon,
			"hoan_thanh": donHoanThanh,
			"ti_le":      math.Round(tiLe),
		},
	})
}

func GetTopMonAnBanChay(c *gin.Context) {

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "5"))

	var result []MonAnBanChayDTO

	err := config.DB.Raw(`
	SELECT 
		ma.ma_mon_an,
		ma.ten_mon_an,
		SUM(cthd.so_luong) AS so_luong
	FROM chi_tiet_hoa_dons cthd
	JOIN hoa_dons hd ON hd.ma_hoa_don = cthd.ma_hoa_don
	JOIN mon_ans ma ON ma.ma_mon_an = cthd.ma_mon_an
	WHERE hd.trang_thai = 'da_giao'
	  AND hd.trang_thai_thanh_toan = 'da_thanh_toan'
	GROUP BY ma.ma_mon_an, ma.ten_mon_an
	ORDER BY so_luong DESC
	LIMIT ?
`, limit).Scan(&result).Error

	if err != nil {
		c.JSON(500, gin.H{"error": "Không thể lấy top món ăn"})
		return
	}

	c.JSON(200, gin.H{"data": result})
}

func formatMoneyVND(n float64) string {
	// ép về int trước cho an toàn
	v := int64(n)

	s := fmt.Sprintf("%d", v)

	// format thủ công dấu chấm
	nStr := ""
	for i, c := range reverse(s) {
		if i != 0 && i%3 == 0 {
			nStr += "."
		}
		nStr += string(c)
	}

	return reverse(nStr) + " ₫"
}
func reverse(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}
func ExportDoanhThuThang(c *gin.Context) {

	thang := c.Query("thang")
	nam := c.Query("nam")

	if thang == "" || nam == "" {
		now := time.Now()
		thang = fmt.Sprintf("%d", int(now.Month()))
		nam = fmt.Sprintf("%d", now.Year())
	}

	// tạo ngày đầu tháng
	startDate := fmt.Sprintf("%s-%02s-01", nam, thang)

	var hoaDons []models.HoaDon

	config.DB.
		Preload("ChiTietHoaDons").
		Where(`
			DATE_TRUNC('month', ngay) = DATE_TRUNC('month', ?::date)
			AND trang_thai = ?
			AND trang_thai_thanh_toan = ?
		`, startDate, "da_giao", "da_thanh_toan").
		Find(&hoaDons)

	f := excelize.NewFile()
	sheet := "DoanhThuThang"
	f.NewSheet(sheet)

	// ======================
	// KHAI BÁO STYLES (ĐỒNG BỘ CHUẨN MOBILE & PC)
	// ======================
	vndFormat := "#,##0"

	// Style cho Tiêu đề lớn
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "2C3E50"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Style cho Header của bảng
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"34495E"}, Pattern: 1}, // Xám xanh đậm Navy tinh tế
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	// Style cho dòng dữ liệu thông thường (Chữ/SĐT)
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "333333"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "E5E7EB", Style: 1}, // Đường gạch ngang mảnh phân cách các dòng dữ liệu
		},
	})

	// Style cho dòng số điện thoại (Căn giữa để nhìn ngay ngắn hơn)
	sdtStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "333333"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "E5E7EB", Style: 1},
		},
	})

	// Style cho các ô Tiền tệ của dữ liệu
	moneyStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 10, Color: "333333"},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &vndFormat,
		Border: []excelize.Border{
			{Type: "bottom", Color: "E5E7EB", Style: 1},
		},
	})

	// Style cho dòng Tổng kết cuối bảng (Nhãn chữ)
	grandTotalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "C0392B", Size: 11}, // Đỏ sẫm nổi bật thanh lịch
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})

	// Style cho dòng Tổng kết cuối bảng (Số tiền tổng)
	grandTotalMoneyStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Color: "C0392B", Size: 12},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &vndFormat,
	})

	// ======================
	// TITLE
	// ======================
	title := fmt.Sprintf("BÁO CÁO DOANH THU THÁNG %s/%s", thang, nam)
	f.SetCellValue(sheet, "A1", title)
	f.MergeCell(sheet, "A1", "E1")
	f.SetCellStyle(sheet, "A1", "E1", titleStyle)
	f.SetRowHeight(sheet, 1, 35) // Tăng độ cao hàng tiêu đề cho thoáng đạt

	row := 3
	var grandTotal float64

	// ======================
	// HEADER TABLE
	// ======================
	headers := []string{"HỌ TÊN KHÁCH HÀNG", "SỐ ĐIỆN THOẠI", "TẠM TÍNH", "TIỀN GIẢM", "TỔNG TIỀN"}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheet, row, 26)

	row++

	// ======================
	// DATA PROCESSING
	// ======================
	for _, hd := range hoaDons {

		// Đẩy dữ liệu thô vào ô (Giữ nguyên kiểu số gốc của hd.TamTinh, hd.TienGiam, hd.TongTien)
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), hd.HoTen)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), hd.SDT)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), hd.TamTinh)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), hd.TienGiam)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), hd.TongTien)

		// Gán Style tương ứng cho từng loại cột dữ liệu
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), sdtStyle)
		
		for col := 3; col <= 5; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellStyle(sheet, cell, cell, moneyStyle)
		}

		f.SetRowHeight(sheet, row, 22) // Chiều cao hàng vừa vặn, dễ bấm chọn trên mobile
		grandTotal += hd.TongTien
		row++
	}

	// Thêm một hàng trống nhẹ cách biệt trước khi hiển thị dòng tổng doanh thu
	f.SetRowHeight(sheet, row, 10)
	row++

	// ======================
	// GRAND TOTAL (TỔNG KẾT DOANH THU)
	// ======================
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), "TỔNG DOANH THU THÁNG:")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), grandTotal)

	f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), grandTotalStyle)
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), grandTotalMoneyStyle)
	f.SetRowHeight(sheet, row, 28)

	// ======================
	// CẤU HÌNH ĐỘ RỘNG CỘT AN TOÀN TRÊN SMARTPHONE
	// ======================
	f.SetColWidth(sheet, "A", "A", 30) // Tên khách hàng dài không bị nuốt chữ
	f.SetColWidth(sheet, "B", "B", 16) // Số điện thoại hiển thị ngay ngắn ở giữa
	f.SetColWidth(sheet, "C", "E", 18) // Các cột tiền tệ rộng rãi tránh hoàn toàn lỗi "###"

	f.DeleteSheet("Sheet1")

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=doanh_thu_thang.xlsx")

	_ = f.Write(c.Writer)
}
func ExportDoanhThuNam(c *gin.Context) {

	nam := c.Query("nam")

	if nam == "" {
		nam = fmt.Sprintf("%d", time.Now().Year())
	}

	namInt, _ := strconv.Atoi(nam)

	var hoaDons []models.HoaDon

	config.DB.
		Preload("ChiTietHoaDons").
		Where(`
			EXTRACT(YEAR FROM ngay) = ?
			AND trang_thai = ?
			AND trang_thai_thanh_toan = ?
		`, namInt, "da_giao", "da_thanh_toan").
		Find(&hoaDons)

	f := excelize.NewFile()
	sheet := "DoanhThuNam"
	f.NewSheet(sheet)

	// ======================
	// KHAI BÁO STYLES (ĐỒNG BỘ CHUẨN FLAT DESIGN)
	// ======================
	vndFormat := "#,##0"

	// Style cho Tiêu đề lớn
	titleStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Size: 16, Color: "2C3E50"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})

	// Style cho Header của bảng
	headerStyle, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Color: "FFFFFF", Size: 11},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"34495E"}, Pattern: 1}, // Tông Navy đậm sang trọng
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "top", Color: "000000", Style: 1},
			{Type: "bottom", Color: "000000", Style: 1},
		},
	})

	// Style cho dòng dữ liệu thông thường (Họ tên)
	dataStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "333333"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "E5E7EB", Style: 1}, // Đường phân cách hàng mảnh giúp mobile dễ nhìn
		},
	})

	// Style riêng cho Số điện thoại (Căn giữa)
	sdtStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Size: 10, Color: "333333"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border: []excelize.Border{
			{Type: "bottom", Color: "E5E7EB", Style: 1},
		},
	})

	// Style cho các ô số tiền (Căn phải tự động qua định dạng số thực)
	moneyStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Size: 10, Color: "333333"},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &vndFormat,
		Border: []excelize.Border{
			{Type: "bottom", Color: "E5E7EB", Style: 1},
		},
	})

	// Style nhãn chữ Tổng kết
	grandTotalStyle, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "C0392B", Size: 11}, // Đỏ sẫm tinh tế
		Alignment: &excelize.Alignment{Horizontal: "right", Vertical: "center"},
	})

	// Style số tiền Tổng doanh thu năm
	grandTotalMoneyStyle, _ := f.NewStyle(&excelize.Style{
		Font:         &excelize.Font{Bold: true, Color: "C0392B", Size: 12},
		Alignment:    &excelize.Alignment{Horizontal: "right", Vertical: "center"},
		CustomNumFmt: &vndFormat,
	})

	// ======================
	// TITLE
	// ======================
	title := fmt.Sprintf("BÁO CÁO DOANH THU NĂM %s", nam)
	f.SetCellValue(sheet, "A1", title)
	f.MergeCell(sheet, "A1", "E1")
	f.SetCellStyle(sheet, "A1", "E1", titleStyle)
	f.SetRowHeight(sheet, 1, 35) // Tăng độ cao tiêu đề chống tràn chữ trên mobile

	row := 3
	var grandTotal float64

	// ======================
	// HEADER TABLE
	// ======================
	headers := []string{"HỌ TÊN KHÁCH HÀNG", "SỐ ĐIỆN THOẠI", "TẠM TÍNH", "TIỀN GIẢM", "TỔNG TIỀN"}

	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, row)
		f.SetCellValue(sheet, cell, h)
		f.SetCellStyle(sheet, cell, cell, headerStyle)
	}
	f.SetRowHeight(sheet, row, 26)

	row++

	// ======================
	// DATA PROCESSING
	// ======================
	for _, hd := range hoaDons {

		// Đẩy dữ liệu thô dạng số nguyên bản vào ô tính để Excel hiển thị mượt nhất
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), hd.HoTen)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), hd.SDT)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), hd.TamTinh)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), hd.TienGiam)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), hd.TongTien)

		// Áp dụng định dạng hiển thị tương ứng từng cột
		f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), dataStyle)
		f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), sdtStyle)
		
		for col := 3; col <= 5; col++ {
			cell, _ := excelize.CoordinatesToCellName(col, row)
			f.SetCellStyle(sheet, cell, cell, moneyStyle)
		}

		f.SetRowHeight(sheet, row, 22) // Độ cao tiêu chuẩn, dễ dàng xem và chạm trên smartphone
		grandTotal += hd.TongTien
		row++
	}

	// Thêm 1 hàng trống nhỏ tạo khoảng giãn cách tinh tế trước hàng tổng kết
	f.SetRowHeight(sheet, row, 10)
	row++

	// ======================
	// GRAND TOTAL (TỔNG KẾT DOANH THU NĂM)
	// ======================
	f.SetCellValue(sheet, fmt.Sprintf("D%d", row), "TỔNG DOANH THU NĂM:")
	f.SetCellValue(sheet, fmt.Sprintf("E%d", row), grandTotal)

	f.SetCellStyle(sheet, fmt.Sprintf("D%d", row), fmt.Sprintf("D%d", row), grandTotalStyle)
	f.SetCellStyle(sheet, fmt.Sprintf("E%d", row), fmt.Sprintf("E%d", row), grandTotalMoneyStyle)
	f.SetRowHeight(sheet, row, 28)

	// ======================
	// CẤU HÌNH ĐỘ RỘNG CỘT TỐI ƯU MOBILE
	// ======================
	f.SetColWidth(sheet, "A", "A", 30) // Tên khách hàng hiển thị trọn vẹn, không chồng lấp
	f.SetColWidth(sheet, "B", "B", 16) // Số điện thoại căn giữa thoáng đãng
	f.SetColWidth(sheet, "C", "E", 18) // Cột tiền tệ mở rộng an toàn, loại bỏ triệt để lỗi "###"

	f.DeleteSheet("Sheet1")

	c.Header("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Header("Content-Disposition", "attachment; filename=doanh_thu_nam.xlsx")

	_ = f.Write(c.Writer)
}
