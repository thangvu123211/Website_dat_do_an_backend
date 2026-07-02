package controllers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"

	//"strconv"
	"time"

	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/gin-gonic/gin"
	"github.com/vpa/quanlynhahang-backend/config"
	"github.com/vpa/quanlynhahang-backend/dto"
	"github.com/vpa/quanlynhahang-backend/internal/websocket"
	"github.com/vpa/quanlynhahang-backend/models"
	"golang.org/x/crypto/bcrypt"
)

type DoiMatKhauInput struct {
	MatKhauCu           string `form:"mat_khau_cu" binding:"required"`
	MatKhauMoi          string `form:"mat_khau_moi" binding:"required"`
	XacNhanMatKhauMoi   string `form:"xac_nhan_mat_khau_moi" binding:"required"`
}

// 🧱 Thêm nhân viên
func CreateNhanVien(c *gin.Context) {
	var nv models.NguoiDung

	// ✅ Lấy dữ liệu từ form-data
	if err := c.ShouldBind(&nv); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Dữ liệu form không hợp lệ: " + err.Error()})
		return
	}
	var count int64
	config.DB.Model(&models.NguoiDung{}).
		Where("email = ?", nv.Email).
		Count(&count)

	if count > 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email đã tồn tại trong hệ thống",
		})
		return
	}
	if nv.NgaySinh != "" {
		ngaySinh, err := time.Parse("2006-01-02", nv.NgaySinh)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Ngày sinh không đúng định dạng YYYY-MM-DD",
			})
			return
		}

		if !ngaySinh.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Ngày sinh phải nhỏ hơn ngày hiện tại",
			})
			return
		}
	}

	// ✅ Kiểm tra loại nhân viên chỉ được phép là "user" hoặc "admin"
	if nv.LoaiNguoiDung != "user" && nv.LoaiNguoiDung != "admin" && nv.LoaiNguoiDung != "shipper" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Loại nhân viên không hợp lệ. Chỉ chấp nhận 'user' hoặc 'admin'."})
		return
	}

	// ✅ Kiểm tra mật khẩu
	if nv.MatKhau == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Mật khẩu không được để trống"})
		return
	}

	// ✅ Hash mật khẩu
	hashedPassword, err := bcrypt.GenerateFromPassword(
		[]byte(nv.MatKhau),
		bcrypt.DefaultCost,
	)

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể mã hóa mật khẩu",
		})
		return
	}

	// Gán lại mật khẩu đã mã hóa
	nv.MatKhau = string(hashedPassword)

	// ✅ Lưu nhân viên trước để có MaNV (ID)
	if err := config.DB.Create(&nv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể tạo nhân viên: " + err.Error()})
		return
	}

	// ✅ Upload ảnh (nếu có)
	file, err := c.FormFile("image")
	if err == nil && file != nil {
		src, err := file.Open()
		if err == nil {
			defer src.Close()

			uploadResult, err := config.CLD.Upload.Upload(c, src, uploader.UploadParams{
				Folder: "nhanvien",
			})
			if err == nil {
				img := models.HinhAnh{
					OwnerID:   nv.MaNguoiDung,
					OwnerType: "nguoi_dung",
					Url:       uploadResult.SecureURL,
				}
				config.DB.Create(&img)
			}
		}
	}

	// ✅ Preload ảnh khi trả về
	config.DB.Preload("AnhNhanVien").First(&nv, nv.MaNguoiDung)

	c.JSON(http.StatusCreated, gin.H{
		"message": "Tạo nhân viên thành công",
		"data":    nv,
	})
}

// 📋 Lấy danh sách nhân viên
func GetAllNhanVien(c *gin.Context) {
	var nhanViens []models.NguoiDung
	if err := config.DB.Preload("AnhNhanVien").Find(&nhanViens).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nhanViens)
}

// 🔍 Lấy 1 nhân viên theo ID
func GetNhanVienByID(c *gin.Context) {
	id := c.Param("id")
	var nv models.NguoiDung
	if err := config.DB.Preload("AnhNhanVien").Find(&nv, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, nv)
}

