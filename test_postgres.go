package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"semantic-text-processor/config"
	"semantic-text-processor/database"
	"semantic-text-processor/models"
)

func main() {
	fmt.Println("=== PostgreSQL 連接測試 ===\n")

	// 載入配置
	cfg := config.LoadConfig()

	// 建立 PostgreSQL 服務
	dbCfg := &database.PostgresConfig{
		Host:        cfg.Database.Host,
		Port:        cfg.Database.Port,
		Database:    cfg.Database.Database,
		User:        cfg.Database.User,
		Password:    cfg.Database.Password,
		SSLMode:     cfg.Database.SSLMode,
		MaxConns:    int32(cfg.Database.MaxConns),
		MinConns:    int32(cfg.Database.MinConns),
		MaxConnLife: time.Hour,
	}

	fmt.Printf("連接到 PostgreSQL:\n")
	fmt.Printf("  Host: %s:%d\n", dbCfg.Host, dbCfg.Port)
	fmt.Printf("  Database: %s\n", dbCfg.Database)
	fmt.Printf("  User: %s\n\n", dbCfg.User)

	db, err := database.NewPostgresService(dbCfg)
	if err != nil {
		log.Fatalf("❌ 無法連接資料庫: %v\n", err)
	}
	defer db.Close()

	fmt.Println("✅ 資料庫連接成功！\n")

	// 測試資料庫健康狀態
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.Health(ctx); err != nil {
		log.Fatalf("❌ 資料庫健康檢查失敗: %v\n", err)
	}

	fmt.Println("✅ 資料庫健康狀態正常\n")

	// 顯示連接池統計
	stats := db.Stats()
	fmt.Printf("連接池統計:\n")
	fmt.Printf("  總連接數: %d\n", stats.TotalConns())
	fmt.Printf("  閒置連接數: %d\n", stats.IdleConns())
	fmt.Printf("  取得連接次數: %d\n\n", stats.AcquireCount())

	// 建立 Chunk Repository
	chunkRepo := database.NewChunkRepository(db)

	// 測試 1: 建立 chunk
	fmt.Println("=== 測試 1: 建立 chunk ===")
	testChunk := &models.UnifiedChunkRecord{
		Contents:  "PostgreSQL 直接連接測試 - 成功！",
		IsPage:    false,
		Metadata: map[string]interface{}{
			"test":   true,
			"source": "postgres_test",
			"連接方式": "直接連接（不使用 Supabase REST API）",
		},
	}

	if err := chunkRepo.Create(ctx, testChunk); err != nil {
		log.Fatalf("❌ 建立 chunk 失敗: %v\n", err)
	}

	fmt.Printf("✅ Chunk 已建立，ID: %s\n\n", testChunk.ChunkID)

	// 測試 2: 查詢 chunk
	fmt.Println("=== 測試 2: 查詢 chunk ===")
	retrievedChunk, err := chunkRepo.GetByID(ctx, testChunk.ChunkID)
	if err != nil {
		log.Fatalf("❌ 查詢 chunk 失敗: %v\n", err)
	}

	fmt.Printf("✅ Chunk 查詢成功:\n")
	fmt.Printf("  ID: %s\n", retrievedChunk.ChunkID)
	fmt.Printf("  內容: %s\n", retrievedChunk.Contents)
	fmt.Printf("  Is Page: %v\n", retrievedChunk.IsPage)
	fmt.Printf("  Metadata: %+v\n\n", retrievedChunk.Metadata)

	// 測試 3: 列出所有 chunks
	fmt.Println("=== 測試 3: 列出 chunks ===")
	chunks, err := chunkRepo.List(ctx, 5, 0)
	if err != nil {
		log.Fatalf("❌ 列出 chunks 失敗: %v\n", err)
	}

	fmt.Printf("✅ 找到 %d 個 chunks:\n", len(chunks))
	for i, chunk := range chunks {
		fmt.Printf("  %d. %s (%.50s...)\n", i+1, chunk.ChunkID, chunk.Contents)
	}
	fmt.Println()

	// 測試 4: 搜尋
	fmt.Println("=== 測試 4: 搜尋 chunks ===")
	searchResults, err := chunkRepo.SearchByContent(ctx, "PostgreSQL", 5)
	if err != nil {
		log.Fatalf("❌ 搜尋失敗: %v\n", err)
	}

	fmt.Printf("✅ 搜尋 'PostgreSQL' 找到 %d 個結果:\n", len(searchResults))
	for i, chunk := range searchResults {
		fmt.Printf("  %d. %s\n", i+1, chunk.Contents)
	}
	fmt.Println()

	// 測試 5: 更新 chunk
	fmt.Println("=== 測試 5: 更新 chunk ===")
	testChunk.Contents = "PostgreSQL 直接連接測試 - 更新成功！"
	testChunk.Metadata["updated"] = true

	if err := chunkRepo.Update(ctx, testChunk); err != nil {
		log.Fatalf("❌ 更新 chunk 失敗: %v\n", err)
	}

	fmt.Println("✅ Chunk 更新成功\n")

	// 驗證更新
	updated, err := chunkRepo.GetByID(ctx, testChunk.ChunkID)
	if err != nil {
		log.Fatalf("❌ 查詢更新後的 chunk 失敗: %v\n", err)
	}

	fmt.Printf("  新內容: %s\n", updated.Contents)
	fmt.Printf("  新 Metadata: %+v\n\n", updated.Metadata)

	// 測試 6: 批次建立
	fmt.Println("=== 測試 6: 批次建立 chunks ===")
	batchChunks := []models.UnifiedChunkRecord{
		{
			Contents: "批次測試 chunk 1",
			IsPage:   false,
			Metadata: map[string]interface{}{"batch": 1},
		},
		{
			Contents: "批次測試 chunk 2",
			IsPage:   false,
			Metadata: map[string]interface{}{"batch": 2},
		},
		{
			Contents: "批次測試 chunk 3",
			IsPage:   false,
			Metadata: map[string]interface{}{"batch": 3},
		},
	}

	if err := chunkRepo.BatchCreate(ctx, batchChunks); err != nil {
		log.Fatalf("❌ 批次建立失敗: %v\n", err)
	}

	fmt.Printf("✅ 成功批次建立 %d 個 chunks\n\n", len(batchChunks))

	// 最終統計
	fmt.Println("=== 測試完成 ===")
	finalChunks, _ := chunkRepo.List(ctx, 100, 0)
	fmt.Printf("✅ 資料庫中目前有 %d 個 chunks\n", len(finalChunks))

	fmt.Println("\n🎉 所有測試通過！PostgreSQL 直接連接運作正常。")
}
