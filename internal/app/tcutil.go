package app

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"
	"unicode"
)

var (
	templateIncludeCache = make(map[string]string)
	templateCache        = make(map[string]*template.Template)
	funcMap              = /* const */ template.FuncMap{
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
		"fmtYearAndMonth": formatYearAndMonth,
		"toLowerCase":     strings.ToLower,
		"normalizeTagURI": normalizeTagURI,
	}
)

func compilePageTemplate(p page, resLoader resourceLoader) *template.Template {
	pageTemplateMarkup, err := readTemplateFile(pageTemplateFileName, resLoader)
	check(err)
	pageTemplate := compileFullTemplate(pageTemplateFileName, pageTemplateMarkup,
		func(mainTemplateMarkup string) string {
			return strings.Replace(mainTemplateMarkup, pageHeadTemplatePlaceholder,
				"{{# "+pageHeadIncludePrefix+p.Id+contentFileExtension+" #}}", 1)
		}, resLoader)
	return pageTemplate
}

func compileArchiveTemplate(resLoader resourceLoader) *template.Template {
	return compileStandalonePageTemplate(archiveTemplateFileName, resLoader)
}

func compileTagIndexTemplate(resLoader resourceLoader) *template.Template {
	return compileStandalonePageTemplate(tagIndexTemplateFileName, resLoader)
}

func compileSearchTemplate(resLoader resourceLoader) *template.Template {
	return compileStandalonePageTemplate(searchTemplateFileName, resLoader)
}

func compileStandalonePageTemplate(singlePageTemplateFileName string, resLoader resourceLoader) *template.Template {
	singlePageTemplateMarkup, err := readTemplateFile(singlePageTemplateFileName, resLoader)
	check(err)
	singlePageTemplateMarkup = strings.Replace(singlePageTemplateMarkup, pageHeadTemplatePlaceholder, "", 1)
	singlePageTemplate := compileFullTemplate(singlePageTemplateFileName, singlePageTemplateMarkup, nil, resLoader)
	return singlePageTemplate
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
	postTemplate, ok := templateCache[postTemplateFileName]
	if !ok {
		postTemplateMarkup, err := readTemplateFile(postTemplateFileName, resLoader)
		check(err)
		postTemplateMarkup = processDirectives(postTemplateMarkup, resLoader)
		tmplt, err := template.New(postTemplateMarkup).Funcs(funcMap).Parse(postTemplateMarkup)
		check(err)
		postTemplate = tmplt
		templateCache[postTemplateFileName] = tmplt
	}
	return postTemplate
}

func compileContentDirectiveTemplate(directive string, resLoader resourceLoader) (*template.Template, error) {
	templateFileName := fmt.Sprintf(contentDirectiveTemplateFileNameFormat, directive)
	contentDirectiveTemplate, ok := templateCache[templateFileName]
	if !ok {
		contentDirectiveMarkup, err := readTemplateFile(templateFileName, resLoader)
		if err != nil {
			return nil, err
		}
		contentDirectiveMarkup = processDirectives(contentDirectiveMarkup, resLoader)
		tmplt, err := template.New(contentDirectiveMarkup).Funcs(funcMap).Parse(contentDirectiveMarkup)
		check(err)
		contentDirectiveTemplate = tmplt
		templateCache[templateFileName] = tmplt
	}
	return contentDirectiveTemplate, nil
}

func compileMediaTemplate(resLoader resourceLoader) *template.Template {
	inlineMediaTemplate, ok := templateCache[mediaTemplateFileName]
	if !ok {
		inlineMediaTemplateMarkup, err := readTemplateFile(mediaTemplateFileName, resLoader)
		check(err)
		inlineMediaTemplateMarkup = processDirectives(inlineMediaTemplateMarkup, resLoader)
		tmplt, err := template.New(mediaTemplateFileName).Funcs(funcMap).Parse(inlineMediaTemplateMarkup)
		check(err)
		inlineMediaTemplate = tmplt
		templateCache[mediaTemplateFileName] = tmplt
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
		includeMarkup, ok := templateIncludeCache[ticKey]
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
			templateIncludeCache[ticKey] = includeMarkup
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

func normalizeTagURI(tag string) string {
	var sb strings.Builder
	for _, r := range tag {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			sb.WriteRune(r)
		} else {
			sb.WriteRune('-')
		}
	}
	result := strings.Trim(sb.String(), "-")
	return strings.ToLower(result)
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