// ✏️ Cập nhật nhân viên
func UpdateNhanVien(c *gin.Context) {
	id := c.Param("id")
	var nv models.NguoiDung

	// 1️⃣ Tìm nhân viên
	if err := config.DB.First(&nv, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy nhân viên"})
		return
	}

	// 2️⃣ LẤY FORM DATA TRƯỚC
	hoTen := c.PostForm("ho_ten")
	email := c.PostForm("email")
	sdt := c.PostForm("sdt")
	ngaySinh := c.PostForm("ngay_sinh")
	gioiTinh := c.PostForm("gioi_tinh")
	trangThai := c.PostForm("trang_thai")
	loaiNhanVien := c.PostForm("loai_nhan_vien")
	matKhau := c.PostForm("mat_khau")

	// ======================
	// ✅ VALIDATE NGÀY SINH
	// ======================
	if ngaySinh != "" {
		parsedDate, err := time.Parse("2006-01-02", ngaySinh)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Ngày sinh không đúng định dạng YYYY-MM-DD",
			})
			return
		}

		if !parsedDate.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Ngày sinh phải nhỏ hơn ngày hiện tại",
			})
			return
		}

		nv.NgaySinh = ngaySinh
	}

	// ======================
	// ✅ VALIDATE EMAIL TRÙNG
	// ======================
	if email != "" && email != nv.Email {
		var count int64
		config.DB.Model(&models.NguoiDung{}).
			Where("email = ? AND ma_nguoi_dung <> ?", email, nv.MaNguoiDung).
			Count(&count)

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Email đã tồn tại trong hệ thống",
			})
			return
		}

		nv.Email = email
	}

	// ======================
	// ✅ VALIDATE SDT
	// ======================
	if sdt != "" {
		matched, _ := regexp.MatchString(`^0\d{9}$`, sdt)
		if !matched {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Số điện thoại phải gồm 10 số và bắt đầu bằng 0",
			})
			return
		}
		nv.SDT = sdt
	}

	// ======================
	// UPDATE CÁC FIELD KHÁC
	// ======================
	if hoTen != "" {
		nv.HoTen = hoTen
	}
	if gioiTinh != "" {
		nv.GioiTinh = gioiTinh
	}
	if trangThai != "" {
		nv.TrangThai = trangThai
	}
	if loaiNhanVien != "" {
		nv.LoaiNguoiDung = loaiNhanVien
	}

	// ======================
	// UPDATE MẬT KHẨU
	// ======================
	if matKhau != "" {
		hashed, err := bcrypt.GenerateFromPassword([]byte(matKhau), bcrypt.DefaultCost)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Mã hóa mật khẩu thất bại"})
			return
		}
		nv.MatKhau = string(hashed)
	}

	// ======================
	// LƯU DB
	// ======================
	if err := config.DB.Save(&nv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Không thể cập nhật nhân viên"})
		return
	}

	file, err := c.FormFile("image")
	if err == nil && file != nil {

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Không thể mở file ảnh"})
			return
		}
		defer src.Close()

		// Upload Cloudinary
		uploadResult, err := config.CLD.Upload.Upload(
			c,
			src,
			uploader.UploadParams{
				Folder: "nhanvien",
			},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Upload ảnh thất bại"})
			return
		}

		// ❗ XÓA ẢNH CŨ (nếu có)
		config.DB.
			Where("owner_id = ? AND owner_type = ?", nv.MaNguoiDung, "nguoi_dung").
			Delete(&models.HinhAnh{})

		// ✅ LƯU ẢNH MỚI
		img := models.HinhAnh{
			OwnerID:   nv.MaNguoiDung,
			OwnerType: "nguoi_dung",
			Url:       uploadResult.SecureURL,
		}

		config.DB.Create(&img)
	}

	config.DB.Preload("AnhNhanVien").First(&nv, nv.MaNguoiDung)

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật thành công",
		"data":    nv,
	})
}

