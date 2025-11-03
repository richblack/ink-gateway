#!/bin/bash

echo "🔍 Finding all Obsidian plugin installations..."
echo "=============================================="

# 搜索所有可能的 Obsidian 插件位置
PLUGIN_DIRS=$(find ~ -name "obsidian-ink-plugin*" -type d 2>/dev/null | grep -E "\.obsidian/plugins")

if [ -z "$PLUGIN_DIRS" ]; then
    echo "❌ No Obsidian Ink Plugin installations found"
    exit 1
fi

echo "📍 Found plugin installations:"
echo "$PLUGIN_DIRS"
echo ""

# 檢查每個安裝的版本
for dir in $PLUGIN_DIRS; do
    echo "📂 Checking: $dir"
    
    if [ -f "$dir/manifest.json" ]; then
        version=$(grep '"version"' "$dir/manifest.json" | cut -d'"' -f4)
        echo "   Version: $version"
        
        # 檢查是否是符號鏈接
        if [ -L "$dir" ]; then
            target=$(readlink "$dir")
            echo "   Type: Symlink → $target"
        else
            echo "   Type: Regular directory"
        fi
        
        # 檢查 main.js 的修復狀態
        if [ -f "$dir/main.js" ]; then
            if grep -q "API key is recommended" "$dir/main.js"; then
                echo "   ✅ Contains API key fix"
            else
                echo "   ❌ Missing API key fix"
            fi
            
            if grep -q "localhost:8081" "$dir/main.js"; then
                echo "   ✅ Contains URL fix"
            else
                echo "   ❌ Missing URL fix"
            fi
        else
            echo "   ❌ No main.js found"
        fi
    else
        echo "   ❌ No manifest.json found"
    fi
    echo ""
done

echo "🔧 Updating all installations..."
echo "================================"

# 更新每個安裝
for dir in $PLUGIN_DIRS; do
    echo "📂 Updating: $dir"
    
    # 如果是符號鏈接，跳過（因為會自動更新）
    if [ -L "$dir" ]; then
        echo "   ⏭️  Skipping symlink (auto-updates)"
        continue
    fi
    
    # 複製新文件
    if [ -f "obsidian-ink-plugin/main.js" ] && [ -f "obsidian-ink-plugin/manifest.json" ]; then
        cp "obsidian-ink-plugin/main.js" "$dir/"
        cp "obsidian-ink-plugin/manifest.json" "$dir/"
        echo "   ✅ Updated files"
    else
        echo "   ❌ Source files not found"
    fi
done

echo ""
echo "🎯 Recommendations:"
echo "=================="
echo "1. Restart Obsidian completely"
echo "2. Or disable/enable the plugin in each vault"
echo "3. Check that all installations show the same version"
echo "4. Test the API key setting in each vault"