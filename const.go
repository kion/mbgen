package main

import (
	_ "embed"
	"regexp"
	"time"
)

const (
	appVersion                                  = "1.7.1"
	defaultGitHubRepoUrl                        = "github.com/kion/mbgen"
	defaultGitHubRepoThemesUrl                  = defaultGitHubRepoUrl + "/themes"
	defaultGitHubRepoPageContentSamplesUrl      = defaultGitHubRepoUrl + "/content-samples/pages"
	defaultGitHubRepoPostContentSamplesUrl      = defaultGitHubRepoUrl + "/content-samples/posts"
	defaultGitHubRepoDeployDirContentSamplesUrl = defaultGitHubRepoUrl + "/content-samples/deploy"
	themesDirName                               = "themes"
	resourcesDirName                            = "resources"
	templatesDirName                            = "templates"
	includeDirName                              = "include"
	templateFileExtension                       = ".html"
	mainTemplateFileName                        = "main" + templateFileExtension
	pageTemplateFileName                        = "page" + templateFileExtension
	postTemplateFileName                        = "post" + templateFileExtension
	mediaTemplateFileName                       = "media" + templateFileExtension
	archiveTemplateFileName                     = "archive" + templateFileExtension
	tagIndexTemplateFileName                    = "tag-index" + templateFileExtension
	searchTemplateFileName                      = "search" + templateFileExtension
	pagerTemplateFileName                       = "pager" + templateFileExtension
	contentDirectiveTemplateFileNameFormat      = "content-%s" + templateFileExtension
	contentFileExtension                        = ".html"
	indexPageFileName                           = "index" + contentFileExtension
	searchPageFileName                          = "search" + contentFileExtension
	searchIndexFileName                         = "search.json"
	searchJSFileName                            = "search.js"
	directivePlaceholderReplacementFormat       = ":@@@:%s:@@@:"
	hashTagMarkdownReplacementFormat            = "[#%s](/" + deployTagsDirName + "/%s/)"
	markdownPagesDirName                        = "pages"
	markdownPostsDirName                        = "posts"
	markdownFileExtension                       = ".md"
	mediaDirName                                = "media"
	deployDirName                               = "deploy"
	deployPostDirName                           = "post"
	deployPostsDirName                          = "posts"
	deployPageDirName                           = "page"
	deployArchiveDirName                        = "archive"
	deployTagsDirName                           = "tags"
	metaDataKeyDate                             = "date"
	metaDataKeyTime                             = "time"
	metaDataKeyTitle                            = "title"
	metaDataKeyTags                             = "tags"
	configFileName                              = "config.yml"
	defaultGenerateArchive                      = true
	defaultGenerateTagIndex                     = true
	defaultEnableSearch                         = true
	defaultPageSize                             = 10
	defaultResizeOrigImages                     = false
	defaultMaxImgSize                           = 1920
	minAllowedMaxImgSize                        = 1080
	minAllowedThumbWidth                        = 320
	minAllowedThumbThreshold                    = 0.3
	defaultThumbThreshold                       = 0.5
	defaultJPEGQuality                          = 85
	minAllowedJPEGQuality                       = 70
	maxAllowedJPEGQuality                       = 100
	defaultPNGCompressionLevel                  = DefaultCompression
	defaultUseThumbs                            = true
	defaultServeHost                            = "localhost"
	defaultServePort                            = 8888
	defaultFeedPostCnt                          = 20
	defaultFeedPostContinueReadingText          = "Continue Reading ->"
	feedExcerptSentenceCnt                      = 3
	feedExcerptFallbackWordCnt                  = 20
	feedFormatRSS                               = "rss"
	feedFormatAtom                              = "atom"
	feedFormatJSON                              = "json"
	feedFileNameRSS                             = "rss.xml"
	feedFileNameAtom                            = "atom.xml"
	feedFileNameJSON                            = "feed.json"
	thumbImgFileSuffix                          = "_thumb"
	pageHeadIncludePrefix                       = "page-head--"
	defaultThemeName                            = "pretty-dark"
	defaultThemeAlias                           = "default"
	downloadedThemeDirSuffix                    = "-downloaded"
	pageHeadTemplatePlaceholder                 = "{{@ page-head @}}"
	subTemplatePlaceholder                      = "{{@ sub-template @}}"
	commandInspectOptionFix                     = "--fix"
	commandCleanupTargetContent                 = "content"
	commandCleanupTargetThumbs                  = "thumbs"
	commandCleanupTargetTags                    = "tags"
	commandCleanupTargetTagIndex                = "tag-index"
	commandCleanupTargetArchive                 = "archive"
	commandCleanupTargetSearch                  = "search"
	commandServeOptionAdmin                     = "--admin"
	commandServeOptionWatchReload               = "--watch-reload"
	commandThemeActionActivate                  = "activate"
	commandThemeActionInstall                   = "install"
	commandThemeActionUpdate                    = "update"
	commandThemeActionRefresh                   = "refresh"
	commandThemeActionDelete                    = "delete"
	httpProtocol                                = "http://"
	httpsProtocol                               = "https://"
	websocketProtocol                           = "ws://"
	websocketPath                               = "/--ws--"
	websocketPingPeriod                         = 60 * time.Second
	jsOpeningTag                                = "<script type='text/javascript'>"
	jsClosingTag                                = "</script>"
	styleOpeningTag                             = "<style>"
	styleClosingTag                             = "</style>"
	headClosingTag                              = "</head>"
	bodyClosingTag                              = "</body>"
	mainOpeningTag                              = "<main>"
	mainClosingTag                              = "</main>"
	deployCommandAvailablePlaceholder           = ":@@@:deploy-command-available:@@@:"
	errPostDateMissing                          = "post '%s' is missing a date, which is required for feed generation"
)

