package app

import (
	"testing"
)

func img(uri string) media {
	// thumbs mirror buildMediaItem's fallback behavior (original URI when no thumb file exists)
	return media{Type: Image, Uri: uri, thumbs: []thumb{{Uri: uri, Size: 480}, {Uri: uri, Size: 960}}}
}

// posts are passed newest-first (the order parsePosts returns them in)
func TestAggregateCollections(t *testing.T) {
	posts := []post{
		{
			Id: "newest",
			Collections: []postCollectionRef{
				{Collection: "Board Games", Item: "Zeta Game", Media: []media{img("/media/post/newest/zeta1.jpg")}},
				{Collection: "Board Games", Item: "Alpha Game"},
				{Collection: "Books", Item: "Alpha Game", Media: []media{img("/media/shared/alpha-book.jpg")}},
			},
		},
		{
			Id: "older",
			Collections: []postCollectionRef{
				{Collection: "board games", Item: "zeta game", Media: []media{
					img("/media/post/older/zeta2.jpg"),
					img("/media/post/newest/zeta1.jpg"), // same URI as in "newest" — must dedup
				}},
				{Collection: "Board Games", Item: "Alpha Game", Media: []media{img("/media/post/older/alpha.jpg")}},
			},
		},
		{
			Id: "oldest",
			Collections: []postCollectionRef{
				// same item referenced twice within one post — counts once
				{Collection: "Board Games", Item: "Zeta Game"},
				{Collection: "Board Games", Item: "Zeta Game", Media: []media{img("/media/post/oldest/zeta3.jpg")}},
			},
		},
	}

	collections := aggregateCollections(posts)

	if len(collections) != 2 {
		t.Fatalf("expected 2 collections, got %d: %+v", len(collections), collections)
	}

	// collections sorted by title
	bg, books := collections[0], collections[1]
	if bg.Title != "Board Games" || bg.URI != "board-games" {
		t.Errorf("expected first collection Board Games/board-games, got %s/%s", bg.Title, bg.URI)
	}
	if books.Title != "Books" || books.URI != "books" {
		t.Errorf("expected second collection Books/books, got %s/%s", books.Title, books.URI)
	}

	// distinct posts referencing any item of the collection
	if bg.PostCnt != 3 {
		t.Errorf("expected Board Games PostCnt 3, got %d", bg.PostCnt)
	}
	if books.PostCnt != 1 {
		t.Errorf("expected Books PostCnt 1, got %d", books.PostCnt)
	}

	// items ordered by first appearance, newest post first
	// (the newest post references Zeta Game before Alpha Game)
	if len(bg.Items) != 2 {
		t.Fatalf("expected 2 Board Games items, got %d: %+v", len(bg.Items), bg.Items)
	}
	zeta, alpha := bg.Items[0], bg.Items[1]
	// title variant: first-seen (newest post) title wins
	if zeta.Title != "Zeta Game" || zeta.URI != "zeta-game" {
		t.Errorf("expected first item Zeta Game/zeta-game, got %s/%s", zeta.Title, zeta.URI)
	}
	if alpha.Title != "Alpha Game" || alpha.URI != "alpha-game" {
		t.Errorf("expected second item Alpha Game/alpha-game, got %s/%s", alpha.Title, alpha.URI)
	}

	// per-item distinct post counts (double ref within one post counts once)
	if alpha.PostCnt != 2 {
		t.Errorf("expected Alpha Game PostCnt 2, got %d", alpha.PostCnt)
	}
	if zeta.PostCnt != 3 {
		t.Errorf("expected Zeta Game PostCnt 3, got %d", zeta.PostCnt)
	}

	// item media: aggregated across posts, deduped by URI, newest-post-first
	expectedZetaMedia := []string{
		"/media/post/newest/zeta1.jpg",
		"/media/post/older/zeta2.jpg",
		"/media/post/oldest/zeta3.jpg",
	}
	if len(zeta.Media) != len(expectedZetaMedia) {
		t.Fatalf("expected %d Zeta Game media, got %d: %+v", len(expectedZetaMedia), len(zeta.Media), zeta.Media)
	}
	for i, uri := range expectedZetaMedia {
		if zeta.Media[i].Uri != uri {
			t.Errorf("Zeta Game media[%d]: expected %q, got %q", i, uri, zeta.Media[i].Uri)
		}
	}

	// item with no image in the newest post picks up the image from an older post
	if len(alpha.Media) != 1 || alpha.Media[0].Uri != "/media/post/older/alpha.jpg" {
		t.Errorf("expected Alpha Game media [/media/post/older/alpha.jpg], got %+v", alpha.Media)
	}

	// same item title in a different collection is a separate item
	if len(books.Items) != 1 {
		t.Fatalf("expected 1 Books item, got %d", len(books.Items))
	}
	if books.Items[0].Title != "Alpha Game" || books.Items[0].PostCnt != 1 {
		t.Errorf("expected Books item Alpha Game with PostCnt 1, got %+v", books.Items[0])
	}
	if len(books.Items[0].Media) != 1 || books.Items[0].Media[0].Uri != "/media/shared/alpha-book.jpg" {
		t.Errorf("expected Books item media [/media/shared/alpha-book.jpg], got %+v", books.Items[0].Media)
	}
}

