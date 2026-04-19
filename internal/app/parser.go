package app

import (
	"bytes"
	"fmt"
	"html"
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
	gmhtml "github.com/yuin/goldmark/renderer/html"
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
		gmhtml.WithHardWraps(),
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
			raw := strings.TrimSpace(v.(string))
			if raw == "" {
				continue
			}
			if !slices.Contains(post.Tags, raw) {
				post.Tags = append(post.Tags, raw)
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

	phReps := make(map[string]string)
	var expListMedia []string

	// extract {cols}...{//} blocks first so inner {col}...{/} tokens don't get
	// misinterpreted as generic wrap directives by wrapPlaceholderRegexp downstream
	var colsBlocks []colsBlock
	colsMatches := colsPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	for _, m := range colsMatches {
		ph := fmt.Sprintf(directivePlaceholderReplacementFormat, uuid.New().String())
		colsBlocks = append(colsBlocks, colsBlock{ph: ph, weights: m[2], inner: m[3], original: m[0]})
		content = strings.Replace(content, m[0], ph, 1)
	}

	content = processInnerDirectives(content, ceType, ceId, config, resLoader, phReps, &expListMedia)

	for _, cb := range colsBlocks {
		rendered := processColsBlock(cb, ceType, ceId, config, resLoader, phReps, &expListMedia)
		phReps[cb.ph] = rendered
	}

	return content, rawBodyContent, phReps
}

func processInnerDirectives(content string, ceType contentEntityType, ceId string, config appConfig, resLoader resourceLoader, phReps map[string]string, expListMedia *[]string) string {
	content = hashTagRegex.ReplaceAllStringFunc(content, func(match string) string {
		tag := match[1:] // strip leading '#'; regex guarantees '#' + tag chars
		return fmt.Sprintf(hashTagMarkdownReplacementFormat, tag, normalizeTagURI(tag))
	})
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
			var linkCeType contentEntityType
			switch entityType {
			case "page":
				linkCeType = Page
			case "post":
				linkCeType = Post
			}
			var link string
			if linkCeType != UndefinedContentEntityType {
				link = "/" + strings.ToLower(linkCeType.String()) + "/" + entryId + contentFileExtension
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
	wrapPlaceholders := wrapPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if wrapPlaceholders != nil {
		sortContentDirectivePlaceholders(wrapPlaceholders)
		for _, wp := range wrapPlaceholders {
			if fileNames, _ := splitMediaArg(wp[4]); fileNames != nil {
				*expListMedia = append(*expListMedia, fileNames...)
			}
		}
	}
	mediaPlaceholders := mediaPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if mediaPlaceholders != nil {
		sortContentDirectivePlaceholders(mediaPlaceholders)
		for _, mp := range mediaPlaceholders {
			if fileNames, _ := splitMediaArg(mp[3]); fileNames != nil {
				*expListMedia = append(*expListMedia, fileNames...)
			}
		}
	}
	if wrapPlaceholders != nil {
		for _, wp := range wrapPlaceholders {
			placeholder := wp[0]
			directive := wp[1]
			if directive == "col" {
				// {col}...{/} is a child of a {cols}...{//} block, handled by processColsBlock;
				// skip here so the generic wrap pipeline doesn't try to compile content-col.html
				continue
			}
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
				var captions map[string]string
				if mediaArg == "" {
					mediaFileNames = listAllMedia(ceType, ceId, *expListMedia)
				} else {
					mediaFileNames, captions = splitMediaArg(mediaArg)
				}
				allMedia := parseMediaFileNames(mediaFileNames, ceType, ceId, config, mediaArg != "", captions)
				var contentDirectiveMarkupBuffer bytes.Buffer
				err = contentDirectiveTemplate.Execute(&contentDirectiveMarkupBuffer, contentDirectiveData{
					Text:  text,
					Media: allMedia,
					Props: props,
				})
				check(err)
				ph := fmt.Sprintf(directivePlaceholderReplacementFormat, uuid.New().String())
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
			var captions map[string]string
			if mediaArg == "" {
				mediaFileNames = listAllMedia(ceType, ceId, *expListMedia)
			} else {
				mediaFileNames, captions = splitMediaArg(mediaArg)
			}
			allMedia := parseMediaFileNames(mediaFileNames, ceType, ceId, config, mediaArg != "", captions)
			if allMedia != nil {
				inlineMediaTemplate := compileMediaTemplate(resLoader)
				var inlineMediaMarkupBuffer bytes.Buffer
				err := inlineMediaTemplate.Execute(&inlineMediaMarkupBuffer, contentDirectiveData{
					Media: allMedia,
					Props: props,
				})
				check(err)
				ph := fmt.Sprintf(directivePlaceholderReplacementFormat, uuid.New().String())
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
				phReps[ph] = strings.TrimSpace(inlineMediaMarkupBuffer.String())
				content = strings.Replace(content, placeholder, ph, 1)
			} else {
				content = strings.Replace(content, placeholder, "", 1)
			}
		}
	}
	return content
}

func processColsBlock(cb colsBlock, ceType contentEntityType, ceId string, config appConfig, resLoader resourceLoader, phReps map[string]string, expListMedia *[]string) string {
	weights, weightsErr := parseColsWeights(cb.weights)
	if weightsErr != "" {
		println(" - " + weightsErr + " for " + ceId + "; falling back to equal widths")
		weights = nil
	}

	// pre-process inner directives on the whole cols body first so that nested
	// {/} closers (e.g., from {with-media}...{/}) become UUID placeholders and
	// don't confuse the {col}...{/} matcher's lazy quantifier downstream
	inner := processInnerDirectives(cb.inner, ceType, ceId, config, resLoader, phReps, expListMedia)

	colMatches := colPlaceholderRegexp.FindAllStringSubmatch(inner, -1)
	if len(colMatches) == 0 {
		println(" - {cols} directive has no {col} children for " + ceId + "; leaving raw directive in content")
		return cb.original
	}

	if weights != nil && len(weights) != len(colMatches) {
		println(fmt.Sprintf(" - {cols} weight count (%d) does not match {col} child count (%d) for %s; falling back to equal widths", len(weights), len(colMatches), ceId))
		weights = nil
	}

	columns := make([]colData, 0, len(colMatches))
	for _, cm := range colMatches {
		propsStr := cm[2]
		body := cm[3]
		align := parseColAlign(propsStr, ceId)

		var buf bytes.Buffer
		err := markdown.Convert([]byte(strings.TrimSpace(body)), &buf)
		check(err)
		// Goldmark wraps standalone UUID placeholders in `<p>...</p>`, and with
		// WithHardWraps multiple consecutive placeholders collapse into a single
		// paragraph joined by `<br>` tags. When fixed-point replacement later
		// substitutes block-level directives (e.g., `{with-media}`), the result is
		// `<p><section>...</section>[<br><section>...</section>]*</p>`, which
		// browsers parse into empty `<p></p>` artifacts that misalign adjacent
		// columns. Strip the `<p>` wrapper (and any inter-placeholder `<br>`s) from
		// paragraphs whose content is only placeholders, whitespace, and `<br>`s.
		rendered := pWrapperAroundPlaceholdersRegexp.ReplaceAllStringFunc(buf.String(), func(match string) string {
			sub := pWrapperAroundPlaceholdersRegexp.FindStringSubmatch(match)
			if len(sub) < 2 {
				return match
			}
			inner := brTagRegexp.ReplaceAllString(sub[1], "\n")
			return strings.TrimSpace(inner)
		})
		columns = append(columns, colData{
			Body:  strings.TrimSpace(rendered),
			Align: align,
		})
	}

	gtc := buildGridTemplateColumns(weights, len(columns))

	tmplt, err := compileContentDirectiveTemplate("cols", resLoader)
	if err != nil {
		println(" - failed to process cols directive for " + ceId + ": " + err.Error())
		return cb.original
	}
	var out bytes.Buffer
	err = tmplt.Execute(&out, contentDirectiveData{
		Columns:             columns,
		GridTemplateColumns: gtc,
	})
	check(err)
	return strings.TrimSpace(out.String())
}

func parseColsWeights(raw string) ([]int, string) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, ""
	}
	parts := strings.Split(raw, ":")
	weights := make([]int, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		w, err := strconv.Atoi(p)
		if err != nil || w <= 0 {
			return nil, "failed to parse cols weight '" + p + "'"
		}
		weights = append(weights, w)
	}
	if len(weights) == 0 {
		return nil, ""
	}
	return weights, ""
}

