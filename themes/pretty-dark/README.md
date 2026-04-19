# pretty-dark

The default mbgen theme.

## Includes

* `header.html` - the site header content to be rendered on top of every page

## Content Directives

### Media rendering

The following properties can be used with both `{media}` and `{with-media}` directives:

* `cc=<num>` - defines the number of columns for the rendered media container, e.g.:
  * `{media(cc=3)}`
  * `{with-media(cc=2):1.mp4,2.jpg,3.jpg} ... {/}`
  * `{with-media(cc=4):1.jpg,2.jpg,3.jpg,4.jpg} ... {/}`

  _Note: `cc` applies to **images only**, i.e. videos are always rendered in a single column and take the full width of the parent container._

The following properties can be used with `{with-media}` directive only:

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

### Columns (side-by-side content rendering)

* Wrap two or more blocks of content in a `{cols}` ... `{//}` block to render them
side-by-side as a grid (note the double `/`, which indicates that it's a wrapping block).
* Each child column is wrapped in `{col}` ... `{/}`.

Basic two-column layout (equal widths):

```
{cols}
  {col}
  Left column content.
  
  Any Markdown is allowed here - paragraphs, lists, links, hashtags.
  {/}
  {col}
  Right column content.
  {/}
{//}
```

Optional relative-width weights via `{cols(w1:w2:…)}`. Weights are positive
integers expressed as a colon-separated list; they translate directly to CSS
Grid `fr` units. For example, `{cols(3:1:1)}` produces a grid with the first
column taking 60% and the other two 20% each:

```
{cols(3:1:1)}
  {col} wide column {/}
  {col} narrow column {/}
  {col} narrow column {/}
{//}
```

Per-column text alignment via `{col(a=l|c|r)}`:

* `a=l` - align left
* `a=c` - center
* `a=r` - align right

```
{cols(2:1:2)}
  {col(a=l)} left-aligned text {/}
  {col(a=c)} centered text {/}
  {col(a=r)} right-aligned text {/}
{//}
```

Column bodies may contain other content directives, including `{with-media}`,
`{media}`, `{embed}`, hashtags, and content links. For example, to put a
paragraph on the left and a stack of `{with-media}` blocks on the right:

```
{cols}
  {col}
  A few paragraphs of commentary here.
  {/}
  {col}
  {with-media(p=l,s=s):book-1.jpg}
  Book title
  #tag-a #tag-b
  {/}
  {with-media(p=l,s=s):book-2.jpg}
  Another book
  {/}
  {/}
{//}
```

#### Notes and limitations

* Nested `{cols}` inside a `{col}` is not supported.
* If the weight count doesn't match the column count (e.g., `{cols(3:1)}` with
  three `{col}` children), a warning is printed and the columns render with
  equal widths.
* A `{cols}` block with no `{col}` children is left as literal text.
* Unknown `{col}` prop values (e.g., `a=x`) are ignored with a warning;
  the column still renders without the invalid alignment applied.
* On viewports narrower than 640px, columns stack vertically.
