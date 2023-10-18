package main

import (
	"errors"
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
		siteName:       "",
		theme:          "",
		homePage:       "",
		pageSize:       defaultPageSize,
		useThumbs:      defaultUseThumbs,
		thumbSizes:     defaultThumbSizes,
		thumbThreshold: defaultThumbThreshold,
		serveHost:      defaultServeHost,
		servePort:      defaultServePort,
	}
}

func readConfig() appConfig {
	if !fileExists(configFileName) {
		exitWithError(configFileName + " not found")
	}

	sprintln("[ ------ config ------ ]\n")

	config := defaultConfig()
	configFile, err := os.ReadFile(configFileName)
	check(err)
	cm := make(map[string]string)
	err = yaml.Unmarshal(configFile, &cm)
	check(err)

	config.siteName = cm["siteName"]
	if config.siteName != "" {
		fmt.Println(" - site name: " + config.siteName)
	}

	config.theme = cm["theme"]
	if config.theme == "" {
		panic(errors.New("missing config theme value"))
	} else {
		fmt.Println(" - theme: " + config.theme)
	}

	config.homePage = cm["homePage"]
	if config.homePage != "" {
		fmt.Println(" - home page: " + config.homePage)
	}

	pageSize := cm["pageSize"]
	if pageSize != "" {
		ps, err := strconv.Atoi(pageSize)
		if err != nil || ps <= 0 {
			fmt.Println(" - invalid config thumb size value: " + pageSize)
			fmt.Println(" - will use the default value instead")
		} else {
			config.pageSize = ps
		}
	}
	fmt.Println(fmt.Sprintf(" - page size: %d", config.pageSize))

	useThumbs := cm["useThumbs"]
	if useThumbs != "" {
		v := strings.ToLower(useThumbs)
		config.useThumbs = v != "no" && v != "false"
	}
	var usingThumbs string
	if config.useThumbs {
		usingThumbs = "yes"
	} else {
		usingThumbs = "no"
	}
	fmt.Println(" - using thumbs: " + usingThumbs)

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
				fmt.Println(" - invalid config thumb widths value: " + thumbSizes + errMsg)
				fmt.Println(" - will use the default widths instead")
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
	fmt.Println(" - thumb sizes: " + strings.Trim(strings.Join(strings.Fields(fmt.Sprint(config.thumbSizes)), ", "), "[]"))

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
			fmt.Println(" - invalid config thumb threshold size value: " + thumbThreshold + errMsg)
			fmt.Println(" - will use the default value instead")
		} else {
			config.thumbThreshold = tts
		}
	}
	fmt.Println(fmt.Sprintf(" - thumb threshold: %.2f", config.thumbThreshold))

	if serveHost, ok := cm["serveHost"]; ok && serveHost != "" {
		config.serveHost = serveHost
	}
	fmt.Println(" - serve host: " + config.serveHost)

	if servePort, ok := cm["servePort"]; ok && servePort != "" {
		config.servePort, err = strconv.Atoi(servePort)
		if err != nil {
			fmt.Println(" - invalid config serve port value: " + servePort)
			fmt.Println(" - will use the default value instead")
		}
	}
	fmt.Println(fmt.Sprintf(" - serve port: %d", config.servePort))

	sprintln("[----------------------]")
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
	if defaultPageSize == config.pageSize {
		yml += "#pageSize: " + strconv.Itoa(defaultPageSize)
	} else {
		yml += "pageSize: " + strconv.Itoa(config.pageSize)
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

	writeDataToFileIfChanged(configFileName, []byte(yml))
}
