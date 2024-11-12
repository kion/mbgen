package main

import (
	"time"
)

type pageEntityCacheData struct {
	modTime time.Time
	page    page
}

type postEntityCacheData struct {
	modTime time.Time
	post    post
}

var pageCacheData = make(map[string]pageEntityCacheData)

var postCacheData = make(map[string]postEntityCacheData)

func cacheContentEntity(fileName string, modTime time.Time, ce contentEntity) {
	switch ce.ContentEntityType() {
	case Page:
		pageCacheData[fileName] = pageEntityCacheData{
			modTime: modTime,
			page:    ce.(page),
		}
	case Post:
		postCacheData[fileName] = postEntityCacheData{
			modTime: modTime,
			post:    ce.(post),
		}
	}
}

func getCachedContentEntity(ceType contentEntityType, fileName string, modTime time.Time) contentEntity {
	switch ceType {
	case Page:
		if data, ok := pageCacheData[fileName]; ok {
			if data.modTime == modTime {
				return data.page
			}
		}
	case Post:
		if data, ok := postCacheData[fileName]; ok {
			if data.modTime == modTime {
				return data.post
			}
		}
	}
	return nil
}
