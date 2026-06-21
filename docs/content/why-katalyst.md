+++
title = "Why Katalyst?"
weight = 10
+++

# Why Katalyst?

`katalyst` is a *content consistency layer*, designed for agents that curate persistent memory, wikis, and knowledge bases.

`katalyst` gives you and your agent tools to solve problems like these:

- "My agent takes a long time to find things, and sometimes burns a ton of tokens."
- "I've repeatedly told my agent how to organize content and it still gets it wrong."
- "Sometimes when I go back and look, I discover that my agent has completely lost important information."
- "The content in my knowledge base isn't just text. It also includes metadata that I need to be able to categorize, score, filter, sort, etc."
- "My agent is supposed to store and curate notes for me, but I spend way too much time checking its work."
- "I want to change how I'm storing my data, but migration would be a big pain."

If you want to be confident that your content/data/memory is always in good shape---even when it's maintained by sometimes-sloppy agents and sometimes-sloppy humans---then `katalyst` is for you.

{{% hint info %}}
**New to katalyst?**

[Get started here >>.]({{< relref "getting-started.md" >}}) Install
the CLI, scaffold a `.katalyst/` project, and run your first checks in a few
minutes.
{{% /hint %}}

## Key features

### Catalog the content you already have

`katalyst` comes with tools and skills to {wrong word: inspect} your content, no matter what state it's in today. It can help you (and your agents) figure out what you've got, map out the important concepts, and---if needed---get more organized.

Compared to having an LLM scan every file or write its own bash scripts, this approach can save a ton of tokens. It also lets you take advantage of skills, tools and strategies curated by a community of who've faced similar challenges.

### Define the language and structure that work best for you

Curation always requires shared language and consistent structure. `katalyst` provides tools for declaring structure and rules for your content in your knowledge base.

- *Markdown content* — required sections, naming conventions, templates, etc.
- *File structure* — naming conventions, preferred and required extensions, directory structures, etc.
- *Metadata* — required fields, types, enums, numeric ranges, and full
  JSON Schema validation of frontmatter.
- *Object relationships* — {aspirational: text needed}

### Reshape as needed

As your content evolves, `katalyst` gives you tools to navigate change. {Refactoring or migrating your knowledge base doesn't need to...}

  - *Add or change checks* — add or modify rules, then re-validate in place.
  - *Change the structure of your content* — reshape frontmatter or layout with
    the checks as your guide.
  - *Change your storage layer* — carry the same guarantees from files to a
    database.

## Design principles

### Lightweight, deployable anywhere

You can run `katalyst` as a linter, a CLI, or a server. Use only the infrastructure that you need for your particular use case.

### Model- and backend-agnostic

I'm building `katalyst` to work with a variety of filesystems and databases. It isn't tied to any one data store.

Similarly, you choose which model to use.

### Leans into shared language

Express the same rules in a project's own vocabulary and conventions.

### Built for both humans and agents

Ergonomics matter — especially for agents. An agent should be able to read the rules, find what it needs, and extend them without ceremony:

In development, I take user feedback seriously

  - *Speed* — fast enough to run on every write.
  - *Discoverability* — an agent can find the schemas and structure on its own.
  - *Readability* — rules and content stay legible to humans and agents alike.
  - *Extensibility* — add new checks and rule kinds as needs grow.

## What if structure was light?

Curating your content with the right structure makes it more useful, but it also takes work. Historically, defining the right structure for knowledge was heavy: high-cost, high-risk, and sometimes technically demanding. This was doubly true when changing needs required updates to structure.

As a result, most structured data systems were rigid and hard to change. Most unstructured knowledge bases were either chronically outdated, or very limited in scope.

As AI starts to infuse our work, curating knowledge is going to become even more important---a massive potential unlock for people who want to work more productively and creatively with agents.

What if structure were light---easy to add, easy to maintain, easy to change?

## About me

<!-- TODO: fill in — background, credentials, and motivation for building Katalyst. -->

_Placeholder — to be written._

## Get started

- [Build the CLI and run your first checks]({{< relref "getting-started.md" >}}).
- [Contribute]({{< relref "contributing/_index.md" >}}) — how we plan and
  document changes.
