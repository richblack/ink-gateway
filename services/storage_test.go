package services

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"semantic-text-processor/models"
)

// TestLocalStorageAdapter 測試本地儲存適配器
func TestLocalStorageAdapter(t *testing.T) {
	// 建立臨時目錄
	tempDir := t.TempDir()
	baseURL := "file://" + tempDir
	
	adapter := NewLocalStorageAdapter(tempDir, baseURL)
	
	// 測試資料
	testData := []byte("test image content")
	metadata := &models.MediaMetadata{
		OriginalFilename: "test.png",
		ContentType:      "image/png",
		Size:             int64(len(testData)),
		Width:            100,
		Height:           100,
		Hash:             "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890",
	}
	
	ctx := context.Background()
	
	t.Run("Upload", func(t *testing.T) {
		reader := bytes.NewReader(testData)
		result, err := adapter.Upload(ctx, reader, metadata)
		
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
		
		if result.StorageType != models.StorageTypeLocal {
			t.Errorf("Expected storage type %s, got %s", models.StorageTypeLocal, result.StorageType)
		}
		
		if result.StorageID == "" {
			t.Error("StorageID should not be empty")
		}
		
		if result.URL == "" {
			t.Error("URL should not be empty")
		}
		
		// 檢查檔案是否真的被建立
		fullPath := filepath.Join(tempDir, result.StorageID)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Errorf("File was not created at %s", fullPath)
		}
	})
	
	t.Run("GetURL", func(t *testing.T) {
		// 先上傳一個檔案
		reader := bytes.NewReader(testData)
		result, err := adapter.Upload(ctx, reader, metadata)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
		
		// 測試 GetURL
		url, err := adapter.GetURL(ctx, result.StorageID)
		if err != nil {
			t.Fatalf("GetURL failed: %v", err)
		}
		
		if url != result.URL {
			t.Errorf("Expected URL %s, got %s", result.URL, url)
		}
	})
	
	t.Run("Download", func(t *testing.T) {
		// 先上傳一個檔案
		reader := bytes.NewReader(testData)
		result, err := adapter.Upload(ctx, reader, metadata)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
		
		// 測試下載
		downloadReader, err := adapter.Download(ctx, result.StorageID)
		if err != nil {
			t.Fatalf("Download failed: %v", err)
		}
		defer downloadReader.Close()
		
		// 讀取下載的內容
		downloadedData := make([]byte, len(testData))
		n, err := downloadReader.Read(downloadedData)
		if err != nil && err.Error() != "EOF" {
			t.Fatalf("Failed to read downloaded data: %v", err)
		}
		
		if n != len(testData) {
			t.Errorf("Expected %d bytes, got %d", len(testData), n)
		}
		
		if !bytes.Equal(testData, downloadedData[:n]) {
			t.Error("Downloaded data does not match original data")
		}
	})
	
	t.Run("Delete", func(t *testing.T) {
		// 先上傳一個檔案
		reader := bytes.NewReader(testData)
		result, err := adapter.Upload(ctx, reader, metadata)
		if err != nil {
			t.Fatalf("Upload failed: %v", err)
		}
		
		// 確認檔案存在
		fullPath := filepath.Join(tempDir, result.StorageID)
		if _, err := os.Stat(fullPath); os.IsNotExist(err) {
			t.Fatalf("File should exist before deletion")
		}
		
		// 測試刪除
		err = adapter.Delete(ctx, result.StorageID)
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}
		
		// 確認檔案已被刪除
		if _, err := os.Stat(fullPath); !os.IsNotExist(err) {
			t.Error("File should be deleted")
		}
	})
	
	t.Run("ScanFolder", func(t *testing.T) {
		// 建立測試檔案
		testFiles := []string{"test1.png", "test2.jpg", "test3.gif", "not_image.txt"}
		for _, filename := range testFiles {
			filePath := filepath.Join(tempDir, filename)
			err := os.WriteFile(filePath, []byte("test content"), 0644)
			if err != nil {
				t.Fatalf("Failed to create test file %s: %v", filename, err)
			}
		}
		
		// 測試掃描
		files, err := adapter.ScanFolder(ctx, tempDir)
		if err != nil {
			t.Fatalf("ScanFolder failed: %v", err)
		}
		
		// 應該只找到 3 個圖片檔案
		expectedCount := 3
		if len(files) != expectedCount {
			t.Errorf("Expected %d image files, got %d", expectedCount, len(files))
		}
		
		// 檢查檔案名稱
		foundFiles := make(map[string]bool)
		for _, file := range files {
			foundFiles[file.Filename] = true
		}
		
		expectedFiles := []string{"test1.png", "test2.jpg", "test3.gif"}
		for _, expected := range expectedFiles {
			if !foundFiles[expected] {
				t.Errorf("Expected to find file %s", expected)
			}
		}
		
		// 確認不包含非圖片檔案
		if foundFiles["not_image.txt"] {
			t.Error("Should not include non-image files")
		}
	})
	
	t.Run("HealthCheck", func(t *testing.T) {
		err := adapter.HealthCheck(ctx)
		if err != nil {
			t.Fatalf("HealthCheck failed: %v", err)
		}
	})
	
	t.Run("GetStorageType", func(t *testing.T) {
		storageType := adapter.GetStorageType()
		if storageType != models.StorageTypeLocal {
			t.Errorf("Expected storage type %s, got %s", models.StorageTypeLocal, storageType)
		}
	})
}

