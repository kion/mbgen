package app

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"cloud.google.com/go/civil"
)

const testSiteName = "[TEST SITE]"
const globalIncludeHeaderContent = "[TEST HEADER - GLOBAL LEVEL]"
const themeIncludeHeaderContent = "[TEST HEADER - THEME LEVEL]"

var expectedNonIndexFileContent = /* const */ []string{
	"<title>" + testSiteName + " - %s</title>",
}

var trimRegexp = /* const */ regexp.MustCompile(`\n+\s+`)

func TestWithoutCustomHomePage(t *testing.T) {
	basicTest(t, false)
}

func TestWithCustomHomePage(t *testing.T) {
	basicTest(t, true)
}

func basicTest(t *testing.T, customHomePage bool) {

	page1Id := "test-page-1"
	page1 := page{
		Id:    page1Id,
		Title: "Test Page 1 Title",
		Body:  "Test Page 1 Body",
	}

	page2Id := "test-page-2"
	page2 := page{
		Id:    page2Id,
		Title: "Test Page 2 Title",
		Body:  "Test Page 2 Body",
	}

	pages := []page{page1, page2}

	tag1 := "Tag1"
	tag2 := "Tag2"
	tag3 := "Tag3"

	tags := []string{tag1, tag2, tag3}

	post1Id := "post-1"
	post1 := post{
		Id:    post1Id,
		Title: "Test Post 1 Title",
		Body:  "Test Post 1 Body",
		Tags:  []string{tag1, tag2},
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")
	post1.Time, _ = civil.ParseTime("19:15:00")

	post2Id := "post-2"
	post2 := post{
		Id:    post2Id,
		Title: "Test Post 2 Title",
		Body:  "Test Post 2 Body",
		Tags:  []string{tag1, tag3},
	}
	post2.Date, _ = civil.ParseDate("2023-09-01")
	post2.Time, _ = civil.ParseTime("09:05:00")

	posts := []post{post1, post2}

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	config := defaultConfig()
	config.enableSearch = false
	config.siteName = testSiteName

	if customHomePage {
		config.homePage = page1Id
	}

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	expectedIndexFile := deployDirName + "/" + indexPageFileName
	expectedPost1File := deployDirName + "/" + deployPostDirName + "/" + post1.Id + contentFileExtension
	expectedPost2File := deployDirName + "/" + deployPostDirName + "/" + post2.Id + contentFileExtension
	expectedPage1File := deployDirName + "/" + deployPageDirName + "/" + page1.Id + contentFileExtension
	expectedPage2File := deployDirName + "/" + deployPageDirName + "/" + page2.Id + contentFileExtension
	expectedFiles := []string{
		expectedIndexFile,
		expectedPost1File,
		expectedPost2File,
		expectedPage2File,
	}

	for _, tag := range tags {
		expectedFiles = append(expectedFiles, deployDirName+"/"+deployTagsDirName+"/"+strings.ToLower(tag)+"/"+indexPageFileName)
	}

	if !customHomePage {
		expectedFiles = append(expectedFiles, expectedPage1File)
	}

	expectedAnyFileContent := []string{
		"<header>" + globalIncludeHeaderContent + themeIncludeHeaderContent + "</header>",
	}

	var expectedIndexFileContent []string

	if customHomePage {
		expectedIndexFileContent = append(expectedIndexFileContent,
			"<title>"+testSiteName+" - "+page1.Title+"</title>")
	} else {
		expectedIndexFileContent = append(expectedIndexFileContent,
			"<title>"+testSiteName+"</title>")
		for _, post := range posts {
			expectedIndexFileContent = append(expectedIndexFileContent, "<a href=\"/"+deployPostDirName+"/"+post.Id+contentFileExtension+"\"")
			for _, epc := range getExpectedPostFileContent(post) {
				expectedIndexFileContent = append(expectedIndexFileContent, epc)
			}
		}
	}

	for _, ef := range expectedFiles {
		outputFileContent, ok := output[ef]
		if !ok {
			t.Error("Missing expected output file: " + ef)
		} else {
			for _, ec := range expectedAnyFileContent {
				if !strings.Contains(outputFileContent, ec) {
					missingExpectedContentError(t, ef, ec)
				}
			}
			if ef == expectedIndexFile {
				for _, ec := range expectedIndexFileContent {
					if !strings.Contains(outputFileContent, ec) {
						missingExpectedContentError(t, ef, ec)
					}
				}
			} else {
				for _, page := range pages {
					if strings.Contains(ef, page.Id+contentFileExtension) {
						verifyExpectedNonIndexOutputFileContentContainsPageContent(ef, outputFileContent, page, t)
					}
				}
				for _, post := range posts {
					if strings.Contains(ef, post.Id+contentFileExtension) {
						verifyExpectedNonIndexOutputFileContentContainsPostContent(ef, outputFileContent, post, t)
					}
				}
				for _, tag := range tags {
					if strings.Contains(ef, strings.ToLower(tag)+contentFileExtension) {
						for _, post := range posts {
							if slices.Contains(post.Tags, tag) {
								verifyExpectedNonIndexOutputFileContentContainsPostContent(ef, outputFileContent, post, t)
							}
						}
						break
					}
				}
			}
		}
	}

}

func TestPagination1(t *testing.T) {
	testPagination(t, 19)
}

func TestPagination2(t *testing.T) {
	testPagination(t, 40)
}

func TestPagination3(t *testing.T) {
	testPagination(t, 59)
}

func testPagination(t *testing.T, postCnt int) {

	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	tag1 := "Tag1"
	tag2 := "Tag2"

	tagCnt := map[string]int{tag1: 0, tag2: 0}

	for i := 1; i <= postCnt; i++ {
		pn := strconv.Itoa(i)
		p := post{
			Id:    "post-" + pn,
			Title: "Test Post " + pn + " Title",
			Body:  "Test Post " + pn + " Body",
			Tags:  []string{tag1},
		}
		tagCnt[tag1] += 1
		if i < defaultPageSize {
			p.Tags = append(p.Tags, tag2)
			tagCnt[tag2] += 1
		}
		posts = append(posts, p)
	}

	config := defaultConfig()
	config.enableSearch = false
	config.siteName = testSiteName

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	expectedFiles := []string{
		deployDirName + "/" + indexPageFileName,
	}

	unexpectedFiles := []string{
		deployDirName + "/" + deployTagsDirName + "/" + strings.ToLower(tag2) + "/2" + contentFileExtension,
	}

	totalPageCnt := postCnt / defaultPageSize
	if postCnt%defaultPageSize > 0 {
		totalPageCnt++
	}

	for i := 2; i <= totalPageCnt; i++ {
		expectedFiles = append(expectedFiles, deployDirName+"/"+deployPostsDirName+"/"+strconv.Itoa(i)+contentFileExtension)
	}

	for tag, cnt := range tagCnt {
		expectedFiles = append(expectedFiles, deployDirName+"/"+deployTagsDirName+"/"+strings.ToLower(tag)+"/"+indexPageFileName)
		if cnt > defaultPageSize {
			tagPageCnt := cnt / defaultPageSize
			if cnt%defaultPageSize > 0 {
				tagPageCnt++
			}
			for i := 2; i <= tagPageCnt; i++ {
				expectedFiles = append(expectedFiles, deployDirName+"/"+deployTagsDirName+"/"+strings.ToLower(tag)+"/"+strconv.Itoa(i)+contentFileExtension)
			}
		}
	}

	for _, ef := range expectedFiles {
		_, ok := output[ef]
		if !ok {
			t.Error("Missing expected output file: " + ef)
		}
	}

	for _, ef := range unexpectedFiles {
		_, ok := output[ef]
		if ok {
			t.Error("Unexpected output file: " + ef)
		}
	}

	for i := 1; i <= totalPageCnt; i++ {
		var outputFile string
		if i == 1 {
			outputFile = deployDirName + "/" + indexPageFileName
		} else {
			outputFile = deployDirName + "/" + deployPostsDirName + "/" + strconv.Itoa(i) + contentFileExtension
		}
		outputFileContent := output[outputFile]
		for j := (i - 1) * defaultPageSize; j < i*defaultPageSize && j < postCnt; j++ {
			verifyExpectedOutputFileContentContainsPostContent(outputFile, outputFileContent, posts[j], t)
		}
	}

	for tag, cnt := range tagCnt {
		for i := 1; i <= cnt; i++ {
			var outputFile string
			if i == 1 {
				outputFile = deployDirName + "/" + deployTagsDirName + "/" + strings.ToLower(tag) + "/" + indexPageFileName
			} else {
				outputFile = deployDirName + "/" + deployTagsDirName + "/" + strings.ToLower(tag) + "/" + strconv.Itoa(i) + contentFileExtension
			}
			outputFileContent := output[outputFile]
			for j := (i - 1) * defaultPageSize; j < i*defaultPageSize && j < cnt; j++ {
				verifyExpectedOutputFileContentContainsPostContent(outputFile, outputFileContent, posts[j], t)
			}
		}
	}

}

func processOutput(pages []page, posts []post,
	globalIncludes map[string]string,
	themeIncludes map[string]string,
	config appConfig) map[string]string {
	output := make(map[string]string)
	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "../../themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	process(pages, posts,
		resourceLoader{
			config: config,
			loadTemplate: func(templateFileName string) ([]byte, error) {
				templateFilePath := fmt.Sprintf("%s%c%s", defaultThemeTemplatesDir, os.PathSeparator, templateFileName)
				return os.ReadFile(templateFilePath)
			},
			loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
				var includeContent string
				var ok bool
				switch level {
				case Global:
					includeContent, ok = globalIncludes[includeFileName]
				case Theme:
					includeContent, ok = themeIncludes[includeFileName]
				}
				if ok {
					return []byte(includeContent), nil
				} else {
					return nil, nil
				}
			},
		},
		func(outputFilePath string, data []byte) bool {
			output[outputFilePath] = strings.ReplaceAll(string(trimRegexp.ReplaceAll(data, []byte(""))), "\n", "")
			return false
		})
	return output
}

