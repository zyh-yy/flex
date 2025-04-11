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

// 新增 SceneConfig 和 FieldInfo 结构体
type SceneConfig struct {
	SceneID int         `json:"scene_id"`
	Fields  []FieldInfo `json:"fields"`
}

type FieldInfo struct {
	FieldID     int    `json:"field_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Responsible string `json:"responsible"`
	CreateTime  string `json:"create_time"`
}
