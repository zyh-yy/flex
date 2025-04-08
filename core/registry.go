package core

import (
	"sync"
)

// FieldType 定义字段类型
type FieldType string

const (
	RequestField      FieldType = "request"      // 请求字段
	IntermediateField FieldType = "intermediate" // 中间产物字段
	ResultField       FieldType = "result"       // 结果字段
	ExternalField     FieldType = "external"     // 外部输入字段
)

// Task 定义任务结构
type Task struct {
	Name         string                                                                               // 任务名称
	DepFields    []string                                                                             // 依赖字段
	OutputFields map[FieldType][]string                                                               // 产出字段
	RunFunc      func(map[string]interface{}, map[string]interface{}) (map[string]interface{}, error) // 任务执行函数
}

type Node struct {
	TaskName string
	DepTask  []string
	DepField []string
	task     Task
}

type RunnableNode interface {
	Run(context map[string]interface{}, inputs map[string]interface{}) (map[string]interface{}, error)
	DepTask() []string
}

// 全局任务注册表
var (
	taskRegistry   = make(map[string]*Task)
	fieldWriteMap  = make(map[string]string)   // 记录字段被哪个任务写入
	sceneConfigMap = make(map[string][]string) // 场景与结果字段的映射
	registryLock   sync.Mutex
	fieldWriteLock sync.Mutex
	sceneLock      sync.Mutex
)

// RegisterTask 注册任务
func RegisterTask(name string, depFields []string, outputFields map[FieldType][]string, runFunc func(map[string]interface{}, map[string]interface{}) (map[string]interface{}, error)) {
	registryLock.Lock()
	defer registryLock.Unlock()

	// 检查任务是否已存在
	if _, exists := taskRegistry[name]; exists {
		panic("任务已存在: " + name)
	}

	// 检测字段写入冲突
	fieldWriteLock.Lock()
	defer fieldWriteLock.Unlock()
	for fieldType, fields := range outputFields {
		for _, field := range fields {
			// 跳过对外部字段的冲突检测
			if fieldType == ExternalField {
				continue
			}
			if writer, exists := fieldWriteMap[field]; exists {
				panic("字段冲突: 字段 " + field + " 已被任务 " + writer + " 写入")
			}
			fieldWriteMap[field] = name
		}
	}

	// 注册任务
	taskRegistry[name] = &Task{
		Name:         name,
		DepFields:    depFields,
		OutputFields: outputFields,
		RunFunc:      runFunc,
	}
}

// ConfigureScene 配置场景所需的结果字段
func ConfigureScene(sceneName string, resultFields []string) {
	sceneLock.Lock()
	defer sceneLock.Unlock()

	sceneConfigMap[sceneName] = resultFields
}

// GetSceneConfig 获取场景的配置
func GetSceneConfig(sceneName string) ([]string, bool) {
	sceneLock.Lock()
	defer sceneLock.Unlock()

	fields, exists := sceneConfigMap[sceneName]
	return fields, exists
}

// GetAllSceneConfigs 获取所有场景的配置
func GetAllSceneConfigs() map[string][]string {
	sceneLock.Lock()
	defer sceneLock.Unlock()

	// 返回副本以避免外部修改
	configs := make(map[string][]string, len(sceneConfigMap))
	for k, v := range sceneConfigMap {
		configs[k] = v
	}
	return configs
}

// ClearSceneConfigs 清空场景配置（用于测试或重置）
func ClearSceneConfigs() {
	sceneLock.Lock()
	defer sceneLock.Unlock()

	sceneConfigMap = make(map[string][]string)
}
