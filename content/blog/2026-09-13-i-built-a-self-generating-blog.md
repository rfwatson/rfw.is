---
title: "Weekend project: I built a self-generating blog"
published_at: 2026-09-13T12:00:00+02:00
---

My weekend project: build a new self-hosted blog from scratch, without using an existing static site generator, theme
or&mdash;most importantly&mdash;any LLMs.

Spoiler: [I did it](https://github.com/rfwatson/rfw.is), and you're reading it right now. Woohoo! Before we get to the
technical details, let's briefly discuss the constraints.

### Self-hosted

Constraint number one is that the blog must be self-hosted.

There are lots of good reasons to self-host free software in general. But in this project I am mostly concerned with
longevity. A blog must stand the test of time&mdash;I want it to look and behave exactly the same in ten years as it
does today.

With a hosted blogging service, I can't be certain that it won't:

- change its functionality (very often for the worse)
- update its terms and conditions
- increase its prices
- introduce advertising
- intrusively track my visitors
- close my account
- be acquired by a fascist billionaire
- disappear completely

If I self-host my own blog using software that I control then the only risk is that one day I have to change my
hosting provider&mdash;a relatively easy job with many possible resolutions. Self-hosting immediately rules out blog
providers such as Wordpress, Ghost, Bearblog and so on.

### No existing generator or theme

A static site generator (SSG&mdash;henceforth, simply generator) is a tool that converts a source set of markdown
files&mdash;say, hand-written blog posts&mdash;into HTML format ready for publishing on the web. Established
generators typically provide additional features such as themes, asset pipelines, and even APIs to
connect with hosting services.

My second constraint is to not use a pre-existing generator. There are [hundreds](https://jamstack.org/generators/)
out there, the vast majority of which I have not evaluated. However, without pointing fingers at any specific generator
my experience has been that many are far too heavyweight for my requirements.

A blog is, in its more basic forms, a simple thing. Generating it should not require more than parsing a bunch of
markdown into a couple of varieties of page layouts, rendering a bit of CSS, eventually perhaps a site map and RSS feed. These requirements
do not call for a library with tens of thousands of lines of code, a shifting API surface and&mdash;most
off-puttingly&mdash;a steep learning curve when things don't work as expected.

Closely related is the fact that I also want to design my blog from scratch. This is partly a matter of
practicality&mdash;to my taste and knowledge there are no free blog themes for my choice of existing generators that I
would want to make use of. The idea of painstakingly adapting an existing theme also feels unsatisfying. This rules out the use
of any theme library, and in doing so diminishes a central reason to use an established generator.

But it also reflects another important principle: I want my personal blog to be, as far as possible, an expression of my
own creativity, not somebody else's.

Which brings us to the final constraint.

### No LLMs, anywhere

I use LLMs all day at work. To some extent I have also used them to support personal projects, and am still in the
process of crystallizing how much, and in what forms, I do so in the future. But in this particular project I am drawing
a clear line.

I know that I could prompt an agent and within minutes have a new blog designed, templated and even deployed to some
hosting provider. It would be shiny and modern and beyond the limits of what my own frontend design skills could
reasonably produce. Heck, the LLM would even draft and edit my posts if I wanted.

But the resulting blog would not be mine. We can think of a creative artefact as being the sum of the [thousands of
individual creative decisions](https://craphound.com/news/2025/03/30/why-i-dont-like-ai-art/)&mdash;small and
large&mdash;that are made during its creation. It is these decisions, each unavoidably guided by our own character and
experience, that imbue the artefact with our own creative DNA. It is this process, in large part, that makes the thing ours.

Outsourcing these decisions to an LLM does not make them magically go away. It just subtitutes them with a weighted average of
other people's answers to similar questions, laundered through a neural network but still in no sense my own. It is not
difficult to think of less creative contexts where this mechanism is easier to justify&mdash;refactoring a large codebase at work, for example&mdash;but I
want to be unambiguously the sole author of my own personal blog.

So my lines are clear. While building this blog:

1. No LLM usage for coding-related tasks, including LLM-powered autocomplete, chat or agent usage.
2. No LLM usage for designing the blog.
3. No LLM usage for ideating, drafting, reviewing or editing blog posts.

I _may_ occasionally write about LLMs. Otherwise&mdash;no LLMs, in this project, ever.

Now, on with the technical details.

## The technical details

rfw.is is built with a minimal tech stack:

* Go 1.27.1
* Two third-party dependencies:
    * [goldmark/v2](https://github.com/yuin/goldmark) markdown parser/renderer
    * [chroma/v3](https://pkg.go.dev/github.com/alecthomas/chroma/v3) syntax highlighter
* [Sass](https://sass-lang.com/guide/) CSS preprocessor
* [mise-en-place](https://mise.jdx.dev/) environment/task management

I also used some web fonts&mdash;Inter, Quantico and Roboto Mono&mdash;courtesy of Google Fonts.

The non-test code weighs in at less than five-hundred lines of Go code. You can find the entire source code at
[https://github.com/rfwatson/rfw.is](https://github.com/rfwatson/rfw.is).

### How it works

As described above, the central logic is pretty simple:

1. Given a source set of markdown and SCSS files
2. Convert them into HTML and CSS files
3. Write them to disk

I started by creating a `content` directory to hold the source files, including blog posts, generic pages (such as the
About page), CSS files and static assets (which need copying instead of converting).

The `content` directory is made available to the Go code by way of Go's `embed.FS` feature. This works, but in hindsight
probably wasn't necessary. Go's [`embed`](https://pkg.go.dev/embed) package, which allows arbitrary data to be embedded in the compiled binary,
excels when the binary needs to be run independently from the source code. However, I am unlikely to have this
requirement, at least in the foreseeable future. I will almost certainly always either generate the blog on my local
machine, or in a CI flow&mdash;in both cases the full Git repository is freely available. Instead of `embed` package, I
could have used the more flexible `os` methods to read the files directly from disk.

I then implemented two packages. First the `markdown` package which wraps functionality from the [goldmark](https://github.com/yuin/goldmark) markdown parsing and
rendering module. This module was recently updated to `v2` and I was a little cautious to use it for various reasons.
So far the new version has worked well, but for reasons of caution I isolated it in its own package to allow it to be easily
interchanged or downgraded if I ever need to.

Secondly, the `generator` package implements the actual conversion of source files into publishable assets. It iterates
through the source files, passes them to the `markdown` package for parsing and conversion, and stores the resulting
HTML file and any parsed front matter data in an in-memory data structure. (They are not immediately written to
disk&mdash;this allows for more efficient unit testing). It also handles page layouts, which are effectively
sub-templates which differ based on the type of page being rendered. This was by far the trickiest part of the
implementation&mdash;see below for a few things I learnt.

Finally, the generator package calls out to a `sass` binary to preprocess stylesheets, and adds them to the output
before returning it to the caller. The main package closes things out by writing each file to disk in the desired
location.

### Deployment

I bought the domain with Namecheap, one of the domain registrars which support the `.is` TLD. I also set up Cloudflare
in proxied mode, as much to experiment with some of the AI crawler blocking functionality as anything. There may be some
reasons not to have Cloudflare in the future but I am not really dependent on it for anything, so removing it should be
easy if it's ever needed.

I happen to have a cheap OVH VPS kicking around not doing much except running
[Caddy](https://github.com/caddyserver/caddy). The Caddyfile configuration only requires the Cloudflare origin
certificate and key to be imported.

```caddyfile
rfw.is {
  tls /etc/tls/rfw.is/cert.pem /etc/tls/rfw.is/key.pem
  root * /var/www/rfw.is
  file_server
  log

  handle_errors 404 {
    rewrite /404.html
    file_server
  }

  handle_errors 500 {
    rewrite /500.html
    file_server
  }
}
```

Building and deploying the blog are both then one-liners:

```shell
go run .           // generate blog to dist/
rsync -av dist/* myhost:/path/to/sites/rfw.is
```

### Next steps

I now have a functional blog, very much of my own creation, and I am very happy with it.

On my wishlist for future iterations:

* An RSS feed
* Improved navigation&mdash;tags, next and previous post
* OpenGraph tags and such like
* A site map for crawlers
* A bit of refactoring

But the first priority will be to write some more words.

### Addendum: Some things I learnt about Go templates

Go's deep standard library often makes it unnecessary to reach for third-party dependencies even
with non-trivial projects. I like to make the most of this, which means that I reach for
[`html/template`](https://pkg.go.dev/html/template) often.

For simple use cases, `html/template` is easy enough to use. But things get more complicated when nested templates&mdash;a pre-requisite for
even a simple generator&mdash;come into the picture.

This isn't the first time I've run into problems with nested templates. A few notes of things that I learnt or re-learnt
during the process:

* Calling `template.New("").ParseFiles(...)` and its siblings returns a `*template.Template` which is actually a _mutable
  set_ of parsed but non-executed templates keyed by their filename.
* New templates can be added to the set programatically, with `tmpl.New("template_name").Parse("some template")`.
* After adding a new template&mdash;even programmatically&mdash;it can be referenced from other templates using its
  `template_name`.
* A template set can only be executed (that is&nbsp;written to a buffer) once. But it can be shallow cloned with the
  `Clone()` method, allowing both templates to be added or overridden, and for the template set to be executed multiple
  times.
* After mastering the above rules, it's tempting to reach for templates to render arbitrary data. But templates must be
  valid in the sense that they must be accepted by `Parse()` and friends. This works for arbitrary data until it is not
  a valid template: for example, if it _includes_ a Go template, like a blog post about Go templates might do! This was
  a good reminder that Go templates are based around executing templates against a data object which has no such
  constraints. Passing non-template data through the data object is the simpler and more correct approach.

Together, the above rules helped create a relatively sane template implementation. The only remaining tricky part was
making it select a layout template dynamically, based on post type (for example, allowing a different layout template
for blog posts and generic pages).

For this, I had to fall back to a bit of slighly hacky inline template generation:

```go
// layout is a dynamically created template which does nothing except execute
// another template matching the provided layoutName. In other words, it
// dynamically switches layout for this single cloned template set.
layout := fmt.Sprintf(`{{template "%s"}}" .`, layoutName)

if tmpl, err = tmpl.New("layout").Parse(layout); err != nil {
    // handle error
}

// The `layout` template can now be called from main.html.tmpl...
```

I hope these notes help somebody out, or at least my future self.
