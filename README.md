# mbgen

An easy-to-use, intuitive and fast static **m**icro-**b**log site **gen**erator,
embracing the convention-over-configuration philosophy.

## Features

* Built-in search engine
* Archive generation
* Thumbnail generation
* Watch & Hot Reload mode to preview changes in browser in real-time
* Simple and intuitive to use image and video embedding
* Customizable configuration (pagination, thumbnails, etc.)

## Demo

My personal website - [kion.name](https://kion.name/) - 
is generated using mbgen with the default theme.

## Installation

If you have Go installed:

```sh
go install github.com/kion/mbgen
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
│   │   │   ├── sample-page
│   │   │   ├── sample-post-1
│   │   │   ├── ...
│   ├── include ☜ include files dir
│   │   ├── default ☜ default theme include files dir
│   ├── pages ☜ page content markdown files dir
│   │   ├── sample-page.md
│   ├── posts ☜ post content markdown files dir
│   │   ├── sample-post-1.md
│   │   ├── ...
│   ├── themes ☜ themes dir
│   │   ├── default ☜ default theme dir
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

You can also use the `--watch-reload` flag to automatically regenerate the site 
and see the changes being reflected in the browser in real-time 
when you change any of the `.md` files in the `pages` or `posts` dirs:

```shell
$ mbgen serve --watch-reload
```

Other supported commands:

* parse content folders and print out the corresponding stats:
```shell
$ mbgen stats
```

* install/update and/or activate a theme:
```shell
$ mbgen theme <action> <theme>
```

* print out help/usage information:
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
  │   │   ├── sample-page
  │   │   │   ├── img-1.jpg
  │   │   │   ├── vid-1.mp4
  │   │   ├── sample-post-1
  │   │   │   ├── img-1.jpg
  │   │   │   ├── img-2.jpg
  │   │   │   ├── vid-1.mp4
  │   │   ├── sample-post-2
  │   │   │   ├── cover.png
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
    by placing the corresponding resource files 
    inside the `include` dir (for global level includes) 
    and/or the `include/<theme-name>` dir (for theme level includes)
    * The following are included in **all** generated files: 
      * `styles.css` - for custom CSS styles
      * `head.html` - for `<meta>` tags and `<link>` tags 
        referencing additional resources (e.g. css/js/font files)
    * To include additional `<meta>` and `<link>` tags for _pages_ only 
      (i.e. these are never included for _posts_), use the following include files:
      * `page-head.html` - is included in **all** generated page files 
      * `page-head--<page>.html` - is included in the generated file for the **specific page only** 
        (`<page>` is the name of the corresponding markdown content file without the `.md` extension, 
        e.g. `page-head--sample-page.html`)
    * _Global includes are injected before any theme-specific ones 
      (i.e. theme includes override the global ones)_
    * Each specific theme might also support some additional includes
      * Sample files for the supported includes are automatically copied to the `include/<theme-name>` dir 
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
* [optional] `siteName` - the name of the site
  - if `:` needs to be used in the site name, 
    the property value must be enclosed in double quotes
* [optional] `homePage` - an id of the page to use as the site's home page
  - for example, if there was a `pages/sample-page.md` content file, 
    it would be possible to set the value to `sample-page`
* [optional] `generateArchive` - the posts archive generation is enabled by default, 
  unless this setting is set to `no`
  - `generate` command generates an archive index page, 
    as well as pages with posts for all the relevant year/month combinations 
    (retrieved from the `date` property of each corresponding post content `.md` file) - 
    the generated index page lists all the year/month combinations with links to the corresponding content pages
  - the generated archive index page is available under `/archive/` URI, e.g.:
    - `<a href="/archive/">Archive</a>`
  - set this setting to `no` to disable archive generation
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
* [optional] `serveHost` - host to use for `serve` command
  - if not specified, the default value of `localhost` is used
* [optional] `servePort` - port to use for `serve` command
  - if not specified, the default value of `8080` is used

## License

Distributed under the GNU AGPLv3 license.

See [LICENSE](https://github.com/kion/mbgen/blob/main/LICENSE) for more information.
