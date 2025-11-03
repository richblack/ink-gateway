package main

import (
	"context"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5/pgxpool"
)

func main() {
	ctx := context.Background()

	// 連接字串
	connStr := "host=localhost port=5432 dbname=postgres user=postgres password=postgres sslmode=disable"

	fmt.Printf("連接字串: %s\n\n", connStr)

	pool, err := pgxpool.New(ctx, connStr)
	if err != nil {
		log.Fatalf("無法連接資料庫: %v", err)
	}
	defer pool.Close()

	// 檢查連接資訊
	var currentDatabase, currentUser, currentSchema string
	err = pool.QueryRow(ctx, "SELECT current_database(), current_user, current_schema()").Scan(&currentDatabase, &currentUser, &currentSchema)
	if err != nil {
		log.Fatalf("無法查詢連接資訊: %v", err)
	}

	fmt.Printf("實際連接資訊:\n")
	fmt.Printf("  資料庫: %s\n", currentDatabase)
	fmt.Printf("  使用者: %s\n", currentUser)
	fmt.Printf("  Schema: %s\n\n", currentSchema)

	// 檢查所有 chunks 表
	fmt.Println("所有 chunks 表 (使用 pg_class):")
	rows, err := pool.Query(ctx, `
		SELECT n.nspname, c.relname, c.oid
		FROM pg_class c
		JOIN pg_namespace n ON c.relnamespace = n.oid
		WHERE c.relname = 'chunks'
		ORDER BY n.nspname
	`)
	if err != nil {
		log.Fatalf("查詢失敗: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var schema, table string
		var oid uint32
		rows.Scan(&schema, &table, &oid)
		fmt.Printf("  %s.%s (OID: %d)\n", schema, table, oid)
	}

	// 直接用 INSERT 測試 - 不明確指定 schema
	fmt.Println("\n測試 INSERT INTO chunks (不指定 schema)...")
	_, err = pool.Exec(ctx, `INSERT INTO chunks (contents, is_page, metadata) VALUES ('測試', false, '{}')`)
	if err != nil {
		fmt.Printf("  ❌ 失敗: %v\n", err)
	} else {
		fmt.Println("  ✅ 成功!")
	}

	// 明確指定 public schema
	fmt.Println("\n測試 INSERT INTO public.chunks...")
	_, err = pool.Exec(ctx, `INSERT INTO public.chunks (contents, is_page, metadata) VALUES ('測試', false, '{}')`)
	if err != nil {
		fmt.Printf("  ❌ 失敗: %v\n", err)
	} else {
		fmt.Println("  ✅ 成功!")
	}

	// 嘗試 content_db.chunks
	fmt.Println("\n測試 INSERT INTO content_db.chunks...")
	_, err = pool.Exec(ctx, `INSERT INTO content_db.chunks (content, text_id, metadata) VALUES ('測試', uuid_generate_v4(), '{}')`)
	if err != nil {
		fmt.Printf("  ❌ 失敗: %v\n", err)
	} else {
		fmt.Println("  ✅ 成功!")
	}
}
