#!/usr/bin/env python3
"""
專案語音通知工具
自動偵測並使用最佳的語音助理
"""
import sys
import subprocess
from pathlib import Path

def main():
    """主函數 - 智慧選擇語音助理"""
    if len(sys.argv) < 2:
        print("用法: python3 .claude-voice/claude_notify.py "訊息" [情緒]")
        return
    
    message = sys.argv[1]
    emotion = sys.argv[2] if len(sys.argv) > 2 else "gentle"
    
    # 優先順序：
    # 1. 全域語音助理
    # 2. 本地語音助理 
    # 3. 直接路徑
    
    commands_to_try = [
        # 全域語音助理
        ['python3', '~/Documents/claude-code-voice/claude_notify.py', message, emotion],
        
        # 本地語音助理
        ['python3', str(Path(__file__).parent / 'claude_notify_direct.py'), message, emotion],
        
        # 直接路徑
        ['python3', str(Path.home() / 'Documents' / 'claude-code-voice' / 'claude_notify_direct.py'), message, emotion]
    ]
    
    for cmd in commands_to_try:
        try:
            result = subprocess.run(cmd, capture_output=True, text=True, timeout=10)
            if result.returncode == 0:
                print(f"🔊 語音通知已發送: {message}")
                if result.stdout:
                    print(result.stdout)
                return True
        except Exception as e:
            continue
    
    # 所有方法都失敗
    print(f"❌ 語音通知發送失敗")
    print(f"💡 請檢查語音助理是否已安裝: ~/Documents/claude-code-voice/")
    return False

if __name__ == "__main__":
    main()
