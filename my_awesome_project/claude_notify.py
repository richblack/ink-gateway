#!/usr/bin/env python3
# Claude Code Voice 便捷腳本
import sys
import os
import subprocess

# 嘗試使用完整的 claude-code-voice
claude_voice_path = os.path.join(os.path.dirname(__file__), 'claude-code-voice', 'claude_notify.py')
if os.path.exists(claude_voice_path):
    subprocess.run([sys.executable, claude_voice_path] + sys.argv[1:])
else:
    # 回退到簡單版本
    import platform
    
    def speak(message, emotion="gentle"):
        prefixes = {
            "gentle": "嗨，打擾一下，",
            "urgent": "快來看看！",
            "excited": "太棒了！",
            "worried": "糟糕，",
            "thinking": "嗯...讓我想想，"
        }
        full_message = prefixes.get(emotion, "") + message
        
        if platform.system() == "Darwin":
            try:
                subprocess.run(["say", full_message], check=True)
            except:
                print(f"🔊 {full_message}")
        else:
            print(f"🔊 {full_message}")
    
    if len(sys.argv) >= 2:
        message = sys.argv[1]
        emotion = sys.argv[2] if len(sys.argv) > 2 else "gentle"
        speak(message, emotion)
