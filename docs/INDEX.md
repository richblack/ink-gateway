# 📚 Ink Gateway 技術文檔

## 💡 核心設計理念

Ink Gateway 採用 **RAG 優化的單表結構設計**，將所有內容類型統一存儲，大幅提升檢索效能。

### 為什麼選擇單表架構？

- ✅ **RAG 查詢零 JOIN** - 一次查詢取得所有相關內容
- ✅ **向量搜尋極速** - 單一向量索引覆蓋所有內容類型
- ✅ **AI 友善結構** - 扁平化設計，直接返回完整上下文
- ✅ **彈性擴展** - 新增內容類型無需變更架構

詳細說明請參閱主文檔的[系統架構章節](../README.md#🏗️-系統架構)

---

## 🚀 快速入門

### 核心架構文檔（必讀）
- [🏗️ Unified Chunk 架構設計](unified-chunk-architecture.md) - **RAG 優化單表設計深度解析**
  - 為什麼選擇單表架構
  - 技術實作細節
  - 效能分析與基準測試
  - 最佳實踐與使用範例

### 基礎文檔
- [API 參考手冊](api_reference.md) - 完整的 REST API 端點說明
- [部署指南](deployment.md) - 快速部署說明
- [Google Drive 設定](GOOGLE_DRIVE_SETUP.md) - Google Drive 整合設定

## 📖 進階主題

### 部署與運維
- [部署與運維手冊](deployment_and_operations_manual.md) - 完整的部署與維運指南
- [運維操作指南](operations.md) - 日常維運操作

### 效能優化
- [效能優化指南](performance_optimization_guide.md) - 系統效能調校
- [效能監控指南](performance_monitoring_guide.md) - 監控與追蹤
- [效能調校指南](performance_tuning_guide.md) - 深度效能調整

### 資料管理
- [資料一致性指南](data_consistency_guide.md) - 資料一致性保證
- [資料遷移與升級指南](data_migration_upgrade_guide.md) - 版本升級與遷移
- [容量規劃與擴展指南](capacity_planning_scaling_guide.md) - 系統擴展規劃

### 搜尋功能
- [搜尋 API 文檔](search_api.md) - 搜尋功能詳細說明
- [搜尋快取機制](search_cache.md) - 快取策略與優化

## ❓ 疑難排解

- [常見問題 FAQ](faq_knowledge_base.md) - 常見問題與解答

---

## 📂 文檔組織

本專案的文檔分為兩部分：

### 公開技術文檔（當前目錄）
面向使用者和開發者的技術文檔，包含 API、部署、優化等內容。

### 內部開發文檔（不公開）
包含需求規格、開發筆記、完工報告等內部資料，不納入 Git 版本控制。

---

*如有問題或建議，歡迎提交 [GitHub Issue](https://github.com/yourusername/ink-gateway/issues)*
