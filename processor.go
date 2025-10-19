package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"text/template"
	"time"

	"github.com/gorilla/feeds"
)

func process(pages []page, posts []post,
	resLoader resourceLoader, handleOutput processorOutputHandler) stats {
	var searchIndex = mapSlice{}
	pageCnt := processPages(pages, &searchIndex, resLoader, handleOutput)
	postCnt, tagCnt := processPosts(posts, &searchIndex, resLoader, handleOutput)
	config := resLoader.config
	if len(config.generateFeeds) > 0 {
		generateFeeds(posts, config, handleOutput)
	}
	if config.enableSearch {
		sprintln(" - generating search files ...")
		searchIndexJson, err := json.Marshal(searchIndex)
		check(err)
		searchIndexOutputFilePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, searchIndexFileName)
		writeDataToFileIfChanged(searchIndexOutputFilePath, searchIndexJson)
		searchTemplate := compileSearchTemplate(resLoader)
		outputFilePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, searchPageFileName)
		var searchContentBuffer bytes.Buffer
		err = searchTemplate.Execute(&searchContentBuffer, templateContent{EntityType: Page, Title: config.siteName + " - Search", Config: buildTemplateConfigMap(config)})
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
			if !page.skipProcessing {
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
				err := pageTemplate.Execute(&pageContentBuffer, templateContent{EntityType: Page, Title: pTitle, FileName: outputFileName, Content: page, Config: buildTemplateConfigMap(resLoader.config)})
				check(err)

				if handleOutput != nil {
					handleOutput(outputFilePath, pageContentBuffer.Bytes())
				}
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
		tagTitleCnt := make(map[string]int)
		tagContent := make(map[string][]string)

		for _, post := range posts {
			pagePostCnt++
			pTitle := title
			if post.Title != "" {
				pTitle = title + " - " + post.Title
			}
			outputFileName := post.Id + contentFileExtension

			var postContentBuffer bytes.Buffer
			err := postContentTemplate.Execute(&postContentBuffer, templateContent{EntityType: Post, Title: pTitle, Content: post, FileName: outputFileName, Config: buildTemplateConfigMap(resLoader.config)})
			check(err)

			postContent := strings.TrimSpace(postContentBuffer.String())

			postPageContent += postContent

			if !post.skipProcessing {
				var singlePostContentBuffer bytes.Buffer
				err = postContentTemplate.Execute(&singlePostContentBuffer, templateContent{EntityType: Post, Title: pTitle, Content: post, Config: buildTemplateConfigMap(resLoader.config)})
				check(err)

				fullTemplate := compileFullTemplate(outputFileName, singlePostContentBuffer.String(), nil, resLoader)

				var singlePostFullContentBuffer bytes.Buffer
				err = fullTemplate.Execute(&singlePostFullContentBuffer, templateContent{EntityType: Post, Title: pTitle, Content: post, Config: buildTemplateConfigMap(resLoader.config)})
				check(err)

				outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployPostDirName, os.PathSeparator, outputFileName)
				if handleOutput != nil {
					handleOutput(outputFilePath, singlePostFullContentBuffer.Bytes())
				}
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
					tagTitleCnt[tag]++
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
			err := archiveTemplate.Execute(&archiveContentBuffer, templateContent{EntityType: Page, Title: config.siteName + " - Archive", Content: archIdxData, Config: buildTemplateConfigMap(config)})
			check(err)
			if handleOutput != nil {
				handleOutput(outputFilePath, archiveContentBuffer.Bytes())
			}
			processPaginatedPostContent(archivePostCnt, archiveContent, pageSize, deployArchiveDirName, pagerTemplate, resLoader, handleOutput)
		}

		tagPostCntLen := len(tagPostCnt)
		if tagPostCntLen > 0 {
			if config.generateTagIndex {
				minTagPostCnt := -1
				maxTagPostCnt := 0
				for _, v := range tagPostCnt {
					if minTagPostCnt == -1 || v < minTagPostCnt {
						minTagPostCnt = v
					}
					if v > maxTagPostCnt {
						maxTagPostCnt = v
					}
				}
				// ======================================================================
				// sort the tags by post count
				// ======================================================================
				var sortedTags []tagData
				for tt, tc := range tagTitleCnt {
					tr := float64(tc-minTagPostCnt) / float64(maxTagPostCnt-minTagPostCnt)
					tr = float64(int(tr*100))/100 + 1
					sortedTags = append(sortedTags, tagData{Title: tt, Count: tc, Ratio: tr})
				}
				sort.Slice(sortedTags, func(i, j int) bool {
					iCnt := sortedTags[i].Count
					jCnt := sortedTags[j].Count
					if iCnt == jCnt {
						return sortedTags[i].Title < sortedTags[j].Title
					}
					return iCnt > jCnt
				})
				// ======================================================================
				sprintln(" - generating tag index ...")
				tagIndexTemplate := compileTagIndexTemplate(resLoader)
				outputFilePath := fmt.Sprintf("%s%c%s%c%s", deployDirName, os.PathSeparator, deployTagsDirName, os.PathSeparator, indexPageFileName)
				var tagIndexContentBuffer bytes.Buffer
				err := tagIndexTemplate.Execute(&tagIndexContentBuffer, templateContent{EntityType: Page, Title: config.siteName + " - Tag Index", Content: sortedTags, Config: buildTemplateConfigMap(config)})
				check(err)
				if handleOutput != nil {
					handleOutput(outputFilePath, tagIndexContentBuffer.Bytes())
				}
			}
			sprintln(" - generating tag pages ...")
			tagCnt = tagPostCntLen
			processPaginatedPostContent(tagPostCnt, tagContent, pageSize, deployTagsDirName, pagerTemplate, resLoader, handleOutput)
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
	err := tmplt.Execute(&contentBuffer, templateContent{EntityType: ceType, Title: title, Config: buildTemplateConfigMap(resLoader.config)})
	check(err)
	if handleOutput != nil {
		handleOutput(outputFilePath, contentBuffer.Bytes())
	}
}

func processAndHandleStats(config appConfig, resLoader resourceLoader, useCache bool) {
	generatedCnt := 0
	stats := process(
		parsePages(config, resLoader, processImgThumbnails, useCache),
		parsePosts(config, resLoader, processImgThumbnails, useCache),
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

// generateFeeds creates RSS, Atom, and/or JSON feeds based on configuration
func generateFeeds(posts []post, config appConfig, handleOutput processorOutputHandler) {
	if len(config.generateFeeds) == 0 || len(posts) == 0 {
		return
	}

	sprintln(" - generating feeds ...")

	feedTitle := config.siteBaseURL
	if config.siteName != "" {
		feedTitle = config.siteName
	}

	feedDescription := config.siteBaseURL
	if config.siteDescription != "" {
		feedDescription = config.siteDescription
	} else if config.siteName != "" {
		feedDescription = config.siteName
	}

	// feed created timestamp is based on the date/time of the very first post
	feedCreated := getPostTimestamp(posts[len(posts)-1])

	// feed updated timestamp is based on the date/time of the most recent post
	feedUpdated := getPostTimestamp(posts[0])

	feed := &feeds.Feed{
		Title:       feedTitle,
		Link:        &feeds.Link{Href: config.siteBaseURL},
		Description: feedDescription,
		Created:     feedCreated,
		Updated:     feedUpdated,
	}

	// add posts to feed (limit by feedPostCnt)
	postCount := config.feedPostCnt
	if postCount > len(posts) {
		postCount = len(posts)
	}

	for i := 0; i < postCount; i++ {
		p := posts[i]

		// check if post has a date
		if p.Date.IsZero() {
			exitWithError(fmt.Sprintf(errPostDateMissing, p.Id))
		}

		// build feed item title with date/time/title
		itemTitle := p.FmtDate() // always starts with date: "2023-08-01"

		if !p.Time.IsZero() {
			itemTitle += " " + p.FmtTime() // add time if present: "2023-08-01 19:15"
		}

		if p.Title != "" {
			itemTitle += " | " + p.Title // add title if present: "2023-08-01 19:15 | My Post"
		}

		// optimize images for feeds by replacing srcset with smallest thumbnails
		itemContent := optimizeImagesForFeeds(p.Body, config.thumbSizes)
		// convert relative URLs to absolute in post body
		itemContent = convertRelativeURLsToAbsolute(itemContent, config.siteBaseURL)

		// construct item URL (deployPostDirName is "post", not "deploy/post")
		itemURL := fmt.Sprintf("%s/%s/%s%s", config.siteBaseURL, deployPostDirName, p.Id, contentFileExtension)

		// create timestamp using date at noon UTC for better timezone compatibility
		createdTime := time.Date(
			p.Date.Year, p.Date.Month, p.Date.Day,
			12, 0, 0, 0, // noon UTC
			time.UTC,
		)

		feed.Items = append(feed.Items, &feeds.Item{
			Title:   itemTitle,
			Link:    &feeds.Link{Href: itemURL},
			Id:      itemURL,
			Content: itemContent,
			Created: createdTime,
		})
	}

	// generate requested feed formats
	for _, format := range config.generateFeeds {
		var feedContent string
		var feedFileName string
		var err error

		switch format {
		case feedFormatRSS:
			feedContent, err = feed.ToRss()
			feedFileName = feedFileNameRSS
		case feedFormatAtom:
			feedContent, err = feed.ToAtom()
			feedFileName = feedFileNameAtom
		case feedFormatJSON:
			feedContent, err = feed.ToJSON()
			feedFileName = feedFileNameJSON
		}

		if err != nil {
			exitWithError(fmt.Sprintf("failed to generate %s feed: %s", format, err.Error()))
		}

		outputFilePath := fmt.Sprintf("%s%c%s", deployDirName, os.PathSeparator, feedFileName)
		if handleOutput != nil {
			handleOutput(outputFilePath, []byte(feedContent))
		}
	}
}

func getPostTimestamp(post post) time.Time {
	if post.Date.IsZero() {
		exitWithError(fmt.Sprintf(errPostDateMissing, post.Id))
	}

	loc := time.Now().Location()
	hour, minute, sec := 12, 0, 0
	if !post.Time.IsZero() {
		hour = post.Time.Hour
		minute = post.Time.Minute
		sec = post.Time.Second
	}

	return time.Date(post.Date.Year, post.Date.Month, post.Date.Day, hour, minute, sec, 0, loc).UTC()
}

// optimizeImagesForFeeds replaces srcset attributes with smallest thumbnail in src for better feed reader compatibility
func optimizeImagesForFeeds(htmlContent string, thumbSizes []int) string {
	if len(thumbSizes) == 0 {
		return htmlContent
	}

	smallestThumbSize := thumbSizes[0]
	for _, size := range thumbSizes {
		if size < smallestThumbSize {
			smallestThumbSize = size
		}
	}

	htmlContent = imgWithSrcsetRegexp.ReplaceAllStringFunc(htmlContent, func(match string) string {
		// extract the srcset value to find the smallest thumbnail URL
		srcsetMatch := srcsetAttrRegexp.FindStringSubmatch(match)
		if len(srcsetMatch) < 2 {
			return match
		}

		srcset := srcsetMatch[1]

		// find the smallest thumbnail URL from srcset (format: "url1 480w, url2 960w, url3 1680w")
		// we want the first entry which should be the smallest
		smallestThumbURL := ""
		srcsetParts := srcsetFirstEntryRegexp.FindStringSubmatch(srcset)
		if len(srcsetParts) >= 2 {
			smallestThumbURL = srcsetParts[1]
		}

		if smallestThumbURL == "" {
			// fallback: construct smallest thumb URL from src attribute
			srcMatch := srcAttrRegexp.FindStringSubmatch(match)
			if len(srcMatch) >= 2 {
				srcURL := srcMatch[1]
				// if src already points to a thumbnail, extract base name and reconstruct with smallest size
				if thumbMatch := thumbPatternRegexp.FindStringSubmatch(srcURL); len(thumbMatch) >= 4 {
					smallestThumbURL = fmt.Sprintf("%s_%d_thumb%s", thumbMatch[1], smallestThumbSize, thumbMatch[3])
				} else {
					// src points to original image, keep it
					smallestThumbURL = srcURL
				}
			}
		}

		// extract other attributes (like alt, etc.)
		attrsMatch := imgAttrsRegexp.FindStringSubmatch(match)
		attrs := ""
		if len(attrsMatch) >= 2 {
			attrs = attrsMatch[1]
		}

		// rebuild img tag with smallest thumbnail in src and add inline styles for better feed reader layout
		if smallestThumbURL != "" {
			return fmt.Sprintf(`<img src="%s" width="%d" style="margin-bottom: 10px;" %s>`, smallestThumbURL, smallestThumbSize, attrs)
		}

		// fallback: just remove srcset
		return srcsetAttrRemovalRegexp.ReplaceAllString(match, "")
	})

	return htmlContent
}

// convertRelativeURLsToAbsolute converts all relative URLs in HTML content to absolute URLs
func convertRelativeURLsToAbsolute(htmlContent string, siteBaseURL string) string {
	// convert `href` relative URLs
	htmlContent = relativeURLHrefRegexp.ReplaceAllString(htmlContent, `href="`+siteBaseURL+`$1"`)
	// convert `src` relative URLs
	htmlContent = relativeURLSrcRegexp.ReplaceAllString(htmlContent, `src="`+siteBaseURL+`$1"`)
	return htmlContent
}

func buildTemplateConfigMap(config appConfig) map[string]any {
	configMap := map[string]any{
		"PageSize": config.pageSize,
	}
	if len(config.generateFeeds) > 0 {
		configMap["GenerateFeeds"] = config.generateFeeds
		configMap["SiteBaseURL"] = config.siteBaseURL
		configMap["SiteName"] = config.siteName
	}
	return configMap
}
