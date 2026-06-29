+++
title = "Up-to-dateness"
weight = 18
+++

# Up-to-dateness

Up-to-dateness is the guarantee of external consistency: the state of the content accurately reflects the state of the real world at some point in time. A knowledge base can be internally consistent and complete within its stated scope while still being wrong, because the world changed since the content was last updated.

That makes up-to-dateness different from the other two criteria. It cannot be guaranteed from inside the content alone. It requires contact with an external source of truth: an event stream, a periodic refresh, a source-system query, a human review, or some other verification process. A curated system can record timestamps, sources, freshness windows, and update rules, but the guarantee comes from the process that reconnects the content to the world.

Because curation takes work, there's always some lag between a change in the world and the content that reflects it. As a general rule, less lag is better. Information doesn't need to be perfectly up-to-date in order to be valuable. The important questions are whether the content makes a truthful claim about when it corresponded to the world, and whether the content is updated quickly enough to support valuable decisions.
