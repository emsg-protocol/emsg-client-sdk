package attachments

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Attachment represents a file attachment
type Attachment struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	MimeType    string            `json:"mime_type"`
	Size        int64             `json:"size"`
	Checksum    string            `json:"checksum"`
	CreatedAt   int64             `json:"created_at"`
	Data        []byte            `json:"data,omitempty"`        // For small attachments
	URL         string            `json:"url,omitempty"`         // For large attachments
	Chunks      []*AttachmentChunk `json:"chunks,omitempty"`     // For chunked attachments
	Metadata    map[string]any    `json:"metadata,omitempty"`
	Encrypted   bool              `json:"encrypted,omitempty"`
	Compression string            `json:"compression,omitempty"` // gzip, etc.
}

// AttachmentChunk represents a chunk of a large attachment
type AttachmentChunk struct {
	Index    int    `json:"index"`
	Size     int    `json:"size"`
	Checksum string `json:"checksum"`
	Data     []byte `json:"data"`
}

// AttachmentManager manages file attachments
type AttachmentManager struct {
	maxFileSize   int64
	maxChunkSize  int64
	allowedTypes  map[string]bool
	storageDir    string
	enableChunking bool
}

// AttachmentConfig holds configuration for attachment handling
type AttachmentConfig struct {
	MaxFileSize    int64             // Maximum file size in bytes
	MaxChunkSize   int64             // Maximum chunk size for large files
	AllowedTypes   []string          // Allowed MIME types (empty = all allowed)
	StorageDir     string            // Directory for storing large attachments
	EnableChunking bool              // Enable chunking for large files
	EnableInline   bool              // Enable inline attachments for small files
	InlineLimit    int64             // Maximum size for inline attachments
}

// DefaultAttachmentConfig returns a default attachment configuration
func DefaultAttachmentConfig() *AttachmentConfig {
	return &AttachmentConfig{
		MaxFileSize:    50 * 1024 * 1024, // 50MB
		MaxChunkSize:   1024 * 1024,      // 1MB chunks
		AllowedTypes:   []string{},       // Allow all types
		StorageDir:     "./attachments",
		EnableChunking: true,
		EnableInline:   true,
		InlineLimit:    1024 * 1024, // 1MB inline limit
	}
}

// NewAttachmentManager creates a new attachment manager
func NewAttachmentManager(config *AttachmentConfig) (*AttachmentManager, error) {
	if config == nil {
		config = DefaultAttachmentConfig()
	}

	// Create storage directory if it doesn't exist
	if config.StorageDir != "" {
		if err := os.MkdirAll(config.StorageDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create storage directory: %w", err)
		}
	}

	// Build allowed types map
	allowedTypes := make(map[string]bool)
	for _, mimeType := range config.AllowedTypes {
		allowedTypes[mimeType] = true
	}

	return &AttachmentManager{
		maxFileSize:    config.MaxFileSize,
		maxChunkSize:   config.MaxChunkSize,
		allowedTypes:   allowedTypes,
		storageDir:     config.StorageDir,
		enableChunking: config.EnableChunking,
	}, nil
}

// CreateAttachmentFromFile creates an attachment from a file path
func (am *AttachmentManager) CreateAttachmentFromFile(filePath string) (*Attachment, error) {
	// Open file
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	// Get file info
	fileInfo, err := file.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	// Check file size
	if fileInfo.Size() > am.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum %d", fileInfo.Size(), am.maxFileSize)
	}

	// Determine MIME type
	mimeType := mime.TypeByExtension(filepath.Ext(filePath))
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Check if MIME type is allowed
	if len(am.allowedTypes) > 0 && !am.allowedTypes[mimeType] {
		return nil, fmt.Errorf("MIME type %s not allowed", mimeType)
	}

	// Read file data
	data, err := io.ReadAll(file)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// Calculate checksum
	checksum := am.calculateChecksum(data)

	// Create attachment
	attachment := &Attachment{
		ID:        am.generateID(),
		Name:      filepath.Base(filePath),
		MimeType:  mimeType,
		Size:      fileInfo.Size(),
		Checksum:  checksum,
		CreatedAt: time.Now().Unix(),
		Metadata:  make(map[string]any),
	}

	// Add file metadata
	attachment.Metadata["original_path"] = filePath
	attachment.Metadata["extension"] = filepath.Ext(filePath)

	// Handle based on size
	if fileInfo.Size() <= am.maxChunkSize || !am.enableChunking {
		// Store inline
		attachment.Data = data
	} else {
		// Create chunks
		chunks, err := am.createChunks(data)
		if err != nil {
			return nil, fmt.Errorf("failed to create chunks: %w", err)
		}
		attachment.Chunks = chunks
	}

	return attachment, nil
}

