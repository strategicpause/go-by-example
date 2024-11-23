package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ecs"
)

func main() {
	// Read & unmarshal the request data from a json file.
	jsonFile, err := os.ReadFile("describe_task_req.json")
	if err != nil {
		fmt.Println("Error reading JSON file:", err)
		return
	}
	var taskData map[string]interface{}
	err = json.Unmarshal(jsonFile, &taskData)
	if err != nil {
		fmt.Println("Error unmarshaling JSON:", err)
		return
	}

	// Load the AWS SDK configuration
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Println("Error loading AWS config:", err)
		return
	}

	// Create an ECS client
	client := ecs.NewFromConfig(cfg)

	// Convert the map to ecs.DescribeTasksInput
	input := &ecs.DescribeTasksInput{}
	inputBytes, err := json.Marshal(taskData)
	if err != nil {
		fmt.Println("Error marshaling input:", err)
		return
	}
	err = json.Unmarshal(inputBytes, input)
	if err != nil {
		fmt.Println("Error unmarshaling to DescribeTasksInput:", err)
		return
	}

	// Make the DescribeTasks API call
	result, err := client.DescribeTasks(context.TODO(), input)
	if err != nil {
		fmt.Println("Error describing tasks:", err)
		return
	}

	// Print the result
	fmt.Printf("Tasks described successfully. Number of tasks: %d\n", len(result.Tasks))
	for i, task := range result.Tasks {
		fmt.Printf("Task %d: %s\n", i+1, *task.TaskArn)
		fmt.Printf("  Status: %s\n", task.LastStatus)
		fmt.Printf("  Desired Status: %s\n", task.DesiredStatus)
		fmt.Printf("  CPU: %s\n", task.Cpu)
		fmt.Printf("  Memory: %s\n", task.Memory)
		fmt.Println("  Containers:")
		for _, container := range task.Containers {
			fmt.Printf("    - Name: %s, Status: %s\n", *container.Name, container.LastStatus)
		}
		fmt.Println()
	}
}
