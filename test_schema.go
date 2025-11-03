package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// 連接字串
	connStr := "host=localhost port=5432 dbname=postgres user=postgres password=postgres sslmode=disable"

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("無法連接資料庫: %v", err)
	}
	defer pool.Close()

	// 測試 1: 檢查當前 schema
	var currentSchema string
	err = pool.QueryRow(ctx, "SELECT current_schema()").Scan(&currentSchema)
	if err != nil {
		log.Fatalf("無法查詢 current_schema: %v", err)
	}
	fmt.Printf("當前 schema: %s\n", currentSchema)

	// 測試 2: 檢查 search_path
	var searchPath string
	err = pool.QueryRow(ctx, "SHOW search_path").Scan(&searchPath)
	if err != nil {
		log.Fatalf("無法查詢 search_path: %v", err)
	}
	fmt.Printf("search_path: %s\n", searchPath)

	// 測試 3: 列出所有 chunks 表
	rows, err := pool.Query(ctx, `
		SELECT schemaname, tablename, tableowner
		FROM pg_tables
		WHERE tablename = 'chunks'
	`)
	if err != nil {
		log.Fatalf("無法查詢表格: %v", err)
	}
	defer rows.Close()

	fmt.Println("\n所有 chunks 表:")
	for rows.Next() {
		var schema, table, owner string
		rows.Scan(&schema, &table, &owner)
		fmt.Printf("  %s.%s (owner: %s)\n", schema, table, owner)
	}

	// 測試 4: 檢查 public.chunks 的欄位（使用 pg_catalog）
	rows2, err := pool.Query(ctx, `
		SELECT a.attname, t.typname
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_type t ON a.atttypid = t.oid
		JOIN pg_catalog.pg_class c ON a.attrelid = c.oid
		JOIN pg_catalog.pg_namespace n ON c.relnamespace = n.oid
		WHERE c.relname = 'chunks'
			AND n.nspname = 'public'
			AND a.attnum > 0
			AND NOT a.attisdropped
		ORDER BY a.attnum
	`)
	if err != nil {
		log.Fatalf("無法查詢欄位: %v", err)
	}
	defer rows2.Close()

	fmt.Println("\npublic.chunks 的欄位:")
	for rows2.Next() {
		var colName, dataType string
		rows2.Scan(&colName, &dataType)
		fmt.Printf("  %s (%s)\n", colName, dataType)
	}

	// 測試 5: 嘗試直接 INSERT (不用 pgx 的參數化)
	fmt.Println("\n嘗試直接 INSERT...")
	_, err = pool.Exec(ctx, `
		INSERT INTO public.chunks (contents, is_page, metadata)
		VALUES ('測試內容', false, '{"test": true}')
	`)
	if err != nil {
		log.Printf("❌ 直接 INSERT 失敗: %v", err)
	} else {
		fmt.Println("✅ 直接 INSERT 成功!")
	}

	// 測試 6: 嘗試帶參數的 INSERT
	fmt.Println("\n嘗試參數化 INSERT...")
	_, err = pool.Exec(ctx, `
		INSERT INTO public.chunks (contents, is_page, metadata, created_time)
		VALUES ($1, $2, $3, $4)
	`, "參數化測試", false, `{"test": true}`, time.Now())
	if err != nil {
		log.Printf("❌ 參數化 INSERT 失敗: %v", err)
	} else {
		fmt.Println("✅ 參數化 INSERT 成功!")
	}
}
