package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// 連接字串 - 不設置 search_path
	connStr := "host=localhost port=5432 dbname=postgres user=postgres password=postgres sslmode=disable"

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("無法連接資料庫: %v", err)
	}
	defer pool.Close()

	// 明確設置 search_path
	fmt.Println("設置 search_path TO public...")
	_, err = pool.Exec(ctx, "SET search_path TO public")
	if err != nil {
		log.Fatalf("無法設置 search_path: %v", err)
	}

	// 確認設置成功
	var searchPath string
	err = pool.QueryRow(ctx, "SHOW search_path").Scan(&searchPath)
	if err != nil {
		log.Fatalf("無法查詢 search_path: %v", err)
	}
	fmt.Printf("當前 search_path: %s\n\n", searchPath)

	// 測試 1: 用 \d chunks 的 SQL 等價查詢
	fmt.Println("檢查當前可見的 chunks 表 OID:")
	var oid uint32
	var nspname, relname string
	err = pool.QueryRow(ctx, `
		SELECT c.oid, n.nspname, c.relname
		FROM pg_catalog.pg_class c
		LEFT JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
		WHERE c.relname = 'chunks'
			AND pg_catalog.pg_table_is_visible(c.oid)
	`).Scan(&oid, &nspname, &relname)
	if err != nil {
		log.Printf("查詢失敗: %v", err)
	} else {
		fmt.Printf("可見的 chunks 表: %s.%s (OID: %d)\n\n", nspname, relname, oid)
	}

	// 測試 2: 列出該表的欄位
	fmt.Println("該表的前 5 個欄位:")
	rows, err := pool.Query(ctx, `
		SELECT a.attname, t.typname
		FROM pg_catalog.pg_attribute a
		JOIN pg_catalog.pg_type t ON a.atttypid = t.oid
		WHERE a.attrelid = $1
			AND a.attnum > 0
			AND NOT a.attisdropped
		ORDER BY a.attnum
		LIMIT 5
	`, oid)
	if err != nil {
		log.Fatalf("無法查詢欄位: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var attname, typname string
		rows.Scan(&attname, &typname)
		fmt.Printf("  %s (%s)\n", attname, typname)
	}

	// 測試 3: 嘗試 INSERT
	fmt.Println("\n嘗試 INSERT INTO chunks...")
	_, err = pool.Exec(ctx, `
		INSERT INTO chunks (content, is_template, metadata)
		VALUES ('測試內容', false, '{"test": true}')
	`)
	if err != nil {
		log.Printf("❌ INSERT 失敗: %v\n", err)
	} else {
		fmt.Println("✅ INSERT 成功!")
	}
}
