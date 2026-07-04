package app

import (
	"bytes"
	"path/filepath"
	"sort"
	"text/template"
)

// aggregateCollections builds the aggregated collections model from parsed post frontmatter.
// Posts are expected newest-first (the order parsePosts returns them in),
// which makes first-seen titles and media win: display titles come from the newest referencing post,
// and item media (deduped by URI) is ordered newest-post-first.
// Collections are sorted by title; items within each collection are ordered by first appearance
// (i.e. items referenced by newer posts come first).
func aggregateCollections(posts []post) []collectionData {
	type itemAgg struct {
		title     string
		postIds   map[string]struct{}
		media     []media
		mediaUris map[string]struct{}
	}
	type collAgg struct {
		title     string
		postIds   map[string]struct{}
		items     map[string]*itemAgg
		itemOrder []string
	}
	colls := make(map[string]*collAgg)
	for _, p := range posts {
		for _, ref := range p.Collections {
			collUri := normalizeURIString(ref.Collection)
			if collUri == "" {
				continue
			}
			itemUri := normalizeURIString(ref.Item)
			if itemUri == "" {
				continue
			}
			ca := colls[collUri]
			if ca == nil {
				ca = &collAgg{title: ref.Collection, postIds: map[string]struct{}{}, items: map[string]*itemAgg{}}
				colls[collUri] = ca
			}
			ca.postIds[p.Id] = struct{}{}
			ia := ca.items[itemUri]
			if ia == nil {
				ia = &itemAgg{title: ref.Item, postIds: map[string]struct{}{}, mediaUris: map[string]struct{}{}}
				ca.items[itemUri] = ia
				ca.itemOrder = append(ca.itemOrder, itemUri)
			}
			ia.postIds[p.Id] = struct{}{}
			for _, m := range ref.Media {
				if _, seen := ia.mediaUris[m.Uri]; !seen {
					ia.mediaUris[m.Uri] = struct{}{}
					ia.media = append(ia.media, m)
				}
			}
		}
	}
	collections := make([]collectionData, 0, len(colls))
	for collUri, ca := range colls {
		items := make([]collectionItemData, 0, len(ca.items))
		for _, itemUri := range ca.itemOrder {
			ia := ca.items[itemUri]
			items = append(items, collectionItemData{
				Title:   ia.title,
				URI:     itemUri,
				PostCnt: len(ia.postIds),
				Media:   ia.media,
			})
		}
		collections = append(collections, collectionData{
			Title:   ca.title,
			URI:     collUri,
			Items:   items,
			PostCnt: len(ca.postIds),
		})
	}
	sort.Slice(collections, func(i, j int) bool {
		return collections[i].Title < collections[j].Title
	})
	return collections
}

// inspectCollectionTitleDuplicates returns two maps of normalized URI -> sorted distinct original titles,
// for collection URIs (first) and "<collection-uri>/<item-uri>" item URIs (second)
// that appear with more than one distinct title across the given posts.
func inspectCollectionTitleDuplicates(posts []post) (map[string][]string, map[string][]string) {
	collTitles := map[string]map[string]struct{}{}
	itemTitles := map[string]map[string]struct{}{}
	for _, p := range posts {
		for _, ref := range p.Collections {
			collUri := normalizeURIString(ref.Collection)
			itemUri := normalizeURIString(ref.Item)
			if collUri == "" || itemUri == "" {
				continue
			}
			if collTitles[collUri] == nil {
				collTitles[collUri] = map[string]struct{}{}
			}
			collTitles[collUri][ref.Collection] = struct{}{}
			itemKey := collUri + "/" + itemUri
			if itemTitles[itemKey] == nil {
				itemTitles[itemKey] = map[string]struct{}{}
			}
			itemTitles[itemKey][ref.Item] = struct{}{}
		}
	}
	toDupes := func(byUri map[string]map[string]struct{}) map[string][]string {
		out := map[string][]string{}
		for uri, titles := range byUri {
			if len(titles) < 2 {
				continue
			}
			list := make([]string, 0, len(titles))
			for t := range titles {
				list = append(list, t)
			}
			sort.Strings(list)
			out[uri] = list
		}
		return out
	}
	return toDupes(collTitles), toDupes(itemTitles)
}

// processCollections generates the per-collection shelf pages, the optional top-level collection index,
// and the paginated per-item post list pages (which follow the same structure as tag pages: index.html + 2.html, 3.html, ...).
// collPostCnt/collContent are keyed by "<collection-uri>/<item-uri>".
// Returns the number of collections and the total number of collection items.
func processCollections(collections []collectionData, collPostCnt map[string]int, collContent map[string][]string,
	pagerTemplate *template.Template, resLoader resourceLoader, handleOutput processorOutputHandler) (int, int) {
	if len(collections) == 0 {
		return 0, 0
	}
	config := resLoader.config
	itemCnt := 0
	sprintln(" - generating collection pages ...")
	collectionTemplate := compileCollectionTemplate(resLoader)
	for _, coll := range collections {
		itemCnt += len(coll.Items)
		var collContentBuffer bytes.Buffer
		err := collectionTemplate.Execute(&collContentBuffer,
			templateContent{EntityType: Page, Title: config.siteName + " - " + coll.Title, Content: coll, Config: buildTemplateConfigMap(config)})
		check(err)
		outputFilePath := filepath.Join(deployDirName, deployCollectionsDirName, coll.URI, indexPageFileName)
		if handleOutput != nil {
			handleOutput(outputFilePath, collContentBuffer.Bytes())
		}
	}
	if config.generateCollectionIndex {
		sprintln(" - generating collection index ...")
		collectionIndexTemplate := compileCollectionIndexTemplate(resLoader)
		var collIndexContentBuffer bytes.Buffer
		err := collectionIndexTemplate.Execute(&collIndexContentBuffer,
			templateContent{EntityType: Page, Title: config.siteName + " - Collections", Content: collections, Config: buildTemplateConfigMap(config)})
		check(err)
		outputFilePath := filepath.Join(deployDirName, deployCollectionsDirName, indexPageFileName)
		if handleOutput != nil {
			handleOutput(outputFilePath, collIndexContentBuffer.Bytes())
		}
	}
	processPaginatedPostContent(collPostCnt, collContent, config.pageSize, deployCollectionsDirName, pagerTemplate, resLoader, handleOutput)
	return len(collections), itemCnt
}
