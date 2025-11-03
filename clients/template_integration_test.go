package clients

import (
	"context"
	"testing"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateSystemIntegration(t *testing.T) {
	client := NewMockSupabaseClient()
	ctx := context.Background()

	t.Run("CompleteTemplateWorkflow", func(t *testing.T) {
		// Step 1: Create a comprehensive template
		templateName := "員工入職流程"
		slotNames := []string{
			"員工姓名",
			"部門",
			"職位",
			"入職日期",
			"直屬主管",
			"工作地點",
			"薪資等級",
			"培訓計劃",
		}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)
		require.NotNil(t, template)

		// Verify template structure
		assert.Equal(t, templateName+"#template", template.Template.Content)
		assert.True(t, template.Template.IsTemplate)
		assert.Len(t, template.Slots, 8)

		// Verify all slots have proper #slot marking
		slotMap := make(map[string]bool)
		for _, slot := range template.Slots {
			slotMap[slot.Content] = true
			assert.True(t, slot.IsSlot)
			assert.False(t, slot.IsTemplate)
		}

		for _, slotName := range slotNames {
			assert.True(t, slotMap["#"+slotName], "Slot #%s should exist", slotName)
		}

		// Step 2: Create multiple instances with different completion levels
		employees := []struct {
			name       string
			slotValues map[string]string
		}{
			{
				name: "張小明",
				slotValues: map[string]string{
					"員工姓名": "張小明",
					"部門":   "工程部",
					"職位":   "軟體工程師",
					"入職日期": "2024-02-01",
					"直屬主管": "李經理",
					"工作地點": "台北辦公室",
					"薪資等級": "P3",
					"培訓計劃": "新人訓練 + 技術培訓",
				},
			},
			{
				name: "王小華",
				slotValues: map[string]string{
					"員工姓名": "王小華",
					"部門":   "產品部",
					"職位":   "產品經理",
					"入職日期": "2024-02-15",
					"直屬主管": "陳總監",
					// 其他欄位故意留空，稍後填入
				},
			},
		}

		var instances []models.TemplateInstance
		for _, emp := range employees {
			instanceReq := &models.CreateInstanceRequest{
				TemplateChunkID: template.Template.ID,
				InstanceName:    emp.name,
				SlotValues:      emp.slotValues,
			}

			instance, err := client.CreateTemplateInstance(ctx, instanceReq)
			require.NoError(t, err)
			instances = append(instances, *instance)

			// Verify instance creation
			assert.Equal(t, emp.name, instance.Instance.Content)
			assert.Equal(t, template.Template.ID, *instance.Instance.TemplateChunkID)
		}

		// Step 3: Complete partial information for second employee
		secondEmployee := instances[1]
		updates := map[string]string{
			"工作地點": "新竹辦公室",
			"薪資等級": "M2",
			"培訓計劃": "產品管理培訓 + 領導力課程",
		}

		for slotName, value := range updates {
			err = client.UpdateSlotValue(ctx, secondEmployee.Instance.ID, slotName, value)
			require.NoError(t, err)

			// Verify update
			slotValueID := secondEmployee.Instance.ID + "-slot-" + slotName
			updatedChunk := client.chunks[slotValueID]
			assert.Equal(t, value, updatedChunk.Content)
			assert.Equal(t, value, *updatedChunk.SlotValue)
		}

		// Step 4: Verify template instances list
		allInstances, err := client.GetTemplateInstances(ctx, template.Template.ID)
		require.NoError(t, err)
		assert.Len(t, allInstances, 2)

		// Verify instances are independent
		var zhang, wang *models.TemplateInstance
		for i, instance := range allInstances {
			if instance.Instance.Content == "張小明" {
				zhang = &allInstances[i]
			} else if instance.Instance.Content == "王小華" {
				wang = &allInstances[i]
			}
		}

		require.NotNil(t, zhang, "張小明 instance should exist")
		require.NotNil(t, wang, "王小華 instance should exist")

		// Verify Zhang's data is complete and unchanged
		assert.Equal(t, "工程部", zhang.SlotValues["部門"].Content)
		assert.Equal(t, "軟體工程師", zhang.SlotValues["職位"].Content)
		assert.Equal(t, "台北辦公室", zhang.SlotValues["工作地點"].Content)

		// Verify Wang's data includes updates
		assert.Equal(t, "產品部", wang.SlotValues["部門"].Content)
		assert.Equal(t, "產品經理", wang.SlotValues["職位"].Content)
		assert.Equal(t, "新竹辦公室", wang.SlotValues["工作地點"].Content)
		assert.Equal(t, "M2", wang.SlotValues["薪資等級"].Content)

		// Step 5: Test template retrieval
		retrievedTemplate, err := client.GetTemplateByContent(ctx, templateName+"#template")
		require.NoError(t, err)
		assert.Equal(t, template.Template.ID, retrievedTemplate.Template.ID)
		assert.Len(t, retrievedTemplate.Instances, 2)
	})

	t.Run("MultipleTemplatesManagement", func(t *testing.T) {
		// Create multiple different templates
		templates := []struct {
			name  string
			slots []string
		}{
			{
				name:  "會議記錄",
				slots: []string{"會議主題", "日期", "參與者", "議程", "決議"},
			},
			{
				name:  "專案計劃",
				slots: []string{"專案名稱", "負責人", "開始日期", "結束日期", "預算"},
			},
			{
				name:  "客戶資料",
				slots: []string{"公司名稱", "聯絡人", "電話", "地址", "業務類型"},
			},
		}

		var createdTemplates []models.TemplateWithInstances
		for _, tmpl := range templates {
			template, err := client.CreateTemplate(ctx, tmpl.name, tmpl.slots)
			require.NoError(t, err)
			createdTemplates = append(createdTemplates, *template)
		}

		// Verify all templates can be retrieved
		allTemplates, err := client.GetAllTemplates(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allTemplates), 3)

		// Create instances for each template
		for i, template := range createdTemplates {
			instanceReq := &models.CreateInstanceRequest{
				TemplateChunkID: template.Template.ID,
				InstanceName:    "實例" + string(rune('A'+i)),
				SlotValues: map[string]string{
					template.Slots[0].Content[1:]: "測試值" + string(rune('1'+i)), // Remove # prefix
				},
			}

			instance, err := client.CreateTemplateInstance(ctx, instanceReq)
			require.NoError(t, err)
			assert.NotNil(t, instance)
		}

		// Verify each template has its instance
		for _, template := range createdTemplates {
			instances, err := client.GetTemplateInstances(ctx, template.Template.ID)
			require.NoError(t, err)
			assert.Len(t, instances, 1)
		}
	})

	t.Run("SlotSystemAdvancedFeatures", func(t *testing.T) {
		// Test advanced slot system features
		templateName := "產品開發計劃"
		slotNames := []string{
			"產品名稱",
			"目標市場",
			"核心功能",
			"技術架構",
			"開發階段",
			"里程碑",
			"風險評估",
			"成功指標",
		}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create instance with structured content
		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "AI助手產品",
			SlotValues: map[string]string{
				"產品名稱": "智能客服助手",
				"目標市場": "中小企業客服部門",
				"核心功能": "1. 自動回覆\n2. 情感分析\n3. 工單分類\n4. 知識庫整合",
				"技術架構": "前端: React\n後端: Node.js\nAI: OpenAI GPT\n資料庫: PostgreSQL",
				"開發階段": "第一階段: MVP開發 (3個月)\n第二階段: 功能完善 (2個月)\n第三階段: 上線部署 (1個月)",
				"里程碑":  "M1: 原型完成\nM2: Alpha測試\nM3: Beta發布\nM4: 正式上線",
				"風險評估": "技術風險: 中\n市場風險: 低\n資源風險: 中",
				"成功指標": "用戶滿意度 > 85%\n回應準確率 > 90%\n月活躍用戶 > 1000",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Verify structured content is preserved
		assert.Contains(t, instance.SlotValues["核心功能"].Content, "1. 自動回覆")
		assert.Contains(t, instance.SlotValues["核心功能"].Content, "2. 情感分析")
		assert.Contains(t, instance.SlotValues["技術架構"].Content, "前端: React")
		assert.Contains(t, instance.SlotValues["技術架構"].Content, "後端: Node.js")

		// Test updating structured content
		newMilestones := "M1: 需求分析完成\nM2: 設計文件完成\nM3: 開發完成\nM4: 測試完成\nM5: 正式發布"
		err = client.UpdateSlotValue(ctx, instance.Instance.ID, "里程碑", newMilestones)
		require.NoError(t, err)

		// Verify update
		slotValueID := instance.Instance.ID + "-slot-里程碑"
		updatedChunk := client.chunks[slotValueID]
		assert.Equal(t, newMilestones, updatedChunk.Content)
		assert.Contains(t, updatedChunk.Content, "M5: 正式發布")
	})

	t.Run("TemplateSystemErrorRecovery", func(t *testing.T) {
		// Test error handling and recovery scenarios
		
		// Create a valid template
		template, err := client.CreateTemplate(ctx, "測試模板", []string{"欄位1", "欄位2"})
		require.NoError(t, err)

		// Create a valid instance
		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "測試實例",
			SlotValues: map[string]string{
				"欄位1": "初始值1",
				"欄位2": "初始值2",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Test recovery from failed updates
		validSlotName := "欄位1"
		invalidSlotName := "不存在的欄位"

		// Valid update should succeed
		err = client.UpdateSlotValue(ctx, instance.Instance.ID, validSlotName, "更新值1")
		require.NoError(t, err)

		// Invalid update should fail but not affect valid data
		err = client.UpdateSlotValue(ctx, instance.Instance.ID, invalidSlotName, "無效值")
		assert.Error(t, err)

		// Verify valid data is still intact
		slotValueID := instance.Instance.ID + "-slot-" + validSlotName
		validChunk := client.chunks[slotValueID]
		assert.Equal(t, "更新值1", validChunk.Content)

		// Verify other slot is unchanged
		otherSlotID := instance.Instance.ID + "-slot-欄位2"
		otherChunk := client.chunks[otherSlotID]
		assert.Equal(t, "初始值2", otherChunk.Content)
	})
}