// 🗑️ Xóa nhân viên
func DeleteNhanVien(c *gin.Context) {
	id := c.Param("id")

	var nv models.NguoiDung

	// Tìm nhân viên
	if err := config.DB.First(&nv, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Không tìm thấy nhân viên",
		})
		return
	}

	// Cập nhật trạng thái thành khóa
	if err := config.DB.Model(&nv).Update("trang_thai", "khoa").Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Đã khóa tài khoản nhân viên",
	})
}

func UpdateThongTinCaNhan(c *gin.Context) {
	id := c.Param("id")

	// ======================
	// ✅ AUTH CHECK
	// ======================
	currentUserID := c.GetUint("user_id")
	currentRole := c.GetString("role")

	if currentRole != "admin" && fmt.Sprint(currentUserID) != id {
		c.JSON(http.StatusForbidden, gin.H{
			"error": "Bạn không có quyền chỉnh sửa thông tin người khác",
		})
		return
	}

	var nv models.NguoiDung

	// ======================
	// 1️⃣ FIND USER
	// ======================
	if err := config.DB.
		Preload("AnhNhanVien").
		First(&nv, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Không tìm thấy người dùng"})
		return
	}

	// ======================
	// 2️⃣ GET FORM DATA
	// ======================
	hoTen := c.PostForm("ho_ten")
	email := c.PostForm("email")
	sdt := c.PostForm("sdt")
	ngaySinh := c.PostForm("ngay_sinh")
	gioiTinh := c.PostForm("gioi_tinh")

	oldPassword := c.PostForm("mat_khau_cu")
	newPassword := c.PostForm("mat_khau_moi")
	confirmPassword := c.PostForm("xac_nhan_mat_khau_moi")

	

	// ======================
	// ✅ VALIDATE NGÀY SINH
	// ======================
	if ngaySinh != "" {
		parsedDate, err := time.Parse("2006-01-02", ngaySinh)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Ngày sinh không đúng định dạng YYYY-MM-DD",
			})
			return
		}

		if !parsedDate.Before(time.Now()) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Ngày sinh phải nhỏ hơn ngày hiện tại",
			})
			return
		}

		nv.NgaySinh = ngaySinh
	}

	// ======================
	// ✅ VALIDATE EMAIL TRÙNG
	// ======================
	if email != "" && email != nv.Email {
		var count int64
		config.DB.Model(&models.NguoiDung{}).
			Where("email = ? AND ma_nguoi_dung <> ?", email, nv.MaNguoiDung).
			Count(&count)

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Email đã tồn tại trong hệ thống",
			})
			return
		}

		nv.Email = email
	}

	// ======================
	// ✅ VALIDATE SDT
	// ======================
	if sdt != "" {
		matched, _ := regexp.MatchString(`^0\d{9}$`, sdt)
		if !matched {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Số điện thoại phải gồm 10 số và bắt đầu bằng 0",
			})
			return
		}
		nv.SDT = sdt
	}

	// ======================
	// UPDATE BASIC FIELDS
	// ======================
	if hoTen != "" {
		nv.HoTen = hoTen
	}
	if gioiTinh != "" {
		nv.GioiTinh = gioiTinh
	}

	// ======================
	// 🔐 UPDATE PASSWORD
	// ======================
	if oldPassword != "" || newPassword != "" || confirmPassword != "" {

		if oldPassword == "" || newPassword == "" || confirmPassword == "" {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Cần nhập đủ mật khẩu cũ, mật khẩu mới và xác nhận mật khẩu mới",
			})
			return
		}

		// ❗ Chỉ user tự đổi mới cần check mật khẩu cũ
		if currentRole != "admin" {
			if err := bcrypt.CompareHashAndPassword(
				[]byte(nv.MatKhau),
				[]byte(oldPassword),
			); err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{
					"error": "Mật khẩu cũ không đúng",
				})
				return
			}
		}

		if newPassword != confirmPassword {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Xác nhận mật khẩu mới không khớp",
			})
			return
		}

		hashed, err := bcrypt.GenerateFromPassword(
			[]byte(newPassword),
			bcrypt.DefaultCost,
		)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Mã hóa mật khẩu thất bại",
			})
			return
		}

		nv.MatKhau = string(hashed)
	}

	// ======================
	// 💾 SAVE USER
	// ======================
	if err := config.DB.Save(&nv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể cập nhật thông tin cá nhân",
		})
		return
	}

	// ======================
	// 🖼️ UPDATE IMAGE
	// ======================
	file, err := c.FormFile("image")
	if err == nil && file != nil {

		src, err := file.Open()
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Không thể mở file ảnh",
			})
			return
		}
		defer src.Close()

		uploadResult, err := config.CLD.Upload.Upload(
			c,
			src,
			uploader.UploadParams{Folder: "nhanvien"},
		)

		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Upload ảnh thất bại",
			})
			return
		}

		// ❗ Xóa ảnh cũ
		config.DB.
			Where("owner_id = ? AND owner_type = ?", nv.MaNguoiDung, "nguoi_dung").
			Delete(&models.HinhAnh{})

		// ✅ Lưu ảnh mới
		img := models.HinhAnh{
			OwnerID:   nv.MaNguoiDung,
			OwnerType: "nguoi_dung",
			Url:       uploadResult.SecureURL,
		}
		config.DB.Create(&img)
	}

	config.DB.Preload("AnhNhanVien").First(&nv, nv.MaNguoiDung)

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật thông tin cá nhân thành công",
		"data":    nv,
	})
}
func GetShippers(c *gin.Context) {
	var shippers []models.NguoiDung

	err := config.DB.
		Preload("AnhNhanVien").
		Where("loai_nguoi_dung = ?", "shipper").
		Find(&shippers).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": "Lỗi lấy danh sách shipper",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, shippers)
}

