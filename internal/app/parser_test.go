package app

import (
	"fmt"
	"os"
	"slices"
	"strings"
	"testing"
)

const testPageTitle = "Test Page"
const testPageBody = "Test page body."
const testPostTitle = "Test Post"
const testPostBody = "Test post body."
const testEmbed1Type = Vimeo
const testEmbed1Code = "1234567890"
const testEmbed2Type = YouTube
const testEmbed2Code = "a_BcDeFgHiJ-x"
const testEmbed3Type = YouTube
const testEmbed3Code = "A_bCdEfGhIj-X"
const testEmbed1 = "http://vimeo.com/" + testEmbed1Code
const testEmbed2 = "http://youtu.be/" + testEmbed2Code
const testEmbed3 = "http://www.youtube.com/watch?v=" + testEmbed3Code
const testPageLinkPlaceholder = "{%page:sample-page-1%}"
const testPageLinkURI = "/page/sample-page-1" + contentFileExtension
const testPostLinkPlaceholder = "{%post:sample-post-1%}"
const testPostLinkURI = "/post/sample-post-1" + contentFileExtension
const testSearchLink1Placeholder = "{%search:term1 term2%}"
const testSearchLink1URI = "/search" + contentFileExtension + "?q=term1%20term2"
const testSearchLink2Placeholder = "{%search:term1+term2%}"
const testSearchLink2URI = "/search" + contentFileExtension + "?q=term1%2Bterm2"
const tag1 = "tag1"
const tag2 = "Tag2"
const tag3 = "TAG3"

const testTagAutoLinkText = "Multi Word Tag"
const testTagAutoLinkPlaceholder = "[" + testTagAutoLinkText + "]({%tag%})"
const testTagAutoLinkExpected = `href="/tags/multi-word-tag/">` + testTagAutoLinkText + `</a>`

const testTagExplicitPlaceholder = "{%tag:multi-word-tag%}"
const testTagExplicitURI = "/tags/multi-word-tag/"

const testTagMultiWordMetadata = "Multi Word Tag"
const testTagMultiWordNormalized = "multi-word-tag"

const pageContentTemplate = `---
title: %s
---

%s

{embed:%s}
{embed:%s}
{embed:%s}

[Page Link](%s)
[Post Link](%s)
[Search Link 1](%s)
[Search Link 2](%s)
%s
[Explicit Tag Link](%s)
`
const postContentTemplate = `---
title: %s
tags:
  - %s
  - %s
  - %s
---

%s

{embed:%s}
{embed:%s}
{embed:%s}

[Page Link](%s)
[Post Link](%s)
[Search Link 1](%s)
[Search Link 2](%s)
%s
[Explicit Tag Link](%s)
`

var testPageContent = fmt.Sprintf(pageContentTemplate,
	testPageTitle,
	testPageBody,
	testEmbed1,
	testEmbed2,
	testEmbed3,
	testPageLinkPlaceholder,
	testPostLinkPlaceholder,
	testSearchLink1Placeholder,
	testSearchLink2Placeholder,
	testTagAutoLinkPlaceholder,
	testTagExplicitPlaceholder,
)

var testPostContent = fmt.Sprintf(postContentTemplate,
	testPostTitle,
	tag1,
	tag2,
	tag3,
	testPostBody,
	testEmbed1,
	testEmbed2,
	testEmbed3,
	testPageLinkPlaceholder,
	testPostLinkPlaceholder,
	testSearchLink1Placeholder,
	testSearchLink2Placeholder,
	testTagAutoLinkPlaceholder,
	testTagExplicitPlaceholder,
)

