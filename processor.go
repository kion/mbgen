package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

		// build feed item title
		itemTitle := buildFeedItemTitle(p)

		// build feed item content excerpt
		itemContent := buildFeedItemExcerpt(p, config)

		// construct item URL (deployPostDirName is "post", not "deploy/post")
		itemURL := fmt.Sprintf("%s/%s/%s%s", config.siteBaseURL, deployPostDirName, p.Id, contentFileExtension)

		// create timestamp using date at noon UTC for better timezone compatibility
		createdTime := time.Date(
			p.Date.Year, p.Date.Month, p.Date.Day,
			12, 0, 0, 0, // noon UTC
			time.UTC,
		)

		// prepend image to content if post has images
		mediaFileNames := listAllMedia(Post, p.Id, nil)
		if len(mediaFileNames) > 0 {
			mediaList := parseMediaFileNames(mediaFileNames, Post, p.Id, config)

			// read original markdown content to find first image reference
			rawContent := getRawPostContent(p.Id)
			if rawContent != "" {
				firstImage := getFirstImageFromContent(rawContent, mediaList)
				if firstImage != nil {
					// prefer smallest thumbnail over original
					imageUri := getSmallestThumbnailOrOriginal(*firstImage)
					imageAbsoluteUrl := config.siteBaseURL + imageUri

					// prepend image tag to content
					imageTag := fmt.Sprintf(`<img src="%s" />`, imageAbsoluteUrl)
					itemContent = imageTag + itemContent
				}
			}
		}

		// create feed item
		item := &feeds.Item{
			Title:   itemTitle,
			Link:    &feeds.Link{Href: itemURL},
			Id:      itemURL,
			Content: itemContent,
			Created: createdTime,
		}

		feed.Items = append(feed.Items, item)
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

// buildFeedItemTitle creates the title for a feed item
func buildFeedItemTitle(p post) string {
	itemTitle := p.FmtDate() // always starts with date

	if !p.Time.IsZero() {
		itemTitle += " " + p.FmtTime() // add time if present
	}

	if p.Title != "" {
		itemTitle += " | " + p.Title // add title if present
	} else if len(p.Tags) > 0 {
		// when no title, append tags
		var tagStrings []string
		for _, tag := range p.Tags {
			tagStrings = append(tagStrings, "#"+tag)
		}
		itemTitle += " | " + strings.Join(tagStrings, " ")
	}

	return itemTitle
}

// buildFeedItemExcerpt creates an HTML excerpt from the post's cleaned markdown content
// extracts up to 3 sentences, converts to HTML, and adds a "continue reading" link
func buildFeedItemExcerpt(p post, config appConfig) string {
	// extract first N sentences from the cleaned markdown
	sentences := extractSentences(p.FeedContent, feedExcerptSentenceCnt)

	var excerptMarkdown string
	if len(sentences) > 0 {
		excerptMarkdown = strings.Join(sentences, ". ")
		// add ellipsis if we truncated
		if len(sentences) == feedExcerptSentenceCnt || !strings.HasSuffix(p.FeedContent, sentences[len(sentences)-1]) {
			excerptMarkdown += "..."
		}
	} else {
		// fallback: use first N words if no sentences found
		excerptMarkdown = extractFirstNWords(p.FeedContent, feedExcerptFallbackWordCnt)
		if len(strings.Fields(p.FeedContent)) > feedExcerptFallbackWordCnt {
			excerptMarkdown += "..."
		}
	}

	// convert the markdown excerpt to HTML (preserves formatting)
	var buf bytes.Buffer
	err := markdown.Convert([]byte(excerptMarkdown), &buf)
	check(err)
	htmlExcerpt := strings.TrimSpace(buf.String())

	// convert relative URLs to absolute (for hashtag links, etc.)
	htmlExcerpt = convertRelativeURLsToAbsolute(htmlExcerpt, config.siteBaseURL)

	// build the "continue reading" link
	postURL := fmt.Sprintf("%s/%s/%s%s", config.siteBaseURL, deployPostDirName, p.Id, contentFileExtension)
	continueLink := fmt.Sprintf(`<p><a href="%s">%s</a></p>`, postURL, config.feedPostContinueReadingText)

	return htmlExcerpt + continueLink
}

