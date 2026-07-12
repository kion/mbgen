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
	content, rawBodyContent, cdPhReps, warnings := parseContentDirectives(Page, pageId, content, config, resLoader)
	var buf bytes.Buffer
	context := parser.NewContext()
	err := markdown.Convert([]byte(content), &buf, parser.WithContext(context))
	check(err)
	page.Body = strings.TrimSpace(buf.String())
	page.Body = handleContentDirectivePlaceholderReplacements(page.Body, cdPhReps)
	page.Warnings = appendUnparsedDirectiveWarnings(warnings, page.Body)
	metaData := meta.Get(context)
	rawTitle := ""
	if title, ok := metaData[metaDataKeyTitle].(string); ok {
		rawTitle = title
		if strings.Contains(title, "\n") {
			title = strings.Replace(title, "\n", "<br>", -1)
		}
		page.Title = title
	}
	if metaCollection, ok := metaData[metaDataKeyMetaCollection].(string); ok {
		page.MetaCollection = strings.TrimSpace(metaCollection)
	}
	// collect the (deduplicated) collection URIs embedded via {collection:...} directives
	// — derived from the rendered body placeholders, which are deterministic and cache-safe
	for _, m := range collectionDirectivePlaceholderRegexp.FindAllStringSubmatch(page.Body, -1) {
		if !slices.Contains(page.CollectionRefs, m[1]) {
			page.CollectionRefs = append(page.CollectionRefs, m[1])
		}
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
	content, rawBodyContent, cdPhReps, warnings := parseContentDirectives(Post, postId, content, config, resLoader)
	post.FeedContent = rawBodyContent // store cleaned markdown for feed generation
	var buf bytes.Buffer
	context := parser.NewContext()
	err := markdown.Convert([]byte(content), &buf, parser.WithContext(context))
	check(err)
	post.Body = strings.TrimSpace(buf.String())
	post.Body = handleContentDirectivePlaceholderReplacements(post.Body, cdPhReps)
	post.Warnings = appendUnparsedDirectiveWarnings(warnings, post.Body)
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
	collRefs, collWarnings := parseCollectionsMetaData(metaData, postId, config)
	post.Collections = collRefs
	post.Warnings = append(post.Warnings, collWarnings...)
	if metaCollections := metaData[metaDataKeyMetaCollections]; metaCollections != nil {
		if mcList, ok := metaCollections.([]interface{}); ok {
			for _, v := range mcList {
				if title, ok := v.(string); ok {
					title = strings.TrimSpace(title)
					if title != "" && !slices.Contains(post.MetaCollections, title) {
						post.MetaCollections = append(post.MetaCollections, title)
					}
				} else {
					post.Warnings = append(post.Warnings, fmt.Sprintf("meta-collections: malformed entry: %v", v))
				}
			}
		} else {
			post.Warnings = append(post.Warnings, "meta-collections: malformed metadata (expected a list of meta collection names)")
		}
	}
	post.SearchData = searchData{
		TypeId:  "post/" + post.Id,
		Content: strings.ToLower(rawTitle) + " " + strings.ToLower(rawBodyContent) + " " + strings.ToLower(strings.Join(post.Tags[:], " ")),
	}
	return post
}

// toStringKeyMap normalizes a YAML-decoded map value: depending on the YAML library version,
// nested maps decode as either map[string]interface{} or map[interface{}]interface{}
func toStringKeyMap(v interface{}) (map[string]interface{}, bool) {
	switch m := v.(type) {
	case map[string]interface{}:
		return m, true
	case map[interface{}]interface{}:
		result := make(map[string]interface{}, len(m))
		for k, val := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}
			result[ks] = val
		}
		return result, true
	}
	return nil, false
}