func TestParser(t *testing.T) {
	expectedEmbeddedMedia := []embeddedMedia{
		{
			MediaType: testEmbed1Type,
			Code:      testEmbed1Code,
		},
		{
			MediaType: testEmbed2Type,
			Code:      testEmbed2Code,
		},
		{
			MediaType: testEmbed3Type,
			Code:      testEmbed3Code,
		},
	}

	expectedTags := []string{tag1, tag2, tag3}

	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "../../themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	resLoader := resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			templateFilePath := fmt.Sprintf("%s%c%s", defaultThemeTemplatesDir, os.PathSeparator, templateFileName)
			return os.ReadFile(templateFilePath)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}

	config := defaultConfig()

	page := parsePage("page", testPageContent, config, resLoader)
	verifyStringsEqual(page.Title, testPageTitle, t)
	verifyStringContains(page.Body, testPageBody, t)
	verifyEmbeddedMedia(page.Body, expectedEmbeddedMedia, t)
	verifyStringContains(page.Body, testPageLinkURI, t)
	verifyStringContains(page.Body, testPostLinkURI, t)
	verifyStringContains(page.Body, testSearchLink1URI, t)
	verifyStringContains(page.Body, testSearchLink2URI, t)
	verifyStringContains(page.Body, testTagAutoLinkExpected, t)
	verifyStringContains(page.Body, testTagExplicitURI, t)

	post := parsePost("post", testPostContent, config, resLoader)
	verifyStringsEqual(post.Title, testPostTitle, t)
	verifyStringContains(post.Body, testPostBody, t)
	verifyEmbeddedMedia(post.Body, expectedEmbeddedMedia, t)
	verifyStringContains(post.Body, testPageLinkURI, t)
	verifyStringContains(post.Body, testPostLinkURI, t)
	verifyStringContains(post.Body, testSearchLink1URI, t)
	verifyStringContains(post.Body, testSearchLink2URI, t)
	verifyStringContains(post.Body, testTagAutoLinkExpected, t)
	verifyStringContains(post.Body, testTagExplicitURI, t)
	verifyStringSlicesEqual(post.Tags, expectedTags, t)
}

func TestNormalizeTagURI(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"tag1", "tag1"},
		{"Tag2", "tag2"},
		{"TAG3", "tag3"},
		{"Multi Word Tag", "multi-word-tag"},
		{"Sci-Fi", "sci-fi"},
		{"🚴 Multi Tag", "multi-tag"},
		{"TourDeZwift", "tourdezwift"},
	}
	for _, c := range cases {
		result := normalizeTagURI(c.input)
		if result != c.expected {
			t.Errorf("normalizeTagURI(%q) = %q, want %q", c.input, result, c.expected)
		}
	}
}

func TestMultiWordTagMetadata(t *testing.T) {
	postContent := `---
date: 2025-08-30
tags:
  - ` + testTagMultiWordMetadata + `
---

Post body.`

	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "../../themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	resLoader := resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			templateFilePath := fmt.Sprintf("%s%c%s", defaultThemeTemplatesDir, os.PathSeparator, templateFileName)
			return os.ReadFile(templateFilePath)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}

	config := defaultConfig()
	post := parsePost("test-post", postContent, config, resLoader)

	// verify tag is stored using its original frontmatter title (preserved as-is)
	if len(post.Tags) != 1 || post.Tags[0] != testTagMultiWordMetadata {
		t.Errorf("Expected tag %q, got %v", testTagMultiWordMetadata, post.Tags)
	}
	// verify the tag still normalizes to the expected URI
	if normalizeTagURI(post.Tags[0]) != testTagMultiWordNormalized {
		t.Errorf("Expected normalized URI %q, got %q", testTagMultiWordNormalized, normalizeTagURI(post.Tags[0]))
	}
}

func TestTagLinkRendering(t *testing.T) {
	postContent := `---
date: 2025-08-30
tags:
  - ExplicitTag
---

Post body with #HashTagAlpha and #HashTagBeta inline.`

	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "../../themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	resLoader := resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			templateFilePath := fmt.Sprintf("%s%c%s", defaultThemeTemplatesDir, os.PathSeparator, templateFileName)
			return os.ReadFile(templateFilePath)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}

	config := defaultConfig()
	post := parsePost("test-post", postContent, config, resLoader)

	// only the explicitly listed YAML metadata tag should be present (raw title preserved)
	if len(post.Tags) != 1 || post.Tags[0] != "ExplicitTag" {
		t.Errorf("Expected post.Tags = [ExplicitTag], got %v", post.Tags)
	}

	// hashtag links should be rendered in the body
	if !strings.Contains(post.Body, `href="/tags/hashtagalpha/"`) {
		t.Error("Body should contain rendered link for #HashTagAlpha")
	}
	if !strings.Contains(post.Body, `href="/tags/hashtagbeta/"`) {
		t.Error("Body should contain rendered link for #HashTagBeta")
	}
}

