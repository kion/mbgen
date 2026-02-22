package app

import (
	"bytes"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strconv"
	"strings"

	"cloud.google.com/go/civil"
	"github.com/google/uuid"
	"github.com/yuin/goldmark"
	meta "github.com/yuin/goldmark-meta"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
)

var mainTemplateMarkup /* const */ string

var markdown = /* const */ goldmark.New(
	goldmark.WithExtensions(
		meta.Meta,
		extension.Strikethrough,
		extension.DefinitionList,
		extension.Table,
		extension.Linkify,
		// TODO: support an option to include extension.CJK (?)
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
	),
)

func parsePages(config appConfig, resLoader resourceLoader, thumbHandler imageThumbnailHandler, useCache bool) []page {
	if !dirExists(markdownPagesDirName) {
		return nil
	}

	markdownPageDirEntries, err := os.ReadDir(markdownPagesDirName)
	check(err)

	if len(markdownPageDirEntries) == 0 {
		return nil
	}

	sprintln(" - parsing pages ...")

	var pages []page
	for _, pageEntry := range markdownPageDirEntries {
		pageEntryInfo, err := pageEntry.Info()
		check(err)
		if !pageEntryInfo.IsDir() {
			pageEntryFileName := pageEntryInfo.Name()
			pageEntryModTime := pageEntryInfo.ModTime()
			if useCache {
				ce := getContentEntityFromCache(Page, pageEntryFileName, pageEntryModTime)
				if ce != nil {
					page := ce.(page)
					page.skipProcessing = true
					pages = append(pages, page)
					continue
				}
			}
			pageEntryPath := fmt.Sprintf("%s%c%s", markdownPagesDirName, os.PathSeparator, pageEntryFileName)
			content, err := os.ReadFile(pageEntryPath)
			check(err)
			pageId := pageEntryFileName[:len(pageEntryFileName)-len(filepath.Ext(pageEntryFileName))]
			pageMediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, deployPageDirName, os.PathSeparator, pageId)
			handleThumbnails(pageMediaDirPath, config, thumbHandler)
			page := parsePage(pageId, string(content), config, resLoader)
			if useCache {
				addContentEntityToCache(pageEntryFileName, pageEntryModTime, page)
			}
			pages = append(pages, page)
		}
	}

	return pages
}

func parsePosts(config appConfig, resLoader resourceLoader, thumbHandler imageThumbnailHandler, useCache bool) []post {
	if !dirExists(markdownPostsDirName) {
		return nil
	}

	markdownPostDirEntries, err := os.ReadDir(markdownPostsDirName)
	check(err)

	if len(markdownPostDirEntries) == 0 {
		return nil
	}

	sort.Slice(markdownPostDirEntries, func(i, j int) bool {
		return markdownPostDirEntries[i].Name() > markdownPostDirEntries[j].Name()
	})

	sprintln(" - parsing posts ...")

	var posts []post
	for _, postEntry := range markdownPostDirEntries {
		postEntryInfo, err := postEntry.Info()
		check(err)
		if !postEntryInfo.IsDir() {
			postEntryFileName := postEntryInfo.Name()
			postEntryModTime := postEntryInfo.ModTime()
			if useCache {
				ce := getContentEntityFromCache(Post, postEntryFileName, postEntryModTime)
				if ce != nil {
					post := ce.(post)
					post.skipProcessing = true
					posts = append(posts, post)
					continue
				}
			}
			postEntryPath := fmt.Sprintf("%s%c%s", markdownPostsDirName, os.PathSeparator, postEntryFileName)
			content, err := os.ReadFile(postEntryPath)
			check(err)
			postId := postEntryFileName[:len(postEntryFileName)-len(filepath.Ext(postEntryFileName))]
			postMediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, deployPostDirName, os.PathSeparator, postId)
			handleThumbnails(postMediaDirPath, config, thumbHandler)
			post := parsePost(postId, string(content), config, resLoader)
			if useCache {
				addContentEntityToCache(postEntryFileName, postEntryModTime, post)
			}
			posts = append(posts, post)
		}
	}

	return posts
}

