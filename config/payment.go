package config

import (
	"os"
	"strconv"
)

type PaymentConfig struct {
	BankBin      string
	AccountNo    string
	ReceiverName string
	MCC          string
	DescPrefix   string
	QRSize       int
}

var PaymentCfg PaymentConfig

func LoadPaymentConfig() {
	qrSizeStr := os.Getenv("VIETQR_QR_SIZE")
	qrSize, _ := strconv.Atoi(qrSizeStr)

	PaymentCfg = PaymentConfig{
		BankBin:      os.Getenv("VIETQR_BANK_BIN"),
		AccountNo:    os.Getenv("VIETQR_ACCOUNT_NO"),
		ReceiverName: os.Getenv("VIETQR_RECEIVER_NAME"),
		MCC:          os.Getenv("VIETQR_MCC"),
		DescPrefix:   os.Getenv("VIETQR_DESCRIPTION_PREFIX"),
		QRSize:       qrSize,
	}
}