func TestRepeatedHashTagLinkRendering(t *testing.T) {
	postContent := `---
date: 2025-08-30
---

* Item one
  note: #TagAlpha #TagBeta #TagGamma
* Item two
  note: #TagAlpha #TagBeta
* Item three
  note: #TagAlpha`

	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "../../themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	resLoader := resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			templateFilePath := fmt.Sprintf("%s%c%s", defaultThemeTemplatesDir, os.PathSeparator, templateFileName)
			return os.ReadFile(templateFilePath)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}

	config := defaultConfig()
	post := parsePost("test-post", postContent, config, resLoader)

	// every hashtag occurrence must render as a proper anchor, not leaked markdown link syntax
	expectedCounts := map[string]int{
		`href="/tags/tagalpha/"`: 3,
		`href="/tags/tagbeta/"`:  2,
		`href="/tags/taggamma/"`: 1,
	}
	for href, want := range expectedCounts {
		got := strings.Count(post.Body, href)
		if got != want {
			t.Errorf("expected %d occurrences of %s, got %d\nBody:\n%s", want, href, got, post.Body)
		}
	}

	// no leftover markdown link syntax wrapping an already-rendered anchor
	if strings.Contains(post.Body, `](/tags/`) {
		t.Errorf("body still contains unrendered markdown link wrapper `](/tags/`:\n%s", post.Body)
	}
	if strings.Contains(post.Body, `[<a href="/tags/`) {
		t.Errorf("body contains anchor wrapped in leftover `[`:\n%s", post.Body)
	}
}

func verifyStringsEqual(value string, expected string, t *testing.T) {
	if value != expected {
		t.Errorf("Expected: %s / Found: %s", expected, value)
	}
}

func verifyStringContains(value string, expected string, t *testing.T) {
	if !strings.Contains(value, expected) {
		t.Errorf("Expected to contain: %s / Found: %s", expected, value)
	}
}

func verifyStringSlicesEqual(values []string, expected []string, t *testing.T) {
	if !slices.Equal(values, expected) {
		t.Errorf("Expected: %s / Found: %s", expected, values)
	}
}

func verifyEmbeddedMedia(body string, expected []embeddedMedia, t *testing.T) {
	for _, em := range expected {
		var url string
		switch em.MediaType {
		case YouTube:
			url = "https://www.youtube.com/embed/"
		case Vimeo:
			url = "https://player.vimeo.com/video/"
		}
		url += em.Code
		if !strings.Contains(body, url) {
			t.Errorf("Expected embedded media not found for: %s", em)
		}
	}
}

func TestMetadataStripping(t *testing.T) {
	// test that YAML frontmatter is properly stripped from FeedContent and search data
	// this test uses more complex YAML with various characters (periods, brackets, etc.)
	postContent := `---
date: 2025-08-30
time: 19:30:00
title: Read a new book (and it's great!). Sci-Fi is awesome!
tags:
  - Books
  - Sci-Fi
---

This is the actual post body content. It should appear in feeds and search.

More content here with **formatting** and [links](http://example.com).`

	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "../../themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	resLoader := resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			templateFilePath := fmt.Sprintf("%s%c%s", defaultThemeTemplatesDir, os.PathSeparator, templateFileName)
			return os.ReadFile(templateFilePath)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}

	config := defaultConfig()
	post := parsePost("test-post", postContent, config, resLoader)

	// verify FeedContent does not contain YAML frontmatter
	if strings.Contains(post.FeedContent, "---") {
		t.Error("FeedContent should not contain YAML delimiters (---)")
	}
	if strings.Contains(post.FeedContent, "date: 2025-08-30") {
		t.Error("FeedContent should not contain metadata field 'date: 2025-08-30'")
	}
	if strings.Contains(post.FeedContent, "title: Read a new book") {
		t.Error("FeedContent should not contain metadata field 'title: Read a new book'")
	}
	if strings.Contains(post.FeedContent, "tags:") {
		t.Error("FeedContent should not contain metadata field 'tags:'")
	}
	if strings.Contains(post.FeedContent, "- Books") {
		t.Error("FeedContent should not contain metadata tag '- Books'")
	}
	if strings.Contains(post.FeedContent, "- Sci-Fi") {
		t.Error("FeedContent should not contain metadata tag '- Sci-Fi'")
	}

	// verify FeedContent DOES contain the actual body
	if !strings.Contains(post.FeedContent, "This is the actual post body content") {
		t.Error("FeedContent should contain the post body text")
	}
	if !strings.Contains(post.FeedContent, "More content here with") {
		t.Error("FeedContent should contain the rest of the post body")
	}

	// verify SearchData.Content does not contain YAML frontmatter
	searchContent := post.SearchData.Content
	if strings.Contains(searchContent, "date: 2025-08-30") {
		t.Error("SearchData.Content should not contain metadata field 'date: 2025-08-30'")
	}
	if strings.Contains(searchContent, "title: read a new book") { // note: search content is lowercased
		t.Error("SearchData.Content should not contain metadata field 'title: read a new book'")
	}
	if strings.Contains(searchContent, "tags:") {
		t.Error("SearchData.Content should not contain metadata field 'tags:'")
	}

	// verify SearchData.Content DOES contain the actual body (lowercased)
	if !strings.Contains(searchContent, "this is the actual post body content") {
		t.Error("SearchData.Content should contain the post body text (lowercased)")
	}
}