func parsePage(pageId string, content string, config appConfig, resLoader resourceLoader) page {
	page := page{Id: pageId}
	content, rawBodyContent, cdPhReps := parseContentDirectives(Page, pageId, content, config, resLoader)
	var buf bytes.Buffer
	context := parser.NewContext()
	err := markdown.Convert([]byte(content), &buf, parser.WithContext(context))
	check(err)
	page.Body = strings.TrimSpace(buf.String())
	page.Body = handleContentDirectivePlaceholderReplacements(page.Body, cdPhReps)
	metaData := meta.Get(context)
	rawTitle := ""
	if title, ok := metaData[metaDataKeyTitle].(string); ok {
		rawTitle = title
		if strings.Contains(title, "\n") {
			title = strings.Replace(title, "\n", "<br>", -1)
		}
		page.Title = title
	}
	page.SearchData = searchData{
		TypeId:  "page/" + page.Id,
		Content: strings.ToLower(rawTitle) + " " + strings.ToLower(rawBodyContent),
	}
	return page
}

func parsePost(postId string, content string, config appConfig, resLoader resourceLoader) post {
	post := post{Id: postId}
	// ================================================================================
	// replace metadata tabs with two spaces to avoid markdown parsing issues
	// ================================================================================
	metadataContent := metaDataPlaceholderRegexp.FindString(content)
	if metadataContent != "" {
		metadataContent = strings.Replace(metadataContent, "\t", "  ", -1)
		content = metaDataPlaceholderRegexp.ReplaceAllString(content, metadataContent)
	}
	// ================================================================================
	content, rawBodyContent, cdPhReps := parseContentDirectives(Post, postId, content, config, resLoader)
	post.FeedContent = rawBodyContent // store cleaned markdown for feed generation
	var buf bytes.Buffer
	context := parser.NewContext()
	err := markdown.Convert([]byte(content), &buf, parser.WithContext(context))
	check(err)
	post.Body = strings.TrimSpace(buf.String())
	post.Body = handleContentDirectivePlaceholderReplacements(post.Body, cdPhReps)
	metaData := meta.Get(context)
	if date, ok := metaData[metaDataKeyDate].(string); ok {
		d, err := civil.ParseDate(date)
		check(err)
		post.Date = d
	}
	if time, ok := metaData[metaDataKeyTime].(string); ok {
		if len(time) == 5 {
			time += ":00"
		}
		t, err := civil.ParseTime(time)
		check(err)
		post.Time = t
	}
	rawTitle := ""
	if title, ok := metaData[metaDataKeyTitle].(string); ok {
		rawTitle = title
		if strings.Contains(title, "\n") {
			title = strings.Replace(title, "\n", "<br>", -1)
		}
		post.Title = title
	}
	tags := metaData[metaDataKeyTags]
	if tags != nil {
		ti := tags.([]interface{})
		for _, v := range ti {
			tag := normalizeTagURI(v.(string))
			if !slices.Contains(post.Tags, tag) {
				post.Tags = append(post.Tags, tag)
			}
		}
	}
	post.SearchData = searchData{
		TypeId:  "post/" + post.Id,
		Content: strings.ToLower(rawTitle) + " " + strings.ToLower(rawBodyContent) + " " + strings.ToLower(strings.Join(post.Tags[:], " ")),
	}
	return post
}

func handleThumbnails(mediaDirPath string, config appConfig, thumbHandler imageThumbnailHandler) {
	if thumbHandler != nil {
		thumbHandler(mediaDirPath, config)
	}
}

