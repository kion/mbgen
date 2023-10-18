package main

import (
	"fmt"
	"log"
	"os"
)

func exitWithError(err string) {
	fmt.Println(err)
	os.Exit(-1)
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func sprintln(strings ...string) {
	fmt.Println("")
	for _, s := range strings {
		fmt.Println(s)
	}
}

func logSprintln(strings ...string) {
	fmt.Println("")
	for _, s := range strings {
		log.Println(s)
	}
}
