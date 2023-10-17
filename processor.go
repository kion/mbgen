package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
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
		log.Println(" - processing pages ...")

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
				panic(fmt.Errorf("home page not found: '%s'", homePage))
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
		log.Println(" - processing posts ...")

		title := resLoader.config.siteName
		homePage := resLoader.config.homePage

		pageSize := resLoader.config.pageSize

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

			var pagePostContentBuffer bytes.Buffer
			err = postContentTemplate.Execute(&pagePostContentBuffer, templateContent{EntityType: Post, Title: pTitle, Content: post, FileName: outputFileName})
			check(err)

			pagePostContent := strings.TrimSpace(pagePostContentBuffer.String())

			postPageContent += pagePostContent

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

			if len(post.Tags) > 0 {
				for _, tag := range post.Tags {
					t := strings.ToLower(tag)
					tagPostCnt[t]++
					tagContent[t] = append(tagContent[t], pagePostContent)
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

		tagPostCntLen := len(tagPostCnt)
		if tagPostCntLen > 0 {
			ptCounts.V2 = tagPostCntLen
			for tag, cnt := range tagPostCnt {
				totalTagPageCnt := cnt / pageSize
				if cnt%pageSize > 0 {
					totalTagPageCnt++
				}
				tagDirName := strings.ToLower(tag)
				tagIndexUri := "/" + deployTagDirName + "/" + tagDirName
				tpd := pagerData{
					CurrPageNum:   1,
					TotalPageCnt:  totalTagPageCnt,
					PageUriPrefix: tagIndexUri,
					IndexPageUri:  tagIndexUri,
				}
				tagPagePostCnt := 0
				tagPageContent := ""
				for i := 0; i < cnt; i++ {
					tagPagePostCnt++
					tagPageContent += tagContent[tag][i]
					if tagPagePostCnt == pageSize {
						var pagerBuffer bytes.Buffer
						err := pagerTemplate.Execute(&pagerBuffer, tpd)
						check(err)
						tagPageContent += pagerBuffer.String()
						var fileName string
						if tpd.CurrPageNum == 1 {
							fileName = indexPageFileName
						} else {
							fileName = strconv.Itoa(tpd.CurrPageNum) + contentFileExtension
						}
						outputFilePath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, deployTagDirName, os.PathSeparator, tagDirName, os.PathSeparator, fileName)
						processContent(fileName, Post, tagPageContent, outputFilePath, resLoader, handleOutput)
						tagPagePostCnt = 0
						tagPageContent = ""
						tpd.CurrPageNum++
					}
				}
				if tagPagePostCnt > 0 {
					tagDirName := strings.ToLower(tag)
					var fileName string
					if tpd.CurrPageNum == 1 {
						fileName = indexPageFileName
					} else {
						fileName = strconv.Itoa(tpd.CurrPageNum) + contentFileExtension
						var pagerBuffer bytes.Buffer
						err := pagerTemplate.Execute(&pagerBuffer, tpd)
						check(err)
						tagPageContent += pagerBuffer.String()
					}
					outputFilePath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, deployTagDirName, os.PathSeparator, tagDirName, os.PathSeparator, fileName)
					processContent(fileName, Post, tagPageContent, outputFilePath, resLoader, handleOutput)
				}
			}
		}
	}
	channel <- ptCounts
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
