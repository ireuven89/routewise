package models

import "time"

type ProjectFile struct {
	ID               uint  `json:"id"`
	ProjectID        uint  `json:"project_id"`
	UploadedByUser   *uint `json:"uploaded_by_user,omitempty"`
	UploadedByWorker *uint `json:"uploaded_by_worker,omitempty"`

	// File info
	FileType         string `json:"file_type"`     // 'photo', 'document', 'report'
	FileCategory     string `json:"file_category"` // 'progress', 'contract', 'site_photo', etc.
	FileName         string `json:"file_name"`
	OriginalFileName string `json:"original_file_name"`

	// Metadata
	MimeType      string `json:"mime_type"`
	FileSize      int64  `json:"file_size"`
	FileExtension string `json:"file_extension"`

	// S3 storage
	S3Bucket string `json:"s3_bucket"`
	S3Key    string `json:"s3_key"`
	S3URL    string `json:"s3_url,omitempty"` // Presigned URL (temporary)

	// Optional
	Description string     `json:"description,omitempty"`
	TakenAt     *time.Time `json:"taken_at,omitempty"`

	// Timestamps
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
