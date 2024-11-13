package main

import (
	"bytes"
	"cloud.google.com/go/civil"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"
)

type appConfig struct {
	siteName         string
	theme            string
	homePage         string
	generateArchive  bool
	generateTagIndex bool
	enableSearch     bool
	pageSize         int
	useThumbs        bool
	thumbSizes       []int
	thumbThreshold   float64
	serveHost        string
	servePort        int
}

type appCommandDescriptor struct {
	command     string
	description string
	usage       string
	reqConfig   bool
	reqArgCnt   int
	optArgCnt   int
}

type appCommand func(config appConfig, commandArgs ...string)

type templateIncludeType int

const (
	UndefinedTemplateIncludeType templateIncludeType = iota
	Template
	Content
)

func (t templateIncludeType) String() string {
	switch t {
	case Template:
		return "Template"
	case Content:
		return "Content"
	}
	panic("Undefined Template Include Type")
}

type templateIncludeLevel int

const (
	UndefinedTemplateIncludeLevel templateIncludeLevel = iota
	Global
	Theme
)

var templateIncludeLevels = /* const */ []templateIncludeLevel{Global, Theme}

func (t templateIncludeLevel) String() string {
	switch t {
	case Global:
		return "global"
	case Theme:
		return "theme"
	}
	panic("Unknown Template Include Level")
}

type templateInclude struct {
	includeType templateIncludeType
	placeholder string
	fileName    string
}

type contentEntityType int

const (
	UndefinedContentEntityType contentEntityType = iota
	Page
	Post
)

func (c contentEntityType) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(c.String()))
}

func (c contentEntityType) Page() bool {
	return c == Page
}

func (c contentEntityType) Post() bool {
	return c == Post
}

func (c contentEntityType) String() string {
	switch c {
	case Page:
		return "Page"
	case Post:
		return "Post"
	}
	panic("Undefined Content Entity Type")
}

type templateContent struct {
	EntityType contentEntityType
	Title      string
	FileName   string
	Content    any
	Config     map[string]any
}

type contentDirectiveData struct {
	Text  string
	Media []media
	Embed []embeddedMedia
	Props map[string]string
}

func (c contentDirectiveData) Images() []media {
	return c.filterMediaByType(Image)
}

func (c contentDirectiveData) Videos() []media {
	return c.filterMediaByType(Video)
}

func (c contentDirectiveData) filterMediaByType(mType mediaType) []media {
	var mList []media
	for _, m := range c.Media {
		if m.Type == mType {
			mList = append(mList, m)
		}
	}
	return mList
}

type pagerData struct {
	CurrPageNum   int
	TotalPageCnt  int
	PageUriPrefix string
	IndexPageUri  string
}

type embeddedMediaType int

const (
	UndefinedEmbeddedMediaType embeddedMediaType = iota
	YouTube
	Vimeo
)

var embeddedMediaTypes = /* const */ []embeddedMediaType{YouTube, Vimeo}

var embeddedMediaTypeRegexp = /* const */ map[embeddedMediaType]*regexp.Regexp{
	YouTube: regexp.MustCompile(`(?i)(youtu\.be/|youtube\.com/watch\?v=)([\w_-]+)`),
	Vimeo:   regexp.MustCompile(`(?i)vimeo\.com/([\w_-]+)`),
}

func (c embeddedMediaType) String() string {
	switch c {
	case YouTube:
		return "youtube"
	case Vimeo:
		return "vimeo"
	}
	panic("Undefined Embedded Media Type")
}

func (c embeddedMediaType) getCode(url string) string {
	var codeGroup int
	switch c {
	case YouTube:
		codeGroup = 2
	case Vimeo:
		codeGroup = 1
	default:
		panic("Undefined Embedded Media Type")
	}
	emm := embeddedMediaTypeRegexp[c].FindStringSubmatch(url)
	if emm != nil {
		return emm[codeGroup]
	}
	return ""
}

type embeddedMedia struct {
	MediaType embeddedMediaType
	Code      string
}

type mediaType int

const (
	UndefinedMediaType mediaType = iota
	Image
	Video
)

