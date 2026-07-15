package app

import (
	_ "embed"
	"regexp"
	"time"
)

const (
	appVersion                                  = "2.0.3"
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
	collectionTemplateFileName                  = "collection" + templateFileExtension
	collectionIndexTemplateFileName             = "collection-index" + templateFileExtension
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
	sharedMediaDirName                          = "shared"
	deployDirName                               = "deploy"
	deployPostDirName                           = "post"
	deployPostsDirName                          = "posts"
	deployPageDirName                           = "page"
	deployArchiveDirName                        = "archive"
	deployTagsDirName                           = "tags"
	deployCollectionsDirName                    = "collections"
	metaDataKeyDate                             = "date"
	metaDataKeyTime                             = "time"
	metaDataKeyTitle                            = "title"
	metaDataKeyTags                             = "tags"
	metaDataKeyCollections                      = "collections"
	metaDataKeyMetaCollections                  = "meta-collections"
	metaDataKeyMetaCollection                   = "meta-collection"
	collectionDirectivePlaceholderFormat        = ":@@@:collection:%s:@@@:"
	configFileName                              = "config.yml"
	defaultGenerateArchive                      = true
	defaultGenerateTagIndex                     = true
	defaultGenerateCollectionIndex              = true
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
	defaultFeedPostViewOnWebsiteLinkText        = "View on website ⮵"
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
	commandCleanupTargetCollections             = "collections"
	commandCleanupTargetCollectionIndex         = "collection-index"
	commandCleanupTargetArchive                 = "archive"
	commandCleanupTargetSearch                  = "search"
	commandCleanupTargetMedia                   = "media"
	commandCleanupOptionDryRun                  = "--dry-run"
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
	tagAutoLinkPlaceholderRegexp         = /* const */ regexp.MustCompile(`\[([^\]]+)\]\(\{%\s*tag\s*%\}\)`)
	tagLinkPlaceholderRegexp             = /* const */ regexp.MustCompile(`{%\s*tag\s*:\s*([\w\s-]+)\s*%}`)
	searchLinkPlaceholderRegexp          = /* const */ regexp.MustCompile(`{%\s*search\s*:\s*([^{}%]+)\s*%}`)
	contentLinkPlaceholderRegexp         = /* const */ regexp.MustCompile(`{%\s*([\w-_]+)\s*:\s*([\w-_]+)\s*%}`)
	mediaPlaceholderRegexp               = /* const */ regexp.MustCompile(`{\s*media(\([\s\w=,]+\))?\s*([:|][^{}]*)?\s*}`)
	collectionDirectiveRegexp            = /* const */ regexp.MustCompile(`{\s*collection\s*:\s*([^{}]+?)\s*}`)
	collectionDirectivePlaceholderRegexp = /* const */ regexp.MustCompile(`:@@@:collection:([^:\s]+):@@@:`)
	// a collection directive on its own line gets wrapped in a <p> element by markdown rendering
	// — the wrapper must be stripped along with the placeholder (a <section> inside a <p> is invalid HTML)
	collectionDirectiveWrappedPlaceholderRegexp = /* const */ regexp.MustCompile(`<p>\s*:@@@:collection:([^:\s]+):@@@:\s*</p>`)
	// blankLineRunRegexp matches a newline followed by one or more additional
	// whitespace-only lines; used to collapse runs of blank/whitespace-only
	// lines produced by Go-template conditionals into a single newline
	blankLineRunRegexp = /* const */ regexp.MustCompile(`\n[ \t]*(?:\n[ \t]*)+`)
	// preRegexp matches `<pre>...</pre>` blocks whose inner whitespace is significant
	// (e.g. fenced code blocks rendered by goldmark) and must be left untouched
	preRegexp                        = /* const */ regexp.MustCompile(`(?is)<pre\b[^>]*>.*?</\s*pre\s*>`)
	embedMediaPlaceholderRegexp      = /* const */ regexp.MustCompile(`{\s*embed\s*:\s*([^}]+)\s*}`)
	wrapPlaceholderOpeningRegexp     = /* const */ regexp.MustCompile(`\{\s*([\w-_.]+)\s*(\([\s\w=,]+\))?\s*([:|][^{}]*)?\s*}`)
	wrapPlaceholderRegexp            = /* const */ regexp.MustCompile(`\{\s*([\w-_.]+)\s*(\([\s\w=,]+\))?\s*([:|][^{}]*)?\s*}([^{}]*){/}`)
	colsPlaceholderRegexp            = /* const */ regexp.MustCompile(`(?s)\{\s*cols\s*(\(([\s\d:]+)\))?\s*\}(.*?)\{//\}`)
	colPlaceholderRegexp             = /* const */ regexp.MustCompile(`(?s)\{\s*col\s*(\(([\s\w=,]+)\))?\s*\}(.*?)\{/\}`)
	pWrapperAroundPlaceholdersRegexp = /* const */ regexp.MustCompile(`(?s)<p>\s*((?::@@@:[\w-]+:@@@:\s*(?:<br\s*/?>\s*)?)+)</p>`)
	brTagRegexp                      = /* const */ regexp.MustCompile(`<br\s*/?>`)
	hashTagRegex                     = /* const */ regexp.MustCompile(`#([\p{L}\d][\p{L}\d_-]*)`)
	relativeURLHrefRegexp            = /* const */ regexp.MustCompile(`href="(/[^"]*)"`)
	// unparsedDirectiveRegexp matches a single leftover `{...}` directive (no nested braces or
	// newlines) in rendered HTML — used to warn about typo'd/unknown content directives
	unparsedDirectiveRegexp = /* const */ regexp.MustCompile(`\{[^{}\n]*\}`)
	// codeSpanRegexp matches inline `<code>...</code>` spans (single-backtick code); used together
	// with preRegexp to mask code regions so braces inside code aren't flagged as unparsed directives
	codeSpanRegexp = /* const */ regexp.MustCompile(`(?is)<code\b[^>]*>.*?</\s*code\s*>`)
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
