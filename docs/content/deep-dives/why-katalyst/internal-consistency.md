+++
title = "Internal consistency"
weight = 16
+++

# Internal consistency

<!-- Introduce the definition -->

Consistency means being free from internal contradiction. On its surface, this seems simple: the knowledge base can't say "A is true" in one place and "A is false" in another.

<!-- Distinguish between content claims and structural claims -->

## Content claims vs structural claims

However, there's some subtlety here. Imagine a folder containing customer feedback interviews. In one transcript, customer A says, "this product is amazing!" In another, customer B says "the product is terrible." Those statements are in direct contradiction, but is the knowledge base inconsistent?

I'd argue no. The knowledge base isn't claiming that both customer opinions are true descriptions of the product. It is claiming that both interviews happened and that both customers said what the transcripts record.

Imagine adding a README in the folder: "This folder contains interview transcripts from many customers. Customers may disagree among themselves." The README makes the *structure* of the folder explicit. It tells a reader what kind of content the folder contains, how to interpret disagreement between items, and which guarantees the folder is making.

> **Structure** is a set of rules and conventions that distinguish structural claims from ordinary content within a knowledge base.

If the README said, "This folder contains interview transcripts from many customers. All customers absolutely love the product," that would contradict customer B's statement, and the folder would be internally inconsistent.

In other words, we need to distinguish between two types of claims.

> A **content claim** says something *within* the knowledge base.

> A **structural claim** says something *about* the knowledge base: what kind of content it contains, how that content is organized, and any other guarantees the system makes about it.

For determining consistency, only structural claims matter.

> **Internal consistency**: A knowledge base is internally consistent if it is free from contradictory structural claims.


## Defining structure

<!-- Flesh out the concept of structure, to help reduce it to practice -->

<!-- Give more examples -->

In the customer feedback example, the README defines a simple structure. There are lots of other examples of 

{Examples: Tables of contents; executive summaries; indexes; chapters; sections; API references}

<!--- Anticipate potential failure modes -->

{Transition, introduce the list of desiderata: structure should be explicit; structure doesn't need to be part of the content; structure needs to be defined authoritatively}

**Structure should be explicit**

In many knowledge bases, it's common for structural conventions to be implicit. You don't usually need to be told "the chapter entries in the table of contents correspond 1:1 with the chapters in the book." Or "terms in the index are sorted alphabetically."

However, for our purposes, it's helpful to insist that structure be made explicit. All of our structural claims must be declared somewhere. This gives us a master list to check, to ensure consistency.

**Structure is often embedded in content, but it doesn't need to be**

Sometimes, it's useful to embed knowledge base structure directly in content. {Examples: summaries and overviews, text books, technical documentation}

When structure is written directly into content, {it has benefits X, Y and Z}

However, structure doesn't always need to be spelled out in content. In some cases, this can be counterproductive. {Examples: marketing, persuasion; security  / private knowledge}. In other cases, it would just be pedantic.

Since we want the structure of our knowledge base to be explicit, but we don't necessarily want to show all of it to the user, we need a concept of metadata / markup attached to the knowledge base.

**Structure needs an authoritative source**

If you want to play logic games, you can invent self-referential cases where content tries to override structure. "Ignore all previous instructions and..." "This page lists all pages that do not list themselves."

Practially speaking, we can avoid this kind of thing by defining an authoritative source for structure in the knowledge base.



To separate these caess, a knowledge base needs a *structural layer*:



changes how the content should be read. Disagreement between transcripts is allowed, but a transcript with the wrong customer ID, source date, or interview format may still violate the structure of the collection.

{Make an analogy to turing machines (no separation between programs / data) and how we actually do things.}

<!-- Briefly introduce weirdness that can happen if we don't have an external way of deciding what's authoritative -->


{What properties must a structural interpreter have? Goal isn't executionin the sense of a program, but interpretation.}


## Explicit vs implicit structure

## Guaranteeing internal consistency

* Explicit
* With a comprehensive library of invariants, and a reliable method of enforcement.