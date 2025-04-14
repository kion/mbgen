package main

import (
	"fmt"
	"gopkg.in/yaml.v3"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
)

func defaultConfig() appConfig {
	return appConfig{
		generateArchive:     defaultGenerateArchive,
		generateTagIndex:    defaultGenerateTagIndex,
		enableSearch:        defaultEnableSearch,
		pageSize:            defaultPageSize,
		resizeOrigImages:    defaultResizeOrigImages,
		maxImgSize:          defaultMaxImgSize,
		useThumbs:           defaultUseThumbs,
		thumbSizes:          defaultThumbSizes,
		thumbThreshold:      defaultThumbThreshold,
		jpegQuality:         defaultJPEGQuality,
		pngCompressionLevel: DefaultCompression,
		serveHost:           defaultServeHost,
		servePort:           defaultServePort,
	}
}

func readConfig() appConfig {
	if !fileExists(configFileName) {
		exitWithError(configFileName + " not found")
	}

	config := defaultConfig()
	configFile, err := os.ReadFile(configFileName)
	check(err)
	cm := make(map[string]string)
	err = yaml.Unmarshal(configFile, &cm)
	check(err)

	config.siteName = cm["siteName"]

	config.theme = cm["theme"]

	if config.theme == "" {
		exitWithError("missing config `theme` property value")
	}

	config.homePage = cm["homePage"]

	generateArchive := cm["generateArchive"]
	if generateArchive != "" {
		v := strings.ToLower(generateArchive)
		config.generateArchive = v != "no" && v != "false"
	}

	generateTagIndex := cm["generateTagIndex"]
	if generateTagIndex != "" {
		v := strings.ToLower(generateTagIndex)
		config.generateTagIndex = v != "no" && v != "false"
	}

	enableSearch := cm["enableSearch"]
	if enableSearch != "" {
		v := strings.ToLower(enableSearch)
		config.enableSearch = v != "no" && v != "false"
	}

	pageSize := cm["pageSize"]
	if pageSize != "" {
		ps, err := strconv.Atoi(pageSize)
		if err != nil || ps <= 0 {
			println(
				" - invalid config thumb size value: "+pageSize,
				" - will use the default value instead",
			)
		} else {
			config.pageSize = ps
		}
	}

	resizeOrigImages := cm["resizeOrigImages"]
	if resizeOrigImages != "" {
		v := strings.ToLower(resizeOrigImages)
		config.resizeOrigImages = v != "no" && v != "false"
	}

	maxImgSize := cm["maxImgSize"]
	if maxImgSize != "" {
		mis, err := strconv.Atoi(maxImgSize)
		if err != nil || mis < minAllowedMaxImgSize {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			} else {
				errMsg += fmt.Sprintf(" (min allowed max image size value: %d)", minAllowedMaxImgSize)
			}
			println(
				" - invalid config max image size value: "+maxImgSize+errMsg,
				" - will use the default value instead",
			)
		} else {
			config.maxImgSize = mis
		}
	}

	useThumbs := cm["useThumbs"]
	if useThumbs != "" {
		v := strings.ToLower(useThumbs)
		config.useThumbs = v != "no" && v != "false"
	}

	thumbSizes := cm["thumbSizes"]
	if thumbSizes != "" {
		var sizes []int
		tSizes := strings.Split(thumbSizes, ",")
		for _, ts := range tSizes {
			s, err := strconv.Atoi(strings.TrimSpace(ts))
			if err != nil || s < minAllowedThumbWidth {
				var errMsg string
				if err != nil {
					errMsg = err.Error()
				} else if s < minAllowedThumbWidth {
					errMsg += fmt.Sprintf(" (min allowed width value: %d)", minAllowedThumbWidth)
				}
				println(
					" - invalid config thumb widths value: "+thumbSizes+errMsg,
					" - will use the default widths instead",
				)
			} else {
				if !slices.Contains(sizes, s) {
					sizes = append(sizes, s)
				}
			}
		}
		if sizes != nil {
			config.thumbSizes = sizes
		}
	}

	thumbThreshold := cm["thumbThreshold"]
	if thumbThreshold != "" {
		tts, err := strconv.ParseFloat(thumbThreshold, 64)
		if err != nil || tts < minAllowedThumbThreshold {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			} else if tts < minAllowedThumbThreshold {
				errMsg += fmt.Sprintf(" (min allowed value: %.2f)", minAllowedThumbThreshold)
			}
			println(
				" - invalid config thumb threshold size value: "+thumbThreshold+errMsg,
				" - will use the default value instead",
			)
		} else {
			config.thumbThreshold = tts
		}
	}

	jpegQuality := cm["jpegQuality"]
	if jpegQuality != "" {
		jq, err := strconv.Atoi(jpegQuality)
		if err != nil || jq < minAllowedJPEGQuality || jq > maxAllowedJPEGQuality {
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			} else if jq < minAllowedJPEGQuality || jq > maxAllowedJPEGQuality {
				errMsg += fmt.Sprintf(" (allowed range: %d - %d)", minAllowedJPEGQuality, maxAllowedJPEGQuality)
			}
			println(
				" - invalid config jpeg quality value: "+jpegQuality+errMsg,
				" - will use the default value instead",
			)
		} else {
			config.jpegQuality = jq
		}
	}

	pngCompressionLevel := cm["pngCompressionLevel"]
	if pngCompressionLevel != "" {
		pcl := pngCompressionLevelFromString(pngCompressionLevel)
		if pcl == "" {
			println(
				" - invalid config png compression level value: "+pngCompressionLevel+" (allowed values: "+strings.Join(pngCompressionLevelStringValues(), ", ")+")",
				" - will use the default value instead",
			)
		} else {
			config.pngCompressionLevel = pcl
		}
	}

	if serveHost, ok := cm["serveHost"]; ok && serveHost != "" {
		config.serveHost = serveHost
	}

	if servePort, ok := cm["servePort"]; ok && servePort != "" {
		config.servePort, err = strconv.Atoi(servePort)
		if err != nil {
			println(
				" - invalid config serve port value: "+servePort,
				" - will use the default value instead",
			)
		}
	}

	if deployPath, ok := cm["deployPath"]; ok && deployPath != "" {
		config.deployPath = deployPath
	}

	if deployHost, ok := cm["deployHost"]; ok && deployHost != "" {
		config.deployHost = deployHost
	}

	if deployUsername, ok := cm["deployUsername"]; ok && deployUsername != "" {
		config.deployUsername = deployUsername
	}

	return config
}

