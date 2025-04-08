package infra

type SceneField struct {
	Scene        int           `json:"scene"`
	OutputFields []OutputField `json:"output_fields"`
}

type OutputField struct {
	FieldID string    `json:"field_id"`
	Name    string    `json:"name"`
	Type    FieldType `json:"type"` 
}

// 枚举类型约束字段类型
type FieldType string

const (
	SaleCount     FieldType = "SaleCount"
	Price         FieldType = "Price"
	DiscountPrice FieldType = "DiscountPrice"
)


type TaskConfig struct {
	TaskName      string   `json:"task"`
	OutputFields  []string `json:"output_fields"`
	InputFields   []string `json:"input_fields"`
	ContextFields []string `json:"context_fields"`
}

// 完整数据结构对应顶层数组
type TaskConfigList []TaskConfig






