package main

import (
	"fmt"
	"os"
	"strings"
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

func getResourceLoader(config appConfig) resourceLoader {
	return resourceLoader{
		config: config,
		loadTemplate: func(templateFileName string) ([]byte, error) {
			templateFilePath := fmt.Sprintf("%s%c%s%c%s",
				config.theme, os.PathSeparator, templatesDirName, os.PathSeparator, templateFileName)
			return os.ReadFile(templateFilePath)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			includeFilePath := getIncludeFilePath(includeFileName, level, config)
			if includeFilePath != "" {
				return os.ReadFile(includeFilePath)
			} else {
				return nil, nil
			}
		},
	}
}

func getIncludeFilePath(includeFileName string, level templateIncludeLevel, config appConfig) string {
	var includeFilePath string
	if Global == level {
		includeFilePath = fmt.Sprintf("%s%c%s", includeDirName, os.PathSeparator, includeFileName)
	} else if Theme == level {
		themeName := config.theme
		if strings.ContainsRune(themeName, os.PathSeparator) {
			themePathSegments := strings.Split(themeName, string(os.PathSeparator))
			themeName = themePathSegments[len(themePathSegments)-1]
		}
		includeFilePath = fmt.Sprintf("%s%c%s%c%s", includeDirName, os.PathSeparator, themeName, os.PathSeparator, includeFileName)
	} else {
		panic("invalid template include level for: " + includeDirName)
	}
	if !fileExists(includeFilePath) {
		// ignore non-existing include files (includes are optional)
		includeFilePath = ""
	}
	return includeFilePath
}

func handleStats(stats stats) {
	fmt.Println("")
	fmt.Println("[------- stats --------]")
	fmt.Printf(" - pages: %d\n", stats.pageCnt)
	fmt.Printf(" - posts: %d\n", stats.postCnt)
	fmt.Printf(" - tags: %d\n", stats.tagCnt)
	fmt.Printf(" - files generated: %d\n", stats.genCnt)
	fmt.Println("[----------------------]")
}
