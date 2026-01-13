package description

import (
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

// DescriptionConfig 描述配置结构
type DescriptionConfig struct {
	Mappings           map[string]string `yaml:"mappings"`
	Notes              []string          `yaml:"notes"`
	DefaultExpiryDays  int               `yaml:"default_expiry_days"`
}

// Mapper 描述映射器
type Mapper struct {
	mappings map[string]string
}

// NewMapper 创建描述映射器
func NewMapper() *Mapper {
	return &Mapper{
		mappings: make(map[string]string),
	}
}

// LoadMappings 加载描述映射配置
func (m *Mapper) LoadMappings(filename string) error {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var config DescriptionConfig
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return err
	}

	m.mappings = config.Mappings
	return nil
}

// GetDescription 获取描述信息
func (m *Mapper) GetDescription(globalAddress string) string {
	if desc, exists := m.mappings[globalAddress]; exists {
		return desc
	}
	
	// 如果没有找到映射，返回默认描述
	return "未知服务-" + globalAddress
}