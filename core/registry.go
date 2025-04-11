package core

import (
	"context"
	"encoding/json"
	"flex/infra"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
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
	Name         string       // 任务名称
	DepFields    []string     // 依赖字段
	OutputFields []string     // 产出字段
	RunnableNode RunnableNode // 任务执行函数
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
	RunTaskMap     = make(map[string]RunnableNode)
	fieldWriteMap  = make(map[string]string)   // 记录字段被哪个任务写入
	sceneConfigMap = make(map[string][]string) // 场景与结果字段的映射
	registryLock   sync.Mutex
	fieldWriteLock sync.Mutex
	sceneLock      sync.Mutex
)

type TaskInfo struct {
	InputFields []FieldInfo `json:"input_fields"`
	OutputField []FieldInfo `json:"output_field"`
	TaskID      int         `json:"task_id"`
	TaskName    string      `json:"task_name"`
	Description string      `json:"description"`
}

type FieldInfo struct {
	FieldID     int    `json:"field_id"`
	FieldName   string `json:"field_name"`
	Description string `json:"description"`
}

func RegisterFieldsAndTasks(client *redis.Client, ctx context.Context, fieldID int) error {
	var globalConfig = make(map[int]TaskInfo)
	visitedFields := make(map[int]bool)
	queue := []int{fieldID}

	for len(queue) > 0 {
		currentFieldID := queue[0]
		queue = queue[1:]

		if visitedFields[currentFieldID] {
			continue
		}
		visitedFields[currentFieldID] = true

		// 获取field_id对应的task信息
		taskKey := fmt.Sprintf("field:%d:output_task", currentFieldID)
		taskInfoStr, err := client.Get(ctx, taskKey).Result()
		if err != nil {
			if err == redis.Nil {
				continue // 没有对应的task，跳过
			}
			return err
		}

		var taskInfo TaskInfo

		err = json.Unmarshal([]byte(taskInfoStr), &taskInfo)
		if err != nil {
			return err
		}

		// 新增：读取field:%d:info获取fieldname
		fieldInfoKey := fmt.Sprintf("field:%d:info", currentFieldID)
		fieldInfoStr, err := client.Get(ctx, fieldInfoKey).Result()
		if err != nil && err != redis.Nil {
			return err
		}

		var fieldInfo FieldInfo
		err = json.Unmarshal([]byte(fieldInfoStr), &fieldInfo)
		if err != nil {
			return err
		}

		if tempTaskInfo, ok := globalConfig[taskInfo.TaskID]; ok {
			taskInfo.OutputField = append(tempTaskInfo.OutputField, fieldInfo)
			continue
		}

		taskID := taskInfo.TaskID
		taskInfoKey := fmt.Sprintf("task:%d:input_fields", taskID)

		// 获取task_id对应的依赖字段信息
		inputFields, err := client.SMembers(ctx, taskInfoKey).Result()
		if err != nil {
			return err
		}

		for _, fieldInfoStr := range inputFields {
			var fieldInfo FieldInfo
			err = json.Unmarshal([]byte(fieldInfoStr), &fieldInfo)
			if err != nil {
				return err
			}

			// 将新的字段加入队列
			queue = append(queue, fieldID)

			// 将字段信息添加到taskInfo中
			taskInfo.InputFields = append(taskInfo.InputFields, fieldInfo)
		}

		if tempTaskInfo, ok := globalConfig[taskInfo.TaskID]; ok {
			taskInfo.OutputField = append(tempTaskInfo.OutputField, fieldInfo)
		}
		// 注册任务信息到全局配置
		globalConfig[taskID] = taskInfo
	}

	for _, taskInfo := range globalConfig {
		// 将taskInfo.InputFields, taskInfo.OutputField提取字段名转为map[]string
		fieldInupts := make([]string, 0)
		for _, fieldInfo := range taskInfo.InputFields {
			fieldInupts = append(fieldInupts, fieldInfo.FieldName)
		}

		fieldOutputs := make([]string, 0)
		for _, fieldInfo := range taskInfo.OutputField {
			fieldOutputs = append(fieldOutputs, fieldInfo.FieldName)
		}

		RegisterTask(taskInfo.TaskName, fieldInupts, fieldOutputs)

	}

	return nil
}

func parseInt(s string) int {
	// 这里假设有一个方法将字符串转换为整数
	// 实际使用时请根据实际情况实现
	i, _ := strconv.Atoi(s)
	return i
}

func Init() {
	go func() {
		rdb := redis.NewClient(&redis.Options{
			Addr:     "localhost:6379", // Redis 服务器地址
			Password: "123456",         // 密码
			DB:       0,                // 默认数据库
		})

		// 检查Redis连接是否成功
		pong, err := rdb.Ping(context.Background()).Result()
		if err != nil {
			fmt.Println("Failed to connect to Redis:", err)
			return
		}
		fmt.Println("Connected to Redis:", pong)

		ticker := time.NewTicker(5 * time.Second) // 每5秒轮询一次
		defer ticker.Stop()

		for range ticker.C {
			ctx := context.Background()
			keys, err := rdb.Keys(ctx, "scene:*:config").Result()
			if err != nil {
				fmt.Println("Failed to get keys from Redis:", err)
				continue
			}

			for _, key := range keys {
				val, err := rdb.Get(ctx, key).Result()
				if err != nil {
					fmt.Println("Failed to get value from Redis:", err)
					continue
				}

				var sceneConfig infra.SceneConfig
				err = json.Unmarshal([]byte(val), &sceneConfig)
				if err != nil {
					fmt.Println("Failed to unmarshal scene config:", err)
					continue
				}

				sceneName := fmt.Sprintf("Scene%d", sceneConfig.SceneID)
				sceneLock.Lock()
				sceneConfigMap[sceneName] = make([]string, len(sceneConfig.Fields))
				for i, field := range sceneConfig.Fields {
					sceneConfigMap[sceneName][i] = field.Name
				}
				sceneLock.Unlock()

				// 在场景配置加载完成后调用RegisterFieldsAndTasks
				for _, field := range sceneConfig.Fields {
					fieldID := field.FieldID
					err := RegisterFieldsAndTasks(rdb, ctx, fieldID)
					if err != nil {
						fmt.Println("Failed to register fields and tasks:", err)
					}
				}
			}
		}
	}()
}

// RegisterTask 注册任务
func RegisterTask(name string, depFields []string, outputFields []string) {
	registryLock.Lock()
	defer registryLock.Unlock()

	// 检查任务是否已存在
	if _, exists := taskRegistry[name]; exists {
		panic("任务已存在: " + name)
	}

	// 检测字段写入冲突
	fieldWriteLock.Lock()
	defer fieldWriteLock.Unlock()
	for _, field := range outputFields {
		// 跳过对外部字段的冲突检测
		if writer, exists := fieldWriteMap[field]; exists {
			panic("字段冲突: 字段 " + field + " 已被任务 " + writer + " 写入")
		}
		fieldWriteMap[field] = name
	}

	// 注册任务
	taskRegistry[name] = &Task{
		Name:         name,
		DepFields:    depFields,
		OutputFields: outputFields,
		RunnableNode: RunTaskMap[name],
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