func parseColAlign(propsStr, ceId string) string {
	propsStr = strings.TrimSpace(propsStr)
	if propsStr == "" {
		return ""
	}
	for _, pStr := range strings.Split(propsStr, ",") {
		pStr = strings.TrimSpace(pStr)
		if pStr == "" {
			continue
		}
		kv := strings.SplitN(pStr, "=", 2)
		if len(kv) != 2 {
			println(" - malformed {col} prop '" + pStr + "' for " + ceId)
			continue
		}
		key := strings.TrimSpace(kv[0])
		val := strings.TrimSpace(kv[1])
		if key == "a" {
			if val == "l" || val == "c" || val == "r" {
				return val
			}
			println(" - unknown {col} alignment value '" + val + "' for " + ceId + "; ignoring")
			continue
		}
		println(" - unknown {col} prop '" + key + "' for " + ceId + "; ignoring")
	}
	return ""
}

func buildGridTemplateColumns(weights []int, colCount int) string {
	parts := make([]string, 0, colCount)
	if weights == nil {
		for i := 0; i < colCount; i++ {
			parts = append(parts, "1fr")
		}
	} else {
		for _, w := range weights {
			parts = append(parts, fmt.Sprintf("%dfr", w))
		}
	}
	return strings.Join(parts, " ")
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
	if phReps == nil {
		return content
	}
	// iterate to a fixed point: a replacement may itself contain placeholders
	// (e.g., `{cols}` HTML embeds `{with-media}` UUIDs), and Go's map iteration order
	// is random; each UUID is unique, so termination is bounded by `len(phReps)`
	for {
		changed := false
		for placeholder, replacement := range phReps {
			if strings.Contains(content, placeholder) {
				content = strings.Replace(content, placeholder, replacement, 1)
				changed = true
			}
		}
		if !changed {
			break
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

func listSharedMedia() []string {
	mediaDirPath := fmt.Sprintf("%s%c%s%c%s",
		deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, sharedMediaDirName)
	var allMedia []string
	if dirExists(mediaDirPath) {
		videoFiles, err := listFilesByExt(mediaDirPath, videoFileExtensions...)
		check(err)
		allMedia = append(allMedia, videoFiles...)
		imageFiles, err := listFilesByExt(mediaDirPath, imageFileExtensions...)
		check(err)
		for _, image := range imageFiles {
			if !strings.Contains(image, thumbImgFileSuffix) {
				allMedia = append(allMedia, image)
			}
		}
	}
	return allMedia
}

func buildMediaItem(mediaFileName string, uriSubPath string, dirSubPath string, config appConfig) *media {
	mediaUri := "/" + mediaDirName + "/" + uriSubPath + "/" + mediaFileName
	mediaFileExt := strings.ToLower(filepath.Ext(mediaFileName))
	if slices.Contains(imageFileExtensions, mediaFileExt) {
		var thumbs []thumb
		for _, thSize := range config.thumbSizes {
			thFileSuffix := "_" + strconv.Itoa(thSize) + thumbImgFileSuffix + mediaFileExt
			thumbFile := mediaFileName + thFileSuffix
			thumbFilePath := fmt.Sprintf("%s%c%s%c%s%c%s",
				deployDirName, os.PathSeparator,
				mediaDirName, os.PathSeparator,
				dirSubPath, os.PathSeparator, thumbFile)
			if fileExists(thumbFilePath) {
				thumbs = append(thumbs, thumb{Uri: "/" + mediaDirName + "/" + uriSubPath + "/" + thumbFile, Size: thSize})
			} else {
				thumbs = append(thumbs, thumb{Uri: mediaUri, Size: thSize})
			}
		}
		return &media{Type: Image, Uri: mediaUri, thumbs: thumbs}
	} else if slices.Contains(videoFileExtensions, mediaFileExt) {
		return &media{Type: Video, Uri: mediaUri}
	}
	return nil
}

// splitMediaArg splits the inner argument of a `{media:...}` / `{with-media:...}`
// directive into entries and extracts an optional caption per entry;
// each entry has the form `<filename>` or `<filename>|<caption>|`;
// commas outside of a `|...|` block separate entries — commas inside a caption
// are preserved as part of the caption text; captions may not contain `|`;
// returns the list of filenames in input order and a map of filename -> caption for the entries that specified one
func splitMediaArg(mediaArg string) ([]string, map[string]string) {
	if strings.TrimSpace(mediaArg) == "" {
		return nil, nil
	}
	// ================================================================================
	// split on `,` but skip commas inside `|...|` caption blocks;
	// `inCaption` toggles on every `|`, so an opening `|` enters caption mode
	// and the matching closing `|` leaves it
	// ================================================================================
	var entries []string
	var cur strings.Builder
	inCaption := false
	for _, r := range mediaArg {
		switch r {
		case '|':
			inCaption = !inCaption
			cur.WriteRune(r)
		case ',':
			if inCaption {
				cur.WriteRune(r)
			} else {
				entries = append(entries, cur.String())
				cur.Reset()
			}
		default:
			cur.WriteRune(r)
		}
	}
	if cur.Len() > 0 {
		entries = append(entries, cur.String())
	}
	// ================================================================================
	var fileNames []string
	captions := make(map[string]string)
	for _, a := range entries {
		entry := strings.TrimSpace(a)
		if entry == "" {
			continue
		}
		fileName := entry
		if m := mediaArgEntryRegexp.FindStringSubmatch(entry); m != nil {
			fileName = strings.TrimSpace(m[1])
			if caption := strings.TrimSpace(m[2]); caption != "" {
				// templates use text/template (no auto-escape),
				// so escape here to ensure user-supplied caption text can't inject HTML
				captions[fileName] = html.EscapeString(caption)
			}
		}
		fileNames = append(fileNames, fileName)
	}
	return fileNames, captions
}

func parseMediaFileNames(mediaFileNames []string, contentEntityType contentEntityType, contentEntityId string, config appConfig, isExplicit bool, captions map[string]string) []media {
	var allMedia []media
	ceType := strings.ToLower(contentEntityType.String())
	for _, mediaFileName := range mediaFileNames {
		if strings.Contains(mediaFileName, thumbImgFileSuffix) {
			continue
		}
		uriSubPath := ceType + "/" + contentEntityId
		dirSubPath := ceType + string(os.PathSeparator) + contentEntityId
		if isExplicit {
			contentSpecificFilePath := fmt.Sprintf("%s%c%s%c%s%c%s",
				deployDirName, os.PathSeparator,
				mediaDirName, os.PathSeparator,
				dirSubPath, os.PathSeparator, mediaFileName)
			if !fileExists(contentSpecificFilePath) {
				sharedFilePath := fmt.Sprintf("%s%c%s%c%s%c%s",
					deployDirName, os.PathSeparator,
					mediaDirName, os.PathSeparator,
					sharedMediaDirName, os.PathSeparator, mediaFileName)
				if fileExists(sharedFilePath) {
					uriSubPath = sharedMediaDirName
					dirSubPath = sharedMediaDirName
				}
			}
		}
		if m := buildMediaItem(mediaFileName, uriSubPath, dirSubPath, config); m != nil {
			if caption, ok := captions[mediaFileName]; ok {
				m.Caption = caption
			}
			allMedia = append(allMedia, *m)
		}
	}
	return allMedia
}

func parseSharedMediaFileNames(mediaFileNames []string, config appConfig) []media {
	var allMedia []media
	for _, mediaFileName := range mediaFileNames {
		if strings.Contains(mediaFileName, thumbImgFileSuffix) {
			continue
		}
		if m := buildMediaItem(mediaFileName, sharedMediaDirName, sharedMediaDirName, config); m != nil {
			allMedia = append(allMedia, *m)
		}
	}
	return allMedia
}

// inspectTagTitleDuplicates returns a map of normalized tag URI -> sorted distinct
// original titles, for URIs that appear with more than one distinct title across the given posts.
func inspectTagTitleDuplicates(posts []post) map[string][]string {
	byUri := map[string]map[string]struct{}{}
	for _, p := range posts {
		for _, t := range p.Tags {
			uri := normalizeTagURI(t)
			if byUri[uri] == nil {
				byUri[uri] = map[string]struct{}{}
			}
			byUri[uri][t] = struct{}{}
		}
	}
	out := map[string][]string{}
	for uri, titles := range byUri {
		if len(titles) < 2 {
			continue
		}
		list := make([]string, 0, len(titles))
		for t := range titles {
			list = append(list, t)
		}
		sort.Strings(list)
		out[uri] = list
	}
	return out
}
