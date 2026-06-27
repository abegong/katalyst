+++
title = "What is curation?"
weight = 15
+++

# What is curation?
<!--Motivate the question-->

If AI systems are going to curate content effectively, they need a clear target: what counts as curation, and what good curation should accomplish.

<!-- Provide several specific and diverse examples of curation work; bullet list sentence fragments -->

Let's start from examples. The work of curation shows up in many practical tasks:

- summarizing the useful parts of a conversation thread in a document that people can find later
- grouping related notes so they are easier to browse
- adding dates, owners, tags, or status labels so people know what they are looking at
- rewriting headings so a reader can scan the page before committing to it
- removing duplicate or stale material

<!-- Transition and provide my definition -->

These tasks look different, but they all share the same purpose: making content usable.

> Curation is the act of making content usable.

<!-- Make the analogy to incurring write-time overhead to improve read-time operations in data systems -->

Curation trades work up front for convenience later. An analogy: in data systems, constraints like indexes and schemas add overhead when data is written, in order to make later reads faster and more reliable. Curation applies the same trade to content: extra care when information is written, edited, or organized, in exchange for less work when someone needs to use it.

## Making information usable

Truly first-class curation requires user empathy: putting yourself in the shoes of a user, thinking about their needs, and what will be most helpful to them.

<!-- Provide examples of curation driven by very specific needs. -->

For example, in technical documentation, different readers need different shapes of content. For someone evaluating the project, the most useful content is a landing page or README that explains of what it does and the problems it solves. For a new user, it's an onboarding tutorial that gets them to a working setup. For an experienced user, it might be an API reference with exact details. Each page type curates the same body of knowledge around a different reader need.

Here are some other examples of information curated for specific needs:

- A bug report that includes exact reproduction steps, environment details, expected behavior, actual behavior, logs, screenshots, and why the reporter thinks it matters.
- An account brief helps a customer success manager prepare for a renewal call by gathering usage trends, open risks, support history, and likely expansion opportunities.
- A battle card helps a salesperson prepare for a competitive sales call by summarizing the competitor the prospect already uses and the objections most likely to come up.
- A board packet that condenses financials, customer signals, hiring plans, and risks into the few questions directors need to weigh in on.

Beyond the information itself, curation can also involve the presentation of the information.

- formatting a document so the most important information is visible before the supporting detail
- adding headings, summaries, and tables of contents so readers can scan before reading
- making content searchable through clear titles, stable terminology, tags, and aliases
- choosing UI conventions that match the task, such as filters for comparison, status badges for freshness, or callouts for warnings
- managing information density so a page gives enough context without burying the answer
- exposing affordances for agents, such as stable identifiers, structured metadata, machine-readable links, and clear boundaries between source material and interpretation

<!-- Start to transition from specific to general -->
Curating information to this level of detail is valuable because it saves the reader from reconstructing context at the moment they need it. It is also costly. To do this effectively, the curator must know the audience, anticipate their needs, gather the source material, and shape it for that use. Part of the promise of AI curation is that it should make more of that work possible.

## Universal properties for good curation

<!-- Shift to a discussion of unanticipatable needs -->

However, even if the cost of curation drops significantly, it's often impossible to anticipate every need that every kind of user will have.

* A user might be interested in comparing cases along a new axis. "Which product requests came from customers who are expanding this year rather than customers at risk?"
* Or they might need to answer a question where the answer depends on a unique combination of components. "What guidance applies to a customer using SSO, SCIM, and audit logs, but only on the team plan?"
* Or maybe something fundamental in the environment has changed, and past conclusions need to be reexamined with new foundational assumptions. “Now that we've changed our pricing, which of our old conclusions about customer segmentation still hold?”

<!-- Introduce the three key properties --> 
Happily, there are some properties that are almost always helpful. Even when exact future needs are unknown, content can still be curated toward a few baseline criteria.

This document focuses on three:
* *Internal consistency*: free from internal contradiction
* *Completeness*: covers all the relevant material within some scope
* *Up-to-dateness*: accurately reflects the state of the real world at some point in time

<!-- Explain why they're universal --> 
These properties are powerful because they create a trustworthy substrate for logical reasoning: answering questions, making decisions, and drawing valid conclusions.

They are *universal* because they can support valid reasoning regardless of subject matter. Even without knowing what a body of content contains or how it will be used, it is still a safe bet that well-curated content should have these properties.

As we'll see, these universal properties also impose enough structure to sketch useful technical requirements for AI systems built to curate content. Let's take them one at a time.

## Internal consistency

## Completeness

## Up-to-dateness

Up-to-dateness is the guarantee of external consistency: the state of the content accurately reflects the state of the real world at some point in time. A knowledge base can be internally consistent and complete within its stated scope while still being wrong, because the world changed since the content was last updated.

That makes up-to-dateness different from the other two criteria. It cannot be guaranteed from inside the content alone. It requires contact with an external source of truth: an event stream, a periodic refresh, a source-system query, a human review, or some other verification process. A curated system can record timestamps, sources, freshness windows, and update rules, but the guarantee comes from the process that reconnects the content to the world.

Because curation takes work, there's always some lag between {...}. As a general rule, less lag is better. Information doesn't need to be perfectly up-to-date in order to be valuable. The important questions are whether the content makes a truthful claim about when it corresponded to the world, and whether the content is updated quickly enough to support valuable decisions.

## Curated knowledge bases

<!-- Define the concept of a knowledge base -->

> A knowledge base is a body of information curated so that it is internally consistent, complete within a useful domain, and up to date.