func TestAggregateCollectionsEmpty(t *testing.T) {
	if collections := aggregateCollections([]post{{Id: "p1"}, {Id: "p2"}}); len(collections) != 0 {
		t.Errorf("expected no collections for posts without collection refs, got %+v", collections)
	}
}

func TestInspectCollectionTitleDuplicates(t *testing.T) {
	posts := []post{
		{Id: "a", Collections: []postCollectionRef{
			{Collection: "Board Games", Item: "Game One"},
			{Collection: "Books", Item: "Some Book"},
		}},
		{Id: "b", Collections: []postCollectionRef{
			{Collection: "board games", Item: "game one"},
		}},
		{Id: "c", Collections: []postCollectionRef{
			{Collection: "Board Games", Item: "Game one"},
			{Collection: "Books", Item: "Some Book"},
		}},
	}

	collDupes, itemDupes := inspectCollectionTitleDuplicates(posts)

	if len(collDupes) != 1 {
		t.Fatalf("expected 1 collection URI with duplicates, got %d: %v", len(collDupes), collDupes)
	}
	expectedCollTitles := []string{"Board Games", "board games"}
	if titles, ok := collDupes["board-games"]; !ok || !equalStringSlices(titles, expectedCollTitles) {
		t.Errorf("expected collection duplicates %v under %q, got %v", expectedCollTitles, "board-games", collDupes)
	}

	if len(itemDupes) != 1 {
		t.Fatalf("expected 1 item URI with duplicates, got %d: %v", len(itemDupes), itemDupes)
	}
	expectedItemTitles := []string{"Game One", "Game one", "game one"}
	if titles, ok := itemDupes["board-games/game-one"]; !ok || !equalStringSlices(titles, expectedItemTitles) {
		t.Errorf("expected item duplicates %v under %q, got %v", expectedItemTitles, "board-games/game-one", itemDupes)
	}
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestCollectionGroupsSuppressSinglePostItems(t *testing.T) {
	p := post{
		Id: "p1",
		Collections: []postCollectionRef{
			{Collection: "Board Games", Item: "Chess"},
			{Collection: "Board Games", Item: "Checkers"},
			{Collection: "Books", Item: "Some Book"},
		},
		collItemPostCnt: map[string]int{
			"board-games/chess":    3,
			"board-games/checkers": 1,
			"books/some-book":      1,
		},
	}

	groups := p.CollectionGroups()

	if len(groups) != 2 {
		t.Fatalf("expected 2 groups, got %d: %+v", len(groups), groups)
	}
	// multi-post item is kept, single-post item is suppressed
	if len(groups[0].Items) != 1 || groups[0].Items[0].URI != "chess" {
		t.Errorf("expected Board Games group with only the chess item, got %+v", groups[0].Items)
	}
	// a collection whose only item is single-post still gets a group (collection link only)
	if groups[1].Title != "Books" || len(groups[1].Items) != 0 {
		t.Errorf("expected Books group with no items, got %+v", groups[1])
	}
}

func TestCollectionGroupsWithoutPostCounts(t *testing.T) {
	// without post-count data (nil map), all item links are kept
	p := post{
		Id: "p1",
		Collections: []postCollectionRef{
			{Collection: "Books", Item: "Some Book"},
		},
	}
	groups := p.CollectionGroups()
	if len(groups) != 1 || len(groups[0].Items) != 1 {
		t.Errorf("expected all items kept when no post-count data is available, got %+v", groups)
	}
}
