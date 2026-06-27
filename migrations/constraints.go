package migrations

import (
	"log"

	"gorm.io/gorm"
)

func CreateConstraints(db *gorm.DB) {
	constraints := []string{

		`ALTER TABLE dia_chis
		ADD CONSTRAINT fk_diachi_nguoidung
		FOREIGN KEY (ma_nguoi_dung)
		REFERENCES nguoi_dungs(ma_nguoi_dung)
		ON DELETE CASCADE;`,

		`ALTER TABLE mon_ans
		ADD CONSTRAINT fk_monan_loaimonan
		FOREIGN KEY (ma_loai_mon_an)
		REFERENCES loai_mon_ans(ma_loai_mon_an)
		ON DELETE SET NULL;`,

		`ALTER TABLE nhom_options
		ADD CONSTRAINT fk_nhomoption_monan
		FOREIGN KEY (ma_mon_an)
		REFERENCES mon_ans(ma_mon_an)
		ON DELETE CASCADE;`,

		`ALTER TABLE option_items
		ADD CONSTRAINT fk_optionitem_nhomoption
		FOREIGN KEY (ma_nhom_option)
		REFERENCES nhom_options(ma_nhom_option)
		ON DELETE CASCADE;`,

		`ALTER TABLE dat_bans
		ADD CONSTRAINT fk_datban_nguoidung
		FOREIGN KEY (ma_nguoi_dung)
		REFERENCES nguoi_dungs(ma_nguoi_dung)
		ON DELETE CASCADE;`,

		`ALTER TABLE dat_bans
		ADD CONSTRAINT fk_datban_banan
		FOREIGN KEY (ma_ban_an)
		REFERENCES ban_ans(ma_ban_an)
		ON DELETE SET NULL;`,

		`ALTER TABLE hoa_dons
		ADD CONSTRAINT fk_hoadon_nguoidung
		FOREIGN KEY (ma_nguoi_dung)
		REFERENCES nguoi_dungs(ma_nguoi_dung)
		ON DELETE SET NULL;`,

		`ALTER TABLE hoa_dons
		ADD CONSTRAINT fk_hoadon_giamgia
		FOREIGN KEY (giam_gia_id)
		REFERENCES giam_gia(id)
		ON DELETE SET NULL;`,

		`ALTER TABLE chi_tiet_hoa_dons
		ADD CONSTRAINT fk_cthd_hoadon
		FOREIGN KEY (ma_hoa_don)
		REFERENCES hoa_dons(ma_hoa_don)
		ON DELETE CASCADE;`,

		`ALTER TABLE chi_tiet_hoa_dons
		ADD CONSTRAINT fk_cthd_monan
		FOREIGN KEY (ma_mon_an)
		REFERENCES mon_ans(ma_mon_an)
		ON DELETE SET NULL;`,

		`ALTER TABLE chi_tiet_hoa_don_options
		ADD CONSTRAINT fk_cthdopt_cthd
		FOREIGN KEY (ma_chi_tiet)
		REFERENCES chi_tiet_hoa_dons(ma_chi_tiet)
		ON DELETE CASCADE;`,

		`ALTER TABLE gio_hangs
		ADD CONSTRAINT fk_giohang_nguoidung
		FOREIGN KEY (ma_nguoi_dung)
		REFERENCES nguoi_dungs(ma_nguoi_dung)
		ON DELETE CASCADE;`,

		`ALTER TABLE gio_hang_options
		ADD CONSTRAINT fk_giohangoption_giohang
		FOREIGN KEY (ma_gio_hang)
		REFERENCES gio_hangs(ma_gio_hang)
		ON DELETE CASCADE;`,

		`ALTER TABLE yeu_thiches
		ADD CONSTRAINT fk_yeuthich_user
		FOREIGN KEY (ma_nguoi_dung)
		REFERENCES nguoi_dungs(ma_nguoi_dung)
		ON DELETE CASCADE;`,

		`ALTER TABLE yeu_thiches
		ADD CONSTRAINT fk_yeuthich_monan
		FOREIGN KEY (ma_mon_an)
		REFERENCES mon_ans(ma_mon_an)
		ON DELETE CASCADE;`,

		`ALTER TABLE yeu_thiches
		ADD CONSTRAINT uq_user_mon
		UNIQUE (ma_nguoi_dung, ma_mon_an);`,

		`ALTER TABLE binh_luans
		ADD CONSTRAINT fk_binhluan_nguoidung
		FOREIGN KEY (ma_nguoi_dung)
		REFERENCES nguoi_dungs(ma_nguoi_dung)
		ON DELETE CASCADE;`,

		`ALTER TABLE binh_luans
		ADD CONSTRAINT fk_binhluan_parent
		FOREIGN KEY (parent_id)
		REFERENCES binh_luans(id)
		ON DELETE CASCADE;`,

		`ALTER TABLE danh_gias
		ADD CONSTRAINT fk_danhgia_nguoidung
		FOREIGN KEY (ma_nguoi_dung)
		REFERENCES nguoi_dungs(ma_nguoi_dung)
		ON DELETE CASCADE;`,

		`ALTER TABLE danh_gias
		ADD CONSTRAINT fk_danhgia_monan
		FOREIGN KEY (ma_mon_an)
		REFERENCES mon_ans(ma_mon_an)
		ON DELETE CASCADE;`,

		`ALTER TABLE thread_messages
		ADD CONSTRAINT fk_threadmessage_thread
		FOREIGN KEY (thread_id)
		REFERENCES threads(id)
		ON DELETE CASCADE;`,

		`ALTER TABLE IF EXISTS public.danh_gia
		ADD FOREIGN KEY (ma_mon_an)
		REFERENCES public.mon_ans (ma_mon_an) MATCH SIMPLE
		ON UPDATE NO ACTION
		ON DELETE NO ACTION
		NOT VALID`,
		
		`ALTER TABLE IF EXISTS public.danh_gia
		ADD FOREIGN KEY (ma_nguoi_dung)
		REFERENCES public.nguoi_dungs (ma_nguoi_dung) MATCH SIMPLE
		ON UPDATE NO ACTION
		ON DELETE NO ACTION
		NOT VALID`,

	}

	for _, query := range constraints {
		if err := db.Exec(query).Error; err != nil {
			log.Println("❌ Constraint error:", err)
		}
	}

	log.Println("✅ All constraints created")
}