func TestHorizontalRulesInPostBody(t *testing.T) {
	// test that horizontal rules (---) in post body are preserved
	postContent := `---
date: 2023-06-30
title: Trek Travel - Glacier National Park
tags:
  - Cycling
  - Glacier
---

I and my best friend Javie just returned from our very first Trek Travel trip.

---

Glacier National Park isn't just another protected wildlife site. It's one of Mother Nature's most prized possessions.

---

More content here.`

	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "../../themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	resLoader := resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			templateFilePath := fmt.Sprintf("%s%c%s", defaultThemeTemplatesDir, os.PathSeparator, templateFileName)
			return os.ReadFile(templateFilePath)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}

	config := defaultConfig()
	post := parsePost("test-post", postContent, config, resLoader)

	// verify the Body contains horizontal rules (converted to <hr> tags by markdown processor)
	if !strings.Contains(post.Body, "<hr") {
		t.Error("Post Body should contain <hr> tags from markdown horizontal rules (---)")
	}

	// verify FeedContent contains the horizontal rule markers
	if !strings.Contains(post.FeedContent, "---") {
		t.Error("FeedContent should preserve horizontal rules (---) from post body")
	}

	// count how many --- appear in FeedContent (should be 2, not 0)
	count := strings.Count(post.FeedContent, "---")
	if count != 2 {
		t.Errorf("FeedContent should contain exactly 2 horizontal rules (---), found %d", count)
	}

	// verify FeedContent does NOT contain metadata
	if strings.Contains(post.FeedContent, "date: 2023-06-30") {
		t.Error("FeedContent should not contain metadata field 'date: 2023-06-30'")
	}
	if strings.Contains(post.FeedContent, "title: Trek Travel") {
		t.Error("FeedContent should not contain metadata field 'title: Trek Travel'")
	}

	// verify body content is present
	if !strings.Contains(post.FeedContent, "I and my best friend Javie") {
		t.Error("FeedContent should contain post body text")
	}
	if !strings.Contains(post.FeedContent, "Glacier National Park") {
		t.Error("FeedContent should contain post body text")
	}
}

func TestSharedMediaResolution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mbgen-shared-media-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	// Create content-specific media dir with specific.jpg
	contentSpecificDir := fmt.Sprintf("%s/%s/%s/%s", deployDirName, mediaDirName, "post", "test-post")
	if err := os.MkdirAll(contentSpecificDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(contentSpecificDir+"/specific.jpg", []byte("fake jpg"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create shared media dir with shared.jpg
	sharedDir := fmt.Sprintf("%s/%s/%s", deployDirName, mediaDirName, sharedMediaDirName)
	if err := os.MkdirAll(sharedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(sharedDir+"/shared.jpg", []byte("fake jpg"), 0644); err != nil {
		t.Fatal(err)
	}

	config := defaultConfig()

	// content-specific file: should use content-specific URI (takes precedence)
	result := parseMediaFileNames([]string{"specific.jpg"}, Post, "test-post", config, true, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(result))
	}
	if result[0].Uri != "/media/post/test-post/specific.jpg" {
		t.Errorf("expected /media/post/test-post/specific.jpg, got %s", result[0].Uri)
	}

	// shared file (not in content-specific dir): should fall back to shared URI
	result = parseMediaFileNames([]string{"shared.jpg"}, Post, "test-post", config, true, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(result))
	}
	if result[0].Uri != "/media/shared/shared.jpg" {
		t.Errorf("expected /media/shared/shared.jpg, got %s", result[0].Uri)
	}

	// missing file (not in either dir): should use content-specific URI (no fallback)
	result = parseMediaFileNames([]string{"missing.jpg"}, Post, "test-post", config, true, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(result))
	}
	if result[0].Uri != "/media/post/test-post/missing.jpg" {
		t.Errorf("expected /media/post/test-post/missing.jpg, got %s", result[0].Uri)
	}

	// non-explicit call: shared.jpg exists in shared dir but isExplicit=false, so no fallback
	result = parseMediaFileNames([]string{"shared.jpg"}, Post, "test-post", config, false, nil)
	if len(result) != 1 {
		t.Fatalf("expected 1 media item, got %d", len(result))
	}
	if result[0].Uri != "/media/post/test-post/shared.jpg" {
		t.Errorf("expected /media/post/test-post/shared.jpg, got %s", result[0].Uri)
	}
}