// CreateAttachmentFromData creates an attachment from raw data
func (am *AttachmentManager) CreateAttachmentFromData(name string, data []byte, mimeType string) (*Attachment, error) {
	// Check file size
	if int64(len(data)) > am.maxFileSize {
		return nil, fmt.Errorf("data size %d exceeds maximum %d", len(data), am.maxFileSize)
	}

	// Set default MIME type if not provided
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}

	// Check if MIME type is allowed
	if len(am.allowedTypes) > 0 && !am.allowedTypes[mimeType] {
		return nil, fmt.Errorf("MIME type %s not allowed", mimeType)
	}

	// Calculate checksum
	checksum := am.calculateChecksum(data)

	// Create attachment
	attachment := &Attachment{
		ID:        am.generateID(),
		Name:      name,
		MimeType:  mimeType,
		Size:      int64(len(data)),
		Checksum:  checksum,
		CreatedAt: time.Now().Unix(),
		Metadata:  make(map[string]any),
	}

	// Handle based on size
	if int64(len(data)) <= am.maxChunkSize || !am.enableChunking {
		// Store inline
		attachment.Data = data
	} else {
		// Create chunks
		chunks, err := am.createChunks(data)
		if err != nil {
			return nil, fmt.Errorf("failed to create chunks: %w", err)
		}
		attachment.Chunks = chunks
	}

	return attachment, nil
}

// SaveAttachment saves an attachment to storage
func (am *AttachmentManager) SaveAttachment(attachment *Attachment) error {
	if am.storageDir == "" {
		return fmt.Errorf("no storage directory configured")
	}

	// Create attachment file path
	filePath := filepath.Join(am.storageDir, attachment.ID)

	// Save attachment metadata
	metadataPath := filePath + ".meta"
	metadataData, err := json.Marshal(attachment)
	if err != nil {
		return fmt.Errorf("failed to marshal attachment metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataData, 0644); err != nil {
		return fmt.Errorf("failed to save attachment metadata: %w", err)
	}

	// Save attachment data if inline
	if len(attachment.Data) > 0 {
		if err := os.WriteFile(filePath, attachment.Data, 0644); err != nil {
			return fmt.Errorf("failed to save attachment data: %w", err)
		}
	}

	// Save chunks if chunked
	if len(attachment.Chunks) > 0 {
		for i, chunk := range attachment.Chunks {
			chunkPath := fmt.Sprintf("%s.chunk.%d", filePath, i)
			if err := os.WriteFile(chunkPath, chunk.Data, 0644); err != nil {
				return fmt.Errorf("failed to save chunk %d: %w", i, err)
			}
		}
	}

	return nil
}

// LoadAttachment loads an attachment from storage
func (am *AttachmentManager) LoadAttachment(attachmentID string) (*Attachment, error) {
	if am.storageDir == "" {
		return nil, fmt.Errorf("no storage directory configured")
	}

	// Load attachment metadata
	metadataPath := filepath.Join(am.storageDir, attachmentID+".meta")
	metadataData, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load attachment metadata: %w", err)
	}

	var attachment Attachment
	if err := json.Unmarshal(metadataData, &attachment); err != nil {
		return nil, fmt.Errorf("failed to unmarshal attachment metadata: %w", err)
	}

	// Load attachment data if inline
	filePath := filepath.Join(am.storageDir, attachmentID)
	if _, err := os.Stat(filePath); err == nil {
		data, err := os.ReadFile(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to load attachment data: %w", err)
		}
		attachment.Data = data
	}

	// Load chunks if chunked
	if len(attachment.Chunks) > 0 {
		for i := range attachment.Chunks {
			chunkPath := fmt.Sprintf("%s.chunk.%d", filePath, i)
			chunkData, err := os.ReadFile(chunkPath)
			if err != nil {
				return nil, fmt.Errorf("failed to load chunk %d: %w", i, err)
			}
			attachment.Chunks[i].Data = chunkData
		}
	}

	return &attachment, nil
}

