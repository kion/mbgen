# mbgen

An easy-to-use, intuitive and fast static **m**icro-**b**log site **gen**erator,
embracing the convention-over-configuration philosophy.

## Features

* Built-in admin (content management) interface
* Watch & hot reload mode: monitor changes to the content and media (image/video) files,
  generate the corresponding output files on the fly, and dynamically update the live view in browser
* Built-in search engine
* Archive Index generation
* Tag Index (Tag Cloud) generation
* Feed generation with support for RSS 2.0, Atom, and JSON Feed formats
* Built-in support for pagination, as well as tag-based and date-based filtering
* Simple and intuitive media (image/video) embedding with custom rendering options
* Support for original image file resizing and thumbnail generation
* Easy to use content directives (e.g. hashtags, links to other posts/pages, etc.)
* Support for custom styling and includes
* Customizable configuration (pagination, thumbnails, etc.)
* Built-in support for deploying the generated site to a remote server using `rsync`

## Demo

My personal website - [kion.name](https://kion.name/) -
is generated using mbgen with the default theme.

## Installation

If you have Go installed:

```sh
go install github.com/kion/mbgen@latest
```

Otherwise, download the [latest binary release](https://github.com/kion/mbgen/releases/latest) for your OS,
rename it to `mbgen`, make it executable (`chmod +x mbgen` on any Unix-like system),
and either add it to your `PATH` to make the `mbgen` command available globally
or just place it in any dir to limit its usage from that dir only.

## Quick Start

Create an empty working dir for your site
and run the following command inside it:

```shell
$ mbgen init
```

This creates the following working dir structure:

```
├── config.yml ☜ sample config file
│   ├── deploy ☜ dir to be deployed to server
│   │   ├── favicon.ico ☜ sample favicon
│   │   ├── media ☜ media content dir
│   │   │   │   ├── page
│   │   │   │   │   ├── sample-page
│   │   │   │   ├── post
│   │   │   │   │   ├── sample-post-1
│   │   │   │   │   ├── ...
│   ├── include ☜ include files dir
│   │   ├── pretty-dark ☜ default theme include files dir
│   ├── pages ☜ page content markdown files dir
│   │   ├── sample-page.md
│   ├── posts ☜ post content markdown files dir
│   │   ├── sample-post-1.md
│   │   ├── ...
│   ├── themes ☜ themes dir
│   │   ├── pretty-dark ☜ default theme dir
```

Run the following command to generate your site:

```shell
$ mbgen generate
```

This generates all the static content files (`.html`) inside the `deploy` dir,
copies all the required theme resources (like fonts, styles, etc.) the static files require,
as well as generates the thumbnails for the content images whenever appropriate
_(this default behavior can be disabled in the `config.yml`)_.

Run the following command to start a simple http server to serve the generated site locally:

```shell
$ mbgen serve
```

Then open the following address in a browser to preview your site:

[http://localhost:8888/](http://localhost:8080/)

_(the default host and port values can be modified in the `config.yml`)_

## Admin Commands / Content Management

Use the `--admin` flag to render content management UI components to create/edit/delete pages and posts:

```shell
$ mbgen serve --admin
```

Alternatively, you can use the `--watch-reload` flag
to monitor any changes to the source content (`.md`) files in the `pages` and `posts` dirs,
automatically regenerate the site on the fly, and see the changes dynamically reflected in browser.

```shell
$ mbgen serve --watch-reload
```

## Deployment

You can upload the `deploy` dir to a remote server manually / using any tool of your choice.

Ideally, the sub-dirs and files inside the `deploy` dir should be uploaded in the following order:

* `media` dir - all the media files (images/videos)
* `page` dir - all the generated single page files
* `post` dir - all the generated single post files
* `posts` dir - all the generated paginated post files
* `tags` dir - all the generated tag-scoped post files
* `archive` dir - all the generated archive index & month-scoped post files
* everything else inside the `deploy` dir, excluding the directories listed above

This way you can avoid broken links/refs on the site during the upload process.

However, if you have `rsync` installed and available in your `PATH`,
you don't have to handle the upload process manually and worry about the upload order of the files and directories.

You can simply use the built-in `deploy` command to upload the generated site to a remote server
(which should be accessible via SSH and have `rsync` installed/available as well),
which would make sure to use the appropriate upload order (as outlined above):

```shell
$ mbgen deploy
```

The `rsync` tool itself is always invoked with the following options:

```
--archive --compress --delete --no-t --no-o --no-g --no-p --progress --verbose
```

_(it would upload only the files changed since the last deployment, which is what this tool is designed for! ;))_

In order to use the `mbgen deploy` command, the `deployPath`, `deployHost`, and `deployUsername` config options must be set in the `config.yml` file
(see more details in the corresponding section down below).

## Other Commands

* Inspect content directories and report/fix any issues found:

```shell
$ mbgen inspect
$ mbgen inspect --fix
```

* Parse content directories and print out the corresponding stats:
```shell
$ mbgen stats
```

* Install/update and/or activate a theme:
```shell
$ mbgen theme <action> <theme>
```

* Print out help/usage information:
```shell
$ mbgen help [command]
```

## General Conventions

* Content files should be placed inside the
  `pages` and `posts` dirs and have the `.md` file extension
* The `deploy` dir contains all the content
  (images/videos, static resources, generated static `.html` site pages, etc.)
  to be uploaded to the web hosting server
* All the images/videos are stored inside the `deploy/media` dir
  and should be placed into the sub-dirs with names matching
  the names of the corresponding content files, e.g.:
  ```
  ├── deploy
  │   ├── media
  │   │   │   ├── page
  │   │   │   │   ├── sample-page
  │   │   │   │   │   ├── img-1.jpg
  │   │   │   │   │   ├── vid-1.mp4
  │   │   │   ├── post
  │   │   │   │   ├── sample-post-1
  │   │   │   │   │   ├── img-1.jpg
  │   │   │   │   │   ├── img-2.jpg
  │   │   │   │   │   ├── vid-1.mp4
  │   │   │   │   ├── sample-post-2
  │   │   │   │   │   ├── cover.png
  ├── pages
  │   ├── sample-page.md
  ├── posts
  │   ├── sample-post-1.md
  │   ├── sample-post-2.md
  ```
* Content files are a combination of the content body in the Markdown format
  along with Yaml formatted metadata in the beginning of the file
  * Post content file example:
    ```
    ---
    date: 2023-08-01
    time: 19:15
    title: Post title
    tags:
      - Tag1
      - Tag2
      - Tag3
    ---

    Post body in the markdown format

    * [Link to another post]({%post:sample-post-1%})
    * [Link to another page]({%page:sample-page%})

    #Tag4 #Tag5

    {media}
    ```
  * Page content file example:
    ```
    ---
    title: Page title
    ---

    Page body in the markdown format

    #Tag1 #Tag2

    {embed:youtu.be/A_bCdEfGhIj-X}
    ```
  * The following content directives are supported:
    * `#<tag>` - renders a hashtag link
      * _there's no need to add hashtags to the metadata `tags` section,
        i.e. hashtags are automatically included in the post's tag list_
    * `{%<entry-type>:<entry-id>%}` - renders a URI to the given `<entry-id>` of the given `<entry-type>`, e.g.:
      * `[Sample Page]({%page:sample-page%})` - would render a link with title `Sample Page` to the page defined in the `pages/sample-project.md`
      * `[Sample Post 1]({%post:sample-post-1%})` - would render a link with title `Sample Post 1` to the post defined in the `posts/sample-post-1.md`
    * `{%search:<search query>%}` - renders a URI to the search page with the given `<search query>`
      (_note: this content directive would work as expected only if
      the `enableSearch` configuration option was not disabled - see more details in the corresponding section below)_, e.g.:
      * `[Search for "something interesting"]({%search:something interesting%})` - would render a link with title `Search for "something interesting"` pointing to `/search.html?q=something%20interesting`
        * _Search would return content entries containing the words `something` **or** `interesting`_
      * `[Search for "something+interesting"]({%search:something+interesting%})` - would render a link with title `Search for "something+interesting"` pointing to `/search.html?q=something%2Binteresting`
        * _Search would return content entries containing the words `something` **and** `interesting`_
    * `{media(<properties>):<file(s)>}` - renders images/videos from the corresponding `deploy/media/<entry-id>` dir
      * to render specific media files, list them explicitly, e.g.:
        * `{media:1.jpg}`
        * `{media:1.jpg,2.png}`
        * `{media:1.mp4,1.jpg,2.png}`
      * `{media}` (without any explicitly listed files)
        renders all the media files from the corresponding `deploy/media/<entry-id>` dir,
        _excluding the explicitly listed media files_
      * additional properties might be supported by each specific theme to fine-tune the media container rendering
        * properties should be comma-separated and use `=`
          to separate the key (on the left) and the value (on the right) of each particular property:
          * `{media(key1=val1):1.jpg}`
          * `{media(key1=val1,key2=val2)}`
     * `{with-media(<properties>):<file(s)>} <related-content> {/}` -
       renders images/videos alongside the `<related-content>`
       * otherwise, works the same way as the `{media}` directive above, e.g.:
         * `{with-media:1.jpg} ... {/}`
         * `{with-media:1.jpg,1.mp4} ... {/}`
         * `{with-media} ... {/}`
    * `{embed:<url>}` - allows to embed media from the supported media content hosting platforms, e.g.:
      * `{embed:youtu.be/A_bCdEfGhIj-X}`
      * `{embed:vimeo.com/1234567890}`
      * The following platforms are currently supported:
        * YouTube
        * Vimeo
  * Custom/additional resources can be integrated on the global and/or theme level
    by placing a `head.html` file inside the `include` dir (for global level includes)
    and/or the `include/<theme-name>` dir (for theme level includes)
    * `head.html` can include any number of `<meta>`, `<link>`, `<script>` tags
      (as well as any other tags allowed in the `<head>` section of the HTML document)
      referencing additional resources (e.g. css/js files)
      * the actual additional resource files could simply be placed in the `deploy/res` dir
        and then referenced from the `head.html` file as follows:
        * `<link rel="stylesheet" href="/res/css/custom.css">`
        * `<script src="/res/js/custom.js"></script>`
      * _do **not** place your custom resources in the `deploy/resources` dir,
        as it's reserved for the theme-specific/generated resources/includes_
    * to include/reference additional resources for _pages_ only
      (i.e. these would never be included for _posts_), use the following include files:
      * `page-head.html` - is included in **all** generated page files
      * `page-head--<page>.html` - is included in the generated file for the **specific page only**
        (`<page>` is the name of the corresponding markdown content file without the `.md` extension,
        e.g. `page-head--sample-page.html`)
    * _global includes are injected before any theme-specific ones
      (i.e. theme-specific definitions override the global ones)_
    * each specific theme might also support some additional includes
      * sample files for the supported includes are automatically copied to the `include/<theme-name>` dir
        and should be modified/tweaked to include any custom markup and content

## Theme Specific Conventions

* [pretty-dark](https://github.com/kion/mbgen/blob/main/themes/pretty-dark/README.md) - the default theme

## Configuration

A number of configuration options can be specified in the `config.yml`
to customize the app behavior / generated output
(the options marked as **[required]** must be present and have a value defined,
while the [optional] ones do not require a value to be defined
or can even be completely omitted from the config):

* **[required]** `theme` - path to the active theme dir
  (either relevant to the working dir or an absolute one)
* **[required/optional]** `siteBaseURL` - the base URL of the site
  - must be specified if feed generation is enabled (see the `generateFeeds` option), optional otherwise
  - should include the protocol (either `http://` or `https://`)
  - the trailing slash (`/`) is allowed, but **not** needed
  - e.g. `https://kion.name`
* [optional] `siteName` - the name of the site
  - if `:` needs to be used in the site name,
    the property value must be enclosed in double quotes
  - this value is used in the generated feed files (if feed generation is enabled)
* [optional] `siteDescription` - the description of the site
  - if `:` needs to be used in the site description,
    the property value must be enclosed in double quotes
  - this value is used in the generated feed files (if feed generation is enabled)
* [optional] `homePage` - an id of the page to use as the site's home page
  - for example, if there was a `pages/sample-page.md` content file,
    it would be possible to set the value to `sample-page`
* [optional] `generateArchive` - the posts archive generation is enabled by default,
  unless this setting is set to `no`
  - `generate` command generates an archive index page,
    as well as pages with posts for all the relevant year/month combinations
    (retrieved from the `date` property of each corresponding post content `.md` file) -
    the generated index page lists all the year/month combinations (along with the corresponding post counts)
    with links to the corresponding content pages
  - the generated archive index page is available under `/archive/` URI, e.g.:
    - `<a href="/archive/">Archive</a>`
  - set this setting to `no` to disable archive generation
* [optional] `generateTagIndex` - the tag index generation is enabled by default,
  unless this setting is set to `no`
  - `generate` command generates a tag index page
    (note, that the pages with posts for each tag are generated always, even if tag index is not) -
    the generated tag index page lists all the tags (along with the corresponding post counts)
    with links to the corresponding content pages
  - the generated archive index page is available under `/tags/` URI, e.g.:
    - `<a href="/tags/">Tags</a>`
  - set this setting to `no` to disable archive generation
* [optional] `generateFeeds` - the feed generation is disabled by default,
  unless this setting is set to a comma-separated list of feed formats to generate:
    * `rss` - RSS 2.0 feed
    * `atom` - Atom feed
    * `json` - JSON Feed
  - e.g. to generate all three feed formats, set the value to `rss, atom, json`
  - `generate` command generates the following feed files 
    (and links them within the generated content pages) 
    under the `deploy` dir, depending on the specified formats:
    * `rss.xml` - RSS 2.0 feed
    * `atom.xml` - Atom feed
    * `feed.json` - JSON Feed
* [optional] `feedPostCnt` - controls the maximum number of posts included in generated feeds
  - if not specified, the default value of `20` is used
* [optional] `feedPostContinueReadingText` - controls the "continue reading" text appended to each post content in generated feeds
  - if not specified, the default value of `Continue reading ->` is used
* [optional] `enableSearch` - the built-in search functionality is enabled by default,
  unless this setting is set to `no`
  * `generate` command generates:
    * a search index file (`/search.json`),
      containing searchable text content for all the page and post content (`.md`) files
      (including the content body, as well as the `title` and the `tags` property values)
    * a search UI page, which is available under `/search.html` URI, e.g.:
      - `<a href="/search.html">Search</a>`)
  * _note: the search functionality requires JavaScript to be enabled in the browser_
* [optional] `pageSize` - controls the maximum number of posts
  on any page that renders a list of posts
  - if not specified, the default value of `10` is used
* [optional] `resizeOrigImages` - the original image resizing is disabled by default,
  unless this setting is set to `yes`
  - `generate` command resizes the original images
    (preserving the original image aspect ratio)
    with the width/height (whichever one is larger)
    exceeding the value specified in the `maxImgSize` option
    (whether it's the explicitly specified or the default one)
  - set this setting to `yes` to enable the original image resizing
* [optional] `maxImgSize` - defines the max size (in pixels) for the original images
  - if not specified, the default value of `1920` is used
  - the original images with the width/height (whichever one is larger)
    exceeding this value are resized to fit the specified width/height
    (while preserving the original image aspect ratio) if:
    - an original image is uploaded via the admin interface (see the `serve` command with the `--admin` flag)
    - the `inspect` command is run with the `--fix` flag (in this case all the original images are inspected/resized)
* [optional] `useThumbs` - the thumbnail behavior (to optimize the page size / load time)
  is enabled by default, unless this setting is set to `no`:
    - `generate` command generates a number of thumbnails
      (preserving the original image aspect ratio)
      with widths/heights (whichever one is larger)
      matching the values of the `thumbSizes` option
      for each image inside the `deploy/media` sub-dirs
      with the file size (**in MB**) larger than the `thumbThreshold` value
      (whether it's the explicitly specified or the default one)
    - all the `<img>` tags for the generated output (`.html` files)
      use the original image URL for the `href` attribute
      and the corresponding thumbnail URL for the `src` attribute
      as well as `srcset` attribute with all the URI and width values
      (if the option is disabled, both the `href` and the `src`
      attributes are set to point to the original image URL,
      while the `srcset` is not used at all)
    - the following image formats are supported:
      - JPEG (`.jpg` and `.jpeg` file extensions)
      - PNG (`.png` file extension)
    - set this setting to `no` to disable the thumbnail behavior
* [optional] `thumbSizes` - defines the set of size values for the generated thumbnails,
  where each value corresponds to either the width or the height of the generated thumbnail
  (depending on which one is larger for the original image)
  - if not specified, the default widths are used: `480, 960`
  - ignored if the `useThumbs` option is disabled
* [optional] `thumbThreshold` - defines the min file size of the original image **in MB**
  to trigger the corresponding thumbnail generation
  - if not specified, the default value of `0.5` (i.e. **0.5 MB**) is used
  - ignored if the `useThumbs` option is disabled
* [optional] `jpegQuality` - defines the quality of the generated JPEG thumbnails and original image replacements
  (the value should be in the range from `70` to `100`)
  - if not specified, the default value of `85` is used
  - has no effect if both the `maxImgSize` and the `useThumbs` options are disabled
* [optional] `pngCompressionLevel` - defines the compression level of the generated PNG thumbnails and original image replacements
  (the value should be one of the following: `DefaultCompression`, `NoCompression`, `BestSpeed`, `BestCompression`)
  - if not specified, the default value of `DefaultCompression` is used
  - has no effect if both the `maxImgSize` and the `useThumbs` options are disabled
* [optional] `serveHost` - host to use for `serve` command
  - if not specified, the default value of `localhost` is used
* [optional] `servePort` - port to use for `serve` command
  - if not specified, the default value of `8888` is used
* [optional] `deployPath` - deploy destination path (e.g. `/var/www/html`)
  - must be specified for the `deploy` command to work
  - _if `deployHost` and `deployUsername` config options are not specified,
    the `deployPath` is used as a local path - this is mostly useful for testing purposes only_
* [optional] `deployHost` - remote host (a domain name or an IP address) to deploy the site to
* [optional] `deployUsername` - username for the SSH connection to the remote deployment host

## License

Distributed under the GNU AGPLv3 license.

See [LICENSE](https://github.com/kion/mbgen/blob/main/LICENSE) for more information.
