package main

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
)

var (
	commandVersion = /* const */ appCommandDescriptor{
		command:     "version",
		description: "print out version info",
		usage:       "mbgen version\n",
		reqConfig:   false,
	}
	commandHelp = /* const */ appCommandDescriptor{
		command:     "help",
		description: "print out help/usage information",
		usage: "mbgen help <command>\n\n" +
			"where <command> is one of the following supported commands to print out help/usage information for:\n\n" +
			"init, generate, serve, inspect, cleanup, theme, stats, version\n",
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
		usage: "mbgen cleanup <target>\n\n" +
			" - <target> (optional) is one of the following:\n\n" +
			"   - " + commandCleanupTargetContent + ": deletes all previously generated content (" + contentFileExtension + ") files\n" +
			"     for which markdown (" + markdownFileExtension + ") content files no longer exist\n\n" +
			"   - " + commandCleanupTargetThumbs + ": deletes all previously generated thumbnail files\n\n" +
			"   - " + commandCleanupTargetTags + ": deletes all previously generated tag files\n" +
			"     that are no longer referenced by any markdown (" + markdownFileExtension + ") content files\n\n" +
			"   - " + commandCleanupTargetTagIndex + ": deletes the previously generated tag index file\n\n" +
			"   - " + commandCleanupTargetArchive + ": deletes the previously generated archive files\n\n" +
			"   - " + commandCleanupTargetSearch + ": deletes all previously generated search files\n\n" +
			" - if no <target> is specified, each target is performed based on the following conditions:\n\n" +
			"   - " + commandCleanupTargetContent + ": always\n\n" +
			"   - " + commandCleanupTargetTags + ": always\n\n" +
			"   - " + commandCleanupTargetThumbs + ": if `useThumbs` config option is disabled\n\n" +
			"   - " + commandCleanupTargetTagIndex + ": if `generateTagIndex` config option is disabled\n\n" +
			"   - " + commandCleanupTargetArchive + ": if `generateArchive` config option is disabled\n\n" +
			"   - " + commandCleanupTargetSearch + ": if `enableSearch` config option is disabled\n\n",
		reqConfig: true,
		optArgCnt: 1,
	}
	commandGenerate = /* const */ appCommandDescriptor{
		command:     "generate",
		description: "parse content and generate site",
		usage:       "mbgen generate\n\n",
		reqConfig:   true,
	}
	commandInspect = /* const */ appCommandDescriptor{
		command:     "inspect",
		description: "inspect content and report/fix any issues found",
		usage: "mbgen inspect [" + commandInspectOptionFix + "]\n\n" +
			"the following flags can be specified:\n" +
			" " + commandInspectOptionFix + " - to automatically fix all detected issues; namely:\n" +
			"   - resize and replace the original images that exceed the `maxImgSize` config option value\n\n",
		reqConfig: true,
		optArgCnt: 1,
	}
	commandStats = /* consts */ appCommandDescriptor{
		command:     "stats",
		description: "parse content and print out stats",
		usage:       "mbgen stats\n\n",
		reqConfig:   true,
	}
	commandServe = /* const */ appCommandDescriptor{
		command:     "serve",
		description: "start a web server to serve the site",
		usage: "mbgen serve [" + commandServeOptionWatchReload + "]\n\n" +
			"ONE of the following flags can be specified:\n" +
			" " + commandServeOptionAdmin + " - to render content admin links\n" +
			" " + commandServeOptionWatchReload + " - to automatically regenerate the site and see the changes being reflected in the browser in real-time when you change any of the markdown content (.md) files in the " + markdownPagesDirName + " or " + markdownPostsDirName + " dirs\n\n",
		reqConfig: true,
		optArgCnt: 1,
	}
	commandTheme = /* const */ appCommandDescriptor{
		command:     "theme",
		description: "install/update and/or activate a theme",
		reqArgCnt:   2,
		usage: "mbgen theme <action> <theme-name>\n\n" +
			" - <action> is one of the following:\n\n" +
			"   - " + commandThemeActionActivate + ": checks if the specified theme is installed,\n" +
			"     and modifies the " + configFileName + " file to make it active\n\n" +
			"   - " + commandThemeActionInstall + ": downloads and installs the specified theme if it's not yet installed,\n" +
			"     and copies all the relevant/missing theme include files into the " + includeDirName + "dir inside the working dir\n\n" +
			"   - " + commandThemeActionUpdate + ": downloads and installs the required updates for the specified theme (must be already installed),\n" +
			"     and copies all the relevant/missing theme include files into the " + includeDirName + "dir inside the working dir\n\n" +
			"   - " + commandThemeActionRefresh + ": copies all the relevant/missing theme include files into the " + includeDirName + "dir inside the working dir\n\n" +
			"   - " + commandThemeActionDelete + ": deletes all the assets of the specified theme, if it's not being currently in use\n\n" +
			" - <theme-name> is the name of a theme to perform the specified action on,\n\n" +
			"   - the default theme name is: \"" + defaultThemeName + "\", but you can also use the \"" + defaultThemeAlias + "\" alias instead\n\n",
		reqConfig: true,
	}
	commandDeploy = /* const */ appCommandDescriptor{
		command: "deploy",
		description: "deploy generated site to a remote server\n\n" +
			" - requires the `deployPath`, `deployHost`, and `deployUsername` config options to be set in the `config.yml`",
		usage:     "mbgen deploy\n\n",
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
		commandInspect.command:  {_inspect, commandInspect},
		commandStats.command:    {_stats, commandStats},
		commandServe.command:    {_serve, commandServe},
		commandTheme.command:    {_theme, commandTheme},
		commandDeploy.command:   {_deploy, commandDeploy},
	}
}

