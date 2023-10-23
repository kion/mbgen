package main

import (
	"fmt"
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
	commandCleanup = /* const */ appCommandDescriptor{
		command:     "cleanup",
		description: "perform a cleanup",
		reqArgCnt:   1,
		usage: "mbgen cleanup <target>\n\n" +
			" - <target> is one of the following:\n\n" +
			"   - thumbs: deletes all previously generated thumbnail files",
		reqConfig: true,
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
		commandCleanup.command:  {_cleanup, commandCleanup},
		commandGenerate.command: {_generate, commandGenerate},
		commandStats.command:    {_stats, commandStats},
		commandServe.command:    {_serve, commandServe},
		commandTheme.command:    {_theme, commandTheme},
	}
}

func _version(config appConfig, commandArgs ...string) {
	println("mbgen " + appVersion)
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
		println(" - config file already exists: " + configFileName)
	} else {
		config = defaultConfig()
		config.siteName = "Sample Site Name"
		writeConfig(config)
		sprintln(" - generated sample config file: " + configFileName)
	}
	// download content samples
	if dirExists(markdownPagesDirName) {
		println(" - page content dir already exists: " + markdownPagesDirName)
	} else {
		createDir(markdownPagesDirName)
		err := download(defaultGitHubRepoPageContentSamplesUrl, markdownPagesDirName)
		if err != nil {
			println(fmt.Sprintf("error downloading page content samples:\n\n" + err.Error()))
		} else {
			println(" - downloaded page content samples")
		}
	}
	if dirExists(markdownPostsDirName) {
		println(" - post content dir already exists: " + markdownPostsDirName)
	} else {
		createDir(markdownPostsDirName)
		err := download(defaultGitHubRepoPostContentSamplesUrl, markdownPostsDirName)
		if err != nil {
			println(fmt.Sprintf("error downloading post content samples:\n\n" + err.Error()))
		} else {
			println(" - downloaded post content samples")
		}
	}
	if dirExists(deployDirName) {
		println(" - deploy dir already exists: " + deployDirName)
	} else {
		createDir(deployDirName)
		err := download(defaultGitHubRepoDeployDirContentSamplesUrl, deployDirName)
		if err != nil {
			println(fmt.Sprintf("error downloading deploy dir content samples:\n\n" + err.Error()))
		} else {
			println(" - downloaded deploy dir content samples")
		}
	}
	// install and activate default theme
	_theme(config, "install", defaultThemeName)
	_theme(config, "activate", defaultThemeName)
	copyThemeIncludes(defaultThemeName)
}

func _cleanup(config appConfig, commandArgs ...string) {
	target := commandArgs[0]
	switch target {
	case "thumbs":
		resLoader := getResourceLoader(config)
		parsePages(config, resLoader, deleteImgThumbnails)
		parsePosts(config, resLoader, deleteImgThumbnails)
	case "archive":
		deployArchivePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, deployArchiveDirName)
		deleteIfExists(deployArchivePath)
		println(" - deleted archive dir: " + deployArchivePath)
	default:
		sprintln("error: invalid cleanup command target: " + target)
		usageHelp := "usage:\n\n" + commandCleanup.usage
		usage(usageHelp)
	}
}

func _generate(config appConfig, commandArgs ...string) {
	createDirIfNotExists(deployDirName)

	resLoader := getResourceLoader(config)

	deployResDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, resourcesDirName)
	recreateDir(deployResDirPath)

	themeResourcesDirPath := fmt.Sprintf("%s%c%s", config.theme, os.PathSeparator, resourcesDirName)
	deployResourcesDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, resourcesDirName)
	copyDir(themeResourcesDirPath, deployResourcesDirPath)
	sprintln(" - copied theme resources")

	for _, level := range templateIncludeLevels {
		stylesIncludeFilePath := getIncludeFilePath(stylesFileName, level, resLoader.config)
		if stylesIncludeFilePath != "" {
			copyFile(stylesIncludeFilePath, fmt.Sprintf("%s%c%s", deployResourcesDirPath, os.PathSeparator, fmt.Sprintf(stylesIncludeFileNameFormat, level.String())))
		}
	}

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
				println(" - generated file: " + outputFilePath)
				generatedCnt++
			}
			return generated
		})
	stats.genCnt = generatedCnt
	handleStats(stats)
}

func _stats(config appConfig, commandArgs ...string) {
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
			sprintln("theme is not installed: " + theme)
		} else {
			config.theme = themeDir
			writeConfig(config)
			sprintln(" - " + configFileName + " updated to activate new theme: " + theme)
		}
	case "install":
		if themeInstalled {
			sprintln("theme is already installed: " + theme)
		} else {
			sprintln(" - installing theme: " + theme)
			themeUrl := fmt.Sprintf("%s/%s", defaultGitHubRepoThemesUrl, theme)
			err := download(themeUrl, themeDir)
			if err != nil {
				println(fmt.Sprintf("error installing theme:\n\n" + err.Error()))
			} else {
				copyThemeIncludes(theme)
			}
		}
	case "update":
		if !themeInstalled {
			sprintln("theme is not installed: " + theme)
		} else {
			println(" - updating theme: " + theme)
			themeUrl := fmt.Sprintf("%s/%s", defaultGitHubRepoThemesUrl, theme)
			themeDlDir := themeDir + downloadedThemeDirSuffix
			deleteIfExists(themeDlDir)
			err := download(themeUrl, themeDlDir)
			if err != nil {
				println(fmt.Sprintf("error updating theme:\n\n" + err.Error()))
			} else {
				recreateDir(themeDir)
				copyDir(themeDlDir, themeDir)
				deleteIfExists(themeDlDir)
				copyThemeIncludes(theme)
			}
		}
	case "refresh":
		if !themeInstalled {
			sprintln("theme is not installed: " + theme)
		} else {
			println(" - refreshing theme: " + theme)
			copyThemeIncludes(theme)
		}
	case "delete":
		if !themeInstalled {
			sprintln("theme is not installed: " + theme)
		} else {
			if config.theme == themeDir {
				sprintln("cannot delete theme being currently in use: " + theme)
			} else {
				println(" - deleting theme: " + theme)
				deleteIfExists(themeDir)
				themeDstIncludeDir := fmt.Sprintf("%s%c%s", includeDirName, os.PathSeparator, theme)
				deleteIfExists(themeDstIncludeDir)
			}
		}
	default:
		sprintln("error: invalid theme command action: " + action)
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
					sprintln(
						" - copying include file:",
						"   - src: "+includeFileSrcPath,
						"   - dst: "+includeFileDstPath,
					)
					copyFile(includeFileSrcPath, includeFileDstPath)
				}
			}
		}
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

func handleStats(stats stats) {
	sprintln(
		"[------- stats --------]\n",
		fmt.Sprintf(" - pages: %d", stats.pageCnt),
		fmt.Sprintf(" - posts: %d", stats.postCnt),
		fmt.Sprintf(" - tags: %d", stats.tagCnt),
		fmt.Sprintf(" - files generated: %d\n", stats.genCnt),
		"[----------------------]",
	)
}
