package clients

import (
	"context"
	"strings"
	"testing"

	"semantic-text-processor/config"
	"semantic-text-processor/models"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTemplateOperations(t *testing.T) {
	// Skip if no test environment
	cfg := getTestConfig()
	if cfg == nil {
		t.Skip("Skipping template tests - no test environment configured")
	}

	client := NewSupabaseClient(cfg)
	ctx := context.Background()

	// Clean up any existing test data
	cleanupTestData(t, client, ctx)

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
		assert.NotEmpty(t, template.Template.ID)

		// Verify slots
		assert.Len(t, template.Slots, 3)
		for i, slot := range template.Slots {
			assert.Equal(t, "#"+slotNames[i], slot.Content)
			assert.False(t, slot.IsTemplate)
			assert.True(t, slot.IsSlot)
			assert.Equal(t, template.Template.ID, *slot.ParentChunkID)
			assert.Equal(t, template.Template.ID, *slot.TemplateChunkID)
			assert.Equal(t, i, *slot.SequenceNumber)
		}

		// Verify no instances initially
		assert.Empty(t, template.Instances)
	})

	t.Run("GetTemplateByContent", func(t *testing.T) {
		// First create a template
		templateName := "聯絡人"
		slotNames := []string{"名字", "職位"}

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

		// Verify slots match
		for i, slot := range retrievedTemplate.Slots {
			assert.Equal(t, "#"+slotNames[i], slot.Content)
			assert.True(t, slot.IsSlot)
		}
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

	t.Run("CreateTemplateInstance", func(t *testing.T) {
		// Create a template first
		templateName := "員工資料"
		slotNames := []string{"姓名", "部門", "職位"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create an instance
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

		// Verify instance chunk
		assert.Contains(t, instance.Instance.Content, "張三")
		assert.Contains(t, instance.Instance.Content, templateName)
		assert.False(t, instance.Instance.IsTemplate)
		assert.False(t, instance.Instance.IsSlot)
		assert.Equal(t, template.Template.ID, *instance.Instance.TemplateChunkID)

		// Verify slot values
		assert.Len(t, instance.SlotValues, 3)
		assert.Equal(t, "張三", instance.SlotValues["姓名"].Content)
		assert.Equal(t, "工程部", instance.SlotValues["部門"].Content)
		assert.Equal(t, "軟體工程師", instance.SlotValues["職位"].Content)

		// Verify slot value chunks have correct relationships
		for _, slotValue := range instance.SlotValues {
			assert.Equal(t, instance.Instance.ID, *slotValue.ParentChunkID)
			assert.Equal(t, template.Template.ID, *slotValue.TemplateChunkID)
			assert.NotNil(t, slotValue.SlotValue)
		}
	})

	t.Run("GetTemplateInstances", func(t *testing.T) {
		// Create a template
		templateName := "產品資訊"
		slotNames := []string{"名稱", "價格", "描述"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create multiple instances
		instance1Req := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "產品A",
			SlotValues: map[string]string{
				"名稱": "產品A",
				"價格": "$100",
				"描述": "優質產品A",
			},
		}

		instance2Req := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "產品B",
			SlotValues: map[string]string{
				"名稱": "產品B",
				"價格": "$200",
				"描述": "高級產品B",
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

		// Verify instances
		var foundA, foundB bool
		for _, instance := range instances {
			if instance.SlotValues["名稱"].Content == "產品A" {
				foundA = true
				assert.Equal(t, "$100", instance.SlotValues["價格"].Content)
				assert.Equal(t, "優質產品A", instance.SlotValues["描述"].Content)
			}
			if instance.SlotValues["名稱"].Content == "產品B" {
				foundB = true
				assert.Equal(t, "$200", instance.SlotValues["價格"].Content)
				assert.Equal(t, "高級產品B", instance.SlotValues["描述"].Content)
			}
		}
		assert.True(t, foundA, "Instance A should be found")
		assert.True(t, foundB, "Instance B should be found")
	})

	t.Run("UpdateSlotValue", func(t *testing.T) {
		// Create a template and instance
		templateName := "客戶資料"
		slotNames := []string{"公司名稱", "聯絡人", "電話"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "客戶甲",
			SlotValues: map[string]string{
				"公司名稱": "ABC公司",
				"聯絡人":  "李四",
				"電話":   "123-456-7890",
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Update a slot value
		err = client.UpdateSlotValue(ctx, instance.Instance.ID, "電話", "098-765-4321")
		require.NoError(t, err)

		// Verify the update
		updatedInstances, err := client.GetTemplateInstances(ctx, template.Template.ID)
		require.NoError(t, err)
		require.Len(t, updatedInstances, 1)

		updatedInstance := updatedInstances[0]
		assert.Equal(t, "ABC公司", updatedInstance.SlotValues["公司名稱"].Content)
		assert.Equal(t, "李四", updatedInstance.SlotValues["聯絡人"].Content)
		assert.Equal(t, "098-765-4321", updatedInstance.SlotValues["電話"].Content)

		// Verify slot value was updated
		assert.Equal(t, "098-765-4321", *updatedInstance.SlotValues["電話"].SlotValue)
	})

	t.Run("TemplateWithSlotMarkings", func(t *testing.T) {
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

func TestTemplateSlotSystem(t *testing.T) {
	// Skip if no test environment
	cfg := getTestConfig()
	if cfg == nil {
		t.Skip("Skipping template slot system tests - no test environment configured")
	}

	client := NewSupabaseClient(cfg)
	ctx := context.Background()

	t.Run("SlotAutoStructureGeneration", func(t *testing.T) {
		// Create template with hierarchical slots
		templateName := "專案計劃"
		slotNames := []string{"專案名稱", "開始日期", "結束日期", "負責人", "預算"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Verify slot structure
		assert.Len(t, template.Slots, 5)
		
		// All slots should be children of the template
		for _, slot := range template.Slots {
			assert.Equal(t, template.Template.ID, *slot.ParentChunkID)
			assert.Equal(t, template.Template.ID, *slot.TemplateChunkID)
			assert.Equal(t, 1, slot.IndentLevel) // One level below template
			assert.True(t, slot.IsSlot)
			assert.False(t, slot.IsTemplate)
		}

		// Verify sequence numbers are assigned correctly
		for i, slot := range template.Slots {
			assert.Equal(t, i, *slot.SequenceNumber)
		}
	})

	t.Run("SlotValueInheritance", func(t *testing.T) {
		// Create template
		templateName := "任務清單"
		slotNames := []string{"任務名稱", "優先級", "截止日期"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create instance with partial slot values
		instanceReq := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "開發任務",
			SlotValues: map[string]string{
				"任務名稱": "實作API",
				"優先級":  "高",
				// 截止日期 intentionally omitted
			},
		}

		instance, err := client.CreateTemplateInstance(ctx, instanceReq)
		require.NoError(t, err)

		// Verify all slots have corresponding values (empty if not provided)
		assert.Len(t, instance.SlotValues, 3)
		assert.Equal(t, "實作API", instance.SlotValues["任務名稱"].Content)
		assert.Equal(t, "高", instance.SlotValues["優先級"].Content)
		assert.Equal(t, "", instance.SlotValues["截止日期"].Content) // Empty value

		// Verify slot value chunks have correct metadata
		for slotName, slotValue := range instance.SlotValues {
			assert.Equal(t, instance.Instance.ID, *slotValue.ParentChunkID)
			assert.Equal(t, template.Template.ID, *slotValue.TemplateChunkID)
			assert.NotNil(t, slotValue.SlotValue)
			
			// Verify slot value content matches SlotValue field
			assert.Equal(t, slotValue.Content, *slotValue.SlotValue)
			
			// Find corresponding template slot
			var foundTemplateSlot bool
			for _, templateSlot := range template.Slots {
				if strings.TrimPrefix(templateSlot.Content, "#") == slotName {
					foundTemplateSlot = true
					assert.Equal(t, templateSlot.IndentLevel, slotValue.IndentLevel)
					break
				}
			}
			assert.True(t, foundTemplateSlot, "Should find corresponding template slot for %s", slotName)
		}
	})

	t.Run("MultipleInstancesIndependence", func(t *testing.T) {
		// Create template
		templateName := "學生資料"
		slotNames := []string{"學號", "姓名", "科系"}

		template, err := client.CreateTemplate(ctx, templateName, slotNames)
		require.NoError(t, err)

		// Create multiple instances
		instance1Req := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "學生1",
			SlotValues: map[string]string{
				"學號": "A001",
				"姓名": "張小明",
				"科系": "資訊工程",
			},
		}

		instance2Req := &models.CreateInstanceRequest{
			TemplateChunkID: template.Template.ID,
			InstanceName:    "學生2",
			SlotValues: map[string]string{
				"學號": "B002",
				"姓名": "李小華",
				"科系": "電機工程",
			},
		}

		instance1, err := client.CreateTemplateInstance(ctx, instance1Req)
		require.NoError(t, err)

		_, err = client.CreateTemplateInstance(ctx, instance2Req)
		require.NoError(t, err)

		// Update slot value in instance1
		err = client.UpdateSlotValue(ctx, instance1.Instance.ID, "科系", "軟體工程")
		require.NoError(t, err)

		// Verify instance2 is not affected
		instances, err := client.GetTemplateInstances(ctx, template.Template.ID)
		require.NoError(t, err)
		assert.Len(t, instances, 2)

		for _, instance := range instances {
			if instance.SlotValues["學號"].Content == "A001" {
				// Instance1 should have updated value
				assert.Equal(t, "軟體工程", instance.SlotValues["科系"].Content)
			} else if instance.SlotValues["學號"].Content == "B002" {
				// Instance2 should have original value
				assert.Equal(t, "電機工程", instance.SlotValues["科系"].Content)
			}
		}
	})
}

// Helper function to clean up test data
func cleanupTestData(t *testing.T, client SupabaseClient, ctx context.Context) {
	// This is a simplified cleanup - in a real test environment,
	// you might want to clean up specific test data or use test transactions
	// For now, we'll rely on the test database being reset between test runs
}

// Helper function to get test configuration
func getTestConfig() *config.SupabaseConfig {
	// This should return test configuration
	// In a real implementation, you'd read from test environment variables
	return &config.SupabaseConfig{
		URL:    "http://localhost:54321", // Local Supabase instance
		APIKey: "test-api-key",
	}
}