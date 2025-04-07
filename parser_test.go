package main

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

	defaultThemeTemplatesDir := fmt.Sprintf("%s%c%s%c%s", "themes", os.PathSeparator, defaultThemeName, os.PathSeparator, "templates")
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

	post := parsePost("post", testPostContent, config, resLoader)
	verifyStringsEqual(post.Title, testPostTitle, t)
	verifyStringContains(post.Body, testPostBody, t)
	verifyEmbeddedMedia(post.Body, expectedEmbeddedMedia, t)
	verifyStringContains(post.Body, testPageLinkURI, t)
	verifyStringContains(post.Body, testPostLinkURI, t)
	verifyStringContains(post.Body, testSearchLink1URI, t)
	verifyStringContains(post.Body, testSearchLink2URI, t)
	verifyStringSlicesEqual(post.Tags, expectedTags, t)
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