func _version(config appConfig, commandArgs ...string) {
	println("mbgen " + appVersion)
}

func _help(config appConfig, commandArgs ...string) {
	if commandArgs != nil && len(commandArgs) > 0 {
		arg := commandArgs[0]
		var cmdDescr appCommandDescriptor
		if cmd, ok := getSupportedCommands()[arg]; ok {
			cmdDescr = cmd.V2
			usageHelp := cmdDescr.description + "\n\nusage:\n\n" + cmdDescr.usage
			usage(usageHelp, 0)
		} else {
			sprintln("error: unknown help <command> argument: " + arg)
			usage("", 1)
		}
	} else {
		usage("", 0)
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
			println(fmt.Sprintf("error downloading page content samples:\n\n%s", err.Error()))
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
			println(fmt.Sprintf("error downloading post content samples:\n\n%s", err.Error()))
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
			println(fmt.Sprintf("error downloading deploy dir content samples:\n\n%s", err.Error()))
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
	cleanupContent := false
	cleanupThumbs := false
	cleanupTags := false
	cleanupTagIndex := false
	cleanupArchive := false
	cleanupSearch := false
	if commandArgs == nil || len(commandArgs) == 0 {
		cleanupContent = true
		cleanupTags = true
		cleanupThumbs = !config.useThumbs
		cleanupArchive = !config.generateArchive
		cleanupTagIndex = !config.generateTagIndex
		cleanupSearch = !config.enableSearch
	} else {
		target := commandArgs[0]
		switch target {
		case commandCleanupTargetContent:
			cleanupContent = true
		case commandCleanupTargetThumbs:
			cleanupThumbs = true
		case commandCleanupTargetTags:
			cleanupTags = true
		case commandCleanupTargetTagIndex:
			cleanupTagIndex = true
		case commandCleanupTargetArchive:
			cleanupArchive = true
		case commandCleanupTargetSearch:
			cleanupSearch = true
		default:
			sprintln("error: invalid cleanup command <target> argument: " + target)
			usageHelp := "usage:\n\n" + commandCleanup.usage
			usage(usageHelp, 1)
		}
	}
	if cleanupContent {
		deployPageDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, deployPageDirName)
		deployPageDirEntries, err := os.ReadDir(deployPageDirPath)
		check(err)
		if len(deployPageDirEntries) > 0 {
			for _, deployPageEntry := range deployPageDirEntries {
				deployPageEntryInfo, err := deployPageEntry.Info()
				check(err)
				if !deployPageEntryInfo.IsDir() {
					deployPageEntryFileName := deployPageEntryInfo.Name()
					pageId := deployPageEntryFileName[:len(deployPageEntryFileName)-len(filepath.Ext(deployPageEntryFileName))]
					markdownPageFilePath := fmt.Sprintf("%s%c%s", markdownPagesDirName, os.PathSeparator, pageId+markdownFileExtension)
					if !fileExists(markdownPageFilePath) {
						sprintln(" - page markdown file no longer exists: " + markdownPageFilePath)
						deployPageFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPageDirName, os.PathSeparator, deployPageEntryFileName)
						deleteFile(deployPageFilePath)
						sprintln(" - deleted page content file: " + deployPageFilePath)
					}
				}
			}
		}
		deployPostDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, deployPostDirName)
		deployPostDirEntries, err := os.ReadDir(deployPostDirPath)
		check(err)
		if len(deployPostDirEntries) > 0 {
			for _, deployPostEntry := range deployPostDirEntries {
				deployPostEntryInfo, err := deployPostEntry.Info()
				check(err)
				if !deployPostEntryInfo.IsDir() {
					deployPostEntryFileName := deployPostEntryInfo.Name()
					postId := deployPostEntryFileName[:len(deployPostEntryFileName)-len(filepath.Ext(deployPostEntryFileName))]
					markdownPostFilePath := fmt.Sprintf("%s%c%s", markdownPostsDirName, os.PathSeparator, postId+markdownFileExtension)
					if !fileExists(markdownPostFilePath) {
						sprintln(" - post markdown file no longer exists: " + markdownPostFilePath)
						deployPostFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostDirName, os.PathSeparator, deployPostEntryFileName)
						deleteFile(deployPostFilePath)
						sprintln(" - deleted post content file: " + deployPostFilePath)
					}
				}
			}
		}
	}
	if cleanupThumbs {
		resLoader := getResourceLoader(config)
		parsePages(config, resLoader, deleteImgThumbnails, false)
		parsePosts(config, resLoader, deleteImgThumbnails, false)
	}
	if cleanupTags {
		deployTagsDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, deployTagsDirName)
		deployTagsDirEntries, err := os.ReadDir(deployTagsDirPath)
		check(err)
		if len(deployTagsDirEntries) > 0 {
			posts := parsePosts(config, getResourceLoader(config), nil, false)
			var tags []string
			for _, post := range posts {
				for _, tag := range post.Tags {
					t := strings.ToLower(tag)
					if !slices.Contains(tags, t) {
						tags = append(tags, t)
					}
				}
			}
			for _, deployTagDirEntry := range deployTagsDirEntries {
				deployTagDirEntryInfo, err := deployTagDirEntry.Info()
				check(err)
				if deployTagDirEntryInfo.IsDir() {
					deployTagDirName := deployTagDirEntryInfo.Name()
					if !slices.Contains(tags, deployTagDirName) {
						sprintln(" - tag no longer referenced: " + deployTagDirName)
						deployTagDirPath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployTagsDirName, os.PathSeparator, deployTagDirName)
						deleteIfExists(deployTagDirPath)
						sprintln(" - deleted tag dir: " + deployTagDirPath)
					}
				}
			}
		}
	}
	if cleanupTagIndex {
		deployTagIndexPath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployTagsDirName, os.PathSeparator, indexPageFileName)
		if deleteIfExists(deployTagIndexPath) {
			sprintln(" - deleted tag index file: " + deployTagIndexPath)
		}
	}
	if cleanupArchive {
		deployArchivePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, deployArchiveDirName)
		if deleteIfExists(deployArchivePath) {
			sprintln(" - deleted archive dir: " + deployArchivePath)
		}
	}
	if cleanupSearch {
		deploySearchIndexPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, searchIndexFileName)
		if deleteIfExists(deploySearchIndexPath) {
			sprintln(" - deleted search index file: " + deploySearchIndexPath)
		}
		deploySearchPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, searchPageFileName)
		if deleteIfExists(deploySearchPath) {
			sprintln(" - deleted search page file: " + deploySearchPath)
		}
		deployResourcesDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, resourcesDirName)
		searchJSFilePath := fmt.Sprintf("%s%c%s", deployResourcesDirPath, os.PathSeparator, searchJSFileName)
		if deleteIfExists(searchJSFilePath) {
			sprintln(" - deleted search JS file: " + searchJSFilePath)
		}
	}
}