// ValidateAttachment validates an attachment's integrity
func (am *AttachmentManager) ValidateAttachment(attachment *Attachment) error {
	var data []byte

	// Get data based on storage type
	if len(attachment.Data) > 0 {
		data = attachment.Data
	} else if len(attachment.Chunks) > 0 {
		// Reassemble chunks
		for _, chunk := range attachment.Chunks {
			data = append(data, chunk.Data...)
		}
	} else {
		return fmt.Errorf("attachment has no data")
	}

	// Validate size
	if int64(len(data)) != attachment.Size {
		return fmt.Errorf("size mismatch: expected %d, got %d", attachment.Size, len(data))
	}

	// Validate checksum
	checksum := am.calculateChecksum(data)
	if checksum != attachment.Checksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", attachment.Checksum, checksum)
	}

	return nil
}

// GetAttachmentData returns the complete data of an attachment
func (am *AttachmentManager) GetAttachmentData(attachment *Attachment) ([]byte, error) {
	if len(attachment.Data) > 0 {
		return attachment.Data, nil
	}

	if len(attachment.Chunks) > 0 {
		var data []byte
		for _, chunk := range attachment.Chunks {
			data = append(data, chunk.Data...)
		}
		return data, nil
	}

	return nil, fmt.Errorf("attachment has no data")
}

// createChunks splits data into chunks
func (am *AttachmentManager) createChunks(data []byte) ([]*AttachmentChunk, error) {
	var chunks []*AttachmentChunk
	
	for i := 0; i < len(data); i += int(am.maxChunkSize) {
		end := i + int(am.maxChunkSize)
		if end > len(data) {
			end = len(data)
		}
		
		chunkData := data[i:end]
		chunk := &AttachmentChunk{
			Index:    len(chunks),
			Size:     len(chunkData),
			Checksum: am.calculateChecksum(chunkData),
			Data:     chunkData,
		}
		
		chunks = append(chunks, chunk)
	}
	
	return chunks, nil
}

// calculateChecksum calculates SHA256 checksum of data
func (am *AttachmentManager) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

// generateID generates a unique ID for an attachment
func (am *AttachmentManager) generateID() string {
	// Simple ID generation - in production, use UUID or similar
	return fmt.Sprintf("att_%d", time.Now().UnixNano())
}

// IsInline returns true if the attachment is stored inline
func (a *Attachment) IsInline() bool {
	return len(a.Data) > 0
}

// IsChunked returns true if the attachment is chunked
func (a *Attachment) IsChunked() bool {
	return len(a.Chunks) > 0
}

// ToJSON serializes an attachment to JSON
func (a *Attachment) ToJSON() ([]byte, error) {
	return json.Marshal(a)
}

// FromJSON deserializes an attachment from JSON
func FromJSON(data []byte) (*Attachment, error) {
	var attachment Attachment
	err := json.Unmarshal(data, &attachment)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal attachment: %w", err)
	}
	return &attachment, nil
}

// GetFileExtension returns the file extension from the attachment name
func (a *Attachment) GetFileExtension() string {
	return strings.ToLower(filepath.Ext(a.Name))
}

// IsImage returns true if the attachment is an image
func (a *Attachment) IsImage() bool {
	return strings.HasPrefix(a.MimeType, "image/")
}

// IsVideo returns true if the attachment is a video
func (a *Attachment) IsVideo() bool {
	return strings.HasPrefix(a.MimeType, "video/")
}

// IsAudio returns true if the attachment is audio
func (a *Attachment) IsAudio() bool {
	return strings.HasPrefix(a.MimeType, "audio/")
}

// IsDocument returns true if the attachment is a document
func (a *Attachment) IsDocument() bool {
	docTypes := []string{
		"application/pdf",
		"application/msword",
		"application/vnd.openxmlformats-officedocument.wordprocessingml.document",
		"application/vnd.ms-excel",
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		"text/plain",
		"text/csv",
	}
	
	for _, docType := range docTypes {
		if a.MimeType == docType {
			return true
		}
	}
	
	return false
}