func newColsTestResLoader() resourceLoader {
	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "../../themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	return resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			templateFilePath := fmt.Sprintf("%s%c%s", defaultThemeTemplatesDir, os.PathSeparator, templateFileName)
			return os.ReadFile(templateFilePath)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}
}

func TestColsDirectiveBasic(t *testing.T) {
	postContent := `---
date: 2026-04-18
---

Intro paragraph.

{cols}
{col}
Left column content.
{/}
{col}
Right column content.
{/}
{//}

Outro.`

	post := parsePost("cols-basic", postContent, defaultConfig(), newColsTestResLoader())

	verifyStringContains(post.Body, `<section class="cols"`, t)
	verifyStringContains(post.Body, `grid-template-columns: 1fr 1fr`, t)
	if n := strings.Count(post.Body, `<section class="col"`); n != 2 {
		t.Errorf("expected 2 <section class=\"col\"> elements, got %d\n%s", n, post.Body)
	}
	verifyStringContains(post.Body, "Left column content.", t)
	verifyStringContains(post.Body, "Right column content.", t)
	verifyStringContains(post.Body, "Intro paragraph.", t)
	verifyStringContains(post.Body, "Outro.", t)
}

func TestColsDirectiveWithWeights(t *testing.T) {
	postContent := `---
date: 2026-04-18
---

{cols(3:1:1)}
{col}
Wide.
{/}
{col}
N1.
{/}
{col}
N2.
{/}
{//}`

	post := parsePost("cols-weights", postContent, defaultConfig(), newColsTestResLoader())

	verifyStringContains(post.Body, `grid-template-columns: 3fr 1fr 1fr`, t)
	if n := strings.Count(post.Body, `<section class="col"`); n != 3 {
		t.Errorf("expected 3 cols, got %d\n%s", n, post.Body)
	}
}

func TestColsDirectiveWeightCountMismatch(t *testing.T) {
	postContent := `---
date: 2026-04-18
---

{cols(3:1)}
{col}
A.
{/}
{col}
B.
{/}
{col}
C.
{/}
{//}`

	post := parsePost("cols-mismatch", postContent, defaultConfig(), newColsTestResLoader())

	// weights dropped, equal widths fallback applied
	verifyStringContains(post.Body, `grid-template-columns: 1fr 1fr 1fr`, t)
	if strings.Contains(post.Body, "3fr") {
		t.Errorf("expected no 3fr token after mismatch fallback\n%s", post.Body)
	}
}

func TestColsDirectiveEmpty(t *testing.T) {
	postContent := `---
date: 2026-04-18
---

Before.

{cols}
{//}

After.`

	post := parsePost("cols-empty", postContent, defaultConfig(), newColsTestResLoader())

	// no cols section rendered
	if strings.Contains(post.Body, `class="cols"`) {
		t.Errorf("expected no rendered cols section for empty {cols} block\n%s", post.Body)
	}
	// the raw directive should be visible to the author (left as literal text)
	verifyStringContains(post.Body, "{cols}", t)
	verifyStringContains(post.Body, "{//}", t)
	verifyStringContains(post.Body, "Before.", t)
	verifyStringContains(post.Body, "After.", t)
}

func TestColsDirectiveAlignment(t *testing.T) {
	postContent := `---
date: 2026-04-18
---

{cols}
{col(a=l)}
Left.
{/}
{col(a=c)}
Center.
{/}
{col(a=r)}
Right.
{/}
{col}
Plain.
{/}
{//}`

	post := parsePost("cols-align", postContent, defaultConfig(), newColsTestResLoader())

	verifyStringContains(post.Body, `<section class="col align-l"`, t)
	verifyStringContains(post.Body, `<section class="col align-c"`, t)
	verifyStringContains(post.Body, `<section class="col align-r"`, t)
	// plain {col} without props: class="col" with no align suffix
	if !strings.Contains(post.Body, `<section class="col">`) {
		t.Errorf("expected plain <section class=\"col\"> for unstyled col\n%s", post.Body)
	}
}

func TestColsDirectiveInvalidAlignment(t *testing.T) {
	postContent := `---
date: 2026-04-18
---

{cols}
{col(a=x)}
Bogus alignment.
{/}
{col}
Normal.
{/}
{//}`

	post := parsePost("cols-invalid-align", postContent, defaultConfig(), newColsTestResLoader())

	// column still renders
	verifyStringContains(post.Body, "Bogus alignment.", t)
	// no align-x class
	if strings.Contains(post.Body, "align-x") {
		t.Errorf("expected no align-x class for invalid alignment value\n%s", post.Body)
	}
}