func _generate(config appConfig, commandArgs ...string) {
	createDirIfNotExists(deployDirName)

	resLoader := getResourceLoader(config)

	deployResDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, resourcesDirName)
	recreateDir(deployResDirPath)

	sprintln(" - copying theme resources ...")
	themeResourcesDirPath := fmt.Sprintf("%s%c%s", config.theme, os.PathSeparator, resourcesDirName)
	deployResourcesDirPath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, resourcesDirName)
	copyDir(themeResourcesDirPath, deployResourcesDirPath)

	if config.enableSearch {
		searchJSFilePath := fmt.Sprintf("%s%c%s", deployResourcesDirPath, os.PathSeparator, searchJSFileName)
		writeDataToFileIfChanged(searchJSFilePath, []byte(searchJS))
	}

	processAndHandleStats(config, resLoader, false)
}

func _inspect(config appConfig, commandArgs ...string) {
	if commandArgs != nil && len(commandArgs) > 0 {
		if len(commandArgs) == 1 {
			if commandArgs[0] == commandInspectOptionFix {
				processOriginalMediaFiles(config, false)
			} else {
				sprintln("error: invalid inspect command argument: " + commandArgs[0])
				usageHelp := "usage:\n\n" + commandInspect.usage
				usage(usageHelp, 1)
			}
		} else {
			sprintln("error: invalid number of inspect command arguments (max allowed: " + strconv.Itoa(commandInspect.optArgCnt) + ")")
			usageHelp := "usage:\n\n" + commandInspect.usage
			usage(usageHelp, 1)
		}
	} else {
		if processOriginalMediaFiles(config, true) {
			sprintln(" - run the following command to fix the issues found:\n\n" +
				"   mbgen inspect " + commandInspectOptionFix)
		} else {
			sprintln(" - no issues found")
		}
	}
}