var (
	defaultThumbSizes                    = /* const */ []int{480, 960}
	thumbImgFileNameRegexp               = /* const */ regexp.MustCompile(`_(\d+)` + thumbImgFileSuffix)
	imageFileExtensions                  = /* const */ []string{".jpg", ".jpeg", ".png", ".gif"}
	thumbImageFileExtensions             = /* const */ []string{".jpg", ".jpeg", ".png"}
	videoFileExtensions                  = /* const */ []string{".mp4", ".mkv", ".mov"}
	metaDataPlaceholderRegexp            = /* const */ regexp.MustCompile(`(?s)^---.*?---`)
	contentDirectivePlaceholderRegexp    = /* const */ regexp.MustCompile(`{.*}`)
	whitespacePlaceholderRegexp          = /* const */ regexp.MustCompile(`\s+`)
	includeTemplateFilePlaceholderRegexp = /* const */ regexp.MustCompile(`{{@\s*([\w-_]+\.html)\s*@}}`)
	includeContentFilePlaceholderRegexp  = /* const */ regexp.MustCompile(`{{#\s*([\w-_]+\.html)\s*#}}`)
	searchLinkPlaceholderRegexp          = /* const */ regexp.MustCompile(`{%\s*search\s*:\s*([^{}%]+)\s*%}`)
	contentLinkPlaceholderRegexp         = /* const */ regexp.MustCompile(`{%\s*([\w-_]+)\s*:\s*([\w-_]+)\s*%}`)
	mediaPlaceholderRegexp               = /* const */ regexp.MustCompile(`{\s*media(\([\s\w=,]+\))?(\s*:\s*([\w\s-_.,*]+))?\s*}`)
	embedMediaPlaceholderRegexp          = /* const */ regexp.MustCompile(`{\s*embed\s*:\s*([^}]+)\s*}`)
	wrapPlaceholderRegexp                = /* const */ regexp.MustCompile(`\{\s*([\w-_.]+)\s*(\([\s\w=,]+\))?(\s*:\s*([\w\s-_.,*]+))?\s*}([^{}]*){/}`)
	hashTagRegex                         = /* const */ regexp.MustCompile(`#(\p{L}+[_-]*\p{L}*)`)
	relativeURLHrefRegexp                = /* const */ regexp.MustCompile(`href="(/[^"]*)"`)
)

//go:embed inject-js/admin.js
var adminJS string

//go:embed inject-js/watch-reload.js
var watchReloadJS string

//go:embed inject-js/search.js
var searchJS string

//go:embed inject-js/easymde.min.js
var mdEditorJS string

//go:embed inject-css/easymde.min.css
var mdEditorCSS string