// parseCollectionsMetaData parses the `collections` frontmatter section of a post:
// a map of collection name -> ordered list of items, where each item is either
// a bare string (no images) or a single-entry `name: image(s)` map (a string or a list of strings).
// Images resolve like all other post media (post media dir first, shared media fallback).
// Returned refs are sorted by collection title (YAML map order is not preserved by the metadata parser,
// and deterministic order keeps generated output stable across runs);
// item order within each collection is preserved as written.
func parseCollectionsMetaData(metaData map[string]interface{}, postId string, config appConfig) ([]postCollectionRef, []string) {
	raw := metaData[metaDataKeyCollections]
	if raw == nil {
		return nil, nil
	}
	var warnings []string
	collMap, ok := toStringKeyMap(raw)
	if !ok {
		warnings = append(warnings, "collections: malformed metadata (expected a map of collection name to item list)")
		return nil, warnings
	}
	collTitles := make([]string, 0, len(collMap))
	for collTitle := range collMap {
		collTitles = append(collTitles, collTitle)
	}
	sort.Strings(collTitles)
	var refs []postCollectionRef
	for _, collTitle := range collTitles {
		items, ok := collMap[collTitle].([]interface{})
		if !ok {
			warnings = append(warnings, "collections: malformed item list for collection \""+collTitle+"\" (expected a list of items)")
			continue
		}
		for _, entry := range items {
			var itemTitle string
			var imageRefs []string
			malformed := false
			switch e := entry.(type) {
			case string:
				itemTitle = strings.TrimSpace(e)
			default:
				entryMap, ok := toStringKeyMap(entry)
				if !ok || len(entryMap) != 1 {
					malformed = true
					break
				}
				for name, images := range entryMap {
					itemTitle = strings.TrimSpace(name)
					switch img := images.(type) {
					case nil:
						// `- Item Name:` — same as a bare string item
					case string:
						imageRefs = append(imageRefs, strings.TrimSpace(img))
					case []interface{}:
						for _, v := range img {
							if s, ok := v.(string); ok {
								imageRefs = append(imageRefs, strings.TrimSpace(s))
							} else {
								warnings = append(warnings, fmt.Sprintf("collections: malformed image reference %v (item \"%s\", collection \"%s\")", v, itemTitle, collTitle))
							}
						}
					default:
						warnings = append(warnings, fmt.Sprintf("collections: malformed image value %v (item \"%s\", collection \"%s\")", images, itemTitle, collTitle))
					}
				}
			}
			if malformed || itemTitle == "" {
				warnings = append(warnings, fmt.Sprintf("collections: malformed item entry in collection \"%s\": %v", collTitle, entry))
				continue
			}
			refs = append(refs, postCollectionRef{
				Collection: collTitle,
				Item:       itemTitle,
				Media:      resolveCollectionRefMedia(imageRefs, itemTitle, collTitle, postId, config, &warnings),
			})
		}
	}
	return refs, warnings
}

// resolveCollectionRefMedia resolves collection item image references
// against the post's media dir with a shared-media fallback (same semantics as explicit `{media:...}` references);
// warning on (and skipping) references that don't resolve to an existing image file
func resolveCollectionRefMedia(imageRefs []string, itemTitle string, collTitle string, postId string, config appConfig, warnings *[]string) []media {
	if len(imageRefs) == 0 {
		return nil
	}
	contentDir := filepath.Join(deployDirName, mediaDirName, strings.ToLower(Post.String()), postId)
	sharedDir := filepath.Join(deployDirName, mediaDirName, sharedMediaDirName)
	var result []media
	for _, ref := range imageRefs {
		if ref == "" {
			continue
		}
		ext := strings.ToLower(filepath.Ext(ref))
		var exists bool
		if slices.Contains(imageFileExtensions, ext) || slices.Contains(videoFileExtensions, ext) {
			exists = fileExists(filepath.Join(contentDir, ref)) || fileExists(filepath.Join(sharedDir, ref))
		} else {
			exists = len(expandMediaFileName(ref, contentDir)) > 0 || len(expandMediaFileName(ref, sharedDir)) > 0
		}
		if !exists {
			*warnings = append(*warnings, fmt.Sprintf("collections: unresolved image reference \"%s\" (item \"%s\", collection \"%s\")", ref, itemTitle, collTitle))
			continue
		}
		for _, m := range parseMediaFileNames([]string{ref}, Post, postId, config, true, nil) {
			if m.Type != Image {
				*warnings = append(*warnings, fmt.Sprintf("collections: image reference \"%s\" is not an image (item \"%s\", collection \"%s\")", ref, itemTitle, collTitle))
				continue
			}
			result = append(result, m)
		}
	}
	return result
}

