package main

import "os"

func main() {
	os.Exit(1) // want "direct call to os.Exit in main function of main package is prohibited"
}

func other() {
	os.Exit(0)
}
