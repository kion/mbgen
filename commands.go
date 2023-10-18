package main

import (
	"fmt"
	"log"
	"os"
	"strings"
)

var (
	commandVersion = /* const */ appCommandDescriptor{
		command:     "version",
		description: "print out version info",
		usage:       "mbgen version",
		reqConfig:   false,
	}
	commandHelp = /* const */ appCommandDescriptor{
		command:     "help",
		description: "print out help/usage information",
		usage: "mbgen help [command]\n\n" +
			"where [command] is one of the following supported commands to print out help/usage information for:\n\n" +
			"init, generate, stats, serve, theme",
		reqConfig: false,
		optArgCnt: 1,
	}
	commandInit = /* const */ appCommandDescriptor{
		command: "init",
		description: "initialize working dir:\n\n" +
			" - generate sample " + configFileName + " file\n" +
			" - download content samples\n" +
			" - install and activate default theme",
		usage:     "mbgen init\n\n",
		reqConfig: false,
	}
	commandGenerate = /* const */ appCommandDescriptor{
		command:     "generate",
		description: "parse content and generate site",
		usage: "mbgen generate\n\n" +
			"must be run from a working dir containing " + configFileName + " file",
		reqConfig: true,
	}
	commandStats = /* consts */ appCommandDescriptor{
		command:     "stats",
		description: "parse content and print out stats",
		usage: "mbgen stats\n\n" +
			"must be run from a working dir containing " + configFileName + " file",
		reqConfig: true,
	}
	commandServe = /* const */ appCommandDescriptor{
		command:     "serve",
		description: "start a web server to serve the site",
		usage: "mbgen serve\n\n" +
			"must be run from a working dir containing " + configFileName + " file and " + deployDirName + " directory with generated assets",
		reqConfig: true,
	}
	commandTheme = /* const */ appCommandDescriptor{
		command:     "theme",
		description: "install/update and/or activate a theme",
		reqArgCnt:   2,
		usage: "mbgen theme <action> <theme-name>\n\n" +
			" - <action> is one of the following:\n\n" +
			"   - activate: checks if the specified theme is installed,\n" +
			"     and modifies the " + configFileName + " file to make it active\n\n" +
			"   - install: downloads and installs the specified theme if it's not yet installed,\n" +
			"     and copies all the relevant/missing theme include files into the " + includeDirName + "dir inside the working dir\n\n" +
			"   - update: downloads and installs the required updates for the specified theme (must be already installed),\n" +
			"     and copies all the relevant/missing theme include files into the " + includeDirName + "dir inside the working dir\n\n" +
			"   - refresh: copies all the relevant/missing theme include files into the " + includeDirName + "dir inside the working dir\n\n" +
			"   - delete: deletes all the assets of the specified theme, if it's not being currently in use\n\n" +
			" - <theme-name> is the name of a theme to perform the specified action on,\n\n" +
			"   - the default theme name is: \"" + defaultThemeName + "\", but you can also use the \"" + defaultThemeAlias + "\" alias instead",
		reqConfig: true,
	}
)

func getSupportedCommands() map[string]tuple2[appCommand, appCommandDescriptor] {
	return map[string]tuple2[appCommand, appCommandDescriptor]{
		commandVersion.command:  {_version, commandVersion},
		commandHelp.command:     {_help, commandHelp},
		commandInit.command:     {_init, commandInit},
		commandGenerate.command: {_generate, commandGenerate},
		commandStats.command:    {_stats, commandStats},
		commandServe.command:    {_serve, commandServe},
		commandTheme.command:    {_theme, commandTheme},
	}
}

func _version(config appConfig, commandArgs ...string) {
	fmt.Println("mbgen " + appVersion)
}

func _help(config appConfig, commandArgs ...string) {
	if commandArgs != nil && len(commandArgs) > 0 {
		cmdDescr := getSupportedCommands()[commandArgs[0]].V2
		usageHelp := cmdDescr.description + "\n\nusage:\n\n" + cmdDescr.usage
		usage(usageHelp)
	} else {
		usage("")
	}
}