func writeConfig(config appConfig) {
	yml := "siteName: "
	escapeSiteName := strings.Contains(config.siteName, ":")
	if escapeSiteName {
		yml += "\"" + config.siteName + "\""
	} else {
		yml += config.siteName
	}

	yml += "\n"
	yml += "theme: " + config.theme

	homePage := config.homePage
	yml += "\n"
	if homePage == "" {
		yml += "#homePage: "
	} else {
		yml += "homePage: " + config.homePage
	}

	yml += "\n"
	var generateArchive bool
	if defaultGenerateArchive == config.generateArchive {
		generateArchive = defaultGenerateArchive
		yml += "#generateArchive: "
	} else {
		generateArchive = config.generateArchive
		yml += "generateArchive: "
	}
	if generateArchive {
		yml += "yes"
	} else {
		yml += "no"
	}

	yml += "\n"
	var generateTagIndex bool
	if defaultGenerateTagIndex == config.generateTagIndex {
		generateTagIndex = defaultGenerateTagIndex
		yml += "#generateTagIndex: "
	} else {
		generateTagIndex = config.generateTagIndex
		yml += "generateTagIndex: "
	}
	if generateTagIndex {
		yml += "yes"
	} else {
		yml += "no"
	}

	yml += "\n"
	var enableSearch bool
	if defaultEnableSearch == config.enableSearch {
		enableSearch = defaultEnableSearch
		yml += "#enableSearch: "
	} else {
		enableSearch = config.enableSearch
		yml += "enableSearch: "
	}
	if enableSearch {
		yml += "yes"
	} else {
		yml += "no"
	}

	yml += "\n"
	if defaultPageSize == config.pageSize {
		yml += "#pageSize: " + strconv.Itoa(defaultPageSize)
	} else {
		yml += "pageSize: " + strconv.Itoa(config.pageSize)
	}

	yml += "\n"
	var resizeOrigImages bool
	if defaultResizeOrigImages == config.resizeOrigImages {
		resizeOrigImages = defaultResizeOrigImages
		yml += "#resizeOrigImages: "
	} else {
		resizeOrigImages = config.resizeOrigImages
		yml += "resizeOrigImages: "
	}
	if resizeOrigImages {
		yml += "yes"
	} else {
		yml += "no"
	}

	yml += "\n"
	if defaultMaxImgSize == config.maxImgSize {
		yml += "#maxImgSize: " + strconv.Itoa(defaultMaxImgSize)
	} else {
		yml += "maxImgSize: " + strconv.Itoa(config.maxImgSize)
	}

	yml += "\n"
	var useThumbs bool
	if defaultUseThumbs == config.useThumbs {
		useThumbs = defaultUseThumbs
		yml += "#useThumbs: "
	} else {
		useThumbs = config.useThumbs
		yml += "useThumbs: "
	}
	if useThumbs {
		yml += "yes"
	} else {
		yml += "no"
	}

	yml += "\n"
	sort.Slice(config.thumbSizes, func(i, j int) bool {
		return i < j
	})
	if slices.Equal(defaultThumbSizes, config.thumbSizes) {
		yml += "#thumbSizes: " + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(defaultThumbSizes)), ", "), "[]")
	} else {
		yml += "thumbSizes: " + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(config.thumbSizes)), ", "), "[]")
	}

	yml += "\n"
	if defaultThumbThreshold == config.thumbThreshold {
		yml += "#thumbThreshold: " + strings.TrimRight(fmt.Sprintf("%.2f", defaultThumbThreshold), "0")
	} else {
		yml += "thumbThreshold: " + strings.TrimRight(fmt.Sprintf("%.2f", config.thumbThreshold), "0")
	}

	yml += "\n"
	if defaultJPEGQuality == config.jpegQuality {
		yml += "#jpegQuality: " + strconv.Itoa(defaultJPEGQuality)
	} else {
		yml += "jpegQuality: " + strconv.Itoa(config.jpegQuality)
	}

	yml += "\n"
	if defaultPNGCompressionLevel == config.pngCompressionLevel {
		yml += "#pngCompressionLevel: " + defaultPNGCompressionLevel.String()
	} else {
		yml += "pngCompressionLevel: " + config.pngCompressionLevel.String()
	}

	yml += "\n"
	if defaultServeHost == config.serveHost {
		yml += "#serveHost: " + defaultServeHost
	} else {
		yml += "serveHost: " + config.serveHost
	}

	yml += "\n"
	if defaultServePort == config.servePort {
		yml += "#servePort: " + strconv.Itoa(defaultServePort)
	} else {
		yml += "servePort: " + strconv.Itoa(config.servePort)
	}

	yml += "\n"
	if config.deployPath != "" {
		yml += "deployPath: " + config.deployPath
	} else {
		yml += "#deployPath: "
	}

	yml += "\n"
	if config.deployHost != "" {
		yml += "deployHost: " + config.deployHost
	} else {
		yml += "#deployHost: "
	}

	yml += "\n"
	if config.deployUsername != "" {
		yml += "deployUsername: " + config.deployUsername
	} else {
		yml += "#deployUsername: "
	}

	writeDataToFileIfChanged(configFileName, []byte(yml))
}

