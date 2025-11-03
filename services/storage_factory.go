package services

import (
	"fmt"

	"semantic-text-processor/models"
)

// storageAdapterFactory 儲存適配器工廠實作
type storageAdapterFactory struct {
	adapters map[models.StorageType]func(config map[string]interface{}) (MediaStorageAdapter, error)
}

// NewStorageAdapterFactory 建立新的儲存適配器工廠
func NewStorageAdapterFactory() StorageAdapterFactory {
	factory := &storageAdapterFactory{
		adapters: make(map[models.StorageType]func(config map[string]interface{}) (MediaStorageAdapter, error)),
	}
	
	// 註冊支援的儲存適配器
	factory.registerAdapters()
	
	return factory
}

// registerAdapters 註冊所有支援的儲存適配器
func (f *storageAdapterFactory) registerAdapters() {
	// 註冊本地儲存適配器
	f.adapters[models.StorageTypeLocal] = func(config map[string]interface{}) (MediaStorageAdapter, error) {
		basePath, ok := config["base_path"].(string)
		if !ok {
			return nil, fmt.Errorf("local storage requires 'base_path' config")
		}
		
		baseURL, ok := config["base_url"].(string)
		if !ok {
			return nil, fmt.Errorf("local storage requires 'base_url' config")
		}
		
		return NewLocalStorageAdapter(basePath, baseURL), nil
	}
	
	// 註冊 Supabase 儲存適配器
	f.adapters[models.StorageTypeSupabase] = func(config map[string]interface{}) (MediaStorageAdapter, error) {
		url, ok := config["url"].(string)
		if !ok {
			return nil, fmt.Errorf("supabase storage requires 'url' config")
		}
		
		apiKey, ok := config["api_key"].(string)
		if !ok {
			return nil, fmt.Errorf("supabase storage requires 'api_key' config")
		}
		
		bucket, ok := config["bucket"].(string)
		if !ok {
			return nil, fmt.Errorf("supabase storage requires 'bucket' config")
		}
		
		return NewSupabaseStorageAdapter(url, apiKey, bucket), nil
	}
	
	// 未來可以新增更多適配器
	// f.adapters[models.StorageTypeGoogleDrive] = func(config map[string]interface{}) (MediaStorageAdapter, error) { ... }
	// f.adapters[models.StorageTypeNAS] = func(config map[string]interface{}) (MediaStorageAdapter, error) { ... }
}

// CreateAdapter 根據類型和設定建立儲存適配器
func (f *storageAdapterFactory) CreateAdapter(storageType models.StorageType, config map[string]interface{}) (MediaStorageAdapter, error) {
	creator, exists := f.adapters[storageType]
	if !exists {
		return nil, fmt.Errorf("unsupported storage type: %s", storageType)
	}
	
	return creator(config)
}

// GetSupportedTypes 取得支援的儲存類型列表
func (f *storageAdapterFactory) GetSupportedTypes() []models.StorageType {
	types := make([]models.StorageType, 0, len(f.adapters))
	for storageType := range f.adapters {
		types = append(types, storageType)
	}
	return types
}

// StorageConfig 儲存設定結構
type StorageConfig struct {
	Primary  models.StorageType            `json:"primary" yaml:"primary"`
	Fallback models.StorageType            `json:"fallback,omitempty" yaml:"fallback,omitempty"`
	Configs  map[string]map[string]interface{} `json:"configs" yaml:"configs"`
}

// StorageManager 儲存管理器
type StorageManager struct {
	factory         StorageAdapterFactory
	primaryAdapter  MediaStorageAdapter
	fallbackAdapter MediaStorageAdapter
	config          *StorageConfig
}

// NewStorageManager 建立新的儲存管理器
func NewStorageManager(config *StorageConfig) (*StorageManager, error) {
	factory := NewStorageAdapterFactory()
	
	// 建立主要儲存適配器
	primaryConfig, exists := config.Configs[string(config.Primary)]
	if !exists {
		return nil, fmt.Errorf("missing config for primary storage type: %s", config.Primary)
	}
	
	primaryAdapter, err := factory.CreateAdapter(config.Primary, primaryConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create primary adapter: %w", err)
	}
	
	manager := &StorageManager{
		factory:        factory,
		primaryAdapter: primaryAdapter,
		config:         config,
	}
	
	// 建立備用儲存適配器（如果設定了）
	if config.Fallback != "" {
		fallbackConfig, exists := config.Configs[string(config.Fallback)]
		if exists {
			fallbackAdapter, err := factory.CreateAdapter(config.Fallback, fallbackConfig)
			if err == nil {
				manager.fallbackAdapter = fallbackAdapter
			}
			// 備用適配器建立失敗不影響主要功能，只記錄錯誤
		}
	}
	
	return manager, nil
}

// GetPrimaryAdapter 取得主要儲存適配器
func (m *StorageManager) GetPrimaryAdapter() MediaStorageAdapter {
	return m.primaryAdapter
}

// GetFallbackAdapter 取得備用儲存適配器
func (m *StorageManager) GetFallbackAdapter() MediaStorageAdapter {
	return m.fallbackAdapter
}

// GetConfig 取得儲存設定
func (m *StorageManager) GetConfig() *StorageConfig {
	return m.config
}

// SwitchPrimary 切換主要儲存類型
func (m *StorageManager) SwitchPrimary(newPrimary models.StorageType) error {
	newConfig, exists := m.config.Configs[string(newPrimary)]
	if !exists {
		return fmt.Errorf("missing config for storage type: %s", newPrimary)
	}
	
	newAdapter, err := m.factory.CreateAdapter(newPrimary, newConfig)
	if err != nil {
		return fmt.Errorf("failed to create new primary adapter: %w", err)
	}
	
	// 將舊的主要適配器設為備用
	m.fallbackAdapter = m.primaryAdapter
	m.primaryAdapter = newAdapter
	m.config.Fallback = m.config.Primary
	m.config.Primary = newPrimary
	
	return nil
}