package clients

import (
	"context"
	"testing"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateInstantiationFeatures(t *testing.T) {
	client := NewMockSupabaseClient()
	ctx := context.Background()

	t.Run("PartialSlotValueFilling", func(t *testing.T) {
		// Create template with multiple slots
		templateName := "專案計劃"
		slotNames := []string{"專案名稱", "開始日期", "結束日期", "負責人", "預算"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create instance with only some slot values filled
		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "新專案",
			SlotValues: map[string]string{
				"專案名稱": "AI 開發專案",
				"負責人":  "張工程師",
				// 其他欄位故意留空
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Verify all slots have corresponding values (empty if not provided)
		assert.Len(t, instance.SlotValues, 5) // All slots should be created
		assert.Equal(t, "AI 開發專案", instance.SlotValues["專案名稱"].Content)
		assert.Equal(t, "張工程師", instance.SlotValues["負責人"].Content)
		assert.Equal(t, "", instance.SlotValues["開始日期"].Content) // Empty initially
		assert.Equal(t, "", instance.SlotValues["結束日期"].Content) // Empty initially
		assert.Equal(t, "", instance.SlotValues["預算"].Content)   // Empty initially

		// Update empty slots later
		err = client.UpdateSlotValue(ctx, instance.Instance.ID, "開始日期", "2024-02-01")
		require.NoError(t, err)

		err = client.UpdateSlotValue(ctx, instance.Instance.ID, "結束日期", "2024-06-30")
		require.NoError(t, err)

		err = client.UpdateSlotValue(ctx, instance.Instance.ID, "預算", "$50,000")
		require.NoError(t, err)

		// Verify updates
		slotValueID := instance.Instance.ID + "-slot-開始日期"
		updatedChunk := client.chunks[slotValueID]
		assert.Equal(t, "2024-02-01", updatedChunk.Content)
	})

	t.Run("SlotValueInheritanceAndPropagation", func(t *testing.T) {
		// Create hierarchical template structure
		templateName := "任務清單"
		slotNames := []string{"任務名稱", "優先級", "截止日期", "負責人"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create multiple instances to test independence
		instance1Req := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "開發任務",
			SlotValues: map[string]string{
				"任務名稱": "實作 API",
				"優先級":  "高",
				"截止日期": "2024-01-31",
				"負責人":  "開發團隊",
			},
		}

		instance2Req := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "測試任務",
			SlotValues: map[string]string{
				"任務名稱": "撰寫測試",
				"優先級":  "中",
				"截止日期": "2024-02-15",
				"負責人":  "QA 團隊",
			},
		}

		instance1, err := client.CreateTemplateInstance(ctx, instance1Req)
		require.NoError(t, err)

		instance2, err := client.CreateTemplateInstance(ctx, instance2Req)
		require.NoError(t, err)

		// Verify instances are independent
		assert.NotEqual(t, instance1.Instance.ID, instance2.Instance.ID)
		assert.Equal(t, "實作 API", instance1.SlotValues["任務名稱"].Content)
		assert.Equal(t, "撰寫測試", instance2.SlotValues["任務名稱"].Content)

		// Update one instance and verify the other is not affected
		err = client.UpdateSlotValue(ctx, instance1.Instance.ID, "優先級", "緊急")
		require.NoError(t, err)

		// Verify instance1 is updated
		slotValueID1 := instance1.Instance.ID + "-slot-優先級"
		updatedChunk1 := client.chunks[slotValueID1]
		assert.Equal(t, "緊急", updatedChunk1.Content)

		// Verify instance2 is unchanged
		slotValueID2 := instance2.Instance.ID + "-slot-優先級"
		originalChunk2 := client.chunks[slotValueID2]
		assert.Equal(t, "中", originalChunk2.Content)
	})

	t.Run("DynamicSlotStructureGeneration", func(t *testing.T) {
		// Test automatic structure generation based on slot definitions
		templateName := "會議記錄"
		slotNames := []string{"會議主題", "日期時間", "參與者", "議程", "決議事項", "後續行動"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Verify template structure
		assert.Equal(t, templateName+"#template", template.Template.Content)
		assert.Len(t, template.Slots, 6)

		// Verify each slot has proper #slot marking
		expectedSlots := map[string]bool{
			"#會議主題": false,
			"#日期時間": false,
			"#參與者":  false,
			"#議程":   false,
			"#決議事項": false,
			"#後續行動": false,
		}

		for _, slot := range template.Slots {
			if _, exists := expectedSlots[slot.Content]; exists {
				expectedSlots[slot.Content] = true
				assert.True(t, slot.IsSlot)
				assert.False(t, slot.IsTemplate)
				assert.Equal(t, template.Template.ID, *slot.ParentChunkID)
				assert.Equal(t, template.Template.ID, *slot.TemplateChunkID)
			}
		}

		// Verify all expected slots were found
		for slotContent, found := range expectedSlots {
			assert.True(t, found, "Slot %s should be found", slotContent)
		}

		// Create instance with structured content
		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "週會記錄",
			SlotValues: map[string]string{
				"會議主題": "產品開發進度檢討",
				"日期時間": "2024-01-15 14:00-15:00",
				"參與者":  "產品經理、開發團隊、設計師",
				"議程":   "1. 進度報告\n2. 問題討論\n3. 下週計劃",
				"決議事項": "加速 API 開發，調整 UI 設計",
				"後續行動": "開發團隊本週完成 API，設計師提供新版 mockup",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Verify instance structure
		assert.Equal(t, "週會記錄", instance.Instance.Content)
		assert.Len(t, instance.SlotValues, 6)

		// Verify slot values maintain structure
		assert.Contains(t, instance.SlotValues["議程"].Content, "1. 進度報告")
		assert.Contains(t, instance.SlotValues["議程"].Content, "2. 問題討論")
		assert.Contains(t, instance.SlotValues["參與者"].Content, "產品經理")
	})

	t.Run("SlotValueValidationAndUpdate", func(t *testing.T) {
		// Create template for testing validation
		templateName := "用戶資料"
		slotNames := []string{"用戶名", "電子郵件", "電話", "地址"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create instance
		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "用戶001",
			SlotValues: map[string]string{
				"用戶名":  "張小明",
				"電子郵件": "zhang@example.com",
				"電話":   "123-456-7890",
				"地址":   "台北市信義區",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Test updating individual slot values
		testUpdates := map[string]string{
			"電子郵件": "zhang.xiaoming@newcompany.com",
			"電話":   "098-765-4321",
			"地址":   "新北市板橋區新站路123號",
		}

		for slotName, newValue := range testUpdates {
			err = client.UpdateSlotValue(ctx, instance.Instance.ID, slotName, newValue)
			require.NoError(t, err)

			// Verify update
			slotValueID := instance.Instance.ID + "-slot-" + slotName
			updatedChunk := client.chunks[slotValueID]
			assert.Equal(t, newValue, updatedChunk.Content)
			assert.Equal(t, newValue, *updatedChunk.SlotValue)
		}

		// Verify other slots remain unchanged
		unchangedSlotID := instance.Instance.ID + "-slot-用戶名"
		unchangedChunk := client.chunks[unchangedSlotID]
		assert.Equal(t, "張小明", unchangedChunk.Content)
	})

	t.Run("ComplexTemplateInstanceWorkflow", func(t *testing.T) {
		// Test complete workflow from template creation to instance management
		templateName := "產品規格"
		slotNames := []string{"產品名稱", "版本", "功能描述", "技術需求", "發布日期", "負責團隊"}

		// Step 1: Create template
		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Step 2: Create multiple instances
		products := []map[string]string{
			{
				"產品名稱": "智能助手 App",
				"版本":   "v1.0",
				"功能描述": "語音識別、自然語言處理、任務自動化",
				"技術需求": "React Native, Node.js, AI/ML APIs",
				"發布日期": "2024-03-15",
				"負責團隊": "移動開發團隊",
			},
			{
				"產品名稱": "數據分析平台",
				"版本":   "v2.1",
				"功能描述": "實時數據處理、視覺化報表、預測分析",
				"技術需求": "Python, Apache Spark, React",
				"發布日期": "2024-04-30",
				"負責團隊": "數據工程團隊",
			},
		}

		var instances []models.TemplateInstance
		for _, productData := range products {
			instanceReq := &models.CreateInstanceRequest{
				TemplateChunkID: template.Template.ID,
				InstanceName:    productData["產品名稱"],
				SlotValues:      productData,
			}

			instance, err := client.CreateTemplateInstance(ctx, instanceReq)
			require.NoError(t, err)
			instances = append(instances, *instance)

			// Verify instance creation
			assert.Equal(t, productData["產品名稱"], instance.Instance.Content)
			assert.Len(t, instance.SlotValues, len(productData))
		}

		// Step 3: Update instances
		// Update first product's release date
		err = client.UpdateSlotValue(ctx, instances[0].Instance.ID, "發布日期", "2024-03-01")
		require.NoError(t, err)

		// Update second product's version
		err = client.UpdateSlotValue(ctx, instances[1].Instance.ID, "版本", "v2.2")
		require.NoError(t, err)

		// Step 4: Verify updates
		slotValueID1 := instances[0].Instance.ID + "-slot-發布日期"
		updatedChunk1 := client.chunks[slotValueID1]
		assert.Equal(t, "2024-03-01", updatedChunk1.Content)

		slotValueID2 := instances[1].Instance.ID + "-slot-版本"
		updatedChunk2 := client.chunks[slotValueID2]
		assert.Equal(t, "v2.2", updatedChunk2.Content)

		// Step 5: Verify template instances list
		allInstances, err := client.GetTemplateInstances(ctx, template.Template.ID)
		require.NoError(t, err)
		assert.Len(t, allInstances, 2)
	})

	t.Run("ErrorHandlingInInstantiation", func(t *testing.T) {
		// Test error handling in template instantiation
		
		// Test creating instance with invalid template ID
		invalidReq := &models.CreateInstanceRequest{
			TemplateChunkID: "non-existent-template",
			InstanceName:    "測試實例",
			SlotValues:      map[string]string{"test": "value"},
		}

		_, err := client.CreateTemplateInstance(ctx, invalidReq)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test updating slot value with invalid instance ID
		err = client.UpdateSlotValue(ctx, "invalid-instance-id", "slot", "value")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Create valid template and instance for further error testing
		template, err := client.CreateTemplate(ctx, "測試模板", []string{"欄位1", "欄位2"})
		require.NoError(t, err)

		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "測試實例",
			SlotValues: map[string]string{
				"欄位1": "值1",
				"欄位2": "值2",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Test updating non-existent slot
		err = client.UpdateSlotValue(ctx, instance.Instance.ID, "不存在的欄位", "新值")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}