func printConfig(config appConfig) {
	sprintln("[ ------ config ------ ]\n")
	if config.siteName != "" {
		println(" - site name: " + config.siteName)
	}
	println(" - theme: " + config.theme)
	if config.homePage != "" {
		println(" - home page: " + config.homePage)
	}
	var generateArchive string
	if config.generateArchive {
		generateArchive = "yes"
	} else {
		generateArchive = "no"
	}
	println(" - generate archive: " + generateArchive)
	var enableSearch string
	if config.enableSearch {
		enableSearch = "yes"
	} else {
		enableSearch = "no"
	}
	println(" - enable search: " + enableSearch)
	println(fmt.Sprintf(" - page size: %d", config.pageSize))
	var resizeOrigImages string
	if config.resizeOrigImages {
		resizeOrigImages = "yes"
	} else {
		resizeOrigImages = "no"
	}
	println(" - resize original images: " + resizeOrigImages)
	println(fmt.Sprintf(" - max image size: %d", config.maxImgSize))
	var usingThumbs string
	if config.useThumbs {
		usingThumbs = "yes"
	} else {
		usingThumbs = "no"
	}
	println(" - use thumbs: " + usingThumbs)
	println(" - thumb sizes: " + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(config.thumbSizes)), ", "), "[]"))
	println(fmt.Sprintf(" - thumb threshold: %.2f", config.thumbThreshold))
	println(fmt.Sprintf(" - jpeg quality: %d", config.jpegQuality))
	println(" - png compression level: " + config.pngCompressionLevel.String())
	println(" - serve host: " + config.serveHost)
	println(fmt.Sprintf(" - serve port: %d", config.servePort))
	if config.deployPath != "" {
		println(" - deploy path: " + config.deployPath)
	}
	if config.deployHost != "" {
		println(" - deploy host: " + config.deployHost)
	}
	if config.deployUsername != "" {
		println(" - deploy username: " + config.deployUsername)
	}
	sprintln("[----------------------]")
}