func AssignShipper(hub *websocket.Hub) gin.HandlerFunc {
	return func(c *gin.Context) {

		var input struct {
			MaHoaDon  uint `json:"ma_hoa_don"`
			MaShipper uint `json:"ma_shipper"`
		}

		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(400, gin.H{"error": "invalid input"})
			return
		}

		var hoaDon models.HoaDon

		// 1️⃣ kiểm tra hóa đơn
		if err := config.DB.
			Where("ma_hoa_don = ?", input.MaHoaDon).
			First(&hoaDon).Error; err != nil {

			c.JSON(404, gin.H{"error": "Hóa đơn không tồn tại"})
			return
		}

		// 2️⃣ chỉ assign khi đã xác nhận
		if hoaDon.TrangThai != "da_xac_nhan" {
			c.JSON(400, gin.H{"error": "Hóa đơn chưa sẵn sàng giao"})
			return
		}

		// 3️⃣ update shipper + trạng thái
		if err := config.DB.Model(&hoaDon).Updates(map[string]interface{}{
			"ma_shipper": input.MaShipper,
			"trang_thai": "da_giao_shipper",
		}).Error; err != nil {

			c.JSON(500, gin.H{"error": "Không thể assign shipper"})
			return
		}

		// 4️⃣ LOAD LẠI FULL DATA (RẤT QUAN TRỌNG)
		if err := config.DB.
			Preload("Shipper").
			Preload("ChiTietHoaDons").
			Preload("ChiTietHoaDons.MonAn").
			Preload("ChiTietHoaDons.Options").
			Preload("ChiTietHoaDons.Options.OptionItem").
			Preload("ChiTietHoaDons.Options.OptionItem.NhomOption").
			First(&hoaDon, hoaDon.MaHoaDon).Error; err != nil {

			c.JSON(500, gin.H{"error": "Không load được chi tiết hóa đơn"})
			return
		}

		// 5️⃣ REALTIME CHO SHIPPER
		hub.SendToUser(input.MaShipper, dto.WSMessage{
			Type:    "shipper_new_order",
			Payload: hoaDon,
		})

		//realtime cho user
		hub.SendToUser(hoaDon.MaNguoiDung, dto.WSMessage{
			Type: "assign_shipper_user",
			Payload: gin.H{
				"ma_hoa_don": hoaDon.MaHoaDon,
				"shipper":    hoaDon.Shipper, // ⭐ chỉ cần shipper
			},
		})

		// 6️⃣ REALTIME CHO ADMIN (cập nhật list)
		hub.Broadcast(dto.WSMessage{
			Type:    "admin_assign_shipper",
			Payload: hoaDon,
		})

		c.JSON(200, gin.H{
			"message": "Assign shipper thành công",
			"data":    hoaDon,
		})
	}
}