func _stats(config appConfig, commandArgs ...string) {
	resLoader := getResourceLoader(config)
	handleStats(process(parsePages(config, resLoader, nil, false),
		parsePosts(config, resLoader, nil, false),
		resLoader, nil))
}

func _serve(config appConfig, commandArgs ...string) {
	resLoader := getResourceLoader(config)
	var wChan chan watchReloadData
	var admin bool
	if commandArgs != nil && len(commandArgs) > 0 {
		if len(commandArgs) == 1 {
			arg := commandArgs[0]
			if arg == commandServeOptionAdmin {
				admin = true
			} else if arg == commandServeOptionWatchReload {
				wChan = make(chan watchReloadData)
				mdFileExt := []string{markdownFileExtension}
				go watchDirForChanges(markdownPagesDirName, mdFileExt, false, func(dwEvent dirWatchEvent) {
					filePath := strings.Split(dwEvent.filePath, string(os.PathSeparator))
					fileName := filePath[len(filePath)-1]
					pageId := fileName[:len(fileName)-len(filepath.Ext(fileName))]
					handleMdContentDirWatchEvent(dwEvent, Page, pageId, config, resLoader, wChan)
				})
				go watchDirForChanges(markdownPostsDirName, mdFileExt, false, func(dwEvent dirWatchEvent) {
					filePath := strings.Split(dwEvent.filePath, string(os.PathSeparator))
					fileName := filePath[len(filePath)-1]
					postId := fileName[:len(fileName)-len(filepath.Ext(fileName))]
					handleMdContentDirWatchEvent(dwEvent, Post, postId, config, resLoader, wChan)
				})
				mediaDir := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, mediaDirName)
				go watchDirForChanges(mediaDir, thumbImageFileExtensions, true, func(dwEvent dirWatchEvent) {
					filePath := strings.Split(dwEvent.filePath, string(os.PathSeparator))
					ceType := contentEntityTypeFromString(filePath[len(filePath)-3])
					ceId := filePath[len(filePath)-2]
					handleMediaDirWatchEvent(dwEvent, ceType, ceId, config, resLoader, wChan)
				})
			} else {
				sprintln("error: invalid serve command argument: " + commandArgs[0])
				usageHelp := "usage:\n\n" + commandServe.usage
				usage(usageHelp, 1)
			}
		} else {
			sprintln("error: invalid number of serve command arguments (max allowed: " + strconv.Itoa(commandServe.optArgCnt) + ")")
			usageHelp := "usage:\n\n" + commandServe.usage
			usage(usageHelp, 1)
		}
	}
	listenAndServe(fmt.Sprintf("%s:%d", config.serveHost, config.servePort), admin, wChan, config, resLoader)
}

