package main

import (
	"fmt"
	"os"
)

func exitWithError(err string) {
	println(err)
	os.Exit(-1)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func println(strings ...interface{}) {
	for _, s := range strings {
		fmt.Println(s)
	}
}

func sprintln(strings ...interface{}) {
	fmt.Println("")
	for _, s := range strings {
		fmt.Println(s)
	}
}