func _init(config appConfig, commandArgs ...string) {
	// generate sample config file
	if fileExists(configFileName) {
		log.Println(" - config file already exists: " + configFileName)
	} else {
		config = defaultConfig()
		config.siteName = "Sample Site Name"
		writeConfig(config)
		fmt.Println("")
		log.Println(" - generated sample config file: " + configFileName)
	}
	// download content samples
	if dirExists(markdownPagesDirName) {
		log.Println(" - page content dir already exists: " + markdownPagesDirName)
	} else {
		createDir(markdownPagesDirName)
		err := download(defaultGitHubRepoPageContentSamplesUrl, markdownPagesDirName)
		if err != nil {
			log.Println(fmt.Sprintf("error downloading page content samples:\n\n" + err.Error()))
		} else {
			log.Println(" - downloaded page content samples")
		}
	}
	if dirExists(markdownPostsDirName) {
		log.Println(" - post content dir already exists: " + markdownPostsDirName)
	} else {
		createDir(markdownPostsDirName)
		err := download(defaultGitHubRepoPostContentSamplesUrl, markdownPostsDirName)
		if err != nil {
			log.Println(fmt.Sprintf("error downloading post content samples:\n\n" + err.Error()))
		} else {
			log.Println(" - downloaded post content samples")
		}
	}
	if dirExists(deployDirName) {
		log.Println(" - deploy dir already exists: " + deployDirName)
	} else {
		createDir(deployDirName)
		err := download(defaultGitHubRepoDeployDirContentSamplesUrl, deployDirName)
		if err != nil {
			log.Println(fmt.Sprintf("error downloading deploy dir content samples:\n\n" + err.Error()))
		} else {
			log.Println(" - downloaded deploy dir content samples")
		}
	}
	// install and activate default theme
	_theme(config, "install", defaultThemeName)
	_theme(config, "activate", defaultThemeName)
	copyThemeIncludes(defaultThemeName)
}

func _generate(config appConfig, commandArgs ...string) {
	createDirIfNotExists(deployDirName)

	resLoader := getResourceLoader(config)

	deployResDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, resourcesDirName)
	recreateDir(deployResDirPath)

	themeResourcesDirPath := fmt.Sprintf("%s%c%s", config.theme, os.PathSeparator, resourcesDirName)
	deployResourcesDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, resourcesDirName)
	copyDir(themeResourcesDirPath, deployResourcesDirPath)
	log.Println(" - copied theme resources")

	for _, level := range templateIncludeLevels {
		stylesIncludeFilePath := getIncludeFilePath(stylesFileName, level, resLoader.config)
		if stylesIncludeFilePath != "" {
			copyFile(stylesIncludeFilePath, fmt.Sprintf("%s%c%s", deployResourcesDirPath, os.PathSeparator, fmt.Sprintf(stylesIncludeFileNameFormat, level.String())))
		}
	}

	fmt.Println("")
	generatedCnt := 0
	stats := process(parsePages(config, resLoader, processImgThumbnails),
		parsePosts(config, resLoader, processImgThumbnails),
		resLoader,
		func(outputFilePath string, data []byte) bool {
			idx := strings.LastIndex(outputFilePath, string(os.PathSeparator))
			if idx != -1 {
				outputDir := outputFilePath[:idx]
				createDirIfNotExists(outputDir)
			}
			generated := writeDataToFileIfChanged(outputFilePath, data)
			if generated {
				log.Println(" - generated file: " + outputFilePath)
				generatedCnt++
			}
			return generated
		})
	stats.genCnt = generatedCnt
	handleStats(stats)
}

func _stats(config appConfig, commandArgs ...string) {
	fmt.Println("")
	resLoader := getResourceLoader(config)
	handleStats(process(parsePages(config, resLoader, nil),
		parsePosts(config, resLoader, nil),
		resLoader, nil))
}

func _serve(config appConfig, commandArgs ...string) {
	listenAndServe(fmt.Sprintf("%s:%d", config.serveHost, config.servePort))
}

