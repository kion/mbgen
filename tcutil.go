package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"text/template"
)

var (
	templateIncludeCache = /* const */ make(map[string]string)
	templateCache        = /* const */ make(map[string]*template.Template)
	templateCacheMutex   = /* const */ sync.RWMutex{}

	funcMap = /* const */ template.FuncMap{
		"mod":   func(a, b int) int { return a % b },
		"minus": func(a, b int) int { return a - b },
		"plus":  func(a, b int) int { return a + b },
		"iter": func(count int) []int {
			var i int
			var items []int
			for i = 0; i < count; i++ {
				items = append(items, i)
			}
			return items
		},
		"everyNthMediaItem": func(slice []media, nth int, startIdx int) []media {
			var fSlice []media
			for idx, el := range slice {
				if idx%nth == startIdx {
					fSlice = append(fSlice, el)
				}
			}
			return fSlice
		},
		"toInt": func(str string) int {
			i, err := strconv.Atoi(str)
			if err != nil {
				i = 0
			}
			return i
		},
		"toLowerCase": strings.ToLower,
	}
)

func compilePageTemplate(p page, resLoader resourceLoader) *template.Template {
	pageTemplateMarkup, err := readTemplateFile(pageTemplateFileName, resLoader)
	check(err)
	pageTemplate := compileFullTemplate(pageTemplateFileName, pageTemplateMarkup,
		func(mainTemplateMarkup string) string {
			return strings.Replace(mainTemplateMarkup, pageHeadTemplatePlaceholder,
				"{{# "+pageHeadIncludePrefix+p.id+contentFileExtension+" #}}", 1)
		}, resLoader)
	return pageTemplate
}

func compilePagerTemplate(resLoader resourceLoader) *template.Template {
	pagerTemplateMarkup, err := readTemplateFile(pagerTemplateFileName, resLoader)
	check(err)
	pagerTemplate, err :=
		template.New(pagerTemplateFileName).Funcs(funcMap).Parse(pagerTemplateMarkup)
	check(err)
	return pagerTemplate
}

func compilePostTemplate(resLoader resourceLoader) *template.Template {
	templateCacheMutex.RLock()
	postTemplate, ok := templateCache[postTemplateFileName]
	templateCacheMutex.RUnlock()
	if !ok {
		postTemplateMarkup, err := readTemplateFile(postTemplateFileName, resLoader)
		check(err)
		postTemplateMarkup = processDirectives(postTemplateMarkup, resLoader)
		tmplt, err := template.New(postTemplateMarkup).Funcs(funcMap).Parse(postTemplateMarkup)
		check(err)
		postTemplate = tmplt
		templateCacheMutex.Lock()
		templateCache[postTemplateFileName] = tmplt
		templateCacheMutex.Unlock()
	}

	return postTemplate
}

func compileContentDirectiveTemplate(directive string, resLoader resourceLoader) (*template.Template, error) {
	templateFileName := fmt.Sprintf(contentDirectiveTemplateFileNameFormat, directive)
	templateCacheMutex.RLock()
	contentDirectiveTemplate, ok := templateCache[templateFileName]
	templateCacheMutex.RUnlock()
	if !ok {
		contentDirectiveMarkup, err := readTemplateFile(templateFileName, resLoader)
		if err != nil {
			return nil, err
		}
		contentDirectiveMarkup = processDirectives(contentDirectiveMarkup, resLoader)
		tmplt, err := template.New(contentDirectiveMarkup).Funcs(funcMap).Parse(contentDirectiveMarkup)
		check(err)
		contentDirectiveTemplate = tmplt
		templateCacheMutex.Lock()
		templateCache[templateFileName] = tmplt
		templateCacheMutex.Unlock()
	}
	return contentDirectiveTemplate, nil
}

func compileMediaTemplate(resLoader resourceLoader) *template.Template {
	templateCacheMutex.RLock()
	inlineMediaTemplate, ok := templateCache[mediaTemplateFileName]
	templateCacheMutex.RUnlock()
	if !ok {
		inlineMediaTemplateMarkup, err := readTemplateFile(mediaTemplateFileName, resLoader)
		check(err)
		inlineMediaTemplateMarkup = processDirectives(inlineMediaTemplateMarkup, resLoader)
		tmplt, err := template.New(mediaTemplateFileName).Funcs(funcMap).Parse(inlineMediaTemplateMarkup)
		check(err)
		inlineMediaTemplate = tmplt
		templateCacheMutex.Lock()
		templateCache[mediaTemplateFileName] = tmplt
		templateCacheMutex.Unlock()
	}
	return inlineMediaTemplate
}