func (m mediaType) Image() bool {
	return m == Image
}

func (m mediaType) Video() bool {
	return m == Video
}

type thumb struct {
	Uri  string
	Size int
}

type media struct {
	Type   mediaType
	Uri    string
	thumbs []thumb
}

func (m media) SrcSet() string {
	minThumbSize := defaultThumbSizes[0]
	maxThumbSize := defaultThumbSizes[len(defaultThumbSizes)-1]
	var srcSet []string
	for _, thumb := range m.thumbs {
		srcSet = append(srcSet, fmt.Sprintf("%s %dw", thumb.Uri, thumb.Size))
		if thumb.Size < minThumbSize {
			minThumbSize = thumb.Size
		}
		if thumb.Size > maxThumbSize {
			maxThumbSize = thumb.Size
		}
	}
	srcSet = append(srcSet, fmt.Sprintf("%s %dw", m.Uri, maxThumbSize+(minThumbSize+maxThumbSize)/2))
	return strings.Join(srcSet, ", ")
}

func (m media) ThumbUri(sizeIdx int) string {
	return m.thumbs[sizeIdx-1].Uri
}

type searchData struct {
	TypeId  string
	Content string
}

type contentEntity interface {
	ContentEntityType() contentEntityType
	EntityId() string
}

type page struct {
	Id             string
	Title          string
	Body           string
	Media          []media
	SearchData     searchData
	skipProcessing bool
}

func (p page) ContentEntityType() contentEntityType {
	return Page
}

func (p page) EntityId() string {
	return p.Id
}

type post struct {
	Id             string
	Date           civil.Date
	Time           civil.Time
	Title          string
	Body           string
	Tags           []string
	SearchData     searchData
	skipProcessing bool
}

func (p post) ContentEntityType() contentEntityType {
	return Post
}

func (p post) EntityId() string {
	return p.Id
}

func (p post) HasDateOrTime() bool {
	return !p.Date.IsZero() || !p.Time.IsZero()
}

func (p post) FmtDate() string {
	if !p.Date.IsZero() {
		return p.Date.String()
	}
	return ""
}

func (p post) FmtTime() string {
	if !p.Time.IsZero() {
		t := p.Time.String()
		if strings.HasSuffix(t, ":00") {
			t, _ = strings.CutSuffix(t, ":00")
		}
		return t
	}
	return ""
}

type archiveIndexData struct {
	YearData []archiveYearData
}

type archiveYearData struct {
	Year      int
	MonthData []archiveMonthData
}

type archiveMonthData struct {
	Month   time.Month
	PostCnt int
}

type tuple2[T1, T2 any] struct {
	V1 T1
	V2 T2
}

type mapItem struct {
	Key, Value interface{}
}

type mapSlice []mapItem

func (ms mapSlice) MarshalJSON() ([]byte, error) {
	buf := &bytes.Buffer{}
	buf.Write([]byte{'{'})
	for i, mi := range ms {
		b, err := json.Marshal(&mi.Value)
		if err != nil {
			return nil, err
		}
		buf.WriteString(fmt.Sprintf("%q:", fmt.Sprintf("%v", mi.Key)))
		buf.Write(b)
		if i < len(ms)-1 {
			buf.Write([]byte{','})
		}
	}
	buf.Write([]byte{'}'})
	return buf.Bytes(), nil
}

type resourceLoader struct {
	config       appConfig
	loadTemplate func(templateFileName string) ([]byte, error)
	loadInclude  func(includeFileName string, level templateIncludeLevel) ([]byte, error)
}

type tagData struct {
	Title string
	Count int
	Ratio float64
}

type stats struct {
	pageCnt int
	postCnt int
	tagCnt  int
	genCnt  int
}

type watchReloadData struct {
	Type    contentEntityType `json:"type"`
	Id      string            `json:"id"`
	Deleted bool              `json:"deleted"`
}

type processorOutputHandler func(outputFilePath string, data []byte) bool

type imageThumbnailHandler func(imgDirPath string, config appConfig)

type dirChangeHandler func(changedFilePath string, deleted bool)
