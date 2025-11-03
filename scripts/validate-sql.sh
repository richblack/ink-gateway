#!/bin/bash

# 驗證 SQL 腳本語法
echo "🔍 Validating SQL script syntax..."

# 檢查 SQL 文件是否存在
if [ ! -f "database/reset_and_recreate.sql" ]; then
    echo "❌ SQL file not found: database/reset_and_recreate.sql"
    exit 1
fi

# 基本語法檢查
echo "📋 Checking for common SQL syntax issues..."

# 檢查是否有未修復的 public schema 引用
echo "🔍 Checking for incorrect schema references..."

ISSUES=0

# 檢查 RPC 函數中的表格引用
if grep -q "FROM public\." database/reset_and_recreate.sql; then
    echo "❌ Found incorrect 'public.' schema references in RPC functions"
    grep -n "FROM public\." database/reset_and_recreate.sql
    ISSUES=$((ISSUES + 1))
fi

if grep -q "JOIN public\." database/reset_and_recreate.sql; then
    echo "❌ Found incorrect 'public.' schema references in JOIN clauses"
    grep -n "JOIN public\." database/reset_and_recreate.sql
    ISSUES=$((ISSUES + 1))
fi

# 檢查 RLS 政策 (排除 DROP 語句和函數定義)
if grep -v "DROP POLICY" database/reset_and_recreate.sql | grep -v "CREATE OR REPLACE FUNCTION public\." | grep -q "ON public\."; then
    echo "❌ Found incorrect 'public.' schema references in RLS policies"
    grep -v "DROP POLICY" database/reset_and_recreate.sql | grep -v "CREATE OR REPLACE FUNCTION public\." | grep -n "ON public\."
    ISSUES=$((ISSUES + 1))
fi

# 檢查是否有正確的 schema 引用
if ! grep -q "content_db\." database/reset_and_recreate.sql; then
    echo "❌ Missing content_db schema references"
    ISSUES=$((ISSUES + 1))
fi

if ! grep -q "vector_db\." database/reset_and_recreate.sql; then
    echo "❌ Missing vector_db schema references"
    ISSUES=$((ISSUES + 1))
fi

if ! grep -q "graph_db\." database/reset_and_recreate.sql; then
    echo "❌ Missing graph_db schema references"
    ISSUES=$((ISSUES + 1))
fi

if [ $ISSUES -eq 0 ]; then
    echo "✅ SQL script validation passed!"
    echo ""
    echo "📊 Schema distribution:"
    echo "  content_db: $(grep -c "content_db\." database/reset_and_recreate.sql) references"
    echo "  vector_db:  $(grep -c "vector_db\." database/reset_and_recreate.sql) references"
    echo "  graph_db:   $(grep -c "graph_db\." database/reset_and_recreate.sql) references"
    echo ""
    echo "🚀 Ready to execute in Supabase Dashboard!"
else
    echo ""
    echo "❌ Found $ISSUES issues in SQL script"
    echo "Please fix the issues above before executing"
    exit 1
fi