// extractSentences splits text into sentences and returns up to maxSentences
func extractSentences(text string, maxSentences int) []string {
	var sentences []string
	var current strings.Builder

	runes := []rune(text)
	for i := 0; i < len(runes); i++ {
		current.WriteRune(runes[i])

		// check if we hit a sentence boundary (period followed by space/end, or end of text)
		if runes[i] == '.' {
			if i+1 >= len(runes) || runes[i+1] == ' ' || runes[i+1] == '\n' {
				sentence := strings.TrimSpace(current.String())
				if sentence != "" {
					sentences = append(sentences, sentence)
					if len(sentences) >= maxSentences {
						break
					}
				}
				current.Reset()
			}
		}
	}

	// add any remaining text as the last sentence if we haven't reached the limit
	if current.Len() > 0 && len(sentences) < maxSentences {
		sentence := strings.TrimSpace(current.String())
		if sentence != "" {
			sentences = append(sentences, sentence)
		}
	}

	return sentences
}

// extractFirstNWords extracts the first N words from text
func extractFirstNWords(text string, n int) string {
	words := strings.Fields(text)
	if len(words) <= n {
		return text
	}
	return strings.Join(words[:n], " ")
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

// convertRelativeURLsToAbsolute converts relative URLs in href attributes to absolute URLs
func convertRelativeURLsToAbsolute(htmlContent string, siteBaseURL string) string {
	return relativeURLHrefRegexp.ReplaceAllString(htmlContent, `href="`+siteBaseURL+`$1"`)
}

// getSmallestThumbnailOrOriginal returns the URI of the smallest thumbnail if available, otherwise the original media URI
func getSmallestThumbnailOrOriginal(m media) string {
	if len(m.thumbs) > 0 {
		// find smallest thumbnail by size
		smallest := m.thumbs[0]
		for _, thumb := range m.thumbs[1:] {
			if thumb.Size < smallest.Size {
				smallest = thumb
			}
		}
		return smallest.Uri
	}
	return m.Uri
}

// getFirstImageFromContent parses post content for media directives and returns the first referenced image
func getFirstImageFromContent(content string, mediaList []media) *media {
	if len(mediaList) == 0 {
		return nil
	}

	// check for {media:filename} or {media(props):filename} directives
	mediaMatches := mediaPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	for _, match := range mediaMatches {
		if len(match) > 3 && match[3] != "" {
			// has file specification (group 3 is the filename part)
			files := strings.Split(match[3], ",")
			if len(files) > 0 {
				filename := strings.TrimSpace(files[0])
				// find this file in mediaList
				for i := range mediaList {
					if mediaList[i].Type == Image && strings.Contains(mediaList[i].Uri, filename) {
						return &mediaList[i]
					}
				}
			}
		}
	}

	// check for {with-media:filename} style wrap directives
	wrapMatches := wrapPlaceholderOpeningRegexp.FindAllStringSubmatch(content, -1)
	for _, match := range wrapMatches {
		if len(match) > 4 && match[1] == "with-media" && match[4] != "" {
			// match[1] is the directive name, match[4] is the filename part
			files := strings.Split(match[4], ",")
			if len(files) > 0 {
				filename := strings.TrimSpace(files[0])
				// find this file in mediaList
				for i := range mediaList {
					if mediaList[i].Type == Image && strings.Contains(mediaList[i].Uri, filename) {
						return &mediaList[i]
					}
				}
			}
		}
	}

	// if no specific file reference, check for generic {media} directive
	if strings.Contains(content, "{media}") {
		// return first image from mediaList
		for i := range mediaList {
			if mediaList[i].Type == Image {
				return &mediaList[i]
			}
		}
	}

	return nil
}

// getRawPostContent reads the original markdown content of a post file
func getRawPostContent(postId string) string {
	postFilePath := filepath.Join(markdownPostsDirName, postId+markdownFileExtension)
	data, err := os.ReadFile(postFilePath)
	if err != nil {
		return ""
	}
	return string(data)
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
