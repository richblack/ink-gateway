package clients

import (
	"context"
	"testing"

	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateSystemMock(t *testing.T) {
	client := NewMockSupabaseClient()
	ctx := context.Background()

	t.Run("CreateTemplate", func(t *testing.T) {
		templateName := "通訊錄"
		slotNames := []string{"姓名", "電話", "地址"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)
		require.NotNil(t, template)

		// Verify template chunk
		assert.Equal(t, templateName+"#template", template.Template.Content)
		assert.True(t, template.Template.IsTemplate)
		assert.False(t, template.Template.IsSlot)

		// Verify slots
		assert.Len(t, template.Slots, 3)
		for i, slot := range template.Slots {
			assert.Equal(t, "#"+slotNames[i], slot.Content)
			assert.False(t, slot.IsTemplate)
			assert.True(t, slot.IsSlot)
			assert.Equal(t, i, *slot.SequenceNumber)
		}

		// Verify no instances initially
		assert.Empty(t, template.Instances)
	})

	t.Run("GetTemplateByContent", func(t *testing.T) {
		templateName := "聯絡人"
		slotNames := []string{"名字", "職位"}

		// Create template
		createdTemplate, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Retrieve by content
		retrievedTemplate, err := client.GetTemplateByContent(ctx, templateName+"#template")
		require.NoError(t, err)
		require.NotNil(t, retrievedTemplate)

		// Verify template matches
		assert.Equal(t, createdTemplate.Template.ID, retrievedTemplate.Template.ID)
		assert.Equal(t, createdTemplate.Template.Content, retrievedTemplate.Template.Content)
		assert.Len(t, retrievedTemplate.Slots, 2)
	})

	t.Run("CreateTemplateInstance", func(t *testing.T) {
		templateName := "員工資料"
		slotNames := []string{"姓名", "部門", "職位"}

		// Create template
		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create instance
		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "張三",
			SlotValues: map[string]string{
				"姓名": "張三",
				"部門": "工程部",
				"職位": "軟體工程師",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)
		require.NotNil(t, instance)

		// Verify instance
		assert.Equal(t, "張三", instance.Instance.Content)
		assert.Equal(t, template.Template.ID, *instance.Instance.TemplateChunkID)

		// Verify slot values
		assert.Len(t, instance.SlotValues, 3)
		assert.Equal(t, "張三", instance.SlotValues["姓名"].Content)
		assert.Equal(t, "工程部", instance.SlotValues["部門"].Content)
		assert.Equal(t, "軟體工程師", instance.SlotValues["職位"].Content)
	})

	t.Run("GetTemplateInstances", func(t *testing.T) {
		templateName := "產品資訊"
		slotNames := []string{"名稱", "價格"}

		// Create template
		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create instances
		instance1Req := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "產品A",
			SlotValues: map[string]string{
				"名稱": "產品A",
				"價格": "$100",
			},
		}

		instance2Req := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "產品B",
			SlotValues: map[string]string{
				"名稱": "產品B",
				"價格": "$200",
			},
		}

		_, err = client.CreateTemplateInstance(ctx, instance1Req)
		require.NoError(t, err)

		_, err = client.CreateTemplateInstance(ctx, instance2Req)
		require.NoError(t, err)

		// Get all instances
		instances, err := client.GetTemplateInstances(ctx, template.Template.ID)
		require.NoError(t, err)
		assert.Len(t, instances, 2)
	})

	t.Run("UpdateSlotValue", func(t *testing.T) {
		templateName := "客戶資料"
		slotNames := []string{"公司名稱", "聯絡人"}

		// Create template and instance
		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "客戶甲",
			SlotValues: map[string]string{
				"公司名稱": "ABC公司",
				"聯絡人":  "李四",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Update slot value
		err = client.UpdateSlotValue(ctx, instance.Instance.ID, "聯絡人", "王五")
		require.NoError(t, err)

		// Verify update (in mock, we check the chunk directly)
		slotValueID := instance.Instance.ID + "-slot-聯絡人"
		updatedChunk := client.chunks[slotValueID]
		assert.Equal(t, "王五", updatedChunk.Content)
		assert.Equal(t, "王五", *updatedChunk.SlotValue)
	})

	t.Run("GetAllTemplates", func(t *testing.T) {
		// Create multiple templates
		template1, err := client.CreateTemplate(ctx, "模板1", []string{"欄位1", "欄位2"})
		require.NoError(t, err)

		template2, err := client.CreateTemplate(ctx, "模板2", []string{"欄位A", "欄位B", "欄位C"})
		require.NoError(t, err)

		// Get all templates
		allTemplates, err := client.GetAllTemplates(ctx)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(allTemplates), 2)

		// Find our created templates
		var found1, found2 bool
		for _, template := range allTemplates {
			if template.Template.ID == template1.Template.ID {
				found1 = true
				assert.Len(t, template.Slots, 2)
			}
			if template.Template.ID == template2.Template.ID {
				found2 = true
				assert.Len(t, template.Slots, 3)
			}
		}
		assert.True(t, found1, "Template 1 should be found in all templates")
		assert.True(t, found2, "Template 2 should be found in all templates")
	})

	t.Run("SlotMarkingSystem", func(t *testing.T) {
		// Test template creation with #slot markings in content
		templateName := "會議記錄"
		slotNames := []string{"日期", "參與者", "議題", "結論"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Verify slot content has # prefix
		for i, slot := range template.Slots {
			expectedContent := "#" + slotNames[i]
			assert.Equal(t, expectedContent, slot.Content)
			assert.True(t, slot.IsSlot)
		}

		// Create instance and verify slot structure
		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "週會",
			SlotValues: map[string]string{
				"日期":  "2024-01-15",
				"參與者": "張三, 李四, 王五",
				"議題":  "產品開發進度",
				"結論":  "按計劃進行",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Verify slot values don't have # prefix
		assert.Equal(t, "2024-01-15", instance.SlotValues["日期"].Content)
		assert.Equal(t, "張三, 李四, 王五", instance.SlotValues["參與者"].Content)
		assert.Equal(t, "產品開發進度", instance.SlotValues["議題"].Content)
		assert.Equal(t, "按計劃進行", instance.SlotValues["結論"].Content)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		// Test getting non-existent template
		_, err := client.GetTemplateByContent(ctx, "不存在的模板#template")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")

		// Test creating instance for non-existent template
		invalidReq := &models.CreateInstanceRequest{
			TemplateChunkID: "invalid-template-id",
			InstanceName:    "測試",
			SlotValues:      map[string]string{},
		}
		_, err = client.CreateTemplateInstance(ctx, invalidReq)
		assert.Error(t, err)

		// Test updating slot value for non-existent instance
		err = client.UpdateSlotValue(ctx, "invalid-instance-id", "slot", "value")
		assert.Error(t, err)
	})
}