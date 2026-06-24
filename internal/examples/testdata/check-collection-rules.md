The `posts` collection attaches three checks: an H1 must exist, the frontmatter `title` must match that H1, and the filename must be kebab-case. `hello-world.md` satisfies all three; `Bad_Title.md` violates the casing rule and the title/H1 match, so `check` reports both and exits 1.

## Input

`.katalyst/storage/local.yaml`

```yaml
type: filesystem
root: .
collections:
  posts:
    path: content/posts
    checks:
      - kind: markdown_requires_h1
      - kind: markdown_title_matches_h1
        field: title
      - kind: filesystem_name_case
        style: kebab
```

`content/posts/hello-world.md`

```markdown
---
title: Hello world
---
# Hello world
```

`content/posts/Bad_Title.md`

```markdown
---
title: Bad title
---
# A different heading
```

## Command

```console
$ katalyst check posts
<project>/content/posts/hello-world.md: OK
<project>/content/posts/Bad_Title.md:4: /title: "Bad title" does not match first H1 "A different heading"
<project>/content/posts/Bad_Title.md: /: filename "Bad_Title" must be kebab-case
exit status 1
```

