package main

import (
	"fmt"
	"os"
	"slices"
	"strconv"
	"time"
)

func exitWithError(err string) {
	println(err)
	os.Exit(1)
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

func printErr(err error) {
	_, err = fmt.Fprintf(os.Stderr, "%v\n", err)
	check(err)
}

func formatYearAndMonth(year int, month time.Month) string {
	m := strconv.Itoa(int(month))
	if len(m) == 1 {
		m = "0" + m
	}
	return strconv.Itoa(year) + "-" + m
}

func removeValuesFromSlice[T comparable](slice []T, values ...T) []T {
	modSlice := make([]T, 0, len(slice))
	for _, v := range slice {
		if !slices.Contains(values, v) {
			modSlice = append(modSlice, v)
		}
	}
	return modSlice
}
