# pretty-dark

The default mbgen theme.

## Includes

* `header.html` - the site header content to be rendered on top of every page

## Content Directives

* the following properties can be used with both `{media}` and `{with-media}` directives:
  * `cc=<num>` - defines the number of columns for the rendered media container, e.g.:
    * `{with-media(cc=3)} ... {/}`
    * `{with-media(cc=2):1.mp4,2.jpg,3.jpg} ... {/}`
    * `{with-media(cc=4):1.jpg,2.jpg,3.jpg,4.jpg} ... {/}`

    _(note that `cc` applies to images only,
    i.e. videos are always rendered in a single column
    and take the full width of the parent container)_


* the following properties can be used with `{with-media}` directive only:
  * `p=l|m|r` - defines the positioning of the rendered media container 
    * `m` is the default value and doesn't need to be specified explicitly - 
      it renders the media container before the `<related-content>` and positions it in the middle 
    * `l`/`r` render the media container on the left/right side of the `<related-content>`
    * usage examples: 
      * `{with-media(p=l):1.jpg} ... {/}`
      * `{with-media(p=r):1.jpg,2.jpg} ... {/}`
  * `s=s|m|l|f` - defines the size (small/medium/large/full) of the rendered media container
    * `m` is the default value and doesn't need to be specified explicitly  
    * `f` makes the media container take the full width of the parent container 
    * usage examples:
      * `{with-media(s=l):1.jpg,1.mp4} ... {/}`
      * `{with-media(s=s):1.jpg} ... {/}`
