package main

import (
	"log"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"context"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"gorm.io/gorm"
)

type Server struct {
	DB       *gorm.DB
	Presign  *Presigner
	Bucket   string
	Region   string
}

type UploadURLReq struct {
	Filename    *string `json:"filename"`
	ContentType *string `json:"contentType"`
}

type UploadURLRes struct {
	URL     string            `json:"url"`
	Key     string            `json:"key"`
	Headers map[string]string `json:"headers,omitempty"`
}

type SaveImageReq struct {
	Key         string  `json:"key"`
	URL         string  `json:"url"`
	ContentType *string `json:"contentType"`
	Size        *int64  `json:"size"`
	ETag        *string `json:"eTag"`
}

func main() {

	// DB
	db, err := newDB()
	if err != nil {
		log.Fatalf("db connect error: %v", err)
	}
	if err := autoMigrate(db); err != nil {
		log.Fatalf("auto migrate error: %v", err)
	}

	// S3設定
	bucket := os.Getenv("S3_BUCKET")
	region := os.Getenv("AWS_REGION")
	if bucket == "" || region == "" {
		log.Fatal("S3_BUCKET and AWS_REGION must be set")
	}

	// Presigner（SDK）
	ps, err := NewPresigner(
		context.Background(),
		bucket, region,
	)
	if err != nil {
		log.Fatalf("presigner init error: %v", err)
	}

	s := &Server{
		DB:      db,
		Presign: ps,
		Bucket:  bucket,
		Region:  region,
	}

	e := echo.New()
	e.HideBanner = true
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())

	// Routes
	e.POST("/api/upload-url", s.handleUploadURL)
	e.POST("/api/images", s.handleSaveImage)

	port := os.Getenv("PORT")
	if port == "" {
		port = "3001"
	}
	log.Printf("Echo server listening on :%s", port)
	if err := e.Start(":" + port); err != nil {
		log.Fatal(err)
	}
}

// POST /api/upload-url
// body: { filename?: string, contentType?: string }
// return: { url: string, key: string, headers?: {...} }
func (s *Server) handleUploadURL(c echo.Context) error {
	var req UploadURLReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid json"})
	}
	if req.ContentType != nil && !strings.HasPrefix(*req.ContentType, "image/") {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "contentType must be image/*"})
	}

	ext := "bin"
	if req.Filename != nil {
		if e := strings.ToLower(strings.TrimPrefix(filepath.Ext(*req.Filename), ".")); e != "" {
			ext = e
		}
	}
	key := "uploads/" + strconv.FormatInt(time.Now().UnixMilli(), 10) + "-" + randString(8) + "." + ext

	urlStr, signedHeaders, err := s.Presign.PresignPutObject(c.Request().Context(), key, req.ContentType, 5*time.Minute)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "presign error"})
	}

	return c.JSON(http.StatusOK, UploadURLRes{
		URL:     urlStr,
		Key:     key,
		Headers: signedHeaders, // フロントはこれをそのままPUTに付与
	})
}

// POST /api/images
// body: { key: string, url: string, contentType?: string, size?: number, eTag?: string }
// return: { id: number }
func (s *Server) handleSaveImage(c echo.Context) error {
	var req SaveImageReq
	if err := c.Bind(&req); err != nil {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "invalid json"})
	}
	if req.Key == "" || req.URL == "" {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "key and url are required"})
	}
	if req.ContentType != nil && !strings.HasPrefix(*req.ContentType, "image/") {
		return c.JSON(http.StatusBadRequest, echo.Map{"message": "contentType must be image/*"})
	}

	img := Image{
		Key: req.Key,
		URL: req.URL,
	}
	if req.ContentType != nil {
		img.ContentType = req.ContentType
	}
	if req.Size != nil {
		img.Size = req.Size
	}
	if req.ETag != nil {
		img.ETag = req.ETag
	}

	if err := s.DB.Create(&img).Error; err != nil {
		return c.JSON(http.StatusInternalServerError, echo.Map{"message": "db insert error"})
	}

	return c.JSON(http.StatusCreated, echo.Map{"id": img.ID})
}

const letters = "abcdefghijklmnopqrstuvwxyz0123456789"

func randString(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
