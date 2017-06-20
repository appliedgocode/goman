package main

import "os"

func main() {
	for i := range os.Args[1:] {
		print(os.Args[i+1], " ")
	}
	println()
}
