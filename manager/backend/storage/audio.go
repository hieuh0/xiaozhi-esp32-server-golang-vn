package storage

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// AudioStorage audio file storage utility
type AudioStorage struct {
	BasePath string
	MaxSize  int64
}

// NewAudioStorage creates an audio storage instance
func NewAudioStorage(basePath string, maxSize int64) *AudioStorage {
	// ensure base directory exists
	if err := os.MkdirAll(basePath, 0755); err != nil {
		panic(fmt.Sprintf("failed to create audio storage directory: %v", err))
	}

	return &AudioStorage{
		BasePath: basePath,
		MaxSize:  maxSize,
	}
}

// SaveAudioFile saves an audio file
// userID: user ID
// groupID: speaker group ID
// uuid: UUID identifier
// fileName: original file name
// fileData: file data
// returns: saved file path, file size, error
func (s *AudioStorage) SaveAudioFile(userID uint, groupID uint, uuid, fileName string, fileData io.Reader) (string, int64, error) {
	// build storage path: storage/speakers/{user_id}/{group_id}/{uuid}.wav
	dirPath := filepath.Join(s.BasePath, fmt.Sprintf("%d", userID), fmt.Sprintf("%d", groupID))

	// ensure directory exists
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create directory: %v", err)
	}

	// build file path (use UUID as filename, preserve extension)
	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".wav" // default extension
	}
	filePath := filepath.Join(dirPath, fmt.Sprintf("%s%s", uuid, ext))

	// create file
	file, err := os.Create(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	// write file data (with size limit)
	limitedReader := io.LimitReader(fileData, s.MaxSize)
	written, err := io.Copy(file, limitedReader)
	if err != nil {
		os.Remove(filePath) // remove partially written file
		return "", 0, fmt.Errorf("failed to write file: %v", err)
	}

	// check file size
	if written >= s.MaxSize {
		os.Remove(filePath)
		return "", 0, fmt.Errorf("file size exceeds limit: %d bytes", s.MaxSize)
	}

	return filePath, written, nil
}

// SaveVoiceCloneAudioFile saves a voice clone audio file
func (s *AudioStorage) SaveVoiceCloneAudioFile(userID uint, uuid, fileName string, fileData io.Reader) (string, int64, error) {
	dirPath := filepath.Join(s.BasePath, "voice_clones", fmt.Sprintf("%d", userID))
	if err := os.MkdirAll(dirPath, 0755); err != nil {
		return "", 0, fmt.Errorf("failed to create directory: %v", err)
	}

	ext := filepath.Ext(fileName)
	if ext == "" {
		ext = ".wav"
	}
	filePath := filepath.Join(dirPath, fmt.Sprintf("%s%s", uuid, ext))

	file, err := os.Create(filePath)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create file: %v", err)
	}
	defer file.Close()

	limitedReader := io.LimitReader(fileData, s.MaxSize)
	written, err := io.Copy(file, limitedReader)
	if err != nil {
		os.Remove(filePath)
		return "", 0, fmt.Errorf("failed to write file: %v", err)
	}
	if written >= s.MaxSize {
		os.Remove(filePath)
		return "", 0, fmt.Errorf("file size exceeds limit: %d bytes", s.MaxSize)
	}

	return filePath, written, nil
}

// DeleteAudioFile deletes an audio file
func (s *AudioStorage) DeleteAudioFile(filePath string) error {
	if filePath == "" {
		return nil
	}

	// check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil // file does not exist, nothing to delete
	}

	return os.Remove(filePath)
}

// GetAudioFile retrieves an audio file
func (s *AudioStorage) GetAudioFile(filePath string) (*os.File, error) {
	return os.Open(filePath)
}

// FileExists checks if a file exists
func (s *AudioStorage) FileExists(filePath string) bool {
	_, err := os.Stat(filePath)
	return !os.IsNotExist(err)
}
