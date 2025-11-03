#!/bin/bash

echo "🔍 檢查 Obsidian 插件目錄..."

# 可能的插件目錄路徑
PATHS=(
    "$HOME/Library/Application Support/obsidian/plugins"
    "$HOME/.obsidian/plugins"
    "$HOME/.config/obsidian/plugins"
)

PLUGIN_NAME="obsidian-ink-plugin"
CURRENT_DIR=$(pwd)

echo "當前目錄: $CURRENT_DIR"

# 檢查哪個路徑存在
for path in "${PATHS[@]}"; do
    if [ -d "$path" ]; then
        echo "✅ 找到插件目錄: $path"
        
        # 創建插件子目錄
        mkdir -p "$path/$PLUGIN_NAME"
        
        # 複製文件
        cp "$CURRENT_DIR/main.js" "$path/$PLUGIN_NAME/"
        cp "$CURRENT_DIR/manifest.json" "$path/$PLUGIN_NAME/"
        
        echo "📁 已安裝到: $path/$PLUGIN_NAME"
        echo "📄 文件列表:"
        ls -la "$path/$PLUGIN_NAME"
        
        echo ""
        echo "🎉 插件安裝完成！"
        echo "請重啟 Obsidian 並在 Settings → Community plugins 中啟用插件"
        exit 0
    fi
done

echo "❌ 未找到 Obsidian 插件目錄"
echo "請手動檢查以下位置："
for path in "${PATHS[@]}"; do
    echo "  - $path"
done
echo ""
echo "或者在 Obsidian 中打開 Settings → Community plugins → Browse 來找到正確路徑"