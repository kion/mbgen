package main

import (
	"bytes"
	"fmt"
	"os"
	"slices"
	"strconv"
	"strings"
	"text/template"
)

func process(pages []page, posts []post,
	resLoader resourceLoader, handleOutput processorOutputHandler) stats {
	pagesChan := make(chan int)
	go processPages(pages, pagesChan, resLoader, handleOutput)
	postsChan := make(chan tuple2[int, int])
	go processPosts(posts, postsChan, resLoader, handleOutput)
	pageCnt := <-pagesChan
	ptCnt := <-postsChan
	return stats{
		pageCnt: pageCnt,
		postCnt: ptCnt.V1,
		tagCnt:  ptCnt.V2,
	}
}

func processPages(pages []page, channel chan int,
	resLoader resourceLoader, handleOutput processorOutputHandler) {
	if pages != nil {
		sprintln(" - processing pages ...")

		title := resLoader.config.siteName
		homePage := resLoader.config.homePage

		var homePageFileName string

		if homePage != "" {
			homePageFileName = homePage + contentFileExtension
			found := false
			for _, page := range pages {
				if homePageFileName == page.id+contentFileExtension {
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
			outputFileName := page.id + contentFileExtension
			var outputFilePath string
			if homePageFileName == page.id+contentFileExtension {
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
		}
	}

	channel <- len(pages)
}

func processPosts(posts []post, channel chan tuple2[int, int],
	resLoader resourceLoader, handleOutput processorOutputHandler) {
	ptCounts := tuple2[int, int]{V1: len(posts)}
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
			outputFileName := post.id + contentFileExtension

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
						processContent(indexPageFileName, Post, postPageContent, outputFilePath, resLoader, handleOutput)
					} else {
						outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostsDirName, os.PathSeparator, indexPageFileName)
						processContent(deployPostsDirName+"/"+indexPageFileName, Post, postPageContent, outputFilePath, resLoader, handleOutput)
					}
				} else {
					fileName := strconv.Itoa(ppd.CurrPageNum) + contentFileExtension
					outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostsDirName, os.PathSeparator, fileName)
					processContent(fileName, Post, postPageContent, outputFilePath, resLoader, handleOutput)
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
		}

		if pagePostCnt > 0 {
			var fileName, outputFilePath string
			if ppd.CurrPageNum == 1 {
				if homePage == "" {
					outputFilePath = fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, indexPageFileName)
				} else {
					outputFilePath = fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostsDirName, os.PathSeparator, indexPageFileName)
				}
			} else {
				fileName = strconv.Itoa(ppd.CurrPageNum) + contentFileExtension
				outputFilePath = fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostsDirName, os.PathSeparator, fileName)
				var pagerBuffer bytes.Buffer
				err := pagerTemplate.Execute(&pagerBuffer, ppd)
				check(err)
				postPageContent += pagerBuffer.String()
			}
			processContent(fileName, Post, postPageContent, outputFilePath, resLoader, handleOutput)
		}

		if config.generateArchive && len(archivePostCnt) > 0 {
			archiveTemplate := compileArchiveTemplate(resLoader)
			outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployArchiveDirName, os.PathSeparator, indexPageFileName)
			var archiveContentBuffer bytes.Buffer
			err := archiveTemplate.Execute(&archiveContentBuffer, templateContent{EntityType: Page, Title: "Archive", Content: archIdxData})
			check(err)
			if handleOutput != nil {
				handleOutput(outputFilePath, archiveContentBuffer.Bytes())
			}
			processPaginatedPostContent(archivePostCnt, archiveContent, pageSize, deployArchiveDirName, pagerTemplate, resLoader, handleOutput)
		}

		tagPostCntLen := len(tagPostCnt)
		if tagPostCntLen > 0 {
			ptCounts.V2 = tagPostCntLen
			processPaginatedPostContent(tagPostCnt, tagContent, pageSize, deployTagDirName, pagerTemplate, resLoader, handleOutput)
		}
	}
	channel <- ptCounts
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
					processContent(fileName, Post, pageContent, outputFilePath, resLoader, handleOutput)
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
				processContent(fileName, Post, pageContent, outputFilePath, resLoader, handleOutput)
			}
		}
	}
}

func processContent(templateName string, ceType contentEntityType, content string, outputFilePath string, resLoader resourceLoader, handleOutput processorOutputHandler) {
	tmplt := compileFullTemplate(templateName, content, nil, resLoader)
	var contentBuffer bytes.Buffer
	err := tmplt.Execute(&contentBuffer, templateContent{EntityType: ceType, Title: resLoader.config.siteName})
	check(err)
	if handleOutput != nil {
		handleOutput(outputFilePath, contentBuffer.Bytes())
	}
}
