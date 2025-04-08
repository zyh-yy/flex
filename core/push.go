package core

import (
	"fmt"
)

// DeriveTasksFromScene 根据场景配置推导出所有需要的任务
func DeriveTasksFromScene(sceneName string) (map[string]*Node, error) {
	sceneLock.Lock()
	defer sceneLock.Unlock()

	// 获取场景配置
	resultFields, exists := sceneConfigMap[sceneName]
	if !exists {
		return nil, fmt.Errorf("场景 %s 不存在", sceneName)
	}

	registryLock.Lock()
	defer registryLock.Unlock()

	// 用于存储推导出的任务
	taskSet := make(map[string]bool)
	// 用于存储待解析的字段队列
	fieldQueue := make([]string, len(resultFields))
	copy(fieldQueue, resultFields)

	// 递归推导任务
	for len(fieldQueue) > 0 {
		field := fieldQueue[0]
		fieldQueue = fieldQueue[1:]

		// 找到产出该字段的任务
		writer, exists := fieldWriteMap[field]
		if !exists {
			return nil, fmt.Errorf("字段 %s 没有被任何任务产出", field)
		}

		// 如果任务未被加入集合，则加入并解析其依赖字段
		if !taskSet[writer] {
			taskSet[writer] = true
			task := taskRegistry[writer]

			// 将任务的依赖字段加入队列
			fieldQueue = append(fieldQueue, task.DepFields...)
		}
	}

	// 构造 Node 映射表
	nodeMap := make(map[string]*Node)
	for taskName := range taskSet {
		task := taskRegistry[taskName]
		nodeMap[taskName] = &Node{
			TaskName: taskName,
			DepTask:  []string{}, // 初始为空，后续填充
			DepField: task.DepFields,
			task:     *task,
		}
	}

	// 填充 DepTask
	for _, node := range nodeMap {
		for _, depField := range node.DepField {
			if writer, exists := fieldWriteMap[depField]; exists {
				node.DepTask = append(node.DepTask, writer)
			}
		}
	}

	return nodeMap, nil
}

// BuildSceneEngine 根据场景配置构造 SceneEngine
func BuildSceneEngine(sceneName string) (*SceneEngine, error) {
	// 推导任务
	nodeMap, err := DeriveTasksFromScene(sceneName)
	if err != nil {
		return nil, err
	}

	// 获取场景配置中的输出字段
	sceneLock.Lock()
	defer sceneLock.Unlock()

	resultFields, exists := sceneConfigMap[sceneName]
	if !exists {
		return nil, fmt.Errorf("场景 %s 不存在", sceneName)
	}

	// 构造 SceneEngine
	engine := NewSceneEngine(resultFields)
	engine.Updatetask(nodeMap)

	return engine, nil
}