func handleMdContentDirWatchEvent(dwEvent dirWatchEvent, contentEntityType contentEntityType, contentEntityId string, config appConfig, resLoader resourceLoader, wChan chan watchReloadData) {
	ceType := strings.ToLower(contentEntityType.String())
	switch dwEvent.op {
	case dirWatchOpCreate:
		sprintln(" - [watch] "+ceType+" markdown file created: ", dwEvent.filePath)
	case dirWatchOpUpdate:
		sprintln(" - [watch] "+ceType+" markdown file updated: ", dwEvent.filePath)
	case dirWatchOpRename:
		sprintln(" - [watch] "+ceType+" markdown file renamed: ",
			"   original file: "+*dwEvent.originalFilePath,
			"   renamed to: "+dwEvent.filePath)
		originalMdContentFileName := filepath.Base(*dwEvent.originalFilePath)
		originalContentEntityId := originalMdContentFileName[:len(originalMdContentFileName)-len(filepath.Ext(originalMdContentFileName))]
		// ==================================================
		// delete the corresponding old content file
		// (a new one will be generated)
		// ==================================================
		originalContentFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, ceType, os.PathSeparator, originalContentEntityId+contentFileExtension)
		deleteIfExists(originalContentFilePath)
		sprintln(" - deleted old content file: " + originalContentFilePath)
		// ==================================================
		// rename the corresponding media directory
		// ==================================================
		originalMediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, ceType, os.PathSeparator, originalContentEntityId)
		newMediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, ceType, os.PathSeparator, contentEntityId)
		renameFile(originalMediaDirPath, newMediaDirPath)
		sprintln(" - renamed media dir: " + originalMediaDirPath + " -> " + newMediaDirPath)
		// ==================================================
	case dirWatchOpDelete:
		sprintln(" - [watch] "+ceType+" markdown file deleted: ", dwEvent.filePath)
		// ==================================================
		// delete the corresponding content file
		// ==================================================
		contentFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, ceType, os.PathSeparator, contentEntityId+contentFileExtension)
		deleteIfExists(contentFilePath)
		sprintln(" - deleted content file: " + contentFilePath)
		// ==================================================
		// delete the corresponding media directory
		// ==================================================
		mediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, ceType, os.PathSeparator, contentEntityId)
		deleteIfExists(mediaDirPath)
		sprintln(" - deleted media dir: " + mediaDirPath)
		// ==================================================
		// delete tag files for the no longer referenced tags
		// ==================================================
		_cleanup(config, commandCleanupTargetTags)
		// ==================================================
	}
	removeContentEntityFromCache(contentEntityType, contentEntityId+markdownFileExtension)
	processAndHandleStats(config, resLoader, true)
	wChan <- watchReloadData{
		Type: contentEntityType,
		Id:   contentEntityId,
		Op:   dwEvent.op,
	}
}

