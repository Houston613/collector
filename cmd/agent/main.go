package main

import (
	"collector/internal/agent"
)

func main() {
	a := agent.NewAgent("http://localhost:8080")
	a.Run()
}