func verifyExpectedNonIndexOutputFileContentContainsPageContent(outputFile string, outputFileContent string, p page, t *testing.T) {
	for _, ec := range expectedNonIndexFileContent {
		etc := fmt.Sprintf(ec, p.Title)
		if !strings.Contains(outputFileContent, etc) {
			missingExpectedContentError(t, outputFile, etc)
		}
		for _, epc := range getExpectedPageFileContent(p) {
			if !strings.Contains(outputFileContent, epc) {
				missingExpectedContentError(t, outputFile, epc)
			}
		}
	}
}

func verifyExpectedNonIndexOutputFileContentContainsPostContent(outputFile string, outputFileContent string, p post, t *testing.T) {
	for _, ec := range expectedNonIndexFileContent {
		etc := fmt.Sprintf(ec, p.Title)
		if !strings.Contains(outputFileContent, etc) {
			missingExpectedContentError(t, outputFile, etc)
		}
		for _, epc := range getExpectedPostFileContent(p) {
			if !strings.Contains(outputFileContent, epc) {
				missingExpectedContentError(t, outputFile, epc)
			}
		}
	}
}

func verifyExpectedOutputFileContentContainsPostContent(outputFile string, outputFileContent string, p post, t *testing.T) {
	for _, epc := range getExpectedPostFileContent(p) {
		if !strings.Contains(outputFileContent, epc) {
			missingExpectedContentError(t, outputFile, epc)
		}
	}
}

