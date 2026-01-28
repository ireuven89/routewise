package repository

import (
	"database/sql"

	"github.com/ireuven89/routewise/internal/models"
)

type FileRepository struct {
	db *sql.DB
}

func NewFileRepository(db *sql.DB) *FileRepository {
	return &FileRepository{db: db}
}

func (r *FileRepository) Create(file *models.ProjectFile) error {
	return r.db.QueryRow(`
        INSERT INTO project_files (
            project_id, uploaded_by_user, uploaded_by_worker,
            file_type, file_category, file_name, original_file_name,
            mime_type, file_size, file_extension,
            s3_bucket, s3_key, description, taken_at
        ) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
        RETURNING id, created_at, updated_at
    `,
		file.ProjectID, file.UploadedByUser, file.UploadedByWorker,
		file.FileType, file.FileCategory, file.FileName, file.OriginalFileName,
		file.MimeType, file.FileSize, file.FileExtension,
		file.S3Bucket, file.S3Key, file.Description, file.TakenAt,
	).Scan(&file.ID, &file.CreatedAt, &file.UpdatedAt)
}

func (r *FileRepository) FindByProjectID(projectID uint) ([]*models.ProjectFile, error) {
	rows, err := r.db.Query(`
        SELECT id, project_id, uploaded_by_user, uploaded_by_worker,
               file_type, file_category, file_name, original_file_name,
               mime_type, file_size, file_extension,
               s3_bucket, s3_key, description, taken_at,
               created_at, updated_at
        FROM project_files
        WHERE project_id = $1
        ORDER BY created_at DESC
    `, projectID)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.ProjectFile
	for rows.Next() {
		var f models.ProjectFile
		err := rows.Scan(
			&f.ID, &f.ProjectID, &f.UploadedByUser, &f.UploadedByWorker,
			&f.FileType, &f.FileCategory, &f.FileName, &f.OriginalFileName,
			&f.MimeType, &f.FileSize, &f.FileExtension,
			&f.S3Bucket, &f.S3Key, &f.Description, &f.TakenAt,
			&f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &f)
	}

	return files, nil
}

func (r *FileRepository) FindByID(id uint) (*models.ProjectFile, error) {
	var file models.ProjectFile
	err := r.db.QueryRow(`
        SELECT id, project_id, uploaded_by_user, uploaded_by_worker,
               file_type, file_category, file_name, original_file_name,
               mime_type, file_size, file_extension,
               s3_bucket, s3_key, description, taken_at,
               created_at, updated_at
        FROM project_files
        WHERE id = $1
    `, id).Scan(
		&file.ID, &file.ProjectID, &file.UploadedByUser, &file.UploadedByWorker,
		&file.FileType, &file.FileCategory, &file.FileName, &file.OriginalFileName,
		&file.MimeType, &file.FileSize, &file.FileExtension,
		&file.S3Bucket, &file.S3Key, &file.Description, &file.TakenAt,
		&file.CreatedAt, &file.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	return &file, nil
}

func (r *FileRepository) Delete(id uint) error {
	_, err := r.db.Exec(`DELETE FROM project_files WHERE id = $1`, id)
	return err
}

func (r *FileRepository) FindByType(projectID uint, fileType string) ([]*models.ProjectFile, error) {
	rows, err := r.db.Query(`
        SELECT id, project_id, uploaded_by_user, uploaded_by_worker,
               file_type, file_category, file_name, original_file_name,
               mime_type, file_size, file_extension,
               s3_bucket, s3_key, description, taken_at,
               created_at, updated_at
        FROM project_files
        WHERE project_id = $1 AND file_type = $2
        ORDER BY created_at DESC
    `, projectID, fileType)

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []*models.ProjectFile
	for rows.Next() {
		var f models.ProjectFile
		err := rows.Scan(
			&f.ID, &f.ProjectID, &f.UploadedByUser, &f.UploadedByWorker,
			&f.FileType, &f.FileCategory, &f.FileName, &f.OriginalFileName,
			&f.MimeType, &f.FileSize, &f.FileExtension,
			&f.S3Bucket, &f.S3Key, &f.Description, &f.TakenAt,
			&f.CreatedAt, &f.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		files = append(files, &f)
	}

	return files, nil
}
