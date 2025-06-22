package test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/emsg-protocol/emsg-client-sdk/attachments"
)

func TestAttachmentManager(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	config := &attachments.AttachmentConfig{
		MaxFileSize:    1024 * 1024, // 1MB
		MaxChunkSize:   1024,        // 1KB chunks
		AllowedTypes:   []string{},  // Allow all types
		StorageDir:     tempDir,
		EnableChunking: true,
		EnableInline:   true,
		InlineLimit:    512, // 512 bytes inline limit
	}

	manager, err := attachments.NewAttachmentManager(config)
	if err != nil {
		t.Fatalf("Failed to create attachment manager: %v", err)
	}

	// Test creating attachment from data
	testData := []byte("This is test attachment data for testing purposes.")
	attachment, err := manager.CreateAttachmentFromData("test.txt", testData, "text/plain")
	if err != nil {
		t.Fatalf("Failed to create attachment from data: %v", err)
	}

	if attachment.Name != "test.txt" {
		t.Errorf("Expected name 'test.txt', got %s", attachment.Name)
	}

	if attachment.MimeType != "text/plain" {
		t.Errorf("Expected MIME type 'text/plain', got %s", attachment.MimeType)
	}

	if attachment.Size != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), attachment.Size)
	}

	if !attachment.IsInline() {
		t.Error("Small attachment should be inline")
	}

	// Test attachment validation
	err = manager.ValidateAttachment(attachment)
	if err != nil {
		t.Errorf("Attachment validation failed: %v", err)
	}

	// Test getting attachment data
	retrievedData, err := manager.GetAttachmentData(attachment)
	if err != nil {
		t.Fatalf("Failed to get attachment data: %v", err)
	}

	if string(retrievedData) != string(testData) {
		t.Errorf("Retrieved data doesn't match original")
	}
}

