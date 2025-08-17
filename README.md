# Todostar

Todostar is a collaborative todo tech-demo app hand-crafted using
[Go](https://go.dev) + [Templ](https://templ.guide) + [Datastar](https://data-star.dev) +
[TailwindCSS](https://tailwindcss.com/).

This is a server-driven web application developed primarily in Go with minimal JavaScript.
The only thing I use [bun](https://bun.com/) for is building the tailwind bundle.

[Templier](https://github.com/romshark/templier) is used for interactive development and
hot reload.

Hypermedia Rocks 🤘

🚧 This is still a work in progress. I still need to fix the randomly hanging tab bug 🐛
and finish work on the live-updates 🔃.

## Improvement Ideas

- An archive page under path `/archive`.
- A 404 error page for unknown paths.
- User feedback on internal errors (e.g. "something went wrong").
- Visual loading indication for the folks on dial-up.
  - Skeletons to hide flashy web components and eliminate CLS
    (which is currently the biggest dent in the Lighthouse score).
- Theme toggleable between `system`/`light`/`dark`.
- More filters & sorting
- The due time of a todo entry is currently not dynamically updated
  and went left idle might show a not-up-to-date value.
- Anything else...? Drop an [issue](https://github.com/romshark/todostar/issues)!

## Development

Simply run `make templier` and start coding.