func _theme(config appConfig, commandArgs ...string) {
	action := commandArgs[0]
	theme := commandArgs[1]
	if defaultThemeAlias == theme {
		theme = defaultThemeName
	}
	themeInstalled := false
	var themeDir = fmt.Sprintf("%s%c%s", themesDirName, os.PathSeparator, theme)
	themeInstalled = dirExists(themeDir)
	switch action {
	case "activate":
		if !themeInstalled {
			fmt.Println("")
			fmt.Println("theme is not installed: " + theme)
		} else {
			config.theme = themeDir
			writeConfig(config)
			fmt.Println("")
			log.Println(" - " + configFileName + " updated to activate new theme: " + theme)
		}
	case "install":
		if themeInstalled {
			fmt.Println("")
			fmt.Println("theme is already installed: " + theme)
		} else {
			fmt.Println("")
			log.Println(" - installing theme: " + theme)
			themeUrl := fmt.Sprintf("%s/%s", defaultGitHubRepoThemesUrl, theme)
			err := download(themeUrl, themeDir)
			if err != nil {
				log.Println(fmt.Sprintf("error installing theme:\n\n" + err.Error()))
			} else {
				copyThemeIncludes(theme)
			}
		}
	case "update":
		if !themeInstalled {
			fmt.Println("")
			fmt.Println("theme is not installed: " + theme)
		} else {
			log.Println(" - updating theme: " + theme)
			themeUrl := fmt.Sprintf("%s/%s", defaultGitHubRepoThemesUrl, theme)
			themeDlDir := themeDir + downloadedThemeDirSuffix
			deleteIfExists(themeDlDir)
			err := download(themeUrl, themeDlDir)
			if err != nil {
				log.Println(fmt.Sprintf("error updating theme:\n\n" + err.Error()))
			} else {
				recreateDir(themeDir)
				copyDir(themeDlDir, themeDir)
				deleteIfExists(themeDlDir)
				copyThemeIncludes(theme)
			}
		}
	case "refresh":
		if !themeInstalled {
			fmt.Println("")
			fmt.Println("theme is not installed: " + theme)
		} else {
			log.Println(" - refreshing theme: " + theme)
			copyThemeIncludes(theme)
		}
	case "delete":
		if !themeInstalled {
			fmt.Println("")
			fmt.Println("theme is not installed: " + theme)
		} else {
			if config.theme == themeDir {
				fmt.Println("")
				fmt.Println("cannot delete theme being currently in use: " + theme)
			} else {
				log.Println(" - deleting theme: " + theme)
				deleteIfExists(themeDir)
				themeDstIncludeDir := fmt.Sprintf("%s%c%s", includeDirName, os.PathSeparator, theme)
				deleteIfExists(themeDstIncludeDir)
			}
		}
	default:
		fmt.Println("error: invalid theme command action: " + action)
		usageHelp := "usage:\n\n" + commandTheme.usage
		usage(usageHelp)
	}
}

func copyThemeIncludes(theme string) {
	themeSrcIncludeDir := fmt.Sprintf("%s%c%s%c%s", themesDirName, os.PathSeparator, theme, os.PathSeparator, includeDirName)
	if dirExists(themeSrcIncludeDir) {
		themeDstIncludeDir := fmt.Sprintf("%s%c%s", includeDirName, os.PathSeparator, theme)
		createDirIfNotExists(themeDstIncludeDir)
		includeFiles, err := os.ReadDir(themeSrcIncludeDir)
		check(err)
		if len(includeFiles) > 0 {
			for _, includeFile := range includeFiles {
				includeFileInfo, err := includeFile.Info()
				check(err)
				includeFileName := includeFileInfo.Name()
				includeFileDstPath := fmt.Sprintf("%s%c%s", themeDstIncludeDir, os.PathSeparator, includeFileName)
				if !fileExists(includeFileDstPath) {
					includeFileSrcPath := fmt.Sprintf("%s%c%s", themeSrcIncludeDir, os.PathSeparator, includeFileName)
					fmt.Println("")
					log.Println(" - copying include file:")
					log.Println("   - src: " + includeFileSrcPath)
					log.Println("   - dst: " + includeFileDstPath)
					copyFile(includeFileSrcPath, includeFileDstPath)
				}
			}
		}
	}
}
