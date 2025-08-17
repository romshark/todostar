# Todostar

https://github.com/user-attachments/assets/1069facc-1de8-4731-b86b-d74625814a22

Todostar is a collaborative todo tech-demo app hand-crafted using:

- [Go](https://go.dev) main language.
- [Templ](https://templ.guide) HTML templates.
- [Datastar](https://data-star.dev) JS-killer.
- [TailwindCSS](https://tailwindcss.com/) styling.
- [WebAwesome](https://webawesome.com/) web components.
- [Templier](https://github.com/romshark/templier) hot-reloader.

This is a server-driven web application in just 2k LoC of Go and minimal JavaScript.
The only thing I use [bun](https://bun.com/) for is building the tailwind bundle.
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
  and when left idle might show a not-up-to-date value.
- Anything else...? Drop an [issue](https://github.com/romshark/todostar/issues)!

## Development

Simply run `make templier` and start coding.