func TestColsDirectiveWithNestedWithMedia(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mbgen-cols-nested-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	mediaDir := fmt.Sprintf("%s/%s/%s/%s", deployDirName, mediaDirName, "post", "cols-nested")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(mediaDir+"/book.jpg", []byte("fake jpg"), 0644); err != nil {
		t.Fatal(err)
	}

	// resLoader needs to read templates from the real theme dir, which is relative to the
	// original working dir (not tmpDir). Resolve absolute path before chdir.
	themeTemplatesAbsDir := fmt.Sprintf("%s%c..%c..%c%s%c%s%c%s", origDir, os.PathSeparator, os.PathSeparator, os.PathSeparator, "themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	resLoader := resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			return os.ReadFile(themeTemplatesAbsDir + string(os.PathSeparator) + templateFileName)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}

	postContent := `---
date: 2026-04-18
---

{cols}
{col}
Left text.
{/}
{col}
{with-media(p=l,s=s):book.jpg}
Book caption.
{/}
{/}
{//}`

	post := parsePost("cols-nested", postContent, defaultConfig(), resLoader)

	verifyStringContains(post.Body, `<section class="cols"`, t)
	verifyStringContains(post.Body, `class="with-media`, t)
	verifyStringContains(post.Body, "/media/post/cols-nested/book.jpg", t)
	verifyStringContains(post.Body, "Left text.", t)
	verifyStringContains(post.Body, "Book caption.", t)
	// no stray UUID placeholders in the output
	if strings.Contains(post.Body, ":@@@:") {
		t.Errorf("output contains unresolved placeholder markers\n%s", post.Body)
	}
	// no <p> wrapper around the with-media <section> inside the col (would produce
	// empty <p></p> artifacts in the browser and misalign adjacent columns).
	if strings.Contains(post.Body, `<p><section class="with-media`) {
		t.Errorf("block-level directive should not be wrapped in <p>...</p>\n%s", post.Body)
	}
}

func TestColsDirectiveMultipleNestedWithMedia(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mbgen-cols-multi-nested-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(origDir)

	mediaDir := fmt.Sprintf("%s/%s/%s/%s", deployDirName, mediaDirName, "post", "cols-multi")
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"a.jpg", "b.jpg", "c.jpg"} {
		if err := os.WriteFile(mediaDir+"/"+name, []byte("fake jpg"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	themeTemplatesAbsDir := fmt.Sprintf("%s%c..%c..%c%s%c%s%c%s", origDir, os.PathSeparator, os.PathSeparator, os.PathSeparator, "themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	resLoader := resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			return os.ReadFile(themeTemplatesAbsDir + string(os.PathSeparator) + templateFileName)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}

	// Three consecutive {with-media} blocks inside a single {col}, no blank lines
	// between them — exercises the multi-placeholder-per-paragraph code path
	// (<p>UUID1<br>UUID2<br>UUID3</p>) that goldmark's WithHardWraps produces.
	postContent := `---
date: 2026-04-18
---

{cols}
{col}
{with-media(p=l,s=s):a.jpg}
Caption A
{/}
{with-media(p=l,s=s):b.jpg}
Caption B
{/}
{with-media(p=l,s=s):c.jpg}
Caption C
{/}
{/}
{col}
right text
{/}
{//}`

	post := parsePost("cols-multi", postContent, defaultConfig(), resLoader)

	// all three captions present
	verifyStringContains(post.Body, "Caption A", t)
	verifyStringContains(post.Body, "Caption B", t)
	verifyStringContains(post.Body, "Caption C", t)
	// no <p> wrappers leaking around block-level section output
	if strings.Contains(post.Body, `<p><section class="with-media`) {
		t.Errorf("block-level directives should not be wrapped in <p>...</p>\n%s", post.Body)
	}
	// no leftover <br> between sections (would indicate we stripped <p> but not <br>)
	if strings.Contains(post.Body, `</section><br`) {
		t.Errorf("unexpected <br> between adjacent <section> elements\n%s", post.Body)
	}
	// exactly three with-media sections rendered
	if n := strings.Count(post.Body, `<section class="with-media`); n != 3 {
		t.Errorf("expected 3 with-media sections, got %d\n%s", n, post.Body)
	}
}

func TestColsDirectiveFeedContentStripping(t *testing.T) {
	postContent := `---
date: 2026-04-18
---

{cols}
{col}
Alpha text.
{/}
{col}
Beta text.
{/}
{//}`

	post := parsePost("cols-feed", postContent, defaultConfig(), newColsTestResLoader())

	// no raw directive tokens in feed content
	if strings.Contains(post.FeedContent, "{cols}") || strings.Contains(post.FeedContent, "{//}") || strings.Contains(post.FeedContent, "{col}") || strings.Contains(post.FeedContent, "{/}") {
		t.Errorf("FeedContent should not contain raw directive tokens, got %q", post.FeedContent)
	}
	// inner text preserved for search/excerpt
	verifyStringContains(post.FeedContent, "Alpha text.", t)
	verifyStringContains(post.FeedContent, "Beta text.", t)
}

// setupMediaCaptionFixture creates a temp working dir with the given media
// file names in deploy/media/post/<postId>/, chdirs into it, and returns a
// resourceLoader pointing at the default theme templates (resolved via the
// original cwd). Cleanup is handled via t.Cleanup so the caller doesn't need
// to defer anything. Returns the resourceLoader.
func setupMediaCaptionFixture(t *testing.T, postId string, mediaFiles []string) resourceLoader {
	t.Helper()
	tmpDir, err := os.MkdirTemp("", "mbgen-media-caption-*")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.RemoveAll(tmpDir) })

	origDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(tmpDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(origDir) })

	mediaDir := fmt.Sprintf("%s/%s/%s/%s", deployDirName, mediaDirName, "post", postId)
	if err := os.MkdirAll(mediaDir, 0755); err != nil {
		t.Fatal(err)
	}
	for _, name := range mediaFiles {
		if err := os.WriteFile(mediaDir+"/"+name, []byte("fake"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	themeTemplatesAbsDir := fmt.Sprintf("%s%c..%c..%c%s%c%s%c%s", origDir, os.PathSeparator, os.PathSeparator, os.PathSeparator, "themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
	return resourceLoader{
		config: defaultConfig(),
		loadTemplate: func(templateFileName string) ([]byte, error) {
			return os.ReadFile(themeTemplatesAbsDir + string(os.PathSeparator) + templateFileName)
		},
		loadInclude: func(includeFileName string, level templateIncludeLevel) ([]byte, error) {
			return nil, nil
		},
	}
}

func TestSplitMediaArg(t *testing.T) {
	cases := []struct {
		name             string
		input            string
		expectedFiles    []string
		expectedCaptions map[string]string
	}{
		{
			name:             "bare filenames",
			input:            "a.jpg, b.png ,c.mp4",
			expectedFiles:    []string{"a.jpg", "b.png", "c.mp4"},
			expectedCaptions: map[string]string{},
		},
		{
			name:             "all with captions",
			input:            "a.jpg|Caption A|, b.png|Caption B|",
			expectedFiles:    []string{"a.jpg", "b.png"},
			expectedCaptions: map[string]string{"a.jpg": "Caption A", "b.png": "Caption B"},
		},
		{
			name:             "mixed",
			input:            "a.jpg|Caption A|, b.png, c.jpg|Caption C|",
			expectedFiles:    []string{"a.jpg", "b.png", "c.jpg"},
			expectedCaptions: map[string]string{"a.jpg": "Caption A", "c.jpg": "Caption C"},
		},
		{
			name:             "empty caption treated as no caption",
			input:            "a.jpg||",
			expectedFiles:    []string{"a.jpg"},
			expectedCaptions: map[string]string{},
		},
		{
			name:             "caption with spaces and punctuation",
			input:            "a.jpg|Hello world! This is fine.|",
			expectedFiles:    []string{"a.jpg"},
			expectedCaptions: map[string]string{"a.jpg": "Hello world! This is fine."},
		},
		{
			name:             "caption with commas",
			input:            "a.jpg|one, two, three|,b.png|plain|",
			expectedFiles:    []string{"a.jpg", "b.png"},
			expectedCaptions: map[string]string{"a.jpg": "one, two, three", "b.png": "plain"},
		},
		{
			name:             "empty input",
			input:            "",
			expectedFiles:    nil,
			expectedCaptions: nil,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			files, caps := splitMediaArg(tc.input)
			if !slices.Equal(files, tc.expectedFiles) {
				t.Errorf("files: expected %v, got %v", tc.expectedFiles, files)
			}
			if tc.expectedCaptions == nil {
				if caps != nil {
					t.Errorf("captions: expected nil, got %v", caps)
				}
				return
			}
			if len(caps) != len(tc.expectedCaptions) {
				t.Errorf("captions: expected %v, got %v", tc.expectedCaptions, caps)
			}
			for k, v := range tc.expectedCaptions {
				if caps[k] != v {
					t.Errorf("caption[%q]: expected %q, got %q", k, v, caps[k])
				}
			}
		})
	}
}

func TestMediaDirectiveWithCaptions(t *testing.T) {
	resLoader := setupMediaCaptionFixture(t, "caps-basic", []string{"a.jpg", "b.png", "c.jpg"})

	postContent := `---
date: 2026-04-18
---

{media:a.jpg|Caption A|,b.png|Caption B|,c.jpg}`

	post := parsePost("caps-basic", postContent, defaultConfig(), resLoader)

	verifyStringContains(post.Body, "/media/post/caps-basic/a.jpg", t)
	verifyStringContains(post.Body, "/media/post/caps-basic/b.png", t)
	verifyStringContains(post.Body, "/media/post/caps-basic/c.jpg", t)
	verifyStringContains(post.Body, `<div class="caption">Caption A</div>`, t)
	verifyStringContains(post.Body, `<div class="caption">Caption B</div>`, t)

	// c.jpg has no caption — exactly two caption divs should render.
	if n := strings.Count(post.Body, `class="caption"`); n != 2 {
		t.Errorf("expected 2 caption divs, got %d\n%s", n, post.Body)
	}
}

func TestWithMediaDirectiveWithCaption(t *testing.T) {
	resLoader := setupMediaCaptionFixture(t, "caps-wm", []string{"book.jpg"})

	postContent := `---
date: 2026-04-18
---

{with-media(p=l,s=s):book.jpg|Book title|}
Commentary about the book.
{/}`

	post := parsePost("caps-wm", postContent, defaultConfig(), resLoader)

	verifyStringContains(post.Body, `class="with-media`, t)
	verifyStringContains(post.Body, "/media/post/caps-wm/book.jpg", t)
	verifyStringContains(post.Body, `<div class="caption">Book title</div>`, t)
	verifyStringContains(post.Body, "Commentary about the book.", t)
}

func TestMediaCaptionHtmlEscaping(t *testing.T) {
	resLoader := setupMediaCaptionFixture(t, "caps-esc", []string{"a.jpg"})

	postContent := `---
date: 2026-04-18
---

{media:a.jpg|<script>alert(1)</script>|}`

	post := parsePost("caps-esc", postContent, defaultConfig(), resLoader)

	// Raw script tag must not appear; the escaped form must.
	if strings.Contains(post.Body, "<script>alert(1)</script>") {
		t.Errorf("raw <script> tag leaked into output:\n%s", post.Body)
	}
	verifyStringContains(post.Body, "&lt;script&gt;alert(1)&lt;/script&gt;", t)
}

func TestMediaCaptionEmpty(t *testing.T) {
	resLoader := setupMediaCaptionFixture(t, "caps-empty", []string{"a.jpg"})

	postContent := `---
date: 2026-04-18
---

{media:a.jpg||}`

	post := parsePost("caps-empty", postContent, defaultConfig(), resLoader)

	verifyStringContains(post.Body, "/media/post/caps-empty/a.jpg", t)
	if strings.Contains(post.Body, `class="caption"`) {
		t.Errorf("empty caption should not render a caption div:\n%s", post.Body)
	}
}

func TestInspectTagTitleDuplicates(t *testing.T) {
	posts := []post{
		{Id: "a", Tags: []string{"Three Word Tag", "Books"}},
		{Id: "b", Tags: []string{"Three word tag"}},
		{Id: "c", Tags: []string{"THREE WORD TAG", "Books"}},
		{Id: "d", Tags: []string{"Sci-Fi"}},
	}
	dupes := inspectTagTitleDuplicates(posts)

	// "three-word-tag" has 3 distinct titles; "books" and "sci-fi" have 1 each
	if len(dupes) != 1 {
		t.Fatalf("expected 1 URI with duplicates, got %d: %v", len(dupes), dupes)
	}
	titles, ok := dupes["three-word-tag"]
	if !ok {
		t.Fatalf("expected duplicates under key %q, got keys %v", "three-word-tag", dupes)
	}
	expected := []string{"THREE WORD TAG", "Three Word Tag", "Three word tag"}
	if !slices.Equal(titles, expected) {
		t.Errorf("expected sorted titles %v, got %v", expected, titles)
	}

	// confirm single-title URIs are excluded
	if _, ok := dupes["books"]; ok {
		t.Error("URI with only one distinct title should not be reported as a duplicate")
	}
}
