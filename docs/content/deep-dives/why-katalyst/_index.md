+++
title = "Why Katalyst?"
weight = 10
bookCollapseSection = true
+++

# Why Katalyst?

In order for agents to become capable of real work, the next frontier seems to revolve around two things:

1. Making operational context more legible to agents.
2. Enabling agents to curate their own memory—individual or shared—in a way that's robust, durable, and efficient.

These problems have several things in common:
* Content that's a mix of text and more structured data. 
* A compute model that's a mix of LLMs and deterministic software.
* The need for humans and agents need to make sense of the same information.
* UI/UX questions that end up being grounded in shared primitives.

I've come to see the two problems as two faces of the same coin. By enabling agents to curate internally consistent, always-up-to-date knowledge bases, I believe we can serve  both needs.

Katalyst is designed to provide the right content primitives and large fraction of the deterministic compute required to solve this problem.

## How this section is organized

This section contains the first-principles reasoning underlying Katalyst's primitives. This isn't necessary if you just want to use the library. It will mostly be useful for those who want a solid, well-grounded perspective on how to build AI knowledge bases.

- [What is curation?]({{< relref "what-is-curation.md" >}}) defines curation and the criteria that make curated information useful.
- [Internal consistency]({{< relref "internal-consistency.md" >}}) explains how a knowledge base decides which contradictions count.
- [Completeness]({{< relref "completeness.md" >}}) covers the scope of information a knowledge base claims to contain.
- [Up-to-dateness]({{< relref "up-to-dateness.md" >}}) describes how a knowledge base stays connected to the world it represents.
- [Progressive operations]({{< relref "progressive-operations.md" >}}) explains how storage backends grow richer as query complexity increases.