func parseContentDirectives(ceType contentEntityType, ceId string, content string, config appConfig, resLoader resourceLoader) (string, string, map[string]string) {
	rawBodyContent := metaDataPlaceholderRegexp.ReplaceAllString(content, "")
	rawBodyContent = contentDirectivePlaceholderRegexp.ReplaceAllString(rawBodyContent, "")
	rawBodyContent = whitespacePlaceholderRegexp.ReplaceAllString(rawBodyContent, " ")
	rawBodyContent = strings.TrimSpace(rawBodyContent)
	var phReps map[string]string
	hashTagPlaceholders := hashTagRegex.FindAllStringSubmatch(content, -1)
	if hashTagPlaceholders != nil {
		for _, htp := range hashTagPlaceholders {
			placeholder := htp[0]
			tag := htp[1]
			replacement := fmt.Sprintf(hashTagMarkdownReplacementFormat, tag, normalizeTagURI(tag))
			content = strings.Replace(content, placeholder, replacement, 1)
		}
	}
	tagAutoLinkPlaceholders := tagAutoLinkPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if tagAutoLinkPlaceholders != nil {
		for _, talp := range tagAutoLinkPlaceholders {
			placeholder := talp[0]
			linkText := talp[1]
			tagURI := normalizeTagURI(linkText)
			replacement := fmt.Sprintf("[%s](/%s/%s/)", linkText, deployTagsDirName, tagURI)
			content = strings.Replace(content, placeholder, replacement, 1)
		}
	}
	tagLinkPlaceholders := tagLinkPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if tagLinkPlaceholders != nil {
		for _, tlp := range tagLinkPlaceholders {
			placeholder := tlp[0]
			tagText := strings.TrimSpace(tlp[1])
			link := "/" + deployTagsDirName + "/" + normalizeTagURI(tagText) + "/"
			content = strings.Replace(content, placeholder, link, 1)
		}
	}
	contentLinkPlaceholders := contentLinkPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if contentLinkPlaceholders != nil {
		for _, clp := range contentLinkPlaceholders {
			placeholder := clp[0]
			entityType := strings.ToLower(clp[1])
			entryId := clp[2]
			var ceType contentEntityType
			switch entityType {
			case "page":
				ceType = Page
			case "post":
				ceType = Post
			}
			var link string
			if ceType != UndefinedContentEntityType {
				link = "/" + strings.ToLower(ceType.String()) + "/" + entryId + contentFileExtension
			}
			content = strings.Replace(content, placeholder, link, 1)
		}
	}
	searchLinkPlaceholders := searchLinkPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if searchLinkPlaceholders != nil {
		for _, slp := range searchLinkPlaceholders {
			placeholder := slp[0]
			searchQuery := url.QueryEscape(strings.ToLower(strings.TrimSpace(slp[1])))
			searchQuery = strings.ReplaceAll(searchQuery, "+", "%20")
			link := "/" + searchPageFileName + "?q=" + searchQuery
			content = strings.Replace(content, placeholder, link, 1)
		}
	}
	var expListMedia []string
	wrapPlaceholders := wrapPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if wrapPlaceholders != nil {
		sortContentDirectivePlaceholders(wrapPlaceholders)
		for _, wp := range wrapPlaceholders {
			mediaArg := wp[4]
			if mediaArg != "" {
				for _, a := range strings.Split(mediaArg, ",") {
					m := strings.TrimSpace(a)
					expListMedia = append(expListMedia, m)
				}
			}
		}
	}
	mediaPlaceholders := mediaPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if mediaPlaceholders != nil {
		sortContentDirectivePlaceholders(mediaPlaceholders)
		for _, mp := range mediaPlaceholders {
			mediaArg := mp[3]
			if mediaArg != "" {
				for _, a := range strings.Split(mediaArg, ",") {
					m := strings.TrimSpace(a)
					expListMedia = append(expListMedia, m)
				}
			}
		}
	}
	if wrapPlaceholders != nil {
		for _, wp := range wrapPlaceholders {
			placeholder := wp[0]
			directive := wp[1]
			contentDirectiveTemplate, err := compileContentDirectiveTemplate(directive, resLoader)
			if err != nil {
				println(" - failed to process " + directive + " directive for " + ceId + ": " + err.Error())
			} else {
				propStr := strings.Trim(wp[2], "()")
				mediaArg := wp[4]
				text := strings.TrimSpace(wp[5])
				var buf bytes.Buffer
				err := markdown.Convert([]byte(text), &buf)
				check(err)
				text = strings.TrimSpace(buf.String())
				props := make(map[string]string)
				if propStr != "" {
					for _, pStr := range strings.Split(propStr, ",") {
						prop := strings.Split(strings.TrimSpace(pStr), "=")
						key := strings.TrimSpace(prop[0])
						val := strings.TrimSpace(prop[1])
						props[key] = val
					}
				}
				var mediaFileNames []string
				if mediaArg == "" {
					mediaFileNames = listAllMedia(ceType, ceId, expListMedia)
				} else {
					for _, a := range strings.Split(mediaArg, ",") {
						m := strings.TrimSpace(a)
						mediaFileNames = append(mediaFileNames, m)
					}
				}
				allMedia := parseMediaFileNames(mediaFileNames, ceType, ceId, config)
				var contentDirectiveMarkupBuffer bytes.Buffer
				err = contentDirectiveTemplate.Execute(&contentDirectiveMarkupBuffer, contentDirectiveData{
					Text:  text,
					Media: allMedia,
					Props: props,
				})
				check(err)
				ph := fmt.Sprintf(directivePlaceholderReplacementFormat, uuid.New().String())
				if phReps == nil {
					phReps = make(map[string]string)
				}
				phReps[ph] = strings.TrimSpace(contentDirectiveMarkupBuffer.String())
				content = strings.Replace(content, placeholder, ph, 1)
			}
		}
	}
	if mediaPlaceholders != nil {
		for _, mp := range mediaPlaceholders {
			placeholder := mp[0]
			propStr := strings.Trim(mp[1], "()")
			mediaArg := mp[3]
			props := make(map[string]string)
			if propStr != "" {
				for _, pStr := range strings.Split(propStr, ",") {
					prop := strings.Split(strings.TrimSpace(pStr), "=")
					key := strings.TrimSpace(prop[0])
					val := strings.TrimSpace(prop[1])
					props[key] = val
				}
			}
			var mediaFileNames []string
			if mediaArg == "" {
				mediaFileNames = listAllMedia(ceType, ceId, expListMedia)
			} else {
				for _, a := range strings.Split(mediaArg, ",") {
					m := strings.TrimSpace(a)
					mediaFileNames = append(mediaFileNames, m)
				}
			}
			allMedia := parseMediaFileNames(mediaFileNames, ceType, ceId, config)
			if allMedia != nil {
				inlineMediaTemplate := compileMediaTemplate(resLoader)
				var inlineMediaMarkupBuffer bytes.Buffer
				err := inlineMediaTemplate.Execute(&inlineMediaMarkupBuffer, contentDirectiveData{
					Media: allMedia,
					Props: props,
				})
				check(err)
				ph := fmt.Sprintf(directivePlaceholderReplacementFormat, uuid.New().String())
				if phReps == nil {
					phReps = make(map[string]string)
				}
				phReps[ph] = strings.TrimSpace(inlineMediaMarkupBuffer.String())
				content = strings.Replace(content, placeholder, ph, 1)
			} else {
				content = strings.Replace(content, placeholder, "", 1)
			}
		}
	}
	embedMediaPlaceholders := embedMediaPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if embedMediaPlaceholders != nil {
		for _, emp := range embedMediaPlaceholders {
			var em *embeddedMedia
			placeholder := emp[0]
			emUrl := emp[1]
			for _, emt := range embeddedMediaTypes {
				code := emt.getCode(emUrl)
				if code != "" {
					em = &embeddedMedia{
						MediaType: emt,
						Code:      code,
					}
					break
				}
			}
			if em != nil {
				inlineMediaTemplate := compileMediaTemplate(resLoader)
				var inlineMediaMarkupBuffer bytes.Buffer
				err := inlineMediaTemplate.Execute(&inlineMediaMarkupBuffer, contentDirectiveData{
					EmbeddedMedia: em,
				})
				check(err)
				ph := fmt.Sprintf(directivePlaceholderReplacementFormat, uuid.New().String())
				if phReps == nil {
					phReps = make(map[string]string)
				}
				phReps[ph] = strings.TrimSpace(inlineMediaMarkupBuffer.String())
				content = strings.Replace(content, placeholder, ph, 1)
			} else {
				content = strings.Replace(content, placeholder, "", 1)
			}
		}
	}
	return content, rawBodyContent, phReps
}

