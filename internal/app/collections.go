package app

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
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

// metaCollection is a meta collection definition: a named grouping declared by a page
// (via the `meta-collection` frontmatter key) that embeds collections via `{collection:...}` directives;
// posts reference it via the `meta-collections` frontmatter key, and their footers link back to the defining page
type metaCollection struct {
	Title  string
	PageId string
}

// buildMetaCollections returns a map of normalized URI -> meta collection definition;
// on duplicates, the first defining page (in slice order) wins
// — duplicates are fatal generate errors anyway (see validateCollectionUsage)
func buildMetaCollections(pages []page) map[string]metaCollection {
	metaColls := map[string]metaCollection{}
	for _, p := range pages {
		if p.MetaCollection == "" {
			continue
		}
		uri := normalizeURIString(p.MetaCollection)
		if uri == "" {
			continue
		}
		if _, ok := metaColls[uri]; !ok {
			metaColls[uri] = metaCollection{Title: p.MetaCollection, PageId: p.Id}
		}
	}
	return metaColls
}

// validateCollectionUsage validates collection directive and meta collection usage across pages and posts.
// It returns fatal errors (the generate command must fail without writing any files until they are fixed):
//   - the same meta collection title (URI-normalized) defined by more than one page
//   - a meta collection URI colliding with a regular collection URI
//
// and appends non-fatal warnings to the pages/posts (by index, so callers holding the slices see them):
//   - a page {collection:...} directive referencing an unknown collection
//   - a page defining a meta collection without embedding any collections
//   - a post referencing an unknown meta collection
func validateCollectionUsage(pages []page, posts []post, collections []collectionData) []string {
	var errs []string
	collUris := map[string]string{} // URI -> title
	for _, coll := range collections {
		collUris[coll.URI] = coll.Title
	}
	metaCollPages := map[string][]string{} // meta URI -> defining page ids
	for i := range pages {
		p := &pages[i]
		if p.MetaCollection != "" {
			uri := normalizeURIString(p.MetaCollection)
			metaCollPages[uri] = append(metaCollPages[uri], p.Id)
			if collTitle, ok := collUris[uri]; ok {
				errs = append(errs, fmt.Sprintf(
					"meta collection \"%s\" (defined by page \"%s\") collides with collection \"%s\" (/%s/%s/)",
					p.MetaCollection, p.Id, collTitle, deployCollectionsDirName, uri))
			}
			if len(p.CollectionRefs) == 0 {
				p.Warnings = append(p.Warnings, fmt.Sprintf(
					"meta collection \"%s\" is defined, but the page does not embed any collections via {collection:...} directives",
					p.MetaCollection))
			}
		}
		for _, ref := range p.CollectionRefs {
			if _, ok := collUris[ref]; !ok {
				p.Warnings = append(p.Warnings, "collection directive references an unknown collection: \""+ref+"\"")
			}
		}
	}
	metaUris := make([]string, 0, len(metaCollPages))
	for uri := range metaCollPages {
		metaUris = append(metaUris, uri)
	}
	sort.Strings(metaUris)
	for _, uri := range metaUris {
		if pageIds := metaCollPages[uri]; len(pageIds) > 1 {
			errs = append(errs, fmt.Sprintf(
				"meta collection \"%s\" is defined by multiple pages (titles must be unique): %s",
				uri, strings.Join(pageIds, ", ")))
		}
	}
	for i := range posts {
		p := &posts[i]
		for _, title := range p.MetaCollections {
			if _, ok := metaCollPages[normalizeURIString(title)]; !ok {
				p.Warnings = append(p.Warnings, "post references an unknown meta collection: \""+title+"\"")
			}
		}
	}
	return errs
}

// renderEmbeddedCollections substitutes collection directive placeholders in a page body
// with the rendered collection shelf blocks (reusing the standalone collection.html template,
// compiled bare, i.e. without the main.html page wrapper);
// placeholders referencing unknown collections render nothing
// (validateCollectionUsage warned about them already)
func renderEmbeddedCollections(body string, collections []collectionData, resLoader resourceLoader) string {
	collByUri := make(map[string]collectionData, len(collections))
	for _, coll := range collections {
		collByUri[coll.URI] = coll
	}
	return collectionDirectivePlaceholderRegexp.ReplaceAllStringFunc(body, func(match string) string {
		m := collectionDirectivePlaceholderRegexp.FindStringSubmatch(match)
		coll, ok := collByUri[m[1]]
		if !ok {
			return ""
		}
		var collContentBuffer bytes.Buffer
		err := compileCollectionBlockTemplate(resLoader).Execute(&collContentBuffer,
			templateContent{EntityType: Page, Content: coll, Config: buildTemplateConfigMap(resLoader.config)})
		check(err)
		return collContentBuffer.String()
	})
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