func TestAttachmentFromFile(t *testing.T) {
	// Create temporary directory and file for testing
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.txt")
	testData := []byte("This is a test file for attachment testing.")

	err := os.WriteFile(testFile, testData, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	config := attachments.DefaultAttachmentConfig()
	config.StorageDir = tempDir

	manager, err := attachments.NewAttachmentManager(config)
	if err != nil {
		t.Fatalf("Failed to create attachment manager: %v", err)
	}

	// Create attachment from file
	attachment, err := manager.CreateAttachmentFromFile(testFile)
	if err != nil {
		t.Fatalf("Failed to create attachment from file: %v", err)
	}

	if attachment.Name != "test.txt" {
		t.Errorf("Expected name 'test.txt', got %s", attachment.Name)
	}

	if attachment.MimeType != "text/plain; charset=utf-8" {
		t.Errorf("Expected MIME type 'text/plain; charset=utf-8', got %s", attachment.MimeType)
	}

	// Test saving and loading attachment
	err = manager.SaveAttachment(attachment)
	if err != nil {
		t.Fatalf("Failed to save attachment: %v", err)
	}

	loadedAttachment, err := manager.LoadAttachment(attachment.ID)
	if err != nil {
		t.Fatalf("Failed to load attachment: %v", err)
	}

	if loadedAttachment.ID != attachment.ID {
		t.Errorf("Loaded attachment ID doesn't match")
	}

	if loadedAttachment.Name != attachment.Name {
		t.Errorf("Loaded attachment name doesn't match")
	}
}

func TestAttachmentChunking(t *testing.T) {
	tempDir := t.TempDir()

	config := &attachments.AttachmentConfig{
		MaxFileSize:    10 * 1024, // 10KB
		MaxChunkSize:   1024,      // 1KB chunks
		AllowedTypes:   []string{},
		StorageDir:     tempDir,
		EnableChunking: true,
		EnableInline:   true,
		InlineLimit:    512, // 512 bytes inline limit
	}

	manager, err := attachments.NewAttachmentManager(config)
	if err != nil {
		t.Fatalf("Failed to create attachment manager: %v", err)
	}

	// Create large data that will be chunked
	largeData := make([]byte, 2048) // 2KB data
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	attachment, err := manager.CreateAttachmentFromData("large.bin", largeData, "application/octet-stream")
	if err != nil {
		t.Fatalf("Failed to create large attachment: %v", err)
	}

	if !attachment.IsChunked() {
		t.Error("Large attachment should be chunked")
	}

	if len(attachment.Chunks) != 2 {
		t.Errorf("Expected 2 chunks, got %d", len(attachment.Chunks))
	}

	// Validate chunk sizes
	if attachment.Chunks[0].Size != 1024 {
		t.Errorf("Expected first chunk size 1024, got %d", attachment.Chunks[0].Size)
	}

	if attachment.Chunks[1].Size != 1024 {
		t.Errorf("Expected second chunk size 1024, got %d", attachment.Chunks[1].Size)
	}

	// Test reassembling data
	retrievedData, err := manager.GetAttachmentData(attachment)
	if err != nil {
		t.Fatalf("Failed to get chunked attachment data: %v", err)
	}

	if len(retrievedData) != len(largeData) {
		t.Errorf("Retrieved data length doesn't match original: %d vs %d", len(retrievedData), len(largeData))
	}

	for i, b := range retrievedData {
		if b != largeData[i] {
			t.Errorf("Data mismatch at byte %d: expected %d, got %d", i, largeData[i], b)
			break
		}
	}
}

func TestAttachmentTypes(t *testing.T) {
	config := attachments.DefaultAttachmentConfig()
	manager, err := attachments.NewAttachmentManager(config)
	if err != nil {
		t.Fatalf("Failed to create attachment manager: %v", err)
	}

	// Test different attachment types
	testCases := []struct {
		name     string
		mimeType string
		isImage  bool
		isVideo  bool
		isAudio  bool
		isDoc    bool
	}{
		{"image.jpg", "image/jpeg", true, false, false, false},
		{"video.mp4", "video/mp4", false, true, false, false},
		{"audio.mp3", "audio/mpeg", false, false, true, false},
		{"document.pdf", "application/pdf", false, false, false, true},
		{"text.txt", "text/plain", false, false, false, true},
	}

	for _, tc := range testCases {
		attachment, err := manager.CreateAttachmentFromData(tc.name, []byte("test"), tc.mimeType)
		if err != nil {
			t.Fatalf("Failed to create attachment %s: %v", tc.name, err)
		}

		if attachment.IsImage() != tc.isImage {
			t.Errorf("IsImage() for %s: expected %v, got %v", tc.name, tc.isImage, attachment.IsImage())
		}

		if attachment.IsVideo() != tc.isVideo {
			t.Errorf("IsVideo() for %s: expected %v, got %v", tc.name, tc.isVideo, attachment.IsVideo())
		}

		if attachment.IsAudio() != tc.isAudio {
			t.Errorf("IsAudio() for %s: expected %v, got %v", tc.name, tc.isAudio, attachment.IsAudio())
		}

		if attachment.IsDocument() != tc.isDoc {
			t.Errorf("IsDocument() for %s: expected %v, got %v", tc.name, tc.isDoc, attachment.IsDocument())
		}
	}
}

func TestAttachmentSerialization(t *testing.T) {
	config := attachments.DefaultAttachmentConfig()
	manager, err := attachments.NewAttachmentManager(config)
	if err != nil {
		t.Fatalf("Failed to create attachment manager: %v", err)
	}

	// Create test attachment
	testData := []byte("Test attachment data")
	attachment, err := manager.CreateAttachmentFromData("test.txt", testData, "text/plain")
	if err != nil {
		t.Fatalf("Failed to create attachment: %v", err)
	}

	// Test JSON serialization
	jsonData, err := attachment.ToJSON()
	if err != nil {
		t.Fatalf("Failed to serialize attachment to JSON: %v", err)
	}

	// Test JSON deserialization
	deserializedAttachment, err := attachments.FromJSON(jsonData)
	if err != nil {
		t.Fatalf("Failed to deserialize attachment from JSON: %v", err)
	}

	if deserializedAttachment.ID != attachment.ID {
		t.Errorf("Deserialized ID doesn't match: %s vs %s", deserializedAttachment.ID, attachment.ID)
	}

	if deserializedAttachment.Name != attachment.Name {
		t.Errorf("Deserialized name doesn't match: %s vs %s", deserializedAttachment.Name, attachment.Name)
	}

	if deserializedAttachment.MimeType != attachment.MimeType {
		t.Errorf("Deserialized MIME type doesn't match: %s vs %s", deserializedAttachment.MimeType, attachment.MimeType)
	}

	if deserializedAttachment.Size != attachment.Size {
		t.Errorf("Deserialized size doesn't match: %d vs %d", deserializedAttachment.Size, attachment.Size)
	}
}

func TestAttachmentConfig(t *testing.T) {
	config := attachments.DefaultAttachmentConfig()

	if config.MaxFileSize != 50*1024*1024 {
		t.Errorf("Expected MaxFileSize 50MB, got %d", config.MaxFileSize)
	}

	if config.MaxChunkSize != 1024*1024 {
		t.Errorf("Expected MaxChunkSize 1MB, got %d", config.MaxChunkSize)
	}

	if !config.EnableChunking {
		t.Error("Expected EnableChunking to be true")
	}

	if !config.EnableInline {
		t.Error("Expected EnableInline to be true")
	}

	if config.InlineLimit != 1024*1024 {
		t.Errorf("Expected InlineLimit 1MB, got %d", config.InlineLimit)
	}
}

func TestAttachmentValidation(t *testing.T) {
	config := &attachments.AttachmentConfig{
		MaxFileSize:    1024,                   // 1KB limit
		MaxChunkSize:   512,                    // 512 byte chunks
		AllowedTypes:   []string{"text/plain"}, // Only allow text files
		StorageDir:     t.TempDir(),
		EnableChunking: true,
		EnableInline:   true,
		InlineLimit:    256,
	}

	manager, err := attachments.NewAttachmentManager(config)
	if err != nil {
		t.Fatalf("Failed to create attachment manager: %v", err)
	}

	// Test file size limit
	largeData := make([]byte, 2048) // 2KB - exceeds limit
	_, err = manager.CreateAttachmentFromData("large.txt", largeData, "text/plain")
	if err == nil {
		t.Error("Expected error for file size exceeding limit")
	}

	// Test MIME type restriction
	_, err = manager.CreateAttachmentFromData("image.jpg", []byte("fake image"), "image/jpeg")
	if err == nil {
		t.Error("Expected error for disallowed MIME type")
	}

	// Test valid attachment
	validData := []byte("Valid text data")
	attachment, err := manager.CreateAttachmentFromData("valid.txt", validData, "text/plain")
	if err != nil {
		t.Errorf("Valid attachment creation failed: %v", err)
	}

	if attachment == nil {
		t.Error("Valid attachment is nil")
	}
}