func sortContentDirectivePlaceholders(cdPlaceholders [][]string) {
	// ==================================================
	// directives with explicitly listed media
	// should be processed before the ones
	// that do not explicitly list any media
	// ==================================================
	sort.Slice(cdPlaceholders, func(i, j int) bool {
		return strings.Compare(
			cdPlaceholders[i][0],
			cdPlaceholders[j][0]) == -1
	})
	// ==================================================
}

func handleContentDirectivePlaceholderReplacements(content string, phReps map[string]string) string {
	if phReps != nil {
		for placeholder, replacement := range phReps {
			content = strings.Replace(content, placeholder, replacement, 1)
		}
	}
	return content
}

func listAllMedia(contentEntityType contentEntityType, contentEntityId string, skipFiles []string) []string {
	ceType := strings.ToLower(contentEntityType.String())
	var allMedia []string
	mediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, ceType, os.PathSeparator, contentEntityId)
	if dirExists(mediaDirPath) {
		videoFiles, err := listFilesByExt(mediaDirPath, videoFileExtensions...)
		check(err)
		for _, video := range videoFiles {
			if !slices.Contains(skipFiles, video) {
				allMedia = append(allMedia, video)
			}
		}
		imageFiles, err := listFilesByExt(mediaDirPath, imageFileExtensions...)
		check(err)
		for _, image := range imageFiles {
			if !slices.Contains(skipFiles, image) && !strings.Contains(image, thumbImgFileSuffix) {
				allMedia = append(allMedia, image)
			}
		}
	}
	return allMedia
}