// TestHashService 測試雜湊服務
func TestHashService(t *testing.T) {
	service := NewHashService()
	
	t.Run("CalculateHash", func(t *testing.T) {
		testData := []byte("test data for hashing")
		reader := bytes.NewReader(testData)
		
		hash, err := service.CalculateHash(reader)
		if err != nil {
			t.Fatalf("CalculateHash failed: %v", err)
		}
		
		// SHA256 雜湊應該是 64 字元的十六進位字串
		if len(hash) != 64 {
			t.Errorf("Expected hash length 64, got %d", len(hash))
		}
		
		// 檢查是否為有效的十六進位字串
		for _, char := range hash {
			if !((char >= '0' && char <= '9') || (char >= 'a' && char <= 'f')) {
				t.Errorf("Hash contains invalid character: %c", char)
				break
			}
		}
	})
	
	t.Run("CalculateHashFromBytes", func(t *testing.T) {
		testData := []byte("test data for hashing")
		
		hash := service.CalculateHashFromBytes(testData)
		
		// 應該與 CalculateHash 產生相同結果
		reader := bytes.NewReader(testData)
		expectedHash, err := service.CalculateHash(reader)
		if err != nil {
			t.Fatalf("CalculateHash failed: %v", err)
		}
		
		if hash != expectedHash {
			t.Errorf("Hash mismatch: expected %s, got %s", expectedHash, hash)
		}
	})
	
	t.Run("VerifyHash", func(t *testing.T) {
		testData := []byte("test data for verification")
		
		// 計算正確的雜湊
		correctHash := service.CalculateHashFromBytes(testData)
		
		// 測試正確的雜湊
		reader := bytes.NewReader(testData)
		isValid, err := service.VerifyHash(reader, correctHash)
		if err != nil {
			t.Fatalf("VerifyHash failed: %v", err)
		}
		
		if !isValid {
			t.Error("Hash verification should succeed with correct hash")
		}
		
		// 測試錯誤的雜湊
		reader = bytes.NewReader(testData)
		wrongHash := strings.Repeat("0", 64)
		isValid, err = service.VerifyHash(reader, wrongHash)
		if err != nil {
			t.Fatalf("VerifyHash failed: %v", err)
		}
		
		if isValid {
			t.Error("Hash verification should fail with wrong hash")
		}
	})
}

// TestStreamingHashCalculator 測試串流雜湊計算器
func TestStreamingHashCalculator(t *testing.T) {
	calculator := NewStreamingHashCalculator()
	
	t.Run("StreamingCalculation", func(t *testing.T) {
		testData := []byte("streaming hash test data")
		
		// 分批寫入資料
		chunkSize := 5
		for i := 0; i < len(testData); i += chunkSize {
			end := i + chunkSize
			if end > len(testData) {
				end = len(testData)
			}
			
			n, err := calculator.Write(testData[i:end])
			if err != nil {
				t.Fatalf("Write failed: %v", err)
			}
			
			if n != end-i {
				t.Errorf("Expected to write %d bytes, wrote %d", end-i, n)
			}
		}
		
		// 取得最終雜湊
		hash := calculator.GetHash()
		size := calculator.GetSize()
		
		// 驗證大小
		if size != int64(len(testData)) {
			t.Errorf("Expected size %d, got %d", len(testData), size)
		}
		
		// 驗證雜湊與一次性計算的結果相同
		hashService := NewHashService()
		expectedHash := hashService.CalculateHashFromBytes(testData)
		
		if hash != expectedHash {
			t.Errorf("Hash mismatch: expected %s, got %s", expectedHash, hash)
		}
	})
	
	t.Run("Reset", func(t *testing.T) {
		// 寫入一些資料
		calculator.Write([]byte("some data"))
		
		// 重置
		calculator.Reset()
		
		// 檢查是否已重置
		if calculator.GetSize() != 0 {
			t.Error("Size should be 0 after reset")
		}
		
		// 重置後的雜湊應該是空資料的雜湊
		emptyHash := calculator.GetHash()
		hashService := NewHashService()
		expectedEmptyHash := hashService.CalculateHashFromBytes([]byte{})
		
		if emptyHash != expectedEmptyHash {
			t.Error("Hash should be empty data hash after reset")
		}
	})
}

// TestTeeHashReader 測試 TeeHashReader
func TestTeeHashReader(t *testing.T) {
	testData := []byte("test data for tee hash reader")
	reader := bytes.NewReader(testData)
	
	teeReader := NewTeeHashReader(reader)
	
	// 讀取所有資料
	readData := make([]byte, len(testData))
	n, err := teeReader.Read(readData)
	if err != nil && err.Error() != "EOF" {
		t.Fatalf("Read failed: %v", err)
	}
	
	if n != len(testData) {
		t.Errorf("Expected to read %d bytes, read %d", len(testData), n)
	}
	
	// 檢查讀取的資料是否正確
	if !bytes.Equal(testData, readData[:n]) {
		t.Error("Read data does not match original data")
	}
	
	// 檢查雜湊是否正確
	hash := teeReader.GetHash()
	size := teeReader.GetSize()
	
	if size != int64(len(testData)) {
		t.Errorf("Expected size %d, got %d", len(testData), size)
	}
	
	// 驗證雜湊
	hashService := NewHashService()
	expectedHash := hashService.CalculateHashFromBytes(testData)
	
	if hash != expectedHash {
		t.Errorf("Hash mismatch: expected %s, got %s", expectedHash, hash)
	}
}