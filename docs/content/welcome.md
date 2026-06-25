+++
title = "Welcome"
weight = 10
aliases = ["why-katalyst"]
+++

# Welcome

Katalyst is a *content consistency layer*, designed for people and agents who curate persistent memory, wikis, and knowledge bases.

Katalyst gives you and your agent tools to solve problems like these:

- "My agent takes a long time to find things, and sometimes burns a ton of tokens."
- "I've repeatedly told my agent how to organize content and it still gets it wrong."
- "Sometimes when I go back and look, I discover that my agent has completely lost important information."
- "The content in my knowledge base isn't just text. It also includes metadata that I need to be able to categorize, score, filter, sort, etc."
- "My agent is supposed to store and curate notes for me, but I spend way too much time checking its work."
- "I want to change how I'm storing my data, but migration would be a big pain."

If you want to be confident that your content/data/memory is always in good shape, even when it's maintained by sometimes-sloppy agents and sometimes-sloppy humans, then Katalyst is for you.

> [!NOTE]
> **New to Katalyst?**
>
> [Get started »]({{< relref "getting-started.md" >}}), install
> the CLI, scaffold a `.katalyst/` project, and run your first checks in a few
> minutes.

## Key features

### Catalog the content you already have

Katalyst comes with tools and skills to take stock of your content, no matter
what state it's in today. It can help you (and your agents) figure out what
you've got, map out the important concepts, and, if needed, get more
organized.

Compared to having an LLM scan every file or write its own bash scripts, this
approach can save a ton of tokens. It also lets you take advantage of skills,
tools and strategies curated by a community of people who've faced similar
challenges.

### Define the language and structure that work best for you

Curation always requires shared language and consistent structure. Katalyst
provides tools for declaring structure and rules for your content in your
knowledge base.

Rule layers include:

- *Markdown content*: required sections, naming conventions, templates, and
  related conventions.
- *File structure*: naming conventions, preferred and required extensions, and
  directory structures.
- *Metadata*: required fields, types, enums, numeric ranges, and full JSON
  Schema validation of frontmatter.
- *Object relationships*: links, summaries, tables of contents, and sequential
  numbering.

### Reshape as needed

As your content evolves, Katalyst gives you tools to navigate change.

Common updates include:

- *Rules*: add or change checks.
- *Content shape*: change the structure of your content.
- *Bases*: change where content lives.

## Design principles

### Lightweight, deployable anywhere

You can run Katalyst as a linter, a CLI, or a server. Use only the
infrastructure that you need for your particular use case.

### Model- and backend-agnostic

I'm building Katalyst to work with a variety of filesystems and databases. It
isn't tied to any one data store.

Similarly, you choose which model to use.

### Leans into shared language

Express the same rules in a project's own vocabulary and conventions.

### Built for both humans and agents

Ergonomics matter: especially for agents. An agent should be able to read the
rules, find what it needs, and extend them without ceremony:

This means optimizing for:

- *Speed*: fast enough to run on every write.
- *Discoverability*: an agent can find the schemas and structure on its own.
- *Readability*: rules and content stay legible to humans and agents alike.
- *Extensibility*: add new check types as needs grow.

## What if structure was light?

Curating your content with the right structure makes it more useful, but it also takes work. Historically, defining the right structure for knowledge was heavy: high-cost, high-risk, and sometimes technically demanding. This was doubly true when changing needs required updates to structure.

As a result, most structured data systems were rigid and hard to change. Most unstructured knowledge bases were either chronically outdated, or very limited in scope.

As AI starts to infuse our work, curating knowledge is going to become even more important, a massive potential unlock for people who want to work more productively and creatively with agents.

What if structure were light: easy to add, easy to maintain, easy to change?

In a world of unbounded creative collaboration with agents, the limiting factor isn't generating new ideas or gathering more information: it's having a shared language and structure to organize what we've learned, and to act on it together.

For more on these ideas, see [Deep dives]({{< relref "deep-dives/_index.md" >}}), especially [Vision and scope]({{< relref "deep-dives/vision.md" >}}).

## No, really, you should try it out

- [Build the CLI and run your first checks]({{< relref "getting-started.md" >}}).
- [Contribute]({{< relref "contributing/_index.md" >}}), how we plan and
  document changes.

## About me

I'm Abe Gong, a technical founder with a deep love for data/ML/AI and open source. I'm the co-creator of [Great
Expectations](https://greatexpectations.io), the leading open-source tool for
data quality.

I'm fascinated by AI and the way it's changing how we work and collaborate, and
I'm building Katalyst in the open to explore it. I take user feedback
seriously: if you're trying Katalyst, I'd love to hear from you.

More about me: [LinkedIn](https://www.linkedin.com/in/abe-gong-8a77034/) · [twitter/x](https://x.com/AbeGong) · [personal site](https://www.abegong.com/).