func compileFullTemplate(name string, content string,
	mainTemplateMarkupHandler func(mainTemplateMarkup string) string,
	resLoader resourceLoader) *template.Template {
	if mainTemplateMarkup == "" {
		markup, err := readTemplateFile(mainTemplateFileName, resLoader)
		check(err)
		mainTemplateMarkup = markup
	}
	stylesTemplatePlaceholderReplacement := ""
	for _, level := range templateIncludeLevels {
		stylesIncludeFilePath := getIncludeFilePath(stylesFileName, level, resLoader.config)
		if stylesIncludeFilePath != "" {
			stylesTemplatePlaceholderReplacement += fmt.Sprintf(`<link rel="stylesheet" href="/resources/%s">`, fmt.Sprintf(stylesIncludeFileNameFormat, level.String()))
		}
	}
	mainTemplateMarkup = strings.Replace(mainTemplateMarkup, stylesTemplatePlaceholder, stylesTemplatePlaceholderReplacement, 1)
	if mainTemplateMarkupHandler != nil {
		mainTemplateMarkup = mainTemplateMarkupHandler(mainTemplateMarkup)
	}
	templateMarkup := compileSubTemplate(mainTemplateMarkup, content, resLoader)
	tmplt, err := template.New(name).Funcs(funcMap).Parse(templateMarkup)
	check(err)
	return tmplt
}

func compileSubTemplate(mainTemplateMarkup string, subTemplateMarkup string, resLoader resourceLoader) string {
	fullTemplateMarkup := strings.Replace(mainTemplateMarkup, subTemplatePlaceholder, subTemplateMarkup, 1)
	fullTemplateMarkup = processDirectives(fullTemplateMarkup, resLoader)
	fullTemplateMarkup = strings.Replace(fullTemplateMarkup, pageHeadTemplatePlaceholder, "", 1)
	return fullTemplateMarkup
}

func processDirectives(templateMarkup string, resLoader resourceLoader) string {
	var templateIncludes []templateInclude
	includeTemplateFilePlaceholders := includeTemplateFilePlaceholderRegexp.FindAllStringSubmatch(templateMarkup, -1)
	if includeTemplateFilePlaceholders != nil {
		for _, itfp := range includeTemplateFilePlaceholders {
			templateIncludes = append(templateIncludes,
				templateInclude{includeType: Template, placeholder: itfp[0], fileName: itfp[1]})
		}
	}
	includeContentFilePlaceholders := includeContentFilePlaceholderRegexp.FindAllStringSubmatch(templateMarkup, -1)
	if includeContentFilePlaceholders != nil {
		for _, icfp := range includeContentFilePlaceholders {
			templateIncludes = append(templateIncludes,
				templateInclude{includeType: Content, placeholder: icfp[0], fileName: icfp[1]})
		}
	}

	for _, ti := range templateIncludes {
		ticKey := ti.includeType.String() + "/" + ti.fileName
		templateCacheMutex.RLock()
		includeMarkup, ok := templateIncludeCache[ticKey]
		templateCacheMutex.RUnlock()
		if !ok {
			switch ti.includeType {
			case Template:
				ic, err := resLoader.loadTemplate(ti.fileName)
				check(err)
				includeMarkup = string(ic)
			case Content:
				includeMarkup = ""
				ic, err := resLoader.loadInclude(ti.fileName, Global)
				if err != nil {
					println("failed to load global include: " + err.Error())
				} else if ic != nil && len(ic) > 0 { // includes are optional
					includeMarkup += string(ic)
				}
				ic, err = resLoader.loadInclude(ti.fileName, Theme)
				if err != nil {
					println("failed to load theme include: " + err.Error())
				} else if ic != nil && len(ic) > 0 { // includes are optional
					includeMarkup += string(ic)
				}
			}
			templateCacheMutex.Lock()
			templateIncludeCache[ticKey] = includeMarkup
			templateCacheMutex.Unlock()
		}
		templateMarkup = strings.Replace(templateMarkup, ti.placeholder, includeMarkup, 1)

		contentLinkPlaceholders := contentLinkPlaceholderRegexp.FindAllStringSubmatch(templateMarkup, -1)
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
				templateMarkup = strings.Replace(templateMarkup, placeholder, link, 1)
			}
		}
	}

	return templateMarkup
}

func readTemplateFile(templateFileName string, resLoader resourceLoader) (string, error) {
	templateMarkupBytes, err := resLoader.loadTemplate(templateFileName)
	return string(templateMarkupBytes), err
}

func getIncludeFilePath(includeFileName string, level templateIncludeLevel, config appConfig) string {
	var includeFilePath string
	if Global == level {
		includeFilePath = fmt.Sprintf("%s%c%s", includeDirName, os.PathSeparator, includeFileName)
	} else if Theme == level {
		themeName := config.theme
		if strings.ContainsRune(themeName, os.PathSeparator) {
			themePathSegments := strings.Split(themeName, string(os.PathSeparator))
			themeName = themePathSegments[len(themePathSegments)-1]
		}
		includeFilePath = fmt.Sprintf("%s%c%s%c%s", includeDirName, os.PathSeparator, themeName, os.PathSeparator, includeFileName)
	} else {
		panic("invalid template include level for: " + includeDirName)
	}
	if !fileExists(includeFilePath) {
		// ignore non-existing include files (includes are optional)
		includeFilePath = ""
	}
	return includeFilePath
}
