package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

func process(pages []page, posts []post,
	resLoader resourceLoader, handleOutput processorOutputHandler) stats {
	var searchIndex = mapSlice{}
	pageCnt := processPages(pages, &searchIndex, resLoader, handleOutput)
	postCnt, tagCnt := processPosts(posts, &searchIndex, resLoader, handleOutput)
	config := resLoader.config
	if config.enableSearch {
		sprintln(" - generating search files ...")
		searchIndexJson, err := json.Marshal(searchIndex)
		check(err)
		searchIndexOutputFilePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, searchIndexFileName)
		writeDataToFileIfChanged(searchIndexOutputFilePath, searchIndexJson)
		searchTemplate := compileSearchTemplate(resLoader)
		outputFilePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, searchPageFileName)
		var searchContentBuffer bytes.Buffer
		err = searchTemplate.Execute(&searchContentBuffer, templateContent{EntityType: Page, Title: config.siteName + " - Search", Config: map[string]any{"PageSize": config.pageSize}})
		check(err)
		if handleOutput != nil {
			handleOutput(outputFilePath, searchContentBuffer.Bytes())
		}
	}
	return stats{
		pageCnt: pageCnt,
		postCnt: postCnt,
		tagCnt:  tagCnt,
	}
}

func processPages(pages []page, searchIndex *mapSlice,
	resLoader resourceLoader, handleOutput processorOutputHandler) int {
	if pages != nil {
		sprintln(" - processing pages ...")

		title := resLoader.config.siteName
		homePage := resLoader.config.homePage

		if homePage != "" {
			found := false
			for _, page := range pages {
				if homePage == page.Id {
					found = true
					break
				}
			}
			if !found {
				exitWithError(fmt.Sprintf("home page not found: '%s'", homePage))
			}
		}

		for _, page := range pages {
			pageTemplate := compilePageTemplate(page, resLoader)
			outputFileName := page.Id + contentFileExtension
			var outputFilePath string
			if homePage == page.Id {
				outputFilePath = fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, indexPageFileName)
			} else {
				outputFilePath = fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPageDirName, os.PathSeparator, outputFileName)
			}

			pTitle := title
			if page.Title != "" {
				pTitle = title + " - " + page.Title
			}

			var pageContentBuffer bytes.Buffer
			err := pageTemplate.Execute(&pageContentBuffer, templateContent{EntityType: Page, Title: pTitle, FileName: outputFileName, Content: page})
			check(err)

			if handleOutput != nil {
				handleOutput(outputFilePath, pageContentBuffer.Bytes())
			}

			*searchIndex = append(*searchIndex, mapItem{Key: page.SearchData.TypeId, Value: page.SearchData.Content})
		}
	}
	return len(pages)
}