func handleThumbnails(mediaDirPath string, config appConfig, thumbHandler imageThumbnailHandler) {
	if thumbHandler != nil {
		thumbHandler(mediaDirPath, config)
	}
}

func parseContentDirectives(ceType contentEntityType, ceId string, content string, config appConfig, resLoader resourceLoader) (string, string, map[string]string, []string) {
	rawBodyContent := metaDataPlaceholderRegexp.ReplaceAllString(content, "")
	rawBodyContent = contentDirectivePlaceholderRegexp.ReplaceAllString(rawBodyContent, "")
	rawBodyContent = whitespacePlaceholderRegexp.ReplaceAllString(rawBodyContent, " ")
	rawBodyContent = strings.TrimSpace(rawBodyContent)

	phReps := make(map[string]string)
	var expListMedia []string
	var warnings []string

	// extract {cols}...{//} blocks first so inner {col}...{/} tokens don't get
	// misinterpreted as generic wrap directives by wrapPlaceholderRegexp downstream
	var colsBlocks []colsBlock
	colsMatches := colsPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	for _, m := range colsMatches {
		ph := fmt.Sprintf(directivePlaceholderReplacementFormat, uuid.New().String())
		colsBlocks = append(colsBlocks, colsBlock{ph: ph, weights: m[2], inner: m[3], original: m[0]})
		content = strings.Replace(content, m[0], ph, 1)
	}

	content = processInnerDirectives(content, ceType, ceId, config, resLoader, phReps, &expListMedia, &warnings)

	for _, cb := range colsBlocks {
		rendered := processColsBlock(cb, ceType, ceId, config, resLoader, phReps, &expListMedia, &warnings)
		phReps[cb.ph] = rendered
	}

	return content, rawBodyContent, phReps, warnings
}

// resolveDirectiveMedia parses a media/with-media directive argument into rendered media items,
// collecting any malformed-caption warnings; an empty file list means "all media".
func resolveDirectiveMedia(mediaArg string, directive string, ceType contentEntityType, ceId string, config appConfig, expListMedia *[]string, warnings *[]string) []media {
	fileNames, captions, malformed := splitMediaArg(mediaArg)
	for _, bad := range malformed {
		*warnings = append(*warnings, "malformed caption \""+strings.TrimSpace(bad)+"\" in {"+directive+"} directive (expected: name: caption)")
	}
	isExplicit := len(fileNames) > 0
	if !isExplicit {
		fileNames = listAllMedia(ceType, ceId, *expListMedia)
	}
	return parseMediaFileNames(fileNames, ceType, ceId, config, isExplicit, captions)
}

