package main

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
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
		id:    page1Id,
		Title: "Test Page 1 Title",
		Body:  "Test Page 1 Body",
	}

	page2Id := "test-page-2"
	page2 := page{
		id:    page2Id,
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
		id:    post1Id,
		Date:  "2023-08-01",
		Title: "Test Post 1 Title",
		Body:  "Test Post 1 Body",
		Tags:  []string{tag1, tag2},
	}

	post2Id := "post-2"
	post2 := post{
		id:    post2Id,
		Date:  "2023-09-01",
		Title: "Test Post 2 Title",
		Body:  "Test Post 2 Body",
		Tags:  []string{tag1, tag3},
	}

	posts := []post{post1, post2}

	globalIncludes := map[string]string{
		"header.html": globalIncludeHeaderContent,
	}

	themeIncludes := map[string]string{
		"header.html": themeIncludeHeaderContent,
	}

	config := defaultConfig()
	config.siteName = testSiteName

	if customHomePage {
		config.homePage = page1Id
	}

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	expectedIndexFile := deployDirName + "/" + indexPageFileName
	expectedPost1File := deployDirName + "/" + deployPostDirName + "/" + post1.id + contentFileExtension
	expectedPost2File := deployDirName + "/" + deployPostDirName + "/" + post2.id + contentFileExtension
	expectedPage1File := deployDirName + "/" + deployPageDirName + "/" + page1.id + contentFileExtension
	expectedPage2File := deployDirName + "/" + deployPageDirName + "/" + page2.id + contentFileExtension
	expectedFiles := []string{
		expectedIndexFile,
		expectedPost1File,
		expectedPost2File,
		expectedPage2File,
	}

	for _, tag := range tags {
		expectedFiles = append(expectedFiles, deployDirName+"/"+deployTagDirName+"/"+strings.ToLower(tag)+"/"+indexPageFileName)
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
			expectedIndexFileContent = append(expectedIndexFileContent, "<a href=\"/"+deployPostDirName+"/"+post.id+contentFileExtension+"\">")
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
					if strings.Contains(ef, page.id+contentFileExtension) {
						verifyExpectedNonIndexOutputFileContentContainsPageContent(ef, outputFileContent, page, t)
					}
				}
				for _, post := range posts {
					if strings.Contains(ef, post.id+contentFileExtension) {
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
			id:    "post-" + pn,
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
	config.siteName = testSiteName

	output := processOutput(pages, posts, globalIncludes, themeIncludes, config)

	expectedFiles := []string{
		deployDirName + "/" + indexPageFileName,
	}

	unexpectedFiles := []string{
		deployDirName + "/" + deployTagDirName + "/" + strings.ToLower(tag2) + "/2" + contentFileExtension,
	}

	totalPageCnt := postCnt / defaultPageSize
	if postCnt%defaultPageSize > 0 {
		totalPageCnt++
	}

	for i := 2; i <= totalPageCnt; i++ {
		expectedFiles = append(expectedFiles, deployDirName+"/"+deployPostsDirName+"/"+strconv.Itoa(i)+contentFileExtension)
	}

	for tag, cnt := range tagCnt {
		expectedFiles = append(expectedFiles, deployDirName+"/"+deployTagDirName+"/"+strings.ToLower(tag)+"/"+indexPageFileName)
		if cnt > defaultPageSize {
			tagPageCnt := cnt / defaultPageSize
			if cnt%defaultPageSize > 0 {
				tagPageCnt++
			}
			for i := 2; i <= tagPageCnt; i++ {
				expectedFiles = append(expectedFiles, deployDirName+"/"+deployTagDirName+"/"+strings.ToLower(tag)+"/"+strconv.Itoa(i)+contentFileExtension)
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
				outputFile = deployDirName + "/" + deployTagDirName + "/" + strings.ToLower(tag) + "/" + indexPageFileName
			} else {
				outputFile = deployDirName + "/" + deployTagDirName + "/" + strings.ToLower(tag) + "/" + strconv.Itoa(i) + contentFileExtension
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
	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
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
		fmt.Sprintf("<header class=\"title\">%s</header>", p.Title),
		fmt.Sprintf("<section class=\"content\">%s</section>", p.Body),
	}
	return expectedContent
}

func getExpectedPostFileContent(p post) []string {
	expectedContent := []string{
		fmt.Sprintf("<span class=\"date\">%s</span>", p.Date),
		fmt.Sprintf("<span class=\"title\">%s</span>", p.Title),
		fmt.Sprintf("<section class=\"content\">%s</section>", p.Body),
	}
	if len(p.Tags) > 0 {
		expectedTagsContent := "<span class=\"tags\">"
		for _, tag := range p.Tags {
			expectedTagsContent += fmt.Sprintf("<a class=\"tag\" href=\"/tag/%s/\">%s</a>", strings.ToLower(tag), tag)
		}
		expectedTagsContent += "</span>"
		expectedContent = append(expectedContent, expectedTagsContent)
	}
	return expectedContent
}

func missingExpectedContentError(t *testing.T, fileName string, expectedContent string) {
	t.Errorf("Processed file - %s - is missing expected content: %s", fileName, expectedContent)
}
