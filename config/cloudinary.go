package config

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/cloudinary/cloudinary-go/v2"
	"github.com/cloudinary/cloudinary-go/v2/api/uploader"
	"github.com/joho/godotenv"
)

var CLD *cloudinary.Cloudinary

func InitCloudinary() {
	// Load .env
	err := godotenv.Load()
	if err != nil {
		log.Println("No .env file found")
	}

	cld, err := cloudinary.NewFromParams(
		os.Getenv("CLOUDINARY_CLOUD_NAME"),
		os.Getenv("CLOUDINARY_API_KEY"),
		os.Getenv("CLOUDINARY_API_SECRET"),
	)
	if err != nil {
		log.Fatal("Failed to initialize Cloudinary:", err)
		return
	}
	fmt.Println("✅ Cloudinary connected successfully")
	CLD = cld
}

func UploadFile(filePath string) (string, error) {
	resp, err := CLD.Upload.Upload(context.Background(), filePath, uploader.UploadParams{})
	if err != nil {
		return "", err
	}
	return resp.SecureURL, nil
}