func processInnerDirectives(content string, ceType contentEntityType, ceId string, config appConfig, resLoader resourceLoader, phReps map[string]string, expListMedia *[]string, warnings *[]string) string {
	content = hashTagRegex.ReplaceAllStringFunc(content, func(match string) string {
		tag := match[1:] // strip leading '#'; regex guarantees '#' + tag chars
		return fmt.Sprintf(hashTagMarkdownReplacementFormat, tag, normalizeURIString(tag))
	})
	tagAutoLinkPlaceholders := tagAutoLinkPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if tagAutoLinkPlaceholders != nil {
		for _, talp := range tagAutoLinkPlaceholders {
			placeholder := talp[0]
			linkText := talp[1]
			tagURI := normalizeURIString(linkText)
			replacement := fmt.Sprintf("[%s](/%s/%s/)", linkText, deployTagsDirName, tagURI)
			content = strings.Replace(content, placeholder, replacement, 1)
		}
	}
	tagLinkPlaceholders := tagLinkPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if tagLinkPlaceholders != nil {
		for _, tlp := range tagLinkPlaceholders {
			placeholder := tlp[0]
			tagText := strings.TrimSpace(tlp[1])
			link := "/" + deployTagsDirName + "/" + normalizeURIString(tagText) + "/"
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
	// collection directives are handled before the generic wrap pipeline
	// so a stray `{/}` further down the content can never swallow one as a wrap directive;
	// they resolve to deterministic placeholders substituted at process time
	// (collection data is aggregated from posts, while pages are parsed/cached independently)
	content = collectionDirectiveRegexp.ReplaceAllStringFunc(content, func(match string) string {
		m := collectionDirectiveRegexp.FindStringSubmatch(match)
		if ceType != Page {
			*warnings = append(*warnings, "collection directive is only supported in pages: "+match)
			return ""
		}
		uri := normalizeURIString(strings.TrimSpace(m[1]))
		if uri == "" {
			*warnings = append(*warnings, "collection directive references an invalid collection name: "+match)
			return ""
		}
		return fmt.Sprintf(collectionDirectivePlaceholderFormat, uri)
	})
	wrapPlaceholders := wrapPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if wrapPlaceholders != nil {
		sortContentDirectivePlaceholders(wrapPlaceholders)
		for _, wp := range wrapPlaceholders {
			if fileNames, _, _ := splitMediaArg(wp[3]); fileNames != nil {
				*expListMedia = append(*expListMedia, fileNames...)
			}
		}
	}
	mediaPlaceholders := mediaPlaceholderRegexp.FindAllStringSubmatch(content, -1)
	if mediaPlaceholders != nil {
		sortContentDirectivePlaceholders(mediaPlaceholders)
		for _, mp := range mediaPlaceholders {
			if fileNames, _, _ := splitMediaArg(mp[2]); fileNames != nil {
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
				mediaArg := wp[3]
				text := strings.TrimSpace(wp[4])
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
				allMedia := resolveDirectiveMedia(mediaArg, directive, ceType, ceId, config, expListMedia, warnings)
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
			mediaArg := mp[2]
			props := make(map[string]string)
			if propStr != "" {
				for _, pStr := range strings.Split(propStr, ",") {
					prop := strings.Split(strings.TrimSpace(pStr), "=")
					key := strings.TrimSpace(prop[0])
					val := strings.TrimSpace(prop[1])
					props[key] = val
				}
			}
			allMedia := resolveDirectiveMedia(mediaArg, "media", ceType, ceId, config, expListMedia, warnings)
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

func processColsBlock(cb colsBlock, ceType contentEntityType, ceId string, config appConfig, resLoader resourceLoader, phReps map[string]string, expListMedia *[]string, warnings *[]string) string {
	weights, weightsErr := parseColsWeights(cb.weights)
	if weightsErr != "" {
		println(" - " + weightsErr + " for " + ceId + "; falling back to equal widths")
		weights = nil
	}

	// pre-process inner directives on the whole cols body first so that nested
	// {/} closers (e.g., from {with-media}...{/}) become UUID placeholders and
	// don't confuse the {col}...{/} matcher's lazy quantifier downstream
	inner := processInnerDirectives(cb.inner, ceType, ceId, config, resLoader, phReps, expListMedia, warnings)

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

// findUnparsedDirectives returns any leftover `{...}` directives in rendered HTML body
// that no handler consumed (typos / invalid directives).
// Code regions (`<pre>` fenced blocks and inline `<code>` spans)
// are masked first so braces inside code are not flagged.
func findUnparsedDirectives(body string) []string {
	masked := preRegexp.ReplaceAllString(body, "")
	masked = codeSpanRegexp.ReplaceAllString(masked, "")
	return unparsedDirectiveRegexp.FindAllString(masked, -1)
}

// appendUnparsedDirectiveWarnings appends a warning for each leftover `{...}` directive
// found in the rendered body to the given (parse-time) warnings list.
func appendUnparsedDirectiveWarnings(warnings []string, body string) []string {
	for _, match := range findUnparsedDirectives(body) {
		warnings = append(warnings, "unparsed directive: "+match)
	}
	return warnings
}

func listAllMedia(contentEntityType contentEntityType, contentEntityId string, skipFiles []string) []string {
	ceType := strings.ToLower(contentEntityType.String())
	var allMedia []string
	mediaDirPath := fmt.Sprintf("%s%c%s%c%s%c%s", deployDirName, os.PathSeparator, mediaDirName, os.PathSeparator, ceType, os.PathSeparator, contentEntityId)
	// skip an explicitly-listed file by its full name or its extension-less base name
	// (an explicit reference may omit the extension, e.g. `{media:3}` excludes `3.jpg`)
	skip := func(file string) bool {
		return slices.Contains(skipFiles, file) || slices.Contains(skipFiles, stripExt(file))
	}
	if dirExists(mediaDirPath) {
		videoFiles, err := listFilesByExt(mediaDirPath, videoFileExtensions...)
		check(err)
		for _, video := range videoFiles {
			if !skip(video) {
				allMedia = append(allMedia, video)
			}
		}
		imageFiles, err := listFilesByExt(mediaDirPath, imageFileExtensions...)
		check(err)
		for _, image := range imageFiles {
			if !skip(image) && !strings.Contains(image, thumbImgFileSuffix) {
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

// splitMediaArg parses the inner argument of a `{media...}` / `{with-media...}` directive.
// The argument is `[: ]file1,file2,...` (a comma-separated file list, optionally empty for all media)
// optionally followed by `|`-separated caption specs:
//   - `name: caption` — targeted caption for a specific file (name with or without extension)
//   - `caption`       — nameless caption, valid only when exactly one file is listed
//
// `::` is an escape for a literal `:` inside caption text. Empty/dangling/trailing `|` segments are rejected.
// It returns the list of filenames in input order, a map of name -> caption
// for the captions that were specified, and the list of malformed/ambiguous caption segments (so callers can warn).
// Captions are HTML-escaped because templates are text/template (no auto-escaping).
func splitMediaArg(mediaArg string) ([]string, map[string]string, []string) {
	mediaArg = strings.TrimSpace(mediaArg)
	// strip a single leading file-list separator `:` (the media regex captures it)
	mediaArg = strings.TrimSpace(strings.TrimPrefix(mediaArg, ":"))
	if mediaArg == "" {
		return nil, nil, nil
	}

	// the file list is everything before the first `|`; the rest is the caption section
	fileListPart := mediaArg
	captionSection := ""
	hasCaptionSection := false
	if i := strings.IndexByte(mediaArg, '|'); i >= 0 {
		fileListPart = mediaArg[:i]
		captionSection = mediaArg[i+1:]
		hasCaptionSection = true
	}

	var fileNames []string
	for _, f := range strings.Split(fileListPart, ",") {
		if f = strings.TrimSpace(f); f != "" {
			fileNames = append(fileNames, f)
		}
	}

	if !hasCaptionSection {
		return fileNames, nil, nil
	}

	var captions map[string]string
	var malformed []string
	specs := strings.Split(captionSection, "|")
	const sentinel = "\x00" // placeholder for the `::` literal-colon escape
	for _, raw := range specs {
		spec := strings.TrimSpace(raw)
		if spec == "" {
			// dangling/trailing/double pipe — always an error, never silently ignored
			malformed = append(malformed, raw)
			continue
		}
		masked := strings.ReplaceAll(spec, "::", sentinel)
		unescape := func(s string) string {
			return strings.ReplaceAll(s, sentinel, ":")
		}
		if i := strings.IndexByte(masked, ':'); i >= 0 {
			// targeted: `name: caption`
			name := strings.TrimSpace(unescape(masked[:i]))
			caption := strings.TrimSpace(unescape(masked[i+1:]))
			if name == "" || caption == "" {
				malformed = append(malformed, spec)
				continue
			}
			if captions == nil {
				captions = make(map[string]string)
			}
			captions[name] = html.EscapeString(caption)
			continue
		}
		// nameless: valid only when it is the sole spec and exactly one file is listed
		if len(specs) != 1 || len(fileNames) != 1 {
			malformed = append(malformed, spec)
			continue
		}
		if captions == nil {
			captions = make(map[string]string)
		}
		captions[fileNames[0]] = html.EscapeString(unescape(masked))
	}
	return fileNames, captions, malformed
}

// stripExt returns the file name without its (last) extension.
func stripExt(name string) string {
	return strings.TrimSuffix(name, filepath.Ext(name))
}

// expandMediaFileName resolves a possibly extension-less media reference to the actual file name(s) present in dir.
// If name already carries a supported image/video extension and that file exists,
// it is returned as-is. Otherwise name is treated as a base name and every `<name>.<ext>`
// (for supported video/image extensions) that exists in dir is returned.
func expandMediaFileName(name string, dir string) []string {
	ext := strings.ToLower(filepath.Ext(name))
	if slices.Contains(imageFileExtensions, ext) || slices.Contains(videoFileExtensions, ext) {
		if fileExists(filepath.Join(dir, name)) {
			return []string{name}
		}
		return nil
	}
	var matches []string
	for _, e := range videoFileExtensions {
		if fileExists(filepath.Join(dir, name+e)) {
			matches = append(matches, name+e)
		}
	}
	for _, e := range imageFileExtensions {
		if fileExists(filepath.Join(dir, name+e)) {
			matches = append(matches, name+e)
		}
	}
	return matches
}

func parseMediaFileNames(mediaFileNames []string, contentEntityType contentEntityType, contentEntityId string, config appConfig, isExplicit bool, captions map[string]string) []media {
	var allMedia []media
	ceType := strings.ToLower(contentEntityType.String())
	contentUriSubPath := ceType + "/" + contentEntityId
	contentDirSubPath := ceType + string(os.PathSeparator) + contentEntityId
	mediaDirBasePath := filepath.Join(deployDirName, mediaDirName)
	contentDir := filepath.Join(mediaDirBasePath, contentDirSubPath)
	sharedDir := filepath.Join(mediaDirBasePath, sharedMediaDirName)
	// attach a caption to a resolved media item by exact file name first, then base name
	// (so a base-name caption applies to every file sharing that base name)
	attach := func(m *media, fileName string) {
		if caption, ok := captions[fileName]; ok {
			m.Caption = caption
		} else if caption, ok := captions[stripExt(fileName)]; ok {
			m.Caption = caption
		}
	}
	hasSupportedExt := func(name string) bool {
		ext := strings.ToLower(filepath.Ext(name))
		return slices.Contains(imageFileExtensions, ext) || slices.Contains(videoFileExtensions, ext)
	}
	for _, mediaFileName := range mediaFileNames {
		if strings.Contains(mediaFileName, thumbImgFileSuffix) {
			continue
		}
		if !isExplicit {
			// names came from listAllMedia — already actual content-specific file names
			if m := buildMediaItem(mediaFileName, contentUriSubPath, contentDirSubPath, config); m != nil {
				attach(m, mediaFileName)
				allMedia = append(allMedia, *m)
			}
			continue
		}
		if hasSupportedExt(mediaFileName) {
			// explicit reference carrying an extension: prefer the content-specific file,
			// fall back to shared when missing there; still render with the content URI when absent from both
			uriSubPath, dirSubPath := contentUriSubPath, contentDirSubPath
			if !fileExists(filepath.Join(contentDir, mediaFileName)) && fileExists(filepath.Join(sharedDir, mediaFileName)) {
				uriSubPath, dirSubPath = sharedMediaDirName, sharedMediaDirName
			}
			if m := buildMediaItem(mediaFileName, uriSubPath, dirSubPath, config); m != nil {
				attach(m, mediaFileName)
				allMedia = append(allMedia, *m)
			}
			continue
		}
		// extension-less reference: expand to the actual file(s) on disk, content dir then shared
		uriSubPath, dirSubPath := contentUriSubPath, contentDirSubPath
		resolved := expandMediaFileName(mediaFileName, contentDir)
		if len(resolved) == 0 {
			uriSubPath, dirSubPath = sharedMediaDirName, sharedMediaDirName
			resolved = expandMediaFileName(mediaFileName, sharedDir)
		}
		for _, fileName := range resolved {
			if m := buildMediaItem(fileName, uriSubPath, dirSubPath, config); m != nil {
				attach(m, fileName)
				allMedia = append(allMedia, *m)
			}
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
			uri := normalizeURIString(t)
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
