# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**mbgen** is a static microblog site generator written in Go. It follows the convention-over-configuration philosophy and generates HTML sites from Markdown content files with YAML frontmatter. The tool supports a built-in admin interface, watch & hot reload mode, search engine, archive/tag indices, feed generation (RSS, Atom, JSON Feed), image processing with thumbnails, and rsync-based deployment.

## Key Commands

### Development
- `go build` - Build the mbgen binary
- `go test` - Run all tests
- `go test -v` - Run tests with verbose output
- `go test -run TestName` - Run a specific test

### Using mbgen
- `go run . init` - Initialize a new site (creates config.yml, downloads samples, installs default theme)
- `go run . generate` - Parse content and generate the static site
- `go run . serve` - Start local web server (default: http://localhost:8888)
- `go run . serve --admin` - Start server with admin UI for content management
- `go run . serve --watch-reload` - Start server with hot reload on markdown file changes
- `go run . inspect` - Inspect content and report issues
- `go run . inspect --fix` - Automatically fix detected issues (resize large images)
- `go run . stats` - Parse content and print statistics
- `go run . cleanup [target]` - Clean up generated files (content/thumbs/tags/archive/search)
- `go run . deploy` - Deploy site to remote server via rsync
- `go run . theme <action> <theme>` - Manage themes (activate/install/update/refresh/delete)

## Architecture

### Core Components

**Entry Point (`main.go`)**
- Command dispatcher that routes to appropriate handlers
- All commands are defined in `commands.go` with descriptors specifying usage, required args, and config requirements

**Configuration (`config.go`, `const.go`)**
- `config.yml` is the main configuration file
- Config is read once at startup and passed to command handlers
- Default values defined in `const.go` with constants for all paths, file extensions, regexes, etc.
- Embedded JavaScript/CSS resources for admin UI, search, and watch-reload functionality

**Parser (`parser.go`, `parser_cache.go`)**
- Parses Markdown content files (.md) from `pages/` and `posts/` directories
- Uses goldmark for Markdown parsing with extensions (strikethrough, tables, linkify, etc.)
- Extracts YAML frontmatter metadata (date, time, title, tags)
- Processes content directives: `{media}`, `{embed:url}`, `{with-media}`, hashtags (#tag), content links ({%page:id%}, {%post:id%})
- Converts embedded media URLs (YouTube, Vimeo) to embeddable formats
- Generates two versions of post content:
  - `Body`: fully processed HTML with all directives resolved (for web pages)
  - `FeedContent`: cleaned markdown with media directives removed (for feed generation)
- Caches parsed content by file modification time for performance (`parser_cache.go`)
- Returns `page` and `post` structs from `types.go`

**Processor (`processor.go`)**
- Takes parsed pages/posts and generates HTML output files
- Compiles Go templates from theme directories
- Generates paginated post lists, tag pages, archive pages
- Creates search index JSON file when `enableSearch` is enabled
- Generates feeds (RSS, Atom, JSON Feed) when `generateFeeds` is configured
- Outputs files to `deploy/` directory with structure:
  - `deploy/page/` - individual page HTML files
  - `deploy/post/` - individual post HTML files
  - `deploy/posts/` - paginated post lists
  - `deploy/tags/` - tag-filtered post lists
  - `deploy/archive/` - date-filtered post lists
  - `deploy/media/` - images/videos organized by content ID
  - `deploy/resources/` - theme resources (CSS, fonts, etc.)

**Image Processing (`imgutil.go`)**
- Resizes original images when `resizeOrigImages` is enabled and images exceed `maxImgSize`
- Generates thumbnails at multiple sizes (default: 480px, 960px) when `useThumbs` is enabled
- Supports JPEG and PNG formats with configurable quality/compression
- Thumbnails are named with `_<size>_thumb` suffix (e.g., `image_480_thumb.jpg`)
- Only generates thumbnails for images above `thumbThreshold` file size (default: 0.5 MB)

**Utilities**
- `fsutil.go` - File system operations (directory checks, file writing, etc.)
- `netutil.go` - HTTP server, WebSocket handling for watch-reload, admin API endpoints
- `tcutil.go` - Template compilation and rendering helpers
- `common.go` - Common utility functions

**Deployment (`deploy.go`)**
- Uses rsync to upload `deploy/` directory to remote server
- Uploads in specific order: media → pages → posts → post lists → tags → archive → everything else
- Requires `deployPath`, `deployHost`, and `deployUsername` in config.yml
- rsync options: --archive --compress --delete --no-t --no-o --no-g --no-p --progress --verbose

**Types (`types.go`)**
- Central type definitions for the entire application
- `appConfig` - all configuration options
- `page` and `post` - content entity types implementing `contentEntity` interface
- `media`, `thumb`, `embeddedMedia` - media handling types
- `templateContent` - data passed to templates during rendering
- `archiveIndexData`, `tagData`, `pagerData` - data structures for generated indices

**Feed Generation (`processor.go`)**
- Generates RSS 2.0, Atom, and JSON Feed formats using github.com/gorilla/feeds library
- Feed files: `rss.xml`, `atom.xml`, `feed.json`
- **Theme-independent implementation**: feeds use cleaned markdown content from `post.FeedContent` field
- Feed excerpts are generated from first 3 sentences (configurable via `feedExcerptSentenceCnt` constant)
- Fallback to first 20 words if no sentences found (configurable via `feedExcerptFallbackWordCnt` constant)
- Relative URLs in excerpts are converted to absolute URLs for feed reader compatibility
- Feed item titles include tags when post has no title: `2025-08-30 | #Tag1 #Tag2`
- Configurable "Continue Reading" link text via `feedPostContinueReadingText` config option
- Requires `siteBaseURL` to be configured

### Data Flow

1. User runs command (e.g., `mbgen generate`)
2. `main.go` validates command and loads config from `config.yml`
3. Parser reads markdown files from `pages/` and `posts/` directories
4. Parser extracts metadata, processes directives, handles media references
5. Processor compiles templates from active theme
6. Processor generates HTML by executing templates with parsed data
7. Output written to `deploy/` directory
8. Image processor generates thumbnails as needed

### Content Directive System

Content directives are special placeholders in Markdown that get transformed:
- `#tag` → tag links to `/tags/tag/`
- `{%page:id%}` → `/page/id.html`
- `{%post:id%}` → `/post/id.html`
- `{%search:query%}` → `/search.html?q=query`
- `{media}` → renders all media files from content's media directory
- `{media:file1.jpg,file2.png}` → renders specific media files
- `{media(prop1=val1,prop2=val2):file.jpg}` → media with properties
- `{with-media:file.jpg} content {/}` → media alongside content
- `{embed:youtu.be/VIDEO_ID}` → embedded YouTube player
- `{embed:vimeo.com/VIDEO_ID}` → embedded Vimeo player

Directives are processed using regex patterns defined in `const.go` and handled in `parser.go`.

### Theme System

Themes are located in `themes/` directory. The default theme is `pretty-dark`. Each theme contains:
- `templates/` - Go HTML templates (main.html, page.html, post.html, etc.)
- `resources/` - CSS, fonts, images copied to `deploy/resources/`
- `README.md` - theme documentation

Templates support includes:
- Global includes: `include/*.html` (injected in all pages)
- Theme includes: `include/<theme-name>/*.html` (theme-specific)
- Page-specific includes: `include/page-head--<page-id>.html`
- Template placeholders: `{{@ file.html @}}` for templates, `{{# file.html #}}` for content

### Admin & Watch-Reload Features

**Admin Mode** (`--admin` flag)
- Injects `admin.js` and markdown editor (EasyMDE) into pages
- Provides UI controls to create/edit/delete pages and posts
- API endpoints in `netutil.go` handle content operations
- Images auto-resize on upload if exceeding `maxImgSize`

**Watch-Reload Mode** (`--watch-reload` flag)
- Uses fsnotify to monitor `pages/` and `posts/` directories
- Automatically regenerates site when .md files change
- WebSocket connection pushes updates to browser
- `watch-reload.js` reloads the page when updates received

### Testing

Tests are in `*_test.go` files:
- `parser_test.go` - tests content parsing, directives, metadata extraction
- `processor_test.go` - tests HTML generation and template rendering

Run tests with `go test` or `go test -v` for verbose output.

## Important Implementation Notes

### File Organization
- All Go source files are in the root directory (flat structure)
- No package subdirectories except `feeds/`
- Markdown content goes in `pages/` and `posts/` directories
- Media files organized as `deploy/media/<page|post>/<content-id>/`

### Configuration System
- Config is read once at startup in `readConfig()` function
- `defaultConfig()` provides fallback values for all optional settings
- `writeConfig()` generates config.yml with comments for disabled options
- Config validation happens during read (e.g., feed formats, image sizes)

### Parser Cache
- Improves performance by caching parsed content indexed by filename and modification time
- Cache is in-memory only (not persisted between runs)
- Use `useCache` parameter in `parsePages()` and `parsePosts()` to enable

### Media Handling
- Images/videos must be in `deploy/media/<type>/<id>/` matching content file names
- Thumbnail generation is conditional based on file size threshold
- Original image resizing is opt-in via `resizeOrigImages` config option
- Supported image formats: JPEG (.jpg, .jpeg), PNG (.png), GIF (.gif)
- Supported video formats: MP4 (.mp4), MKV (.mkv), MOV (.mov)

### Search Implementation
- Client-side search using JavaScript
- Search index built at generation time contains all page/post content
- Search terms are OR'ed by default, use + for AND (e.g., "term1+term2")
- `search.js` handles the search UI and query execution

### Deployment
- rsync must be installed and available in PATH
- Remote server must have rsync and SSH access
- Local deployment possible by omitting `deployHost` and `deployUsername`
- Upload order ensures no broken links during deployment
