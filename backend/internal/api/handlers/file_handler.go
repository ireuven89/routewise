package handlers

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/ireuven89/routewise/internal/models"
	"github.com/ireuven89/routewise/internal/repository"
	"github.com/ireuven89/routewise/services"
)

type FileHandler struct {
	fileRepo    *repository.FileRepository
	projectRepo *repository.JobRepository
	s3Service   *services.S3Service
}

func NewFileHandler(fileRepo *repository.FileRepository, projectRepo *repository.JobRepository, s3Service *services.S3Service) *FileHandler {
	return &FileHandler{
		fileRepo:    fileRepo,
		projectRepo: projectRepo,
		s3Service:   s3Service,
	}
}

func (h *FileHandler) Upload(c *gin.Context) {
	projectID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	orgID := c.GetUint("organization_id")
	userID := c.GetUint("organization_user_id")
	userType := c.GetString("user_type")

	// Verify project belongs to org
	project, err := h.projectRepo.FindByID(uint(projectID), orgID)
	if err != nil || project.OrganizationID != orgID {
		c.JSON(404, gin.H{"error": "Project not found"})
		return
	}

	// Get uploaded file
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(400, gin.H{"error": "No file provided"})
		return
	}
	defer file.Close()

	// Get optional fields
	category := c.PostForm("category")       // Optional: 'progress', 'contract', etc.
	description := c.PostForm("description") // Optional

	// Detect file type from MIME
	mimeType := header.Header.Get("Content-Type")
	fileType := determineFileType(mimeType)

	if fileType == "" {
		c.JSON(400, gin.H{"error": "Unsupported file type"})
		return
	}

	// Generate S3 key
	s3Key := services.GenerateS3Key(orgID, uint(projectID), fileType, header.Filename)

	// Upload to S3
	ctx := context.Background()
	err = h.s3Service.UploadFile(ctx, file, s3Key, mimeType)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to upload file"})
		return
	}

	var uploadedByUser, uploadedByWorker *uint

	if userType == "worker" {
		uploadedByWorker = &userID
		log.Printf("  - Setting uploaded_by_worker: %d", userID)
	} else {
		uploadedByUser = &userID
		log.Printf("  - Setting uploaded_by_user: %d", userID)
	}

	log.Printf("  - uploadedByUser: %v", uploadedByUser)
	log.Printf("  - uploadedByWorker: %v", uploadedByWorker)

	// Save to database
	projectFile := &models.ProjectFile{
		ProjectID:        uint(projectID),
		UploadedByUser:   uploadedByUser,
		UploadedByWorker: uploadedByWorker,
		FileType:         fileType,
		FileCategory:     category,
		FileName:         header.Filename,
		OriginalFileName: header.Filename,
		MimeType:         mimeType,
		FileSize:         header.Size,
		FileExtension:    strings.TrimPrefix(filepath.Ext(header.Filename), "."),
		S3Bucket:         os.Getenv("S3_BUCKET_NAME"),
		S3Key:            s3Key,
		Description:      description,
	}

	err = h.fileRepo.Create(projectFile)
	if err != nil {
		h.s3Service.DeleteFile(ctx, s3Key) // Rollback S3 upload
		c.JSON(500, gin.H{"error": "Failed to save file"})
		return
	}

	c.JSON(201, gin.H{
		"message": "File uploaded successfully",
		"file":    projectFile,
	})
}

func determineFileType(mimeType string) string {
	if strings.HasPrefix(mimeType, "image/") {
		return "photo"
	}
	if mimeType == "application/pdf" || strings.HasPrefix(mimeType, "application/") {
		return "document"
	}
	return ""
}

func (h *FileHandler) ListFiles(c *gin.Context) {
	projectID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	orgID := c.GetUint("organization_id")

	// Verify project belongs to org
	project, err := h.projectRepo.FindByID(uint(projectID), orgID)
	if err != nil || project.OrganizationID != orgID {
		c.JSON(404, gin.H{"error": "Project not found"})
		return
	}

	// Optional filter by type: ?type=photo or ?type=document
	fileType := c.Query("type")

	var files []*models.ProjectFile

	if fileType != "" {
		files, err = h.fileRepo.FindByType(uint(projectID), fileType)
	} else {
		files, err = h.fileRepo.FindByProjectID(uint(projectID))
	}

	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to fetch files"})
		return
	}

	// Generate presigned URLs
	ctx := context.Background()
	for _, file := range files {
		url, err := h.s3Service.GetSignedURL(ctx, file.S3Key)
		if err == nil {
			file.S3URL = url
		}
	}

	c.JSON(200, gin.H{
		"files": files,
		"count": len(files),
	})
}

func (h *FileHandler) GetFile(c *gin.Context) {
	fileID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	orgID := c.GetUint("organization_id")

	// Get file
	file, err := h.fileRepo.FindByID(uint(fileID))
	if err != nil {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}

	// Verify belongs to org
	project, err := h.projectRepo.FindByID(file.ProjectID, orgID)
	if err != nil || project.OrganizationID != orgID {
		c.JSON(403, gin.H{"error": "Unauthorized"})
		return
	}

	// Generate presigned URL
	ctx := context.Background()
	url, err := h.s3Service.GetSignedURL(ctx, file.S3Key)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to generate download URL"})
		return
	}

	file.S3URL = url

	c.JSON(200, gin.H{"file": file})
}

func (h *FileHandler) DeleteFile(c *gin.Context) {
	fileID, _ := strconv.ParseUint(c.Param("id"), 10, 32)
	orgID := c.GetUint("organization_id")

	// Get file
	file, err := h.fileRepo.FindByID(uint(fileID))
	if err != nil {
		c.JSON(404, gin.H{"error": "File not found"})
		return
	}

	// Verify belongs to org
	project, err := h.projectRepo.FindByID(file.ProjectID, orgID)
	if err != nil || project.OrganizationID != orgID {
		c.JSON(403, gin.H{"error": "Unauthorized"})
		return
	}

	// Delete from S3
	ctx := context.Background()
	err = h.s3Service.DeleteFile(ctx, file.S3Key)
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete from storage"})
		return
	}

	// Delete from database
	err = h.fileRepo.Delete(uint(fileID))
	if err != nil {
		c.JSON(500, gin.H{"error": "Failed to delete file record"})
		return
	}

	c.JSON(200, gin.H{"message": "File deleted successfully"})
}