func handleMediaDirWatchEvent(dwEvent dirWatchEvent, contentEntityType contentEntityType, contentEntityId string, config appConfig, resLoader resourceLoader, wChan chan watchReloadData) {
	ceType := strings.ToLower(contentEntityType.String())
	var removedImageFileName *string
	switch dwEvent.op {
	case dirWatchOpCreate:
		sprintln(" - [watch] "+ceType+" media file created: ", dwEvent.filePath)
	case dirWatchOpUpdate:
		sprintln(" - [watch] "+ceType+" media file updated: ", dwEvent.filePath)
	case dirWatchOpRename:
		sprintln(" - [watch] "+ceType+" media file renamed: ",
			"   original file: "+*dwEvent.originalFilePath,
			"   renamed to: "+dwEvent.filePath)
		originalMediaFileName := filepath.Base(*dwEvent.originalFilePath)
		originalMediaFileNameExt := filepath.Ext(originalMediaFileName)
		if slices.Contains(thumbImageFileExtensions, originalMediaFileNameExt) {
			removedImageFileName = &originalMediaFileName
		}
	case dirWatchOpDelete:
		sprintln(" - [watch] "+ceType+" media file deleted: ", dwEvent.filePath)
		removedMediaFileName := filepath.Base(dwEvent.filePath)
		removedMediaFileNameExt := filepath.Ext(removedMediaFileName)
		if slices.Contains(thumbImageFileExtensions, removedMediaFileNameExt) {
			removedImageFileName = &removedMediaFileName
		}
	}
	if removedImageFileName != nil {
		mediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, ceType, os.PathSeparator, contentEntityId)
		if dirExists(mediaDirPath) {
			// ==================================================
			// remove the original/old image file thumbnails
			// ==================================================
			imageFileNames, err := listFilesByExt(mediaDirPath, thumbImageFileExtensions...)
			if err == nil && len(imageFileNames) > 0 {
				for _, imageFileName := range imageFileNames {
					if strings.HasPrefix(imageFileName, *removedImageFileName) {
						oldThumbFilePath := fmt.Sprintf("%s%c%s", mediaDirPath, os.PathSeparator, imageFileName)
						err = os.Remove(oldThumbFilePath)
						if err != nil {
							sprintln(" - error deleting old thumbnail file: "+oldThumbFilePath, err)
						} else {
							sprintln(" - deleted old thumbnail file: " + oldThumbFilePath)
						}
					}
				}
			}
			// ==================================================
		}
	}
	mdContentFileName := contentEntityId + markdownFileExtension
	mdContentFilePath := fmt.Sprintf("%s%c%s", ceType+"s", os.PathSeparator, mdContentFileName)
	if fileExists(mdContentFilePath) {
		removeContentEntityFromCache(contentEntityType, mdContentFileName)
		processAndHandleStats(config, resLoader, true)
		wChan <- watchReloadData{
			Type: contentEntityType,
			Id:   contentEntityId,
			Op:   dirWatchOpUpdate,
		}
	} else {
		processOriginalMediaFile(dwEvent.filePath, config, false)
		mediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, ceType, os.PathSeparator, contentEntityId)
		processImgThumbnails(mediaDirPath, config)
	}
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
	case commandThemeActionActivate:
		if !themeInstalled {
			sprintln("theme is not installed: " + theme)
		} else {
			config.theme = themeDir
			writeConfig(config)
			sprintln(" - " + configFileName + " updated to activate new theme: " + theme)
		}
	case commandThemeActionInstall:
		if themeInstalled {
			sprintln("theme is already installed: " + theme)
		} else {
			sprintln(" - installing theme: " + theme)
			themeUrl := fmt.Sprintf("%s/%s", defaultGitHubRepoThemesUrl, theme)
			err := download(themeUrl, themeDir)
			if err != nil {
				println(fmt.Sprintf("error installing theme:\n\n%s", err.Error()))
			} else {
				copyThemeIncludes(theme)
			}
		}
	case commandThemeActionUpdate:
		if !themeInstalled {
			sprintln("theme is not installed: " + theme)
		} else {
			println(" - updating theme: " + theme)
			themeUrl := fmt.Sprintf("%s/%s", defaultGitHubRepoThemesUrl, theme)
			themeDlDir := themeDir + downloadedThemeDirSuffix
			deleteIfExists(themeDlDir)
			err := download(themeUrl, themeDlDir)
			if err != nil {
				println(fmt.Sprintf("error updating theme:\n\n%s", err.Error()))
			} else {
				recreateDir(themeDir)
				copyDir(themeDlDir, themeDir)
				deleteIfExists(themeDlDir)
				copyThemeIncludes(theme)
			}
		}
	case commandThemeActionRefresh:
		if !themeInstalled {
			sprintln("theme is not installed: " + theme)
		} else {
			println(" - refreshing theme: " + theme)
			copyThemeIncludes(theme)
		}
	case commandThemeActionDelete:
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
		sprintln("error: invalid theme command <action> argument: " + action)
		usageHelp := "usage:\n\n" + commandTheme.usage
		usage(usageHelp, 1)
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

func _deploy(config appConfig, commandArgs ...string) {
	if config.deployPath == "" {
		sprintln("error: no deploy path specified in the config file")
	} else {
		deployDestination := config.deployPath
		if config.deployHost != "" {
			deployDestination = config.deployHost + ":" + deployDestination
			if config.deployUsername != "" {
				deployDestination = config.deployUsername + "@" + deployDestination
			}
		}
		rsyncDeploy(deployDestination)
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