func getExpectedPageFileContent(p page) []string {
	expectedContent := []string{
		fmt.Sprintf("<header class=\"title\"><span class=\"title\">%s</span></header>", p.Title),
		fmt.Sprintf("<section class=\"content\">%s</section>", p.Body),
	}
	return expectedContent
}

func getExpectedPostFileContent(p post) []string {
	expectedContent := []string{
		fmt.Sprintf("<span class=\"title\">%s</span>", p.Title),
		fmt.Sprintf("<section class=\"content\">%s</section>", p.Body),
	}
	if !p.Date.IsZero() {
		expectedContent = append(expectedContent, fmt.Sprintf("<span class=\"date\">%s</span>", p.FmtDate()))
	}
	if !p.Time.IsZero() {
		expectedContent = append(expectedContent, fmt.Sprintf("<span class=\"time\">%s</span>", p.FmtTime()))
	}
	if len(p.Tags) > 0 {
		expectedTagsContent := "<span class=\"tags\">"
		for _, tag := range p.Tags {
			expectedTagsContent += fmt.Sprintf("<a class=\"tag\" href=\"/tags/%s/\">%s</a>", strings.ToLower(tag), tag)
		}
		expectedTagsContent += "</span>"
		expectedContent = append(expectedContent, expectedTagsContent)
	}
	return expectedContent
}

