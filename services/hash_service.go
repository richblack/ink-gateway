package services

import (
	"crypto/sha256"
	"fmt"
	"hash"
	"io"
)

// HashService 檔案雜湊計算服務
type HashService struct {
	algorithm string
}

// NewHashService 建立新的雜湊服務
func NewHashService() *HashService {
	return &HashService{
		algorithm: "sha256",
	}
}

// CalculateHash 計算檔案的 SHA256 雜湊值
func (h *HashService) CalculateHash(reader io.Reader) (string, error) {
	hasher := sha256.New()
	
	// 使用緩衝區讀取，避免一次性載入大檔案到記憶體
	buffer := make([]byte, 64*1024) // 64KB 緩衝區
	
	for {
		n, err := reader.Read(buffer)
		if n > 0 {
			hasher.Write(buffer[:n])
		}
		
		if err == io.EOF {
			break
		}
		
		if err != nil {
			return "", fmt.Errorf("failed to read data for hashing: %w", err)
		}
	}
	
	// 計算最終雜湊值
	hashBytes := hasher.Sum(nil)
	hashString := fmt.Sprintf("%x", hashBytes)
	
	return hashString, nil
}

// CalculateHashFromBytes 從位元組陣列計算雜湊值
func (h *HashService) CalculateHashFromBytes(data []byte) string {
	hasher := sha256.New()
	hasher.Write(data)
	hashBytes := hasher.Sum(nil)
	return fmt.Sprintf("%x", hashBytes)
}

// VerifyHash 驗證檔案雜湊值
func (h *HashService) VerifyHash(reader io.Reader, expectedHash string) (bool, error) {
	actualHash, err := h.CalculateHash(reader)
	if err != nil {
		return false, fmt.Errorf("failed to calculate hash for verification: %w", err)
	}
	
	return actualHash == expectedHash, nil
}

// GetAlgorithm 取得雜湊演算法名稱
func (h *HashService) GetAlgorithm() string {
	return h.algorithm
}

// StreamingHashCalculator 串流雜湊計算器
type StreamingHashCalculator struct {
	hasher hash.Hash
	size   int64
}

// NewStreamingHashCalculator 建立新的串流雜湊計算器
func NewStreamingHashCalculator() *StreamingHashCalculator {
	return &StreamingHashCalculator{
		hasher: sha256.New(),
		size:   0,
	}
}

// Write 實作 io.Writer 介面
func (s *StreamingHashCalculator) Write(p []byte) (n int, err error) {
	n, err = s.hasher.Write(p)
	s.size += int64(n)
	return n, err
}

// GetHash 取得計算的雜湊值
func (s *StreamingHashCalculator) GetHash() string {
	hashBytes := s.hasher.Sum(nil)
	return fmt.Sprintf("%x", hashBytes)
}

// GetSize 取得處理的資料大小
func (s *StreamingHashCalculator) GetSize() int64 {
	return s.size
}

// Reset 重置計算器
func (s *StreamingHashCalculator) Reset() {
	s.hasher.Reset()
	s.size = 0
}

// TeeHashReader 同時讀取和計算雜湊的 Reader
type TeeHashReader struct {
	reader     io.Reader
	calculator *StreamingHashCalculator
}

// NewTeeHashReader 建立新的 TeeHashReader
func NewTeeHashReader(reader io.Reader) *TeeHashReader {
	return &TeeHashReader{
		reader:     reader,
		calculator: NewStreamingHashCalculator(),
	}
}

// Read 實作 io.Reader 介面
func (t *TeeHashReader) Read(p []byte) (n int, err error) {
	n, err = t.reader.Read(p)
	if n > 0 {
		t.calculator.Write(p[:n])
	}
	return n, err
}

// GetHash 取得計算的雜湊值
func (t *TeeHashReader) GetHash() string {
	return t.calculator.GetHash()
}

// GetSize 取得讀取的資料大小
func (t *TeeHashReader) GetSize() int64 {
	return t.calculator.GetSize()
}

// HashAndSizeReader 同時計算雜湊和大小的 Reader 包裝器
type HashAndSizeReader struct {
	*TeeHashReader
}

// NewHashAndSizeReader 建立新的 HashAndSizeReader
func NewHashAndSizeReader(reader io.Reader) *HashAndSizeReader {
	return &HashAndSizeReader{
		TeeHashReader: NewTeeHashReader(reader),
	}
}

// GetMetadata 取得檔案元資料（雜湊和大小）
func (h *HashAndSizeReader) GetMetadata() (hash string, size int64) {
	return h.GetHash(), h.GetSize()
}