func DoiMatKhau(c *gin.Context) {
	// ✅ Parse ma_nguoi_dung (uint)
	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Không xác thực người dùng",
		})
		return
	}

	maNguoiDung := maNguoiDungAny.(uint)


	var input DoiMatKhauInput
	if err := c.ShouldBind(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vui lòng nhập đầy đủ thông tin",
		})
		return
	}

	// 🔹 Validate
	if len(input.MatKhauMoi) < 6 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Mật khẩu mới phải ít nhất 6 ký tự",
		})
		return
	}

	if input.MatKhauMoi != input.XacNhanMatKhauMoi {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Xác nhận mật khẩu không khớp",
		})
		return
	}

	// 🔹 Lấy người dùng đúng theo ma_nguoi_dung
	var user models.NguoiDung
	if err := config.DB.
		Where("ma_nguoi_dung = ?", maNguoiDung).
		First(&user).Error; err != nil {

		c.JSON(http.StatusNotFound, gin.H{
			"error": "Người dùng không tồn tại",
		})
		return
	}

	// 🔹 Check mật khẩu cũ
	if err := bcrypt.CompareHashAndPassword(
		[]byte(user.MatKhau),
		[]byte(input.MatKhauCu),
	); err != nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Mật khẩu cũ không đúng",
		})
		return
	}

	// 🔹 Không cho trùng mật khẩu cũ
	if bcrypt.CompareHashAndPassword(
		[]byte(user.MatKhau),
		[]byte(input.MatKhauMoi),
	) == nil {

		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Mật khẩu mới không được trùng mật khẩu cũ",
		})
		return
	}

	// 🔹 Hash mật khẩu mới
	hashed, err := bcrypt.GenerateFromPassword(
		[]byte(input.MatKhauMoi),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Lỗi mã hóa mật khẩu",
		})
		return
	}

	// 🔹 Update DB
	if err := config.DB.
		Model(&models.NguoiDung{}).
		Where("ma_nguoi_dung = ?", maNguoiDung).
		Update("mat_khau", string(hashed)).Error; err != nil {

		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Đổi mật khẩu thất bại",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Đổi mật khẩu thành công",
	})
}
func DoiAnhDaiDien(c *gin.Context) {
	// =====================
	// LẤY ID NGƯỜI DÙNG
	// =====================
	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Không xác thực người dùng",
		})
		return
	}
	maNguoiDung := maNguoiDungAny.(uint)

	// =====================
	// KIỂM TRA NGƯỜI DÙNG
	// =====================
	var user models.NguoiDung
	if err := config.DB.First(&user, maNguoiDung).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Người dùng không tồn tại",
		})
		return
	}

	// =====================
	// LẤY FILE ẢNH
	// =====================
	file, err := c.FormFile("image")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vui lòng chọn ảnh đại diện",
		})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Không thể đọc file ảnh",
		})
		return
	}
	defer src.Close()

	// =====================
	// UPLOAD CLOUDINARY
	// =====================
	uploadResult, err := config.CLD.Upload.Upload(
		c,
		src,
		uploader.UploadParams{
			Folder: "nguoidung",
		},
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Upload ảnh thất bại",
		})
		return
	}

	// =====================
	// TÌM ẢNH CŨ
	// =====================
	var img models.HinhAnh
	err = config.DB.
		Where("owner_id = ? AND owner_type = ?", user.MaNguoiDung, "nguoi_dung").
		First(&img).Error

	if err == nil {
		// ✅ CÓ ẢNH → UPDATE URL
		img.Url = uploadResult.SecureURL
		if err := config.DB.Save(&img).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể cập nhật ảnh đại diện",
			})
			return
		}
	} else {
		// ✅ CHƯA CÓ ẢNH → CREATE
		newImg := models.HinhAnh{
			OwnerID:    user.MaNguoiDung,
			OwnerType: "nguoi_dung",
			Url:       uploadResult.SecureURL,
		}
		if err := config.DB.Create(&newImg).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Không thể lưu ảnh đại diện",
			})
			return
		}
	}

	// =====================
	// PRELOAD TRẢ VỀ
	// =====================
	config.DB.Preload("AnhNhanVien").
		First(&user, user.MaNguoiDung)

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật ảnh đại diện thành công",
		"data":    user,
	})
}