func missingExpectedContentError(t *testing.T, fileName string, expectedContent string) {
	t.Errorf("Processed file - %s - is missing expected content: %s", fileName, expectedContent)
}

func TestFeedGenerationRSS(t *testing.T) {
	testFeedGeneration(t, []string{feedFormatRSS})
}

func TestFeedGenerationAtom(t *testing.T) {
	testFeedGeneration(t, []string{feedFormatAtom})
}

func TestFeedGenerationJSON(t *testing.T) {
	testFeedGeneration(t, []string{feedFormatJSON})
}

func TestFeedGenerationAll(t *testing.T) {
	testFeedGeneration(t, []string{feedFormatRSS, feedFormatAtom, feedFormatJSON})
}

func testFeedGeneration(t *testing.T, feedFormats []string) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	post1 := post{
		Id:          "post-1",
		Title:       "Test Post 1 Title",
		Body:        "Test Post 1 Body",
		FeedContent: "Test Post 1 Body",
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")
	post1.Time, _ = civil.ParseTime("19:15:00")

	post2 := post{
		Id:          "post-2",
		Title:       "Test Post 2 Title",
		Body:        "Test Post 2 Body",
		FeedContent: "Test Post 2 Body",
	}
	post2.Date, _ = civil.ParseDate("2023-07-15")

	posts = []post{post1, post2}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = feedFormats
	config.feedPostCnt = 2
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	for _, format := range feedFormats {
		var expectedFeedFile string
		switch format {
		case feedFormatRSS:
			expectedFeedFile = deployDirName + "/" + feedFileNameRSS
		case feedFormatAtom:
			expectedFeedFile = deployDirName + "/" + feedFileNameAtom
		case feedFormatJSON:
			expectedFeedFile = deployDirName + "/" + feedFileNameJSON
		}

		feedContent, ok := output[expectedFeedFile]
		if !ok {
			t.Error("Missing expected feed file: " + expectedFeedFile)
			continue
		}

		if !strings.Contains(feedContent, config.siteBaseURL) {
			t.Errorf("Feed %s missing siteBaseURL", expectedFeedFile)
		}

		for _, post := range posts {
			if !strings.Contains(feedContent, post.FmtDate()) {
				t.Errorf("Feed %s missing post date: %s", expectedFeedFile, post.FmtDate())
			}
		}
	}
}