func processPosts(posts []post, searchIndex *mapSlice,
	resLoader resourceLoader, handleOutput processorOutputHandler) (int, int) {
	tagCnt := 0
	if posts != nil {
		sprintln(" - processing posts ...")

		config := resLoader.config

		title := config.siteName
		homePage := config.homePage

		pageSize := config.pageSize

		postContentTemplate := compilePostTemplate(resLoader)
		pagerTemplate := compilePagerTemplate(resLoader)

		postCnt := len(posts)
		totalPageCnt := postCnt / pageSize
		if postCnt%pageSize > 0 {
			totalPageCnt++
		}
		ppd := pagerData{
			CurrPageNum:   1,
			TotalPageCnt:  totalPageCnt,
			PageUriPrefix: "/" + deployPostsDirName,
			IndexPageUri:  "/",
		}

		if homePage != "" {
			ppd.IndexPageUri += deployPostsDirName + "/"
		}

		pagePostCnt := 0

		var postPageContent string

		var archIdxData archiveIndexData
		archivePostCnt := make(map[string]int)
		archiveContent := make(map[string][]string)

		tagPostCnt := make(map[string]int)
		tagContent := make(map[string][]string)

		for _, post := range posts {
			pagePostCnt++
			pTitle := title
			if post.Title != "" {
				pTitle = title + " - " + post.Title
			}
			outputFileName := post.Id + contentFileExtension

			var singlePostContentBuffer bytes.Buffer
			err := postContentTemplate.Execute(&singlePostContentBuffer, templateContent{EntityType: Post, Title: pTitle, Content: post})
			check(err)

			var postContentBuffer bytes.Buffer
			err = postContentTemplate.Execute(&postContentBuffer, templateContent{EntityType: Post, Title: pTitle, Content: post, FileName: outputFileName})
			check(err)

			postContent := strings.TrimSpace(postContentBuffer.String())

			postPageContent += postContent

			fullTemplate := compileFullTemplate(outputFileName, singlePostContentBuffer.String(), nil, resLoader)

			var singlePostFullContentBuffer bytes.Buffer
			err = fullTemplate.Execute(&singlePostFullContentBuffer, templateContent{EntityType: Post, Title: pTitle, Content: post})
			check(err)

			outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostDirName, os.PathSeparator, outputFileName)
			if handleOutput != nil {
				handleOutput(outputFilePath, singlePostFullContentBuffer.Bytes())
			}

			if pagePostCnt == pageSize {
				var pagerBuffer bytes.Buffer
				err = pagerTemplate.Execute(&pagerBuffer, ppd)
				check(err)
				postPageContent += pagerBuffer.String()
				if ppd.CurrPageNum == 1 {
					if homePage == "" {
						outputFilePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, indexPageFileName)
						processContent(indexPageFileName, Post, config.siteName, postPageContent, outputFilePath, resLoader, handleOutput)
					}
					outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostsDirName, os.PathSeparator, indexPageFileName)
					processContent(deployPostsDirName+"/"+indexPageFileName, Post, config.siteName, postPageContent, outputFilePath, resLoader, handleOutput)
				} else {
					fileName := strconv.Itoa(ppd.CurrPageNum) + contentFileExtension
					outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostsDirName, os.PathSeparator, fileName)
					processContent(fileName, Post, config.siteName, postPageContent, outputFilePath, resLoader, handleOutput)
				}
				pagePostCnt = 0
				postPageContent = ""
				ppd.CurrPageNum++
			}

			if config.generateArchive && !post.Date.IsZero() {
				year := post.Date.Year
				month := post.Date.Month
				yearIdx := slices.IndexFunc(archIdxData.YearData, func(data archiveYearData) bool {
					return data.Year == year
				})
				if yearIdx != -1 {
					monthData := archIdxData.YearData[yearIdx].MonthData
					monthIdx := slices.IndexFunc(monthData, func(data archiveMonthData) bool {
						return data.Month == month
					})
					if monthIdx != -1 {
						archIdxData.YearData[yearIdx].MonthData[monthIdx].PostCnt++
					} else {
						archIdxData.YearData[yearIdx].MonthData = append(archIdxData.YearData[yearIdx].MonthData,
							archiveMonthData{
								Month:   month,
								PostCnt: 1,
							})
					}
				} else {
					archIdxData.YearData = append(archIdxData.YearData, archiveYearData{
						Year: year,
						MonthData: []archiveMonthData{
							{
								Month:   month,
								PostCnt: 1,
							},
						},
					})
				}
				postYearAndMonth := formatYearAndMonth(year, month)
				archivePostCnt[postYearAndMonth]++
				archiveContent[postYearAndMonth] = append(archiveContent[postYearAndMonth], postContent)
			}

			if len(post.Tags) > 0 {
				for _, tag := range post.Tags {
					t := strings.ToLower(tag)
					tagPostCnt[t]++
					tagContent[t] = append(tagContent[t], postContent)
				}
			}

			*searchIndex = append(*searchIndex, mapItem{Key: post.SearchData.TypeId, Value: post.SearchData.Content})
		}

		if pagePostCnt > 0 {
			if ppd.CurrPageNum == 1 {
				if homePage == "" {
					outputFilePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, indexPageFileName)
					processContent(indexPageFileName, Post, config.siteName, postPageContent, outputFilePath, resLoader, handleOutput)
				}
				outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostsDirName, os.PathSeparator, indexPageFileName)
				processContent(deployPostsDirName+"/"+indexPageFileName, Post, config.siteName, postPageContent, outputFilePath, resLoader, handleOutput)
			} else {
				fileName := strconv.Itoa(ppd.CurrPageNum) + contentFileExtension
				outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostsDirName, os.PathSeparator, fileName)
				var pagerBuffer bytes.Buffer
				err := pagerTemplate.Execute(&pagerBuffer, ppd)
				check(err)
				postPageContent += pagerBuffer.String()
				processContent(fileName, Post, config.siteName, postPageContent, outputFilePath, resLoader, handleOutput)
			}
		}

		if config.generateArchive && len(archivePostCnt) > 0 {
			sprintln(" - generating archive ...")
			archiveTemplate := compileArchiveTemplate(resLoader)
			outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployArchiveDirName, os.PathSeparator, indexPageFileName)
			var archiveContentBuffer bytes.Buffer
			err := archiveTemplate.Execute(&archiveContentBuffer, templateContent{EntityType: Page, Title: config.siteName + " - Archive", Content: archIdxData})
			check(err)
			if handleOutput != nil {
				handleOutput(outputFilePath, archiveContentBuffer.Bytes())
			}
			processPaginatedPostContent(archivePostCnt, archiveContent, pageSize, deployArchiveDirName, pagerTemplate, resLoader, handleOutput)
		}

		tagPostCntLen := len(tagPostCnt)
		if tagPostCntLen > 0 {
			tagCnt = tagPostCntLen
			processPaginatedPostContent(tagPostCnt, tagContent, pageSize, deployTagDirName, pagerTemplate, resLoader, handleOutput)
		}
	}
	return len(posts), tagCnt
}