func UpdateThongTinUserTuThan(c *gin.Context) {
	// ======================
	// 🔐 AUTH CHECK
	// ======================
	maNguoiDungAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Không xác thực người dùng",
		})
		return
	}

	maNguoiDung := maNguoiDungAny.(uint)

	// ======================
	// 🔍 FIND USER
	// ======================
	var nv models.NguoiDung
	if err := config.DB.First(&nv, maNguoiDung).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "Người dùng không tồn tại",
		})
		return
	}

	// ======================
	// 📥 GET FORM DATA
	// ======================
	hoTen := strings.TrimSpace(c.PostForm("ho_ten"))
	email := strings.TrimSpace(c.PostForm("email"))
	sdt := strings.TrimSpace(c.PostForm("sdt"))
	ngaySinh := strings.TrimSpace(c.PostForm("ngay_sinh"))

	// ======================
	// ❗ VALIDATE REQUIRED
	// ======================
	if hoTen == ""  {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vui lòng nhập đầy đủ họ tên",
		})
		return
	}
	if email == ""  {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vui lòng nhập đầy đủ email",
		})
		return
	}
	if sdt == ""  {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vui lòng nhập đầy đủ số điện thoại",
		})
		return
	}
	if ngaySinh == ""  {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vui lòng nhập đầy đủ ngày sinh",
		})
		return
	}
	if hoTen == "" || email == "" || sdt == "" || ngaySinh == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Vui lòng nhập đầy đủ họ tên, email, số điện thoại và ngày sinh",
		})
		return
	}

	// ======================
	// 📛 VALIDATE HỌ TÊN
	// ======================
	if len(hoTen) < 2 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Họ tên phải có ít nhất 2 ký tự",
		})
		return
	}

	// ======================
	// 📧 VALIDATE EMAIL FORMAT
	// ======================
	if !regexp.MustCompile(`^[^\s@]+@[^\s@]+\.[^\s@]+$`).MatchString(email) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Email không đúng định dạng",
		})
		return
	}

	// ======================
	// 📧 VALIDATE EMAIL TRÙNG
	// ======================
	if email != nv.Email {
		var count int64
		config.DB.Model(&models.NguoiDung{}).
			Where("email = ? AND ma_nguoi_dung <> ?", email, maNguoiDung).
			Count(&count)

		if count > 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "Email đã tồn tại trong hệ thống",
			})
			return
		}
	}

	// ======================
	// 📞 VALIDATE SDT
	// ======================
	if !regexp.MustCompile(`^0\d{9}$`).MatchString(sdt) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Số điện thoại phải gồm 10 chữ số và bắt đầu bằng 0",
		})
		return
	}

	// ======================
	// 📅 VALIDATE NGÀY SINH
	// ======================
	parsedDate, err := time.Parse("2006-01-02", ngaySinh)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ngày sinh không đúng định dạng YYYY-MM-DD",
		})
		return
	}

	if !parsedDate.Before(time.Now()) {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Ngày sinh phải nhỏ hơn ngày hiện tại",
		})
		return
	}

	// ======================
	// ✏️ UPDATE DATA
	// ======================
	nv.HoTen = hoTen
	nv.Email = email
	nv.SDT = sdt
	nv.NgaySinh = ngaySinh

	// ======================
	// 💾 SAVE
	// ======================
	if err := config.DB.Save(&nv).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Không thể cập nhật thông tin người dùng",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cập nhật thông tin cá nhân thành công",
		"data":    nv,
	})
}