func TestFeedExcerptGeneration(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// test post with multiple sentences (should extract first 3)
	post1 := post{
		Id:          "post-1",
		Title:       "Multi-Sentence Post",
		Body:        "<p>Test body</p>",
		FeedContent: "First sentence here. Second sentence here. Third sentence here. Fourth sentence should not appear. Fifth sentence also not.",
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")

	posts = []post{post1}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 1
	config.enableSearch = false
	config.feedPostViewOnWebsiteLinkText = "Read more"

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// should contain first three sentences
	if !strings.Contains(feedContent, "First sentence here") {
		t.Error("Feed missing first sentence")
	}
	if !strings.Contains(feedContent, "Second sentence here") {
		t.Error("Feed missing second sentence")
	}
	if !strings.Contains(feedContent, "Third sentence here") {
		t.Error("Feed missing third sentence")
	}

	// should NOT contain fourth and fifth sentences
	if strings.Contains(feedContent, "Fourth sentence should not appear") {
		t.Error("Feed should not contain fourth sentence")
	}
	if strings.Contains(feedContent, "Fifth sentence also not") {
		t.Error("Feed should not contain fifth sentence")
	}

	// should contain ellipsis for truncation
	if !strings.Contains(feedContent, "...") {
		t.Error("Feed should contain ellipsis for truncated content")
	}

	// should contain view on website link
	if !strings.Contains(feedContent, "Read more") {
		t.Error("Feed missing view on website link text")
	}
	if !strings.Contains(feedContent, "https://example.com/post/post-1.html") {
		t.Error("Feed missing view on website link URL")
	}
}

func TestFeedExcerptWordFallback(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// test post with very long single sentence - exceeds 3 sentence limit but tests word handling
	longText := ""
	for i := 1; i <= 100; i++ {
		longText += "word" + strconv.Itoa(i) + " "
	}
	longText = strings.TrimSpace(longText) + "."

	post1 := post{
		Id:          "post-long-sentence",
		Title:       "Post With Long Sentence",
		Body:        "<p>Test body</p>",
		FeedContent: longText,
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")

	posts = []post{post1}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 1
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// should contain the single long sentence (treated as 1 of 3 allowed sentences)
	if !strings.Contains(feedContent, "word1") {
		t.Error("Feed missing first word")
	}
	if !strings.Contains(feedContent, "word100") {
		t.Error("Feed missing last word of sentence")
	}

	// should NOT contain ellipsis since this is the complete content (1 sentence that is the entire post)
	if strings.Contains(feedContent, "...") {
		t.Error("Feed should not contain ellipsis when content is complete")
	}
}

func TestFeedTitleWithTags(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// post without title but with tags
	post1 := post{
		Id:          "post-with-tags",
		Title:       "", // no title
		Body:        "<p>Test body</p>",
		FeedContent: "Post content here.",
		Tags:        []string{"Cycling", "Adventure", "Travel"},
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")
	post1.Time, _ = civil.ParseTime("14:30:00")

	posts = []post{post1}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 1
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// title should contain date, time, and tags with # prefix
	if !strings.Contains(feedContent, "2023-08-01 14:30") {
		t.Error("Feed title missing date and time")
	}
	if !strings.Contains(feedContent, "#Cycling") {
		t.Error("Feed title missing #Cycling tag")
	}
	if !strings.Contains(feedContent, "#Adventure") {
		t.Error("Feed title missing #Adventure tag")
	}
	if !strings.Contains(feedContent, "#Travel") {
		t.Error("Feed title missing #Travel tag")
	}
}

func TestFeedRelativeURLConversion(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// post with relative URLs in markdown (as it would be after parsing hashtags)
	// hashtags are converted to markdown links during parsing: #Technology -> [#Technology](/tags/technology/)
	post1 := post{
		Id:          "post-with-links",
		Title:       "Post With Links",
		Body:        "<p>Test body</p>",
		FeedContent: "Check out [my page](/page/about) and tags like [#Technology](/tags/technology/) and [#Programming](/tags/programming/).",
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")

	posts = []post{post1}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 1
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// relative URLs should be converted to absolute
	if !strings.Contains(feedContent, "https://example.com/page/about") {
		t.Error("Feed should contain absolute URL for page link")
	}
	if !strings.Contains(feedContent, "https://example.com/tags/technology") {
		t.Error("Feed should contain absolute URL for technology tag")
	}
	if !strings.Contains(feedContent, "https://example.com/tags/programming") {
		t.Error("Feed should contain absolute URL for programming tag")
	}

	// should NOT contain relative URLs
	if strings.Contains(feedContent, `href="/page/about"`) {
		t.Error("Feed should not contain relative URL")
	}
}

func TestFeedExcerptNoDoublePeriods(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// test post with multiple sentences - verify no double periods
	post1 := post{
		Id:          "post-periods",
		Title:       "Test Periods",
		Body:        "<p>Test body</p>",
		FeedContent: "First sentence. Second sentence. Third sentence. Fourth sentence.",
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")

	posts = []post{post1}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 1
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// should NOT contain double periods
	if strings.Contains(feedContent, "sentence.. ") {
		t.Error("Feed should not contain double periods")
	}

	// should contain properly spaced sentences
	if !strings.Contains(feedContent, "First sentence.") {
		t.Error("Feed missing first sentence with period")
	}
	if !strings.Contains(feedContent, "Second sentence.") {
		t.Error("Feed missing second sentence with period")
	}
	if !strings.Contains(feedContent, "Third sentence.") {
		t.Error("Feed missing third sentence with period")
	}
}

func TestFeedExcerptEllipsisOnlyWhenTruncated(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// test post with exactly 3 sentences - should NOT have ellipsis
	post1 := post{
		Id:          "post-exact-three",
		Title:       "Exact Three Sentences",
		Body:        "<p>Test body</p>",
		FeedContent: "First sentence here. Second sentence here. Third sentence here.",
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")

	// test post with less than 3 sentences - should NOT have ellipsis
	post2 := post{
		Id:          "post-two-sentences",
		Title:       "Two Sentences",
		Body:        "<p>Test body</p>",
		FeedContent: "First sentence. Second sentence.",
	}
	post2.Date, _ = civil.ParseDate("2023-08-02")

	// test post with more than 3 sentences - SHOULD have ellipsis
	post3 := post{
		Id:          "post-four-sentences",
		Title:       "Four Sentences",
		Body:        "<p>Test body</p>",
		FeedContent: "First sentence. Second sentence. Third sentence. Fourth sentence.",
	}
	post3.Date, _ = civil.ParseDate("2023-08-03")

	posts = []post{post1, post2, post3}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 3
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// Extract individual feed items for each post
	// For post1 (exactly 3 sentences): should NOT have ellipsis
	if strings.Contains(feedContent, "Exact Three Sentences") {
		// Check that this post's content doesn't have ellipsis
		// This is tricky to verify in the full feed, but we can check that
		// "Third sentence here..." doesn't appear (would be double periods + ellipsis)
		if strings.Contains(feedContent, "Third sentence here...") {
			t.Error("Post with exactly 3 sentences should not have ellipsis")
		}
	}

	// For post3 (4 sentences): SHOULD have ellipsis after third sentence
	if strings.Contains(feedContent, "Four Sentences") {
		if !strings.Contains(feedContent, "Third sentence...") {
			t.Error("Post with more than 3 sentences should have ellipsis")
		}
		if strings.Contains(feedContent, "Fourth sentence") {
			t.Error("Fourth sentence should not appear in truncated excerpt")
		}
	}
}

func TestFeedExcerptAbbreviations(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// test post with abbreviations - should not break on "i.e." or "e.g."
	post1 := post{
		Id:          "post-abbreviations",
		Title:       "Post With Abbreviations",
		Body:        "<p>Test body</p>",
		FeedContent: "This is about microblogging, i.e. short form content. You can use it for notes, e.g. daily observations. There are many benefits to this approach.",
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")

	posts = []post{post1}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 1
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// should contain full sentences with abbreviations intact
	if !strings.Contains(feedContent, "i.e. short form content") {
		t.Error("Feed should preserve 'i.e.' abbreviation within sentence")
	}
	if !strings.Contains(feedContent, "e.g. daily observations") {
		t.Error("Feed should preserve 'e.g.' abbreviation within sentence")
	}

	// should extract all 3 sentences (not break on abbreviations)
	if !strings.Contains(feedContent, "This is about microblogging") {
		t.Error("Feed missing first sentence")
	}
	if !strings.Contains(feedContent, "You can use it for notes") {
		t.Error("Feed missing second sentence")
	}
	if !strings.Contains(feedContent, "There are many benefits") {
		t.Error("Feed missing third sentence")
	}
}

func TestFeedExcerptExclamationAndQuestionMarks(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// test post with sentences ending in !, ?, and .
	post1 := post{
		Id:          "post-mixed-punctuation",
		Title:       "Post With Mixed Punctuation",
		Body:        "<p>Test body</p>",
		FeedContent: "This is exciting! Can you believe it? Yes, I can. This fourth sentence should not appear.",
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")

	// test post with only exclamation marks
	post2 := post{
		Id:          "post-exclamations",
		Title:       "Post With Exclamations",
		Body:        "<p>Test body</p>",
		FeedContent: "First exciting thing! Second exciting thing! Third exciting thing! Fourth should not appear!",
	}
	post2.Date, _ = civil.ParseDate("2023-08-02")

	// test post with only question marks
	post3 := post{
		Id:          "post-questions",
		Title:       "Post With Questions",
		Body:        "<p>Test body</p>",
		FeedContent: "What is this? How does it work? Why does it matter? When will it happen?",
	}
	post3.Date, _ = civil.ParseDate("2023-08-03")

	posts = []post{post1, post2, post3}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 3
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// Post 1: mixed punctuation
	if !strings.Contains(feedContent, "This is exciting!") {
		t.Error("Feed should contain sentence ending with exclamation mark")
	}
	if !strings.Contains(feedContent, "Can you believe it?") {
		t.Error("Feed should contain sentence ending with question mark")
	}
	if !strings.Contains(feedContent, "Yes, I can.") {
		t.Error("Feed should contain sentence ending with period")
	}
	if strings.Contains(feedContent, "This fourth sentence should not appear") {
		t.Error("Fourth sentence should not appear in excerpt")
	}

	// Post 2: only exclamations - should extract 3 sentences
	// Since content is truncated, final punctuation will be replaced with ...
	if !strings.Contains(feedContent, "First exciting thing!") {
		t.Error("Feed should contain first exclamation sentence")
	}
	if !strings.Contains(feedContent, "Second exciting thing!") {
		t.Error("Feed should contain second exclamation sentence")
	}
	if !strings.Contains(feedContent, "Third exciting thing...") {
		t.Error("Feed should contain third exclamation sentence with ellipsis (truncated)")
	}
	if strings.Contains(feedContent, "Fourth should not appear") {
		t.Error("Fourth exclamation sentence should not appear in excerpt")
	}

	// Post 3: only questions - should extract 3 sentences
	// Since content is truncated, final punctuation will be replaced with ...
	if !strings.Contains(feedContent, "What is this?") {
		t.Error("Feed should contain first question")
	}
	if !strings.Contains(feedContent, "How does it work?") {
		t.Error("Feed should contain second question")
	}
	if !strings.Contains(feedContent, "Why does it matter...") {
		t.Error("Feed should contain third question with ellipsis (truncated)")
	}
	if strings.Contains(feedContent, "When will it happen") {
		t.Error("Fourth question should not appear in excerpt")
	}
}

func TestFeedExcerptEllipsisReplacementForAllPunctuation(t *testing.T) {
	var pages []page
	var posts []post

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	// test truncated content ending with period - should become ...
	post1 := post{
		Id:          "post-truncated-period",
		Title:       "Truncated Period",
		Body:        "<p>Test body</p>",
		FeedContent: "First sentence. Second sentence. Third sentence. Fourth sentence not shown.",
	}
	post1.Date, _ = civil.ParseDate("2023-08-01")

	// test truncated content ending with exclamation - should become ...
	post2 := post{
		Id:          "post-truncated-exclamation",
		Title:       "Truncated Exclamation",
		Body:        "<p>Test body</p>",
		FeedContent: "First sentence! Second sentence! Third sentence! Fourth not shown!",
	}
	post2.Date, _ = civil.ParseDate("2023-08-02")

	// test truncated content ending with question - should become ...
	post3 := post{
		Id:          "post-truncated-question",
		Title:       "Truncated Question",
		Body:        "<p>Test body</p>",
		FeedContent: "First sentence? Second sentence? Third sentence? Fourth not shown?",
	}
	post3.Date, _ = civil.ParseDate("2023-08-03")

	posts = []post{post1, post2, post3}

	config := defaultConfig()
	config.siteBaseURL = "https://example.com"
	config.siteName = testSiteName
	config.generateFeeds = []string{feedFormatRSS}
	config.feedPostCnt = 3
	config.enableSearch = false

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	feedFile := deployDirName + "/" + feedFileNameRSS
	feedContent, ok := output[feedFile]
	if !ok {
		t.Fatal("Missing expected feed file")
	}

	// Post 1: truncated with period - should have "..." not "...."
	if strings.Contains(feedContent, "Third sentence....") {
		t.Error("Should not have quadruple periods (period + ellipsis)")
	}
	if !strings.Contains(feedContent, "Third sentence...") {
		t.Error("Truncated content ending with period should show ellipsis")
	}

	// Post 2: truncated with exclamation - should replace ! with ...
	if strings.Contains(feedContent, "Third sentence!...") {
		t.Error("Should not have exclamation + ellipsis, should replace ! with ...")
	}
	if !strings.Contains(feedContent, "Third sentence...") {
		t.Error("Truncated content ending with exclamation should replace with ellipsis")
	}

	// Post 3: truncated with question - should replace ? with ...
	if strings.Contains(feedContent, "Third sentence?...") {
		t.Error("Should not have question mark + ellipsis, should replace ? with ...")
	}
	if !strings.Contains(feedContent, "Third sentence...") {
		t.Error("Truncated content ending with question should replace with ellipsis")
	}
}
