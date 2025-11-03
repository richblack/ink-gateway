import { describe, it, expect, beforeEach, afterEach, vi } from 'vitest';
import { TFile } from 'obsidian';
import { ObsidianInkPlugin } from '../../src/main';
import { UnifiedChunk, Template, SearchQuery } from '../../src/types';
import { createMockEnvironment } from '../mock-data/mock-environment';

/**
 * 端到端使用場景測試
 * 模擬真實使用者工作流程
 */
describe('End-to-End User Scenarios', () => {
    let plugin: ObsidianInkPlugin;
    let mockApp: any;
    let mockVault: any;

    beforeEach(async () => {
        const mockEnv = createMockEnvironment();
        mockApp = mockEnv.app;
        mockVault = mockEnv.vault;
        
        plugin = new ObsidianInkPlugin(mockApp, {} as any);
        await plugin.onload();
        
        // 設置基本 API 模擬
        setupBasicAPIMocks();
    });

    afterEach(async () => {
        await plugin.onunload();
        vi.clearAllMocks();
    });

    /**
     * 場景 1: 新使用者首次使用插件
     */
    describe('Scenario 1: New User First Time Setup', () => {
        it('should guide new user through complete setup process', async () => {
            // 1. 使用者安裝插件並首次啟動
            expect(plugin.settings).toBeDefined();
            expect(plugin.settings.inkGatewayUrl).toBe('');
            
            // 2. 使用者配置 Ink-Gateway 連線
            plugin.settings.inkGatewayUrl = 'https://api.ink-gateway.com';
            plugin.settings.apiKey = 'test-api-key';
            await plugin.saveSettings();
            
            // 3. 測試連線
            vi.spyOn(plugin.apiClient, 'searchChunks').mockResolvedValue({
                items: [],
                totalCount: 0,
                searchTime: 100,
                cacheHit: false
            });
            
            const connectionTest = await plugin.settingsManager.testConnection();
            expect(connectionTest.success).toBe(true);
            
            // 4. 使用者創建第一個筆記
            const firstNote = `# My First Note

This is my first note using the Ink plugin.

## Key Points
- Point 1
- Point 2

#important #first-note`;

            const mockFile = createMockFile('first-note.md', firstNote);
            vi.spyOn(mockVault, 'read').mockResolvedValue(firstNote);
            
            // 5. 自動處理內容
            await plugin.contentManager.handleContentChange(mockFile);
            
            // 6. 驗證內容已同步
            expect(plugin.apiClient.batchCreateChunks).toHaveBeenCalled();
        });
    });

    /**
     * 場景 2: 研究員使用模板管理聯絡人
     */
    describe('Scenario 2: Researcher Managing Contacts with Templates', () => {
        it('should support complete contact management workflow', async () => {
            // 1. 創建聯絡人模板
            const contactTemplate: Template = {
                id: 'contact-template',
                name: 'Contact Template',
                slots: [
                    { id: 'name', name: 'Name', type: 'text', required: true },
                    { id: 'email', name: 'Email', type: 'text', required: false },
                    { id: 'organization', name: 'Organization', type: 'text', required: false },
                    { id: 'research_area', name: 'Research Area', type: 'text', required: false }
                ],
                structure: {
                    layout: 'vertical',
                    sections: [
                        { type: 'header', content: '# Contact: {{name}}' },
                        { type: 'field', content: 'Email: {{email}}' },
                        { type: 'field', content: 'Organization: {{organization}}' },
                        { type: 'field', content: 'Research Area: {{research_area}}' }
                    ]
                },
                metadata: {
                    createdAt: new Date(),
                    updatedAt: new Date(),
                    version: '1.0'
                }
            };
            
            vi.spyOn(plugin.apiClient, 'createTemplate').mockResolvedValue(contactTemplate);
            
            const template = await plugin.templateManager.createTemplate(
                'Contact Template',
                contactTemplate.structure
            );
            
            expect(template).toEqual(contactTemplate);
            
            // 2. 使用模板創建聯絡人
            const contactContent = `---
name: Dr. Jane Smith
email: jane.smith@university.edu
organization: MIT
research_area: Machine Learning
---

# Contact: Dr. Jane Smith

Email: jane.smith@university.edu
Organization: MIT
Research Area: Machine Learning

## Notes
- Met at AI conference 2024
- Interested in collaboration on NLP project

#contact #ai-researcher #collaboration`;

            const contactFile = createMockFile('contacts/jane-smith.md', contactContent);
            vi.spyOn(mockVault, 'read').mockResolvedValue(contactContent);
            
            // 3. 應用模板並處理內容
            await plugin.templateManager.applyTemplate('contact-template', contactFile);
            await plugin.contentManager.handleContentChange(contactFile);
            
            // 4. 搜尋聯絡人
            const searchQuery: SearchQuery = {
                tags: ['contact', 'ai-researcher'],
                tagLogic: 'AND',
                searchType: 'exact'
            };
            
            const searchResult = await plugin.searchManager.performSearch(searchQuery);
            expect(searchResult.items.length).toBeGreaterThan(0);
            
            // 5. 查詢所有聯絡人模板實例
            const templateInstances = await plugin.templateManager.getTemplateInstances('contact-template');
            expect(templateInstances.length).toBeGreaterThan(0);
        });
    });

    /**
     * 場景 3: 學生進行研究筆記和 AI 輔助
     */
    describe('Scenario 3: Student Research Notes with AI Assistance', () => {
        it('should support research workflow with AI chat', async () => {
            // 1. 學生創建研究筆記
            const researchNote = `# Machine Learning Research Notes

## Introduction
Machine learning is a subset of artificial intelligence that focuses on algorithms that can learn from data.

## Key Concepts
- Supervised Learning
  - Classification
  - Regression
- Unsupervised Learning
  - Clustering
  - Dimensionality Reduction
- Reinforcement Learning

## Questions
- How does gradient descent work?
- What are the differences between various optimization algorithms?

#research #machine-learning #study`;

            const noteFile = createMockFile('research/ml-notes.md', researchNote);
            vi.spyOn(mockVault, 'read').mockResolvedValue(researchNote);
            
            // 2. 自動處理和同步內容
            await plugin.contentManager.handleContentChange(noteFile);
            
            // 3. 學生使用 AI 聊天詢問問題
            const aiResponse = {
                message: 'Gradient descent is an optimization algorithm used to minimize the cost function...',
                suggestions: [
                    { type: 'link', content: 'Learn more about optimization algorithms' },
                    { type: 'note', content: 'Consider adding a section on backpropagation' }
                ],
                actions: [],
                metadata: { timestamp: new Date(), model: 'gpt-4' }
            };
            
            vi.spyOn(plugin.apiClient, 'chatWithAI').mockResolvedValue(aiResponse);
            
            const response = await plugin.aiManager.sendMessage('How does gradient descent work?');
            expect(response.message).toContain('Gradient descent');
            
            // 4. 學生進行語義搜尋尋找相關內容
            const semanticQuery: SearchQuery = {
                content: 'optimization algorithms machine learning',
                searchType: 'semantic'
            };
            
            const semanticResults = await plugin.searchManager.performSearch(semanticQuery);
            expect(semanticResults).toBeDefined();
            
            // 5. 學生點擊搜尋結果導航到原始位置
            if (semanticResults.items.length > 0) {
                await plugin.searchManager.navigateToResult(semanticResults.items[0]);
                expect(mockApp.workspace.openLinkText).toHaveBeenCalled();
            }
        });
    });

    /**
     * 場景 4: 團隊協作和離線工作
     */
    describe('Scenario 4: Team Collaboration and Offline Work', () => {
        it('should handle offline work and sync when online', async () => {
            // 1. 使用者在線上工作
            expect(plugin.offlineManager.isOnline()).toBe(true);
            
            const onlineNote = `# Team Meeting Notes

## Attendees
- Alice
- Bob
- Charlie

## Action Items
- [ ] Alice: Review design document
- [ ] Bob: Implement API endpoints
- [ ] Charlie: Write tests

#team #meeting #action-items`;

            const meetingFile = createMockFile('meetings/team-meeting-2024.md', onlineNote);
            vi.spyOn(mockVault, 'read').mockResolvedValue(onlineNote);
            
            await plugin.contentManager.handleContentChange(meetingFile);
            
            // 2. 網路中斷，進入離線模式
            vi.spyOn(plugin.offlineManager, 'isOnline').mockReturnValue(false);
            
            const offlineNote = `# Offline Work Notes

Working on the project while offline.

## Progress
- Completed task A
- Started task B

#offline #progress`;

            const offlineFile = createMockFile('work/offline-notes.md', offlineNote);
            vi.spyOn(mockVault, 'read').mockResolvedValue(offlineNote);
            
            // 3. 離線時的變更應該被排隊
            await plugin.contentManager.handleContentChange(offlineFile);
            
            const pendingOps = plugin.offlineManager.getPendingOperations();
            expect(pendingOps.length).toBeGreaterThan(0);
            
            // 4. 網路恢復，自動同步
            vi.spyOn(plugin.offlineManager, 'isOnline').mockReturnValue(true);
            
            await plugin.offlineManager.syncWhenOnline();
            
            // 5. 驗證同步完成
            const finalPendingOps = plugin.offlineManager.getPendingOperations();
            expect(finalPendingOps.length).toBe(0);
        });
    });

    /**
     * 場景 5: 大型文件庫的效能測試
     */
    describe('Scenario 5: Large Vault Performance', () => {
        it('should handle large vault with many files efficiently', async () => {
            const fileCount = 100;
            const files: TFile[] = [];
            
            // 1. 創建大量檔案
            for (let i = 0; i < fileCount; i++) {
                const content = `# Document ${i}

This is document number ${i} in the vault.

## Content
- Point 1 for document ${i}
- Point 2 for document ${i}

#document #batch-${Math.floor(i / 10)}`;

                const file = createMockFile(`docs/document-${i}.md`, content);
                files.push(file);
                
                vi.spyOn(mockVault, 'read').mockResolvedValue(content);
            }
            
            // 2. 批次處理所有檔案
            const startTime = performance.now();
            
            const promises = files.map(file => 
                plugin.contentManager.handleContentChange(file)
            );
            
            await Promise.all(promises);
            
            const endTime = performance.now();
            const totalTime = endTime - startTime;
            
            // 3. 驗證效能
            expect(totalTime).toBeLessThan(30000); // 30秒內完成
            expect(plugin.apiClient.batchCreateChunks).toHaveBeenCalledTimes(fileCount);
            
            // 4. 測試搜尋效能
            const searchStartTime = performance.now();
            
            const searchResult = await plugin.searchManager.performSearch({
                content: 'document',
                searchType: 'semantic'
            });
            
            const searchEndTime = performance.now();
            const searchTime = searchEndTime - searchStartTime;
            
            expect(searchTime).toBeLessThan(1000); // 1秒內完成搜尋
        });
    });

    /**
     * 場景 6: 錯誤恢復和使用者體驗
     */
    describe('Scenario 6: Error Recovery and User Experience', () => {
        it('should provide good user experience during errors', async () => {
            // 1. API 服務暫時不可用
            vi.spyOn(plugin.apiClient, 'batchCreateChunks')
                .mockRejectedValueOnce(new Error('Service unavailable'))
                .mockRejectedValueOnce(new Error('Service unavailable'))
                .mockResolvedValueOnce([]);
            
            const content = `# Test Note

This note will test error recovery.`;

            const testFile = createMockFile('test-error.md', content);
            vi.spyOn(mockVault, 'read').mockResolvedValue(content);
            
            // 2. 處理內容（應該重試並最終成功）
            await plugin.contentManager.handleContentChange(testFile);
            
            // 3. 驗證重試機制
            expect(plugin.apiClient.batchCreateChunks).toHaveBeenCalledTimes(3);
            
            // 4. 測試使用者友好的錯誤訊息
            const errorHandler = plugin.errorHandler;
            const lastError = errorHandler.getLastError();
            
            if (lastError) {
                expect(lastError.recoverable).toBe(true);
            }
            
            // 5. 測試狀態列更新
            const statusBar = plugin.statusBarManager;
            expect(statusBar.getStatus()).toBeDefined();
        });
    });

    // 輔助函數
    function setupBasicAPIMocks() {
        vi.spyOn(plugin.apiClient, 'batchCreateChunks').mockResolvedValue([]);
        vi.spyOn(plugin.apiClient, 'searchChunks').mockResolvedValue({
            items: [],
            totalCount: 0,
            searchTime: 100,
            cacheHit: false
        });
        vi.spyOn(plugin.apiClient, 'searchSemantic').mockResolvedValue({
            items: [],
            totalCount: 0,
            searchTime: 100,
            cacheHit: false
        });
        vi.spyOn(plugin.apiClient, 'chatWithAI').mockResolvedValue({
            message: 'Mock AI response',
            suggestions: [],
            actions: [],
            metadata: { timestamp: new Date(), model: 'gpt-4' }
        });
        vi.spyOn(plugin.templateManager, 'getTemplateInstances').mockResolvedValue([]);
    }

    function createMockFile(path: string, content: string): TFile {
        return {
            path,
            name: path.split('/').pop() || path,
            basename: path.split('/').pop()?.replace(/\.[^/.]+$/, '') || path,
            extension: 'md',
            stat: { ctime: Date.now(), mtime: Date.now(), size: content.length },
            vault: mockVault
        } as TFile;
    }
});