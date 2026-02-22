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
const testTagAutoLinkExpected = `href="/tags/multi_word_tag/">` + testTagAutoLinkText + `</a>`

const testTagExplicitPlaceholder = "{%tag:multi_word_tag%}"
const testTagExplicitURI = "/tags/multi_word_tag/"

const testTagMultiWordMetadata = "Multi Word Tag"
const testTagMultiWordNormalized = "multi_word_tag"

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

	expectedTags := []string{tag1, normalizeTagURI(tag2), normalizeTagURI(tag3)}

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
		{"Multi Word Tag", "multi_word_tag"},
		{"Sci-Fi", "sci-fi"},
		{"🚴 Multi Tag", "multi_tag"},
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

	// verify tag is stored in normalized form
	if len(post.Tags) != 1 || post.Tags[0] != testTagMultiWordNormalized {
		t.Errorf("Expected tag %q, got %v", testTagMultiWordNormalized, post.Tags)
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

	// only the explicitly listed YAML metadata tag should be present (normalized)
	if len(post.Tags) != 1 || post.Tags[0] != "explicittag" {
		t.Errorf("Expected post.Tags = [explicittag], got %v", post.Tags)
	}

	// hashtag links should be rendered in the body
	if !strings.Contains(post.Body, `href="/tags/hashtagalpha/"`) {
		t.Error("Body should contain rendered link for #HashTagAlpha")
	}
	if !strings.Contains(post.Body, `href="/tags/hashtagbeta/"`) {
		t.Error("Body should contain rendered link for #HashTagBeta")
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
