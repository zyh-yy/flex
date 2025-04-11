package main

import (
	"flex/core"
	"fmt"
	"sync"
)

func main() {
	// 配置场景
	// core.ConfigureScene("Scene1", []string{"output1", "output2"})

	// // 注册任务1：无外部依赖，直接产出 output1
	// core.RegisterTask(
	// 	"Task1",
	// 	[]string{}, // 无依赖字段
	// 	map[core.FieldType][]string{
	// 		core.ResultField: {"output1"},
	// 	},
	// 	func(contextInput map[string]interface{}, input map[string]interface{}) (map[string]interface{}, error) {
	// 		return map[string]interface{}{"output1": 42}, nil
	// 	},
	// )

	// // 注册任务2：依赖 Task1 的 output1，产出 output2
	// core.RegisterTask(
	// 	"Task2",
	// 	[]string{"output1"}, // 依赖字段
	// 	map[core.FieldType][]string{
	// 		core.ResultField: {"output2"},
	// 	},
	// 	func(contextInput map[string]interface{}, input map[string]interface{}) (map[string]interface{}, error) {
	// 		value := input["output1"].(int)
	// 		return map[string]interface{}{"output2": value * 2}, nil
	// 	},
	// )

	// // 注册任务3：依赖 Task1 的 output1，产出 output2
	// core.RegisterTask(
	// 	"Task3",
	// 	[]string{"output2"}, // 依赖字段
	// 	map[core.FieldType][]string{
	// 		core.ResultField: {"output3"},
	// 	},
	// 	func(contextInput map[string]interface{}, input map[string]interface{}) (map[string]interface{}, error) {
	// 		value := input["output1"].(int)
	// 		return map[string]interface{}{"output2": value * 2}, nil
	// 	},
	// )
	wg := sync.WaitGroup{}
	core.Init()
	wg.Add(1)

	wg.Wait()

	// 构造 SceneEngine
	engine, err := core.BuildSceneEngine("Scene1")
	if err != nil {
		fmt.Println("构造失败:", err)
		return
	}

	// 执行任务
	engine.Exec()

	// 打印推导出的任务
	fmt.Println("推导出的任务:")
	for _, taskName := range engine.GetDerivedTasks() {
		fmt.Println(taskName)
	}

	// 打印最终结果
	fmt.Println("任务执行完成，最终结果:")
	for field, value := range engine.GetDataBusValues() { // 假设 GetDataBusValues 方法返回最终数据
		fmt.Printf("%s: %v\n", field, value)
	}
}