func processPaginatedPostContent(postCnt map[string]int, content map[string][]string, pageSize int,
	contentDeployDirName string, pagerTemplate *template.Template,
	resLoader resourceLoader, handleOutput processorOutputHandler) {
	postCntLen := len(postCnt)
	if postCntLen > 0 {
		for key, cnt := range postCnt {
			totalPageCnt := cnt / pageSize
			if cnt%pageSize > 0 {
				totalPageCnt++
			}
			subDirName := strings.ToLower(key)
			pdIndexUri := "/" + contentDeployDirName + "/" + subDirName
			pd := pagerData{
				CurrPageNum:   1,
				TotalPageCnt:  totalPageCnt,
				PageUriPrefix: pdIndexUri,
				IndexPageUri:  pdIndexUri,
			}
			pagePostCnt := 0
			pageContent := ""
			for i := 0; i < cnt; i++ {
				pagePostCnt++
				pageContent += content[key][i]
				if pagePostCnt == pageSize {
					var pagerBuffer bytes.Buffer
					err := pagerTemplate.Execute(&pagerBuffer, pd)
					check(err)
					pageContent += pagerBuffer.String()
					var fileName string
					if pd.CurrPageNum == 1 {
						fileName = indexPageFileName
					} else {
						fileName = strconv.Itoa(pd.CurrPageNum) + contentFileExtension
					}
					outputFilePath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, contentDeployDirName, os.PathSeparator, subDirName, os.PathSeparator, fileName)
					processContent(fileName, Post, resLoader.config.siteName+" - "+key, pageContent, outputFilePath, resLoader, handleOutput)
					pagePostCnt = 0
					pageContent = ""
					pd.CurrPageNum++
				}
			}
			if pagePostCnt > 0 {
				var fileName string
				if pd.CurrPageNum == 1 {
					fileName = indexPageFileName
				} else {
					fileName = strconv.Itoa(pd.CurrPageNum) + contentFileExtension
					var pagerBuffer bytes.Buffer
					err := pagerTemplate.Execute(&pagerBuffer, pd)
					check(err)
					pageContent += pagerBuffer.String()
				}
				outputFilePath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, contentDeployDirName, os.PathSeparator, subDirName, os.PathSeparator, fileName)
				processContent(fileName, Post, resLoader.config.siteName+" - "+key, pageContent, outputFilePath, resLoader, handleOutput)
			}
		}
	}
}

func processContent(templateName string, ceType contentEntityType, title string, content string, outputFilePath string, resLoader resourceLoader, handleOutput processorOutputHandler) {
	tmplt := compileFullTemplate(templateName, content, nil, resLoader)
	var contentBuffer bytes.Buffer
	err := tmplt.Execute(&contentBuffer, templateContent{EntityType: ceType, Title: title})
	check(err)
	if handleOutput != nil {
		handleOutput(outputFilePath, contentBuffer.Bytes())
	}
}

func processAndHandleStats(config appConfig, resLoader resourceLoader) {
	generatedCnt := 0
	stats := process(
		parsePages(config, resLoader, processImgThumbnails),
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
