package main

import (
	"regexp"
	"time"
)

const (
	appVersion                                  = "1.2.2"
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
	searchTemplateFileName                      = "search" + templateFileExtension
	pagerTemplateFileName                       = "pager" + templateFileExtension
	contentDirectiveTemplateFileNameFormat      = "content-%s" + templateFileExtension
	contentFileExtension                        = ".html"
	indexPageFileName                           = "index" + contentFileExtension
	searchPageFileName                          = "search" + contentFileExtension
	searchIndexFileName                         = "search.json"
	directivePlaceholderReplacementFormat       = ":@@@:%s:@@@:"
	hashTagMarkdownReplacementFormat            = "[#%s](/" + deployTagDirName + "/%s/)"
	stylesFileName                              = "styles.css"
	stylesIncludeFileNameFormat                 = "styles-include-%s.css"
	markdownPagesDirName                        = "pages"
	markdownPostsDirName                        = "posts"
	markdownFileExtension                       = ".md"
	mediaDirName                                = "media"
	deployDirName                               = "deploy"
	deployPostDirName                           = "post"
	deployPostsDirName                          = "posts"
	deployPageDirName                           = "page"
	deployArchiveDirName                        = "archive"
	deployTagDirName                            = "tag"
	metaDataKeyDate                             = "date"
	metaDataKeyTime                             = "time"
	metaDataKeyTitle                            = "title"
	metaDataKeyTags                             = "tags"
	configFileName                              = "config.yml"
	defaultGenerateArchive                      = true
	defaultEnableSearch                         = true
	defaultPageSize                             = 10
	minAllowedThumbWidth                        = 320
	minAllowedThumbThreshold                    = 0.3
	defaultThumbThreshold                       = 0.5
	defaultUseThumbs                            = true
	defaultServeHost                            = "localhost"
	defaultServePort                            = 8888
	thumbImgFileSuffix                          = "_thumb"
	pageHeadIncludePrefix                       = "page-head--"
	defaultThemeName                            = "pretty-dark"
	defaultThemeAlias                           = "default"
	downloadedThemeDirSuffix                    = "-downloaded"
	stylesTemplatePlaceholder                   = "{{@ styles @}}"
	pageHeadTemplatePlaceholder                 = "{{@ page-head @}}"
	subTemplatePlaceholder                      = "{{@ sub-template @}}"
	commandCleanupTargetContent                 = "content"
	commandCleanupTargetThumbs                  = "thumbs"
	commandCleanupTargetArchive                 = "archive"
	commandCleanupTargetSearch                  = "search"
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
	contentClosingTag                           = "</body>"
)

var (
	defaultThumbSizes                    = /* const */ []int{480, 960}
	thumbImgFileNameRegexp               = /* const */ regexp.MustCompile(`_(\d+)` + thumbImgFileSuffix)
	imageFileExtensions                  = /* const */ []string{".jpg", ".jpeg", ".png", ".gif"}
	thumbImageFileExtensions             = /* const */ []string{".jpg", ".jpeg", ".png"}
	videoFileExtensions                  = /* const */ []string{".mp4", ".mkv", ".mov"}
	metaDataPlaceholderRegexp            = /* const */ regexp.MustCompile(`---[\s\w-:"]*---`)
	contentDirectivePlaceholderRegexp    = /* const */ regexp.MustCompile(`{.*}`)
	whitespacePlaceholderRegexp          = /* const */ regexp.MustCompile(`\s+`)
	includeTemplateFilePlaceholderRegexp = /* const */ regexp.MustCompile(`{{@\s*([\w-_]+\.html)\s*@}}`)
	includeContentFilePlaceholderRegexp  = /* const */ regexp.MustCompile(`{{#\s*([\w-_]+\.html)\s*#}}`)
	contentLinkPlaceholderRegexp         = /* const */ regexp.MustCompile(`{%\s*([\w-_]+):([\w-_]+)\s*%}`)
	mediaPlaceholderRegexp               = /* const */ regexp.MustCompile(`{media(\([\s\w=,]+\))?(:\s*([\w\s-_.,*]+))?}`)
	embedMediaPlaceholderRegexp          = /* const */ regexp.MustCompile(`{embed:\s*([^}]+)}`)
	wrapPlaceholderRegexp                = /* const */ regexp.MustCompile(`\{([\w-_.]+)(\([\s\w=,]+\))?(:\s*([\w\s-_.,*]+))?}([^{}]*){/}`)
	hashTagRegex                         = /* const */ regexp.MustCompile(`#(\p{L}+[_-]*\p{L}*)`)
)
