package core

import (
	"flex/types/assemble"
	"fmt"
	"sync"
)

type SceneEngine struct {
	task         map[string]*Node
	databus      map[int64]*assemble.DataBus
	taskMap      map[string]int // 存储每个任务的入度
	queue        []*Node        // 初始无依赖任务队列
	isPrepared   bool           // 标记是否已经完成前置准备
	lock         sync.Mutex     // 保护并发访问
	outputFields []string       // 记录需要的输出字段的键
}

func NewSceneEngine(outputFields []string) *SceneEngine {
	return &SceneEngine{
		task:         make(map[string]*Node),
		databus:      make(map[int64]*assemble.DataBus),
		taskMap:      make(map[string]int),
		queue:        []*Node{},
		isPrepared:   false,
		outputFields: outputFields,
	}
}

// GetDerivedTasks 返回当前推导出的任务名称列表
func (s *SceneEngine) GetDerivedTasks() []string {
	s.lock.Lock()
	defer s.lock.Unlock()

	var taskNames []string
	for name, _ := range s.task {
		taskNames = append(taskNames, name)
	}
	return taskNames
}

// GetDataBusValues 获取已经配置的输出字段的值，返回值是一个map【输出字段key】结果
func (s *SceneEngine) GetDataBusValues() map[string]interface{} {
	s.lock.Lock()
	defer s.lock.Unlock()

	outputValues := make(map[string]interface{})

	for _, key := range s.outputFields {
		for _, dataBus := range s.databus {
			value := dataBus.GetVal(key)
			if value != nil {
				outputValues[key] = value
				break // 找到一个值后跳出内层循环
			}
		}
	}

	return outputValues
}

// prepareInternal 内部方法用于生成依赖关系图、入度表，并检测循环依赖
func (s *SceneEngine) prepareInternal() {
	s.taskMap = make(map[string]int) // 存储每个任务的入度
	s.queue = []*Node{}              // 初始无依赖任务队列

	// 初始化入度表
	for name, node := range s.task {
		s.taskMap[name] = len(node.DepTask)
		if len(node.DepTask) == 0 {
			s.queue = append(s.queue, node)
		}
	}

	// 拓扑排序检测循环依赖
	processedCount := 0
	tempQueue := make([]*Node, len(s.queue))
	copy(tempQueue, s.queue)

	for len(tempQueue) > 0 {
		current := tempQueue[0]
		tempQueue = tempQueue[1:]
		processedCount++

		for name, node := range s.task {
			for _, dep := range node.DepTask {
				if dep == current.TaskName {
					s.taskMap[name]--
					if s.taskMap[name] == 0 {
						tempQueue = append(tempQueue, node)
					}
				}
			}
		}
	}

	// 如果处理的节点数小于 task 中的总节点数，说明存在循环依赖
	if processedCount < len(s.task) {
		panic("存在循环依赖")
	}

	s.isPrepared = true
}

// EnsurePrepared 确保依赖关系图和入度表已经生成
func (s *SceneEngine) EnsurePrepared() {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.isPrepared {
		s.prepareInternal()
	}
}

// Exec 方法使用协程并发执行任务
func (s *SceneEngine) Exec() {
	s.EnsurePrepared()

	var wg sync.WaitGroup
	queue := make([]*Node, len(s.queue))
	copy(queue, s.queue) // 复制初始队列，避免并发修改原队列

	taskMap := make(map[string]int)
	for k, v := range s.taskMap {
		taskMap[k] = v
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]

		wg.Add(1)
		go func(node *Node) {
			defer wg.Done()

			// 读取输入字段和上下文字段
			inputs := make(map[string]interface{})
			context := make(map[string]interface{})

			for _, depField := range node.DepField {
				for _, dataBus := range s.databus {
					if value := dataBus.GetVal(depField); value != nil {
						inputs[depField] = value
						break
					}
				}
			}

			// 执行任务
			runnableNode, ok := s.task[node.TaskName]
			if !ok {
				panic(fmt.Sprintf("任务 %s 未找到", node.TaskName))
			}

			outputs, err := runnableNode.task.RunnableNode.Run(context, inputs)
			if err != nil {
				panic(fmt.Sprintf("任务 %s 执行失败: %v", node.TaskName, err))
			}

			// 将输出字段写入 databus
			for key, value := range outputs {
				for _, dataBus := range s.databus {
					dataBus.SetVal(key, value)
				}
			}

			// 更新依赖当前任务的其他任务的入度
			s.lock.Lock()
			for name := range taskMap {
				for _, dep := range s.task[name].DepTask {
					if dep == node.TaskName {
						taskMap[name]--
						if taskMap[name] == 0 {
							queue = append(queue, s.task[name])
						}
					}
				}
			}
			s.lock.Unlock()
		}(current)
	}

	wg.Wait()
}

// Updatetask 更新 task 数据并重新生成依赖关系
func (s *SceneEngine) Updatetask(newtask map[string]*Node) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.task = newtask
	s.isPrepared = false // 标记需要重新生成依赖关系
}