func parseMediaFileNames(mediaFileNames []string, contentEntityType contentEntityType, contentEntityId string, config appConfig) []media {
	var allMedia []media
	ceType := strings.ToLower(contentEntityType.String())
	for _, mediaFileName := range mediaFileNames {
		if strings.Contains(mediaFileName, thumbImgFileSuffix) {
			continue
		}
		mediaUri := "/" + mediaDirName + "/" + ceType + "/" + contentEntityId + "/" + mediaFileName
		mediaFileExt := filepath.Ext(mediaFileName)
		var mType mediaType
		if slices.Contains(imageFileExtensions, mediaFileExt) {
			mType = Image
			imgFileExt := filepath.Ext(mediaFileName)
			var thumbs []thumb
			for _, thSize := range config.thumbSizes {
				thFileSuffix := "_" + strconv.Itoa(thSize) + thumbImgFileSuffix + imgFileExt
				thumbFile := mediaFileName + thFileSuffix
				thumbFilePath := fmt.Sprintf("%s%c%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, ceType, os.PathSeparator, contentEntityId, os.PathSeparator, thumbFile)
				if fileExists(thumbFilePath) {
					thumbUri := "/" + mediaDirName + "/" + ceType + "/" + contentEntityId + "/" + thumbFile
					thumbs = append(thumbs, thumb{
						Uri:  thumbUri,
						Size: thSize,
					})
				} else {
					thumbs = append(thumbs, thumb{
						Uri:  mediaUri,
						Size: thSize,
					})
				}
			}
			allMedia = append(allMedia, media{
				Type:   mType,
				Uri:    mediaUri,
				thumbs: thumbs,
			})
		} else if slices.Contains(videoFileExtensions, mediaFileExt) {
			mType = Video
			allMedia = append(allMedia, media{
				Type: mType,
				Uri:  mediaUri,
			})
		}
	}
	return allMedia
}
