# Native collection commands — spec

> **Status: planning.** Exploration for exposing each configured collection as
> a top-level CLI command (`katalyst blog-posts list`) instead of requiring the
> generic noun form (`katalyst item list blog-posts`). The design is settled
> (below), including the **name-normalization approach**: strict names by
> default with an optional explicit `command:` alias (Alternative 5). The
> alternatives that led there are kept for the record; what remains are the
> sub-questions in [Open questions](#open-questions).

## Overview

A project declares collections — `contacts`, `blog-posts`, `articles` — under
`.katalyst/collections/`. Today their CRUD-ish operations are reached through
generic noun commands that take the collection as an argument:

```
katalyst item list   blog-posts
katalyst item get    blog-posts/my-first-post
katalyst item add    blog-posts/draft
katalyst item update blog-posts/draft --set status=published
katalyst item delete blog-posts/draft
```

The goal is to let the collection name lead, because the config already tells us
it is a collection:

```
katalyst blog-posts list
katalyst blog-posts get    my-first-post
katalyst blog-posts add    draft
katalyst blog-posts update draft --set status=published
katalyst blog-posts delete draft
```

## Value

The collection is the thing a user thinks in. `katalyst blog-posts list` reads
the way the user already talks about their data; `katalyst item list blog-posts`
makes them route through an implementation noun first. The native form is also
shorter and more guessable, and it makes `katalyst --help` (in a configured
project) advertise *the user's own* collections, not just abstract machinery.

## Settled decisions

These are decided; recorded here so the eventual plan has a foundation.

1. **Verb is `list`** (not `find`) — the native verb set mirrors the existing
   `item` verbs and shares one implementation: `list`, `get`, `add`, `update`,
   `delete`. Native commands are sugar over the same code path, not a fork.
2. **Reserved names error at load time.** A collection whose name collides with
   a built-in *subcommand* is rejected when config loads, with a clear message.
   The reserved set is the bare-token subcommands a user could type in that
   position: our own (`init`, `check`, `fix`, `item`, `schema`, `collection`)
   plus Cobra's auto-generated `help` and `completion`. **Global flags are *not*
   reserved** — they are always written with a leading `-`/`--` (`katalyst
   --help`), so Cobra never confuses a bare command token (`katalyst help`) for
   a flag; the lexical shape disambiguates them. The collision risk is purely
   between subcommands, which share the bare-token slot. This is a new
   validation (today there is none; see below).
3. **The custom command names are toggleable** via a boolean in
   `.katalyst/config.yaml`:

   ```yaml
   cli:
     includeCustomCommandNames: true   # default
   ```

   - `true` *(default)* — each collection is exposed as a top-level command
     (`katalyst blog-posts list`) **in addition to** the generic
     `item …`/`collection …` forms. The custom names are additive sugar; the
     generic forms remain the stable, scriptable surface.
   - `false` — no custom commands; only the generic `item`/`collection` forms
     (today's behavior; the escape hatch).

   The generic commands are *always* present — the toggle only controls whether
   the per-collection names are layered on top. (This drops the earlier
   third "hide the generics" mode; if a real need for it shows up, it can come
   back as a separate setting.)
4. **Outside a configured project**, no native commands are registered, and the
   CLI shows a generic note that it is running outside a configured katalyst
   directory.
5. **Valid-name pattern is its own definition** — *not* a reuse of the
   `filesystem_filename_kebab_case` check. It is **minimally restrictive while
   still yielding a usable CLI token**: `^[A-Za-z0-9][A-Za-z0-9._-]*$` (must
   start with a letter or digit; thereafter letters, digits, `.`, `_`, `-`).
   The exclusions are exactly what breaks at the command line — a leading `-`
   (parsed as a flag), whitespace and shell metacharacters (`/ * ? | & ; $ ( )`
   …, which force quoting), and `/` in particular (the selector separator in
   `<collection>/<item>`). Mixed case and underscores are **allowed** — unlike
   the stricter kebab-only filename rule — because they work fine as tokens; we
   are not imposing a house style here, only ruling out names that won't work.
   Non-ASCII is excluded for shell/locale safety. See the open question on case
   for the one residual footgun.
6. **The `command:` alias ships in v1**, not as a fast-follow. The slugify
   suggestion in the load error points straight at it, so the strict gate and
   its escape hatch arrive together.
7. **Custom commands render in their own `--help` group**, a "Collections"
   section distinct from the built-in commands, so a project's data-shaped
   commands stay visually separate from tool machinery and scale when there are
   many. (Cobra command groups.)

### Cobra feasibility (settled in principle)

Cobra commands are ordinary structs added via `AddCommand`; nothing requires
static registration. `NewRootCmd()` will call `config.Load(cwd)` *before*
building the tree, and for each discovered collection synthesize a
`*cobra.Command{Use: <token>}` with the five verb subcommands, then
`root.AddCommand` it. By the time `Execute()` runs, the collection commands are
first-class — they appear in `--help` and shell completion for free. Config-load
failures during this step must degrade gracefully (fall back to the static
commands; never abort startup), per decision 4.

---

## Open question: name normalization

**A collection's name must become a CLI command token.** Tokens want to be
predictable, typeable, stable, scriptable, and free of collisions. Collection
names today are not constrained to look like that.

### What a name is today

A collection's `Name` is **either** the stem of its file in
`.katalyst/collections/` (convention mode) **or** the key in the
`collections.defs` map (explicit mode). `internal/config/config.go` performs
**no validation** on it — whatever the filesystem or the YAML key says becomes
the name verbatim. That same string is *already* used as a CLI token in the
selector grammar (`katalyst item list <collection>`,
`katalyst check <collection>/<item>`). So a name that is awkward as a command
token is *already* awkward today; native commands don't introduce the problem,
they raise its stakes.

### The example set

The alternatives below are all evaluated against the same collections, chosen to
hit the edge cases:

| Source (stem or key) | Note |
|---|---|
| `contacts`          | already a clean token |
| `blog-posts`        | already a clean token (kebab) |
| `blogPosts`         | camelCase |
| `my_notes`          | snake_case |
| `Reading List`      | capitals + space |
| `2024`              | leading digit, all numeric |
| `item`              | collides with a built-in command |
| `café`              | non-ASCII |

The central tension across every option: **one identity vs. low friction.** Do
we keep a single string that is the name *and* the selector token *and* the
command token *and* the config key — or do we let the on-CLI token diverge from
the stored name to avoid forcing users to rename things?

---

### Alternative 1 — Strict validation (slug-only names)

Require every collection name to already be a valid command token —
kebab-case, `^[a-z0-9]+(-[a-z0-9]+)*$` — and reject anything else when config
loads. No transformation ever happens. `name == selector token == command
token == config identity`.

Worked examples:

| Source | Result |
|---|---|
| `contacts`     | ✓ `katalyst contacts list` |
| `blog-posts`   | ✓ `katalyst blog-posts list` |
| `blogPosts`    | ✗ load error: *collection name `blogPosts` must be kebab-case (lowercase letters, digits, hyphens)* |
| `my_notes`     | ✗ load error (underscore) |
| `Reading List` | ✗ load error (capital, space) |
| `2024`         | ✓ `katalyst 2024 list` |
| `item`         | ✗ load error (reserved — decision 2) |
| `café`         | ✗ load error (non-ASCII) |

- **Pros:** one identity everywhere — trivial mental model, nothing to reverse-
  map, no ambiguity, docs and scripts read the same as the CLI. The validation
  doubles as the reserved-word check from decision 2. Consistent with the
  selector grammar, which uses the same string.
- **Cons:** imposes a naming policy. Existing collections with `_`, camelCase,
  capitals, or non-ASCII must be **renamed** — and renaming the file/key also
  changes selectors and any scripts that reference them. Hard line for i18n
  names. The error is a wall, not a ramp: the tool says "no" without fixing it.

---

### Alternative 2 — Forgiving auto-slug (derive token from name)

Keep the name verbatim as the stored identity, but *derive* the command token by
slugifying: lowercase, replace spaces/underscores with hyphens, drop other
invalid characters, collapse repeats. The slug is what appears on the CLI; the
original name stays canonical in config and selectors.

Worked examples:

| Source | Token | `katalyst … list` |
|---|---|---|
| `contacts`     | `contacts`     | `katalyst contacts list` |
| `blog-posts`   | `blog-posts`   | `katalyst blog-posts list` |
| `blogPosts`    | `blogposts` *(or* `blog-posts` *if we split camel — a choice)* | `katalyst blogposts list` |
| `my_notes`     | `my-notes`     | `katalyst my-notes list` |
| `Reading List` | `reading-list` | `katalyst reading-list list` |
| `2024`         | `2024`         | `katalyst 2024 list` |
| `item`         | `item`         | ✗ still must error (reserved) |
| `café`         | `caf` *or* `cafe` *(strip vs transliterate — a choice)* | surprising either way |

- **Pros:** accepts any existing name; zero forced renames; "just works" on
  first run.
- **Cons:** **two identities.** The CLI token (`reading-list`) and the canonical
  name (`Reading List`) diverge, and the selector grammar still uses the
  canonical name — so `katalyst reading-list list` works but
  `katalyst item list reading-list` does **not** (it wants `Reading List`).
  That split is the real cost. Plus: **slug collisions** — `blogPosts`,
  `blog_posts`, and `blog-posts` can all slug to the same token, so we need a
  collision error *anyway*. camelCase and non-ASCII handling are full of
  surprising judgment calls (`APIKeys` → `api-keys`? `apikeys`? `café` → `cafe`?
  `caf`?). Requires a token→collection reverse lookup at dispatch.

---

### Alternative 3 — Validate-and-normalize on creation

Runtime stays strict (Alternative 1), but the tool *normalizes for the user at
the moment a collection is introduced* — e.g. a future `katalyst collection add
"Reading List"` writes `reading-list.yaml`. Hand-edited/hand-dropped files are
still subject to strict load validation.

- **Pros:** strict, single-identity runtime *plus* a helpful on-ramp — users who
  go through the tool never hit the wall.
- **Cons:** today collections are created by **dropping a file**, not via a
  command (per `project-layout-spec.md`, `init` deliberately scaffolds
  nothing). There is no creation command to hook yet, so this mostly reduces to
  Alternative 1 until that command exists. More surface to build; does nothing
  for hand-authored files.

---

### Alternative 4 — Explicit alias field (opt-in command name)

Don't transform. Keep names loosely validated (only enough for selectors). Add
an optional per-collection field that names the command explicitly:

```yaml
# .katalyst/collections/blogPosts.yaml
command: posts        # optional native command token
path: notes/blog
schema: post
```

If `command` is present and valid, that is the native token. If absent, the
collection gets a native command **only if its name is already a valid token**
(Alternative 1's rule); otherwise it gets **no native command** (with a hint)
but stays usable via `item list <name>`.

Worked examples:

| Source | `command:` | Result |
|---|---|---|
| `blog-posts`   | —        | `katalyst blog-posts list` |
| `blogPosts`    | —        | no native command (hint: *add `command:` to enable*); `item list blogPosts` still works |
| `blogPosts`    | `posts`  | `katalyst posts list` |
| `Reading List` | `reading`| `katalyst reading list` |
| `item`         | —        | error (reserved) |

- **Pros:** never forces a rename; native commands are opt-in and **explicit**,
  not magically derived; the alias can be *nicer* than the name (`posts` beats
  `blog-posts`); cleanly decouples CLI ergonomics from storage identity *on
  purpose*.
- **Cons:** more config to learn. Reintroduces two names — but intentionally and
  visibly, not silently. Collision checking must now span aliases **and** names
  **and** reserved words. "Why doesn't my collection show up as a command?"
  becomes a discoverability/support issue.

---

### Alternative 5 — Hybrid: strict default + alias escape hatch

Alternative 1 is the default and the common path: names must be clean tokens,
validated at load, one identity. Alternative 4's optional `command:` field
exists as the escape hatch for the genuine cases — a constrained legacy name, or
a deliberately different display vs. CLI name. Most projects never touch the
field and live in the single-identity world; the field is there when reality
doesn't fit.

- **Pros:** keeps the simple, single-identity model for the 95% case while
  leaving a sanctioned exit for the 5%. The escape hatch is explicit, so it
  doesn't carry Alternative 2's silent-divergence surprise.
- **Cons:** two code paths to build and test (validated-name path + alias path),
  and the divergence cost still exists *when the hatch is used* (selector vs.
  command token differ for aliased collections) — just opt-in rather than
  default.

---

### Comparison

| | Single identity | Forced renames | Silent divergence | Collision handling | Build cost |
|---|---|---|---|---|---|
| **1 Strict**        | ✅ always      | ⚠️ yes        | ✅ none          | folds into validation | 🟢 low |
| **2 Auto-slug**     | ❌ name≠token  | ✅ none       | ❌ name vs token | ⚠️ slug collisions, still must error | 🟡 medium |
| **3 Normalize-on-create** | ✅ at runtime | ✅ via tool | ✅ none      | folds into validation | 🔴 high (needs create cmd) |
| **4 Alias field**   | ⚠️ opt-in two  | ✅ none       | ✅ explicit only | spans names+aliases | 🟡 medium |
| **5 Hybrid (1+4)**  | ✅ by default  | ⚠️ default yes / hatch no | ✅ explicit only | validation + alias | 🟡 medium |

### Decision: Alternative 5, with slug-powered suggestions

**We take Alternative 5 — strict validation by default, with an optional,
always-explicit `command:` alias as the escape hatch.** The slug logic from
Alternative 2 is reused, but **only to *suggest* a name, never to apply one**.

The two pieces:

1. **Strict by default (Alt 1).** A collection name must already be a valid
   command token or config load fails. This keeps the single identity — name =
   selector token = command token = config key — and folds the reserved-word
   check (decision 2) into the same gate.
2. **Explicit alias as the hatch (Alt 4).** A collection may set `command:`
   to give its native command a different token than its name. This is the
   *only* way the on-CLI token ever differs from the name, and it is always
   written by hand, so the divergence is visible and intentional — never the
   silent name-vs-token split that sank Alternative 2.

**Slug logic earns its keep in the error message, not in dispatch.** When a name
fails strict validation, we run the Alternative 2 slugify on it to compute a
*suggested* token and put it in the error:

```
collection "Reading List" is not a valid command name (must be kebab-case:
lowercase letters, digits, hyphens). Rename the file to "reading-list.yaml",
or keep the name and add `command: reading-list` to the collection.
```

So the user gets the convenience of auto-slug (they don't have to invent the
token) without the cost (nothing diverges behind their back — they either rename
to the suggestion or paste it into an explicit `command:`). Auto-slug becomes a
*recommendation engine*, not a runtime transform.

Why this over the others: the collection name is *already* a CLI token in the
selector grammar, so pure auto-slug (Alt 2) would permanently split the selector
token from the command token — the exact two-identities confusion the rest of
the model avoids. Strict-by-default keeps one identity for the common case and
is the least code; the explicit alias covers the genuine outliers (constrained
legacy names, a deliberately different display vs. CLI name) without reopening
the silent-divergence door. We are *adding* validation to a surface that has
none today, so the strict gate ships with the suggestion above to make the new
constraint a ramp rather than a wall.

## Open questions

The valid-name pattern, alias timing, the reserved set, and help grouping are
resolved and folded into [Settled decisions](#settled-decisions) (5, 6, 2, 7).
What remains:

### 1. The config key name

**Context.** Decision 3 settled the *shape* — a boolean under `cli:`, default
`true`, that controls whether each collection is also exposed as a top-level
command. What's unsettled is the **key's name**. It's public config surface, so
renaming it after release is a breaking change; worth feeling out now. The
current draft, `includeCustomCommandNames`, also drifts from the prose, which
calls these "collection commands" — so picking the key is partly picking the
**term** we use everywhere (config, `--help` group heading, docs).

**Candidates, in a sentence and in config.** Read each as the line a user writes
and the way we'd describe it in docs:

- **`includeCustomCommandNames`** — *"Set `includeCustomCommandNames: false` to
  turn off the per-collection commands."* Accurate but wordy, and "custom" is
  vague — custom how? It doesn't say *collections*.

  ```yaml
  cli:
    includeCustomCommandNames: true
  ```

- **`includeCollectionCommands`** — *"Each collection becomes a command unless
  you set `includeCollectionCommands: false`."* Says exactly what it does;
  matches the "Collections" help group and the prose. Slightly long.

  ```yaml
  cli:
    includeCollectionCommands: true
  ```

- **`collectionCommands`** — *"`collectionCommands: false` disables
  `katalyst blog-posts list`."* Shortest; reads cleanly as a feature name. The
  boolean-ness is implicit, which some readers dislike for a `true/false` key.

  ```yaml
  cli:
    collectionCommands: true
  ```

- **`collectionShortcuts`** — *"Turn off `collectionShortcuts` to require the
  generic `item …` form."* Frames them honestly as sugar over the generic
  commands; "shortcut" reads friendly. Slightly undersells them (they're
  first-class commands, not just aliases).

  ```yaml
  cli:
    collectionShortcuts: true
  ```

- **`exposeCollectionsAsCommands`** — *"`exposeCollectionsAsCommands: false`
  hides them."* Most self-documenting; verb-y and the longest. Reads more like a
  sentence than a setting.

  ```yaml
  cli:
    exposeCollectionsAsCommands: true
  ```

**Lean.** `includeCollectionCommands` or `collectionCommands` — both name the
thing precisely ("collection") and let the whole doc drop "custom/native" in
favor of one word. Pick one and the prose + help heading follow.

### 2. Forward-compat for future built-ins (minor)

**Context.** The reserved set (decision 2) covers *today's* subcommands. If we
later add a new built-in — say `katalyst sync` — and some project already has a
`sync` collection, the new built-in would collide with (and could shadow) their
command. This is the one residual collision risk strict validation can't catch
at authoring time, because the conflicting name doesn't exist yet.

**Choices & tradeoffs.**

- **Accept the risk (lean).** Adding a built-in is *our* deliberate act; we
  check the name against common collection names and pick non-colliding verbs,
  and the load-time error catches it for any project that does collide (their
  collection name simply joins the reserved set on upgrade, with a clear
  message). Zero cost now. Cost: a project can be forced to rename on a katalyst
  upgrade — a real but rare and loud breakage.
- **Reserve a namespace now.** E.g. promise built-ins never collide by routing
  future ones under a prefix, or publish a reserved-word list. Removes the
  upgrade-breakage risk entirely. Cost: constrains our own naming forever, or
  burdens users with a reserved list that does nothing visible today.

### 3. Case sensitivity (minor)

**Context.** The decided pattern (decision 5) allows mixed case, so `Contacts`
and `contacts` are both valid names — and Cobra treats them as **distinct**
commands. On a case-insensitive filesystem the *files* `Contacts.yaml` and
`contacts.yaml` already collide (caught by the existing duplicate-stem check),
but two collections differing only in case from an explicit `defs` map would
produce two near-identical commands — a footgun.

**Choices & tradeoffs.**

- **Allow mixed case, add no special handling (lean — matches "minimally
  restrictive").** Simplest; consistent with the pattern. Cost: leaves the
  case-twin footgun for the rare explicit-defs project.
- **Lower-case-fold for collision detection.** Treat names that differ only in
  case as colliding and error at load. Closes the footgun. Cost: a little extra
  validation, and it quietly forbids a (weird but legal) name pair.
