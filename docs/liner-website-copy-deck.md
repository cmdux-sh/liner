# Liner website copy deck

Final copy deck for website implementation. The factual review and final Humanizer pass are complete. This artifact changes no website source.

## Brief and application

The reader already uses AI assistants and local developer tools, but has never used Liner. The page should explain the local research workflow, show the saved files, and make the first command clear. The docs should let that reader complete the same workflow without guessing at product terms.

The homepage introduces the tool, inputs, local folder, compiled research, and instruction files in that order. Docs retain exact interface labels after explaining what each step does. Existing commands and file names remain exact unless this deck explicitly relocates or changes them. Examples illustrate possible work; they are not a shipped project library. The examples promise research guidance within the evidence; App Store approval and output quality remain outside those claims.

Each source mapping below is absolute. Tables map individual homepage and shared elements. Each docs section supplies the complete proposed MDX body and metadata; imports are omitted because they are implementation details. Fenced YAML at each route start becomes frontmatter. Fenced MDX file trees become components; unwrap those fences when applying the page. Embedded command snippets, file trees, YAML examples, command tables, and key bindings are supplied in full. Corrections against the current product are included below. Headings use sentence case. The changelog uses complete public summaries of historical releases and keeps internal release-engineering details in the source changelog.

## Homepage

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/pages/index.astro`. Apply rows in DOM order. Repeated marquee strings and duplicated install controls use the same wording at every instance.

### Hero

| Element | Proposed copy |
| --- | --- |
| Eyebrow | Local research toolkit |
| H1, both lines | Build reusable / AI context. |
| Body | Liner is a local CLI and terminal UI for turning focused research into files your AI assistant can use. Bring URLs, YouTube videos, PDFs, local files, or pasted text. Choose the evidence, review the research summary, and save it with instructions for the next session. |
| Version chip | `V. {releaseVersion}` unchanged; source remains `VERSION`. |
| Version status | Current version |
| Visual panel status | Research and instructions |
| Install eyebrow | Start here |
| Install headline | Open the terminal UI / with one command. |
| Command | `npx linersh` unchanged |
| Install support line | Free · MIT licensed · macOS, Linux & Windows x64 |
| GitHub link | View on GitHub |
| Decorative version | `/// v{releaseVersion}` unchanged |
| Marquee 1 | Sources selected /// task defined |
| Marquee 2 | Research compiled /// Markdown ready |
| Marquee 3 | Local tool /// no telemetry |
| Marquee 4 | Research and instructions /// MIXTAPE.md · LINER.md · SKILL.md |

### From task to files

| Element | Proposed copy |
| --- | --- |
| Eyebrow | From task to files |
| Heading | Keep the research behind the answer. |
| Intro | A saved folder keeps your selected sources, research summary, and AI instructions together. Open it in a new AI session to work from the same evidence and boundaries. Liner calls this folder a Project. |
| Example label | Example task |
| Example request | Help my AI write copy for a mental-health app using the App Store review rules. |
| Step 1 title | Clarify the job |
| Step 1 body | Name the situation, desired output, and claims the AI should avoid. |
| Step 2 title | Choose the evidence |
| Step 2 body | Add review rules, privacy guidance, claim examples, and rejection notes. Review the proposed sources before saving them. |
| Step 3 title | Approve the synthesis |
| Step 3 body | Read the summary of the accepted evidence. Edit it if needed, then approve it before the research is compiled. |
| File-tree title | `app-review-health-claims/` unchanged |
| File-tree status | Local files |
| File-tree contents | All existing names and indentation unchanged, including `liner.yaml`, `LINER.md`, `SKILL.md`, and the three example source filenames. |
| MIXTAPE.md card | Compiled research with your synthesis and links to the saved source files. |
| LINER.md card | Instructions for using the evidence, following the method, and respecting its limits. |
| SKILL.md card | The Project's entry point for an AI assistant that can read local files. |

### Examples

| Element | Proposed copy |
| --- | --- |
| Eyebrow | Focused work |
| Heading | Build research for a specific task. |
| Intro | Save a Project when you need to choose the evidence carefully and use it again in another session. You could build research for tasks like these. |
| Card 1 role | App developer |
| Card 1 task | Write mental-health app copy grounded in App Store review rules. |
| Card 1 input | Review rules, privacy guidance, rejected claims |
| Card 1 output | Evidence-backed writing instructions |
| Card 2 role | Product designer |
| Card 2 task | Turn visual references into a web art-direction brief. |
| Card 2 input | Reference notes, layout patterns, web constraints |
| Card 2 output | A direction with explicit design boundaries |
| Card 3 role | Engineer |
| Card 3 task | Plan an API migration using framework docs and failure reports. |
| Card 3 input | Official guides, breaking changes, postmortems |
| Card 3 output | A migration plan and review checklist |
| All input/output labels | Input / Output unchanged |

### Workflow

| Element | Proposed copy |
| --- | --- |
| Eyebrow | From research to reuse |
| Heading | Four steps to a folder your AI can use. |
| Step 01 title | Frame |
| Step 01 body | Define one task and the output you need. The terminal UI helps clarify its situation and boundaries before research begins. |
| Step 01 proof | A specific job |
| Step 02 title | Gather |
| Step 02 body | Add material you already have, then build and review the research with your configured Claude Code or Codex runner. Choose which proposed sources to keep. |
| Step 02 proof | Selected evidence |
| Step 03 title | Review |
| Step 03 body | Approve the synthesis separately from the source selection. Compile then fetches the source content and writes the research packet. |
| Step 03 proof | Research compiled |
| Step 04 title | Use |
| Step 04 body | Create the AI instruction files in the terminal UI. Give the whole folder to a file-reading assistant, or paste MIXTAPE.md into a conversation. |
| Step 04 proof | Instructions written |
| Step numbers | 01 / 02 / 03 / 04 unchanged |

### Optional agent maintenance

| Element | Proposed copy |
| --- | --- |
| Eyebrow | Optional agent maintenance |
| Heading | Update the Project through Codex or Claude Code. |
| Body | Install the Maintenance Adapter to let a coding assistant request changes through Liner's CLI. The CLI checks Project identity, previews the file changes, requires approval for higher-risk operations, and records the result. |
| Command | `liner adapters install codex --yes` unchanged |
| Guide link | Read the maintenance guide |
| Step 01 title | Request a change |
| Step 01 body | Add a source, rename or move a Project, or move an unused Project to Liner Trash. |
| Step 02 title | Review the plan |
| Step 02 body | A Project Change Set shows the exact effects. Operations that require approval wait for your decision on that preview. |
| Step 03 title | Keep the record |
| Step 03 body | A Change Receipt records what was applied. Project deletion moves the folder to recoverable Liner Trash and preserves its files. |
| Step numbers | 01 / 02 / 03 unchanged |

### Fit and trust

| Element | Proposed copy |
| --- | --- |
| Eyebrow | The right size of work |
| Heading | Keep research ready for another session. |
| Saved-folder column label | Keep a Project |
| Saved-folder item 1 | Source choice changes the quality of the output. |
| Saved-folder item 2 | Another AI session needs the same evidence and boundaries. |
| Saved-folder item 3 | The task is specific enough to define and useful enough to repeat. |
| Chat column label | A chat is enough |
| Chat item 1 | You need one quick answer. |
| Chat item 2 | The evidence changes hour by hour. |
| Chat item 3 | The data belongs in a system outside a local folder. |
| Trust card 1 title | Inspectable files |
| Trust card 1 body | Read the sources, synthesis, and instructions in Markdown. Inspect the recipe and status in YAML. |
| Trust card 2 title | Your local tools |
| Trust card 2 body | The installed tool has no telemetry. AI-assisted research uses your configured Codex CLI or Claude Code runner and its account limits or charges. Files are saved locally; material read by the runner is processed under your provider's terms. Compile and instruction-file creation make no model calls. |
| Trust card 3 title | Check before sharing |
| Trust card 3 body | A .mixtape archive can include local source files and saved content. Review its contents before sending or publishing it. |

### Closing install and footer

| Element | Proposed copy |
| --- | --- |
| Install eyebrow | Start here |
| Install heading | Build context / you can use again. |
| Command | `npx linersh` unchanged |
| Docs link | Read the docs |
| Install facts, in order | macOS, Linux & Windows x64 / Free and MIT licensed / Local tool, no telemetry |
| Footer marquee 1 | `/// liner v{releaseVersion}` unchanged |
| Footer marquee 2 | /// local tool · no telemetry |
| Footer marquee 3 | /// free · MIT licensed · open source |
| Footer marquee 4 | /// build reusable AI context |
| Footer logo alt | Liner unchanged |
| Footer description | Focused research, saved sources, and instructions your AI can reuse. Free and MIT licensed. |
| Footer nav | Docs / Changelog / GitHub unchanged |
| Footer nav accessible name | Footer unchanged |
| Copyright | © 2026 Liner unchanged |
| Sequence marker | /// 007 · END OF SEQUENCE unchanged |

## Shared metadata, navigation, and messages

| Source | Element | Proposed copy |
| --- | --- | --- |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/layouts/BaseLayout.astro` | Default title | Liner: Local research for reusable AI context |
| Same | Default description | Turn focused sources into a local research folder with Markdown evidence and AI instructions. Start with Liner's CLI and terminal UI. |
| Same | Open Graph image alt | Liner. Focused research and instructions your AI can reuse. |
| Same | Website name and app title | Liner unchanged |
| Same | Open Graph and Twitter title/description | Inherit the route title and description above; no duplicate wording. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/components/SiteHeader.astro` | Brand accessible name; logo alt; nav accessible name | Liner home / Liner / Primary unchanged |
| Same | All links | Docs / Changelog / GitHub unchanged |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/astro.config.mjs` | Docs site title, logo alt, social link | Liner Docs / Liner / GitHub unchanged |
| Same | Sidebar group 1 and links | Start here / Overview / Install / Build a Project / Maintain with an Agent |
| Same | Sidebar group 2 and links | Concepts / Projects and Mixtapes / Tape format / Sharing |
| Same | Sidebar group 3 and links | Reference / CLI commands / Terminal UI / Troubleshooting / Changelog |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/components/CommandSnippet.astro` | Command pre accessible name | Command to run unchanged |
| Same | Copy button text and success | Copy / Copied unchanged |
| Same | Copy accessible name | `Copy command: {command}` unchanged |
| Same and homepage copy buttons | Install button accessible name | Copy the install command unchanged |
| Same and homepage copy buttons | Pending copy button label | Copying |
| Same and homepage copy buttons | Successful copy status | Command copied. |
| Same and homepage copy buttons | Failed copy status | Copy failed. Select the command and copy it manually. |
| Same and homepage copy buttons | Failed clipboard button label | Retry |
| Homepage visual panel | Motion control | Pause animation / Resume animation |
| Homepage visual panel | Motion control when the system requests reduced motion | Reduced motion |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/components/FileTree.astro` | Default label | Project folder |
| Same | Accessible name | `{root} file tree` unchanged |
| Docs overrides | File-tree labels | Compile output / Archive output |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/components/Icon.astro` | Icons | No authored visible text; decorative use stays hidden from assistive technology. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/components/starlight/Header.astro` | Search and header | Uses the shared header and generated search strings below; no other authored copy. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content.config.ts` | Schema | No public prose. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/styles/global.css` and `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/styles/starlight.css` | CSS | No authored public prose. |

Clipboard success must appear only after a successful write. The failure wording defines the proposed interface state for implementation.

### Search, 404, and generated docs chrome

The current English strings come from installed Starlight translations and Pagefind. Retain stock wording where it is already clear. Configure changes through project-owned translation overrides or components; do not edit dependencies.

| Element | Proposed copy |
| --- | --- |
| Search trigger, dialog accessible name, input placeholder | Search docs |
| Search cancel | Cancel unchanged |
| Search shortcut | Ctrl K / ⌘ K unchanged |
| Development search notice | Search is available in the production preview. Build and preview the site to test it. |
| 404 title | Page not found |
| 404 body | This page couldn't be found. Check the address or search the docs. |
| 404 navigation, if added | Liner home / Read the docs |
| Skip link | Skip to content unchanged |
| Mobile menu accessible name | Menu unchanged |
| Sidebar accessible name | Main unchanged |
| Table of contents | On this page / Overview unchanged |
| Pagination | Previous / Next unchanged |
| Last-updated label | Last updated unchanged |
| Edit link | Edit page unchanged |
| Starlight credit | Built with Starlight unchanged |
| Section anchor label | `Section titled “{{title}}”` unchanged |
| Aside labels | Note / Tip / Caution / Danger unchanged |
| Directory label | Directory unchanged |
| Language selector and untranslated notice | Select language / This content is not available in your language yet. unchanged |
| Theme labels, if rendered | Select theme / Dark / Light / Auto unchanged |
| Draft notice, development only | This content is a draft and will not be included in production builds. unchanged |
| Pagefind clear button | Clear unchanged |
| Pagefind load-more button | Load more results unchanged |
| Pagefind input accessible name | Search docs |
| Pagefind filters | Filters unchanged |
| Pagefind zero results | `No results for [SEARCH_TERM]` unchanged |
| Pagefind multiple results | `[COUNT] results for [SEARCH_TERM]` unchanged |
| Pagefind single result | `[COUNT] result for [SEARCH_TERM]` unchanged |
| Pagefind searching status | Searching for [SEARCH_TERM]... unchanged |
| Pagefind total-result messages | `No results` / `[COUNT] result` / `[COUNT] results` unchanged |
| Pagefind alternate result | `No results for [SEARCH_TERM]. Showing results for [DIFFERENT_TERM] instead` unchanged |
| Pagefind suggested searches | `No results for [SEARCH_TERM]. Try one of the following searches:` unchanged |
| Pagefind result-list accessible name | Search results unchanged |
| Pagefind keyboard hints | navigate / select / clear / close / search unchanged |
| Pagefind error | Search failed unchanged |
| Pagefind filter selection counts | `[COUNT] selected` unchanged for singular and plural |
| Pagefind input hint | Results will appear as you type unchanged |
| Pagefind loading state | Loading unchanged |

## Docs routes

The ten sections that follow are complete proposed page bodies. Retained command/reference blocks keep exact syntax except the stated factual corrections. The install page separates first use from optional browser setup and uninstall. The destructive release-testing reset is removed from public copy; the internal release/development workflow remains its reference and does not belong in the public first-use flow.

### https://www.liner.sh/docs/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/index.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Liner Docs
description: Install Liner, turn focused sources into a local research folder, and write instructions your AI assistant can reuse.
```


Liner is a local CLI and terminal UI for turning focused research into files your AI assistant can reuse.

Bring URLs, YouTube videos, PDFs, local files, or pasted text. The terminal UI helps you choose the evidence, review a research summary called a synthesis, compile the research into Markdown, and write AI instructions. AI-assisted research uses your installed, authenticated Codex CLI or Claude Code runner and that account's limits or charges. Files are saved locally; material read by the runner is processed under your provider's terms. Compile and instruction-file creation make no model calls.

The saved folder is a **Liner Project**. Its research packet is called a **Mixtape**. The packet lives under `mixtape/`; the Project's instruction and status files live at the root.

<CommandSnippet command="npx linersh" />

## Focused work

Use it when source choice changes the quality of the AI's work. It fits focused jobs like reviewing a native iOS flow, comparing API migration patterns, preparing a hiring rubric, or grounding a strategy review in a specific body of evidence.

Use Liner to do the following.

- frame the task and desired output
- gather and evaluate source material
- write a summary the AI should inherit
- fetch source content into local Markdown files
- create instruction files for future AI sessions
- review and record changes to Projects and Sources
- package the folder for reuse or sharing

## The creation workflow

1. Create a Project and describe the task using the prompted Job Story form.
2. Add material you already have. Review the sources and save your selection.
3. Use **Clarify Job** to make the situation, desired outcome, and boundaries explicit.
4. Run **Build Corpus** with your configured local AI runner. A corpus is the body of research for the task.
5. Review the draft sources and accept the ones you want to keep.
6. Review the synthesis separately. Approve it when it accurately represents the accepted evidence, or edit it first.
7. Liner starts **Compile** after synthesis approval and writes the Mixtape.
8. Run **Create Operating Layer** to write the AI instruction files.
9. Give the Project folder to an AI tool that can read files, or paste `mixtape/MIXTAPE.md` into a conversation.

For optional agent-assisted updates, see [Maintain with an Agent](/docs/maintenance/).

## Terms used in these docs

| Term | Meaning |
| --- | --- |
| Tape | The YAML recipe in `mixtape/tape.yaml`. It lists metadata and source entries. |
| Mixtape | The research packet inside the Project. It lives under `mixtape/` and contains the tape, synthesis, extracted sources, working notes, and compiled output. |
| Liner Project | The full working folder. It contains the Mixtape plus AI instructions, a root skill file, status, and maintenance metadata. |
| Synthesis | Your summary and interpretation of the selected evidence. It appears at the top of `MIXTAPE.md`. The interface calls the person choosing and approving the research the curator. |
| Sources | Selected web pages, YouTube transcripts, local files, pasted text, or skill references. Liner extracts their content into Markdown. |
| Project and Source IDs | Immutable identifiers assigned by Liner. Renaming or moving a Project keeps its Project ID. Editing Source metadata or its locator keeps its Source ID. Replacing the meaning of a Source creates a successor ID and records lineage. |
| Instruction files | `LINER.md`, root `SKILL.md`, and `liner.yaml` tell future AI sessions how to use the saved research and what state to trust. The TUI action that creates them is called Create Operating Layer. |
| Optional Maintenance Adapter | A Codex or Claude skill that routes Project and Source changes through the installed CLI. It previews a Project Change Set, waits for Curator approval when required, and leaves a Change Receipt after apply. It works alongside the Project Skill. |
| Project Snapshot, Change Set, and Change Receipt | A Snapshot records current state without writing. A Change Set is an exact write-free plan for one Project revision. A Receipt is the durable redacted record of the applied plan. |

## Next

Start with [Install](/docs/install/), walk through [Build a Project](/docs/build-a-mixtape/), then see [Maintain a Project with an Agent](/docs/maintenance/) when you want Codex or Claude to update it through reviewed changes.

<!-- End proposed route content -->

### https://www.liner.sh/docs/install/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/install.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Install
description: Start Liner with npx or install the CLI and terminal UI globally with npm.
```

You need Node.js 18 or newer. The `linersh` npm package includes the terminal UI and a matching CLI binary, so normal use needs no Python installation.

## Open the terminal UI

<CommandSnippet command="npx linersh" />

For AI-assisted research, install and authenticate Codex CLI or Claude Code first. The research runner uses that account's usage limits or charges and processes the material it reads under your provider's terms. Follow first-run setup to choose the runner and a local Project workspace, then create a Project for one focused task. The optional Maintenance Adapter described below is a separate feature for updating existing Projects.

Compile and instruction-file creation run locally without new model calls.

Check the package version with the same one-shot form.

<CommandSnippet command="npx --yes linersh@latest --version" />

Continue with [Build a Project](/docs/build-a-mixtape/).

## Install globally

<CommandSnippet command={`npm install -g linersh
liner`} />

Running `liner` with no arguments opens the terminal UI. A subcommand runs the bundled CLI operation. For one-shot use, replace `liner <command>` with `npx --yes linersh@latest <command>`.

## Platform support

Version `1.1.1` supports these native npm platform packages. The main `linersh` package declares them as optional dependencies, and npm selects the matching OS and CPU package during install.

| System | npm platform package |
| --- | --- |
| macOS Apple silicon | `linersh-darwin-arm64` |
| macOS Intel | `linersh-darwin-x64` |
| Linux arm64 | `linersh-linux-arm64` |
| Linux x64 | `linersh-linux-x64` |
| Windows x64 | `linersh-win32-x64` |

Each platform package contains the native Go TUI and bundled CLI core for that system. Install the main `linersh` package and let npm select the platform package.

Node.js 18 or newer is required.

## Enable JavaScript rendering

Some source pages need a browser to expose readable content. Set up Playwright Chromium once using the command for your install path.

For one-shot use, run this command.

<CommandSnippet command="npx --yes linersh@latest setup-js" />

For a global install, run this command.

<CommandSnippet command="liner setup-js" />

The browser download is about 150 MB. You can run setup again when needed. The terminal UI also offers setup when compile encounters a page that requires it.

## Optional agent maintenance

A global install lets Codex or Claude Code call the same CLI to update an existing Project. The optional Maintenance Adapter routes those requests through Liner's reviewed change workflow.

Inspect the target and preview installation.

<CommandSnippet command="liner adapters inspect codex" />

<CommandSnippet command="liner adapters install codex" />

After reviewing the plan, authorize installation.

<CommandSnippet command="liner adapters install codex --yes" />

Use `claude` for Claude Code. The adapter preserves user-authored content outside its managed files and blocks. It works alongside the Project's own `SKILL.md`.

See [Maintain with an Agent](/docs/maintenance/) for installation, review, and approval details.

## Uninstall

Uninstall removes Liner configuration and source cache, plus shared Playwright browser downloads and npm's npx execution cache. Other tools may need to download those files again. Inspect the paths with `uninstall --dry-run` first. Saved Projects outside those paths remain in place.

<CommandSnippet command="npx --yes linersh@latest uninstall --dry-run" />

For one-shot use, this command asks for confirmation before removal.

<CommandSnippet command="npx --yes linersh@latest uninstall" />

For a global install, this command asks for confirmation before removal.

<CommandSnippet command="liner uninstall" />

Then remove the globally installed npm package if applicable.

<CommandSnippet command="npm uninstall -g linersh" />

<!-- End proposed route content -->

### https://www.liner.sh/docs/build-a-mixtape/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/build-a-mixtape.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Build a Project
description: Collect sources, compile Markdown research, write instruction files, and use the saved folder with an AI assistant.
```


Create a local folder for one focused task. Add sources, review the research, then write instructions an AI assistant can follow. Liner calls the folder a Project and the research packet inside it a Mixtape.

## Start in the terminal UI

<CommandSnippet command="npx linersh" />

The terminal UI guides creation from the first brief through source selection, synthesis approval, compile, and AI instruction writing. AI-assisted research needs your installed, authenticated Codex CLI or Claude Code runner. It uses that account's limits or charges and processes the material it reads under your provider's terms. Compile and instruction-file creation use local saved artifacts without making new model calls. Use the CLI for scriptable operations and `.mixtape` archives.

The default workspace is `~/liner/projects`. Set `LINER_DIR` to use another location.

<CommandSnippet command="LINER_DIR=~/liner/projects npx linersh" />

## Creation checkpoints

1. **Create Project** asks for a name, the task, your curator name, and whether you already have sources. The task uses Job Story form, such as “When I am designing native mobile interactions, I want grounded touch guidance, so I can avoid brittle gesture patterns.”
2. **Source Inbox** accepts URLs, local files, pasted article text, and skill Sources. Its review screen lets you include or exclude each item, then shows a plain-language summary before saving.
3. **Clarify Job** records the situation, desired outcome, and boundaries before research begins.
4. **Build Corpus** assembles the body of research for the task. It discovers and evaluates evidence, writes working artifacts, and proposes draft Sources.
5. **Review Draft Sources** is the Curator checkpoint for the canonical Source set. Accepting it saves the checked Sources atomically and preserves any already-present Sources.
6. **Review Synthesis** is a separate checkpoint. Approve the current `synthesis.md` unchanged, or edit it and review the exact replacement before anything is written.
7. After Synthesis approval, Liner verifies the Project Snapshot and starts **Compile** automatically. Compile writes `mixtape/MIXTAPE.md` and the extracted source files.
8. **Create Operating Layer** writes `LINER.md`, root `SKILL.md`, and `liner.yaml`, completing the Project.

Source approval and Synthesis approval are intentionally separate. Choosing the evidence does not silently approve the interpretation placed at the top of `MIXTAPE.md`.

## Use the CLI

CLI paths resolve from your shell's working directory. This example creates a Project in the default terminal UI library. If you configured another library, use its path.

<CommandSnippet command={`mkdir -p ~/liner/projects
cd ~/liner/projects
liner init mobile-design-foundations`} />

Add sources through the CLI so Liner can assign a stable ID and check for duplicates.

<CommandSnippet command={`liner sources add mobile-design-foundations \
  --type web \
  --url https://developer.apple.com/app-store/review/guidelines/ \
  --note "Use the current review rules as primary evidence." \
  --priority required`} />

Continue in the terminal UI to build the research and review its proposed Sources and Synthesis. Your accepted Sources and separately approved Synthesis become part of the Mixtape.

After reviewing the recipe and synthesis, run a scriptable compile.

<CommandSnippet command="liner compile mobile-design-foundations" />

The compile step writes the research packet.

```mdx
<FileTree
  root="mobile-design-foundations/"
  label="Compile output"
  items={[
    { name: "mixtape/", kind: "dir" },
    { name: "MIXTAPE.md", kind: "file", depth: 1, accent: true },
    { name: "sources/", kind: "dir", depth: 1 },
    { name: "NN-source-slug.md", kind: "file", depth: 2 },
  ]}
/>
```

## Use the result

After compile, give the AI `mixtape/MIXTAPE.md` and access to `mixtape/sources/`. The compiled file contains a synthesis and links to separate source files.

In the terminal UI, Compile begins automatically after Synthesis approval. Continue to **Create Operating Layer** when the compiled result is usable. That action writes `LINER.md`, root `SKILL.md`, and `liner.yaml`.

After the instruction files are written, give the AI the whole folder when it can read files. For lighter use, paste `mixtape/MIXTAPE.md` into the conversation. It contains the synthesis and a source index; provide the referenced source files separately when the assistant needs their content.

## Recompile

When evidence changes, use `liner sources add`, `update`, or `replace` so identity and lineage stay intact. Then rebuild the affected corpus and compile.

<CommandSnippet command="liner compile mobile-design-foundations" />

In the terminal UI, completed folders open the existing compile result by default. Press `r` from the compile screen to repair or retry compile. To refresh instructions after evidence changes, inspect `liner project guidance <folder> --format markdown` and follow the reviewed maintenance steps or use the Maintenance Adapter. The terminal UI does not regenerate an existing Operating Layer.

<!-- End proposed route content -->

### https://www.liner.sh/docs/mixtapes/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/mixtapes.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Projects and Mixtapes
description: Understand the durable Liner Project, its Mixtape research packet, instruction files, status, and source files.
```


Each task has a local working folder called a Liner Project. Its research packet is called a Mixtape. The tree below shows the research inside `mixtape/` and the AI instructions at the Project root.

```mdx
<FileTree
  root="mobile-design-foundations/"
  items={[
    { name: "liner.yaml", kind: "file" },
    { name: "LINER.md", kind: "file" },
    { name: "SKILL.md", kind: "file" },
    { name: "working/", kind: "dir" },
    { name: "mixtape/", kind: "dir" },
    { name: "MIXTAPE.md", kind: "file", depth: 1, accent: true },
    { name: "tape.yaml", kind: "file", depth: 1 },
    { name: "synthesis.md", kind: "file", depth: 1 },
    { name: "sources/", kind: "dir", depth: 1 },
    { name: "01-touch-target-size.md", kind: "file", depth: 2 },
    { name: "02-apple-gestures.md", kind: "file", depth: 2 },
    { name: "local-sources/", kind: "dir", depth: 1 },
    { name: "touch-target-paper.pdf", kind: "file", depth: 2 },
    { name: "working/", kind: "dir", depth: 1 },
    { name: "01-jtbd-and-knowledge-map.md", kind: "file", depth: 2 },
    { name: "02-candidate-longlist.md", kind: "file", depth: 2 },
    { name: "03-evaluation.yaml", kind: "file", depth: 2 },
    { name: "04-quality-checks.md", kind: "file", depth: 2 },
  ]}
/>
```

## Tape

`mixtape/tape.yaml` is the recipe. It names the mixtape, describes its purpose, identifies the curator, and lists the kept sources.

The tape is small enough to review, fork, and store in Git.

Use Liner's Project and Source maintenance commands for changes after creation. The CLI checks stable IDs, plans the exact file effects, and records the applied result in a Change Receipt. Direct edits bypass those checks.

## Identity and change records

Each current Project has an immutable Project ID in `liner.yaml`. Each canonical Source has an immutable Source ID in `mixtape/tape.yaml`. Renaming a Project, moving its folder, or editing a Source's URL, title, notes, or section keeps the corresponding ID.

- Project rename and move preserve the Project ID.
- Source metadata and locator updates preserve the Source ID.
- Semantic Source replacement creates a successor Source ID and records predecessor lineage.
- Source removal retains content and provenance; purge is a separate destructive action.

Before a supported change, Liner reads a versioned **Project Snapshot**, which records current state, and creates a write-free **Project Change Set**, which plans the exact effects for that revision. After apply, it writes a redacted **Change Receipt** under `mixtape/.liner-runs/maintenance/`. The receipt records identity, effects, validation, stale derived artifacts, and next actions without copying Source bodies or private notes.

## Liner Project

At the Project root, these files guide future AI sessions.

- `LINER.md` tells an AI how to use the saved research.
- Root `SKILL.md` is the Project entry point for compatible local AI assistants.
- `liner.yaml` records status, evidence pointers, and instruction-file state.
- `working/` is for audits and maintenance notes.

## Mixtape

The Mixtape is the research packet. It lives under `mixtape/` and includes the tape, synthesis, extracted sources, optional private source copies, methodology notes, and compiled output.

## Compiled research

`mixtape/MIXTAPE.md` is the research entry point. It includes the curator synthesis and a source index with links to extracted files in `mixtape/sources/`. A written `MIXTAPE.md` means the research packet is compiled, but the AI instruction files may still need to be created.

## AI instruction files

In the terminal UI, **Create Operating Layer** is the guided action that writes `LINER.md`, root `SKILL.md`, and `liner.yaml`.

The instruction files help a new AI session load the folder, follow the method from the research, and check its status before reading the working notes.

## Curator notes

Give each source a note explaining why it belongs. Tell the AI which parts matter, what to skim, and which limitations apply.

## Skills and Projects

Skills carry reusable methods, including steps, style, rules, and taste.

Mixtapes carry evidence for a specific task, including primary docs, examples, competing views, and local files.

A Project combines the evidence with a method for the task in its root skill file.

<!-- End proposed route content -->

### https://www.liner.sh/docs/maintenance/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/maintenance.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Maintain a Project with an Agent
description: Let Codex or Claude Code request Project updates through Liner's CLI, with change previews, required approvals, and a saved record.
```


A completed Project has a root `SKILL.md` that tells an AI how to use its research. This is the **Project Skill**. To let Codex or Claude Code update the Project or its sources, install the separate **Maintenance Adapter**. It routes change requests through Liner's CLI.

The adapter reads the installed CLI's current guidance and routes change requests through its commands. It never edits `liner.yaml`, `tape.yaml`, `LINER.md`, `SKILL.md`, or `MIXTAPE.md` directly.

## Install the optional adapter

Install Liner globally so the agent can call the same compatible CLI for each request.

<CommandSnippet command="npm install -g linersh" />

Inspect the Codex target, then request an installation plan.

<CommandSnippet command="liner adapters inspect codex" />

<CommandSnippet command="liner adapters install codex" />

Review the plan before authorizing its file changes.

<CommandSnippet command="liner adapters install codex --yes" />

For Claude Code, inspect and preview its adapter separately.

<CommandSnippet command="liner adapters inspect claude" />

<CommandSnippet command="liner adapters install claude" />

Then authorize the Claude Code installation.

<CommandSnippet command="liner adapters install claude --yes" />

Inspection reports adapter state, compatibility, and target. When installation needs approval, the command without `--yes` prints the plan and exits with code 2. Adding `--yes` authorizes the file changes immediately and prints a receipt after applying them. Installation preserves user-authored files and text outside Liner's managed content.

## Ask for one concrete change

Open Codex or Claude Code where it can read the Project, then make a scoped request.

> Inspect this Liner Project. Add the current Apple App Review Guidelines as a required Source, tell me which derived artifacts become stale, and show any approval-required plan before applying it.

The Maintenance Adapter tells the agent to start with these inspection commands. Run this example from `~/liner/projects`, where `mobile-design-foundations` is the Project folder, or use your Project's actual path.

<CommandSnippet command={`liner project guidance mobile-design-foundations --format markdown
liner project inspect mobile-design-foundations --json`} />

The running CLI defines the available operations, compatibility rules, and recovery steps.

## Preview before applying

Every supported change follows this sequence.

1. A read-only Project Snapshot records identity, revision, compatibility, Sources, and lifecycle state.
2. A write-free Project Change Set lists the exact operations, file effects, risk, and refresh consequences.
3. When `approval_required` is `true`, the agent stops and waits for a fresh Curator response to that exact preview.
4. Liner revalidates the Project and plan before applying it.
5. A durable, redacted Change Receipt records what changed without copying Source bodies or private curator notes.

The immutable Project ID binds all three records to the same Project even after a rename or move. Source operations target immutable Source IDs, so the requested entry remains explicit even when names or addresses look similar. Receipts are stored inside the Project under `mixtape/.liner-runs/maintenance/`.

Additive and metadata-only changes may continue after validation when Core says approval is not required. Remove, purge, semantic replacement, Project moves, and other higher-risk changes keep their explicit approval gates.

## Supported changes

The installed adapter can route these supported requests.

- inspect Project identity, compatibility, lifecycle state, and Sources
- add or update a Source while preserving Source identity
- replace a Source with explicit predecessor and successor lineage
- remove a Source into the Retention Vault or separately approve a destructive purge
- rename or move a Project without changing Project identity
- refresh stale research and AI instructions through the required review steps
- update managed Project guidance or pointer blocks without overwriting user content

## Recoverable Project deletion

The terminal UI also exposes **Delete Project** under Maintain project. It moves the entire folder while preserving the Project ID and files. You must type the exact Project name and approve the exact destination; Liner then moves the whole folder out of the active Project library into a reviewed Liner Trash path and writes a Change Receipt.

The archived folder keeps its Project ID and user-authored files. To restore it, inspect the trash path and use the ordinary identity-preserving Project move workflow to move it back to an unoccupied destination in your Project library.

## Three skill surfaces

| Surface | Purpose | Can become active instructions? |
| --- | --- | --- |
| **Project Skill** | Lives at the Project root and teaches an AI how to use that Project's research and Operating Layer. | Yes, as the Project's canonical entrypoint. |
| **Maintenance Adapter** | Installs explicitly into Codex or Claude and routes maintenance to `liner project guidance`. | Yes, after you install it. |
| **`type: skill` Source** | Adds another skill's text to the Mixtape as evidence or an example. | No. It remains inert source material. |

The Maintenance Adapter does not replace the Project Skill, and a `type: skill` Source never installs or activates instructions.

## Inspect, update, or remove the adapter

Inspect the installed adapter.

<CommandSnippet command="liner adapters inspect codex" />

Preview an update to its managed content.

<CommandSnippet command="liner adapters update codex" />

After reviewing the plan, authorize the update.

<CommandSnippet command="liner adapters update codex --yes" />

To remove the adapter, preview removal separately.

<CommandSnippet command="liner adapters remove codex" />

After reviewing that plan, authorize removal.

<CommandSnippet command="liner adapters remove codex --yes" />

Use `claude` for Claude Code. Update and removal touch only verified Liner-managed content. If compatibility or managed state is ambiguous, Liner stops and prints the steps needed to resolve it.

<!-- End proposed route content -->

### https://www.liner.sh/docs/cli/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/cli.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: CLI Commands
description: Create, compile, inspect, and maintain local research Projects and Sources from the command line.
```


Run `liner` with no arguments to open the terminal UI. Use subcommands when you want a scriptable workflow.

For one-shot npm use, replace `liner <command>` with `npx --yes linersh@latest <command>`.

## Project commands

| Command | What it does |
| --- | --- |
| `liner init <folder>` | Create a folder with starter research files. |
| `liner replay <folder> [--out <path>]` | Copy a Project's Job to Be Done and Clarify Job answers into a fresh Project while retaining lineage for comparison. |
| `liner compile <folder>` | Fetch sources and write `mixtape/MIXTAPE.md` plus `mixtape/sources/`. |
| `liner share <folder>` | Zip the project into a `.mixtape` archive. |
| `liner import <archive> [dest]` | Unzip a `.mixtape` archive and refetch uncached web and YouTube sources. |
| `liner clone <url-or-path> [dest]` | Fetch a remote tape file or copy a local tape file. |
| `liner list` | List saved folders in the current directory. |

## Reviewed Project maintenance

Maintenance starts with a read-only Project Snapshot, plans an exact Change Set, applies that plan, and records a Change Receipt. Inspection and planning write no Project files. Apply validates the reviewed Change Set again. Operations that require approval wait for the curator to approve that exact preview.

`liner project inspect` reports the immutable Project ID, revision, compatibility, lifecycle state, and immutable Source IDs. Use those IDs when scripting a sequence of changes, since paths and source addresses can change.

| Command | What it does |
| --- | --- |
| `liner project inspect [path]` | Read Project identity, compatibility, lifecycle state, capabilities, and root-discovery evidence without writing. |
| `liner project guidance <folder> --format json` | Print the installed CLI's current maintenance contract and remediation. Use `markdown` for human-readable guidance. |
| `liner project plan <folder> --request-json <json>` | Create a write-free Change Set from a versioned maintenance request. |
| `liner project pointer <folder> --environment <codex\|claude> --action <install\|update\|remove>` | Plan an opt-in managed `AGENTS.md` or `CLAUDE.md` pointer update. |
| `liner project rename <folder> --name <name>` | Plan a Project display-name change without changing Project identity. |
| `liner project move <folder> --destination <path>` | Plan an atomic, identity-preserving Project root move. |
| `liner project apply <folder> --change-set-json <json>` | Apply one exact Change Set atomically and persist a redacted receipt. |

Structural or semantic plans require `--approve`. A Project move also requires `--approved-destination <path>` matching the exact destination in the reviewed Change Set.

## Source maintenance

Source operations use immutable IDs. Updating metadata keeps the same ID. Replacing a Source creates a new ID linked to its predecessor. Removal retains content in the retention vault; purge requires separate approval and deletes retained content.

| Command | What it does |
| --- | --- |
| `liner sources add <folder> --type <type> ...` | Add a Source and record a receipt when approval is not required; otherwise return the exact plan for review. |
| `liner sources update <folder> --source-id <id> ...` | Apply a metadata or locator update while preserving Source identity. |
| `liner sources replace <folder> --source-id <id> --type <type> ...` | Plan a successor Source with explicit lineage; apply the approved Change Set separately. |
| `liner sources remove <folder> --source-id <id>` | Plan removal into the retention vault; apply the approved Change Set separately. |
| `liner sources purge <folder> --source-id <id>` | Plan separately approved deletion of retained Source content. |

## Optional Maintenance Adapters

The Maintenance Adapter is opt-in, delegates to the installed CLI, and never
replaces the Project Skill. Managed operations preserve user-authored content.

| Command | What it does |
| --- | --- |
| `liner adapters inspect <codex\|claude>` | Inspect adapter state without changing it. |
| `liner adapters install <codex\|claude>` | Preview installation when approval is required. Add `--yes` to authorize the file changes and receive an apply receipt. |
| `liner adapters update <codex\|claude>` | Preview an update to Liner-managed content. Add `--yes` to authorize the changes. |
| `liner adapters remove <codex\|claude>` | Preview removal of managed content. Add `--yes` to authorize removal; user-authored siblings and text remain. |

## Process commands

| Command | What it does |
| --- | --- |
| `liner manifest <folder>` | Aggregate `.liner-runs/` into `process.json`. |
| `liner status <folder>` | Show a summary of runs, tokens, tool calls, and fetched URLs. |

Use `liner status <folder> --no-write --json` for a read-only status consumer. Use `--status-only` only when you intend to refresh the Status Snapshot in `liner.yaml` without writing `process.json`.

## Skill source discovery

| Command | What it does |
| --- | --- |
| `liner skills list` | List locally installed skills that Liner can import as reference material. |
| `liner skills list --json` | Emit the same discovery result as JSON. |

## Cache commands

| Command | What it does |
| --- | --- |
| `liner cache info` | Show cache location and size. |
| `liner cache list` | List cached source entries. |
| `liner cache show <url>` | Inspect one cached source. |
| `liner cache clear` | Clear cached source content. |
| `liner cache purge <url>` | Remove one cached URL. |

## Setup commands

| Command | What it does |
| --- | --- |
| `liner setup-js` | Download Playwright Chromium for JavaScript-rendered pages. |
| `liner uninstall` | Remove Liner configuration and source cache, shared Playwright browsers, and npm's npx execution cache after confirmation. Use `--dry-run` to inspect the paths. |

## Useful flags

Stream compile progress as events.

<CommandSnippet command="liner compile mobile-design-foundations --emit-events" />

Fetch source content again instead of reusing valid cache entries.

<CommandSnippet command="liner compile mobile-design-foundations --no-cache" />

Use a browser-session cookies file for source fetching.

<CommandSnippet command="liner compile mobile-design-foundations --cookies ~/cookies.txt" />

Export without the raw local-source directories. Check the [sharing limits](/docs/sharing/#private-material) before sending the archive.

<CommandSnippet command="liner share mobile-design-foundations --no-personal" />

List saved folders as JSON.

<CommandSnippet command="liner list --json" />

Search one level below the selected directory.

<CommandSnippet command="liner list --recursive" />

Inspect uninstall paths without removing them.

<CommandSnippet command="liner uninstall --dry-run" />

To clear saved source content, use `liner cache clear` as a separate operation. It asks for confirmation before clearing the cache.

The CLI creates folders, compiles research, and handles supported Project and Source changes. The terminal UI also creates AI instruction files through **Create Operating Layer** after compile. Both interfaces use Liner Core for maintenance.

The terminal UI uses `--emit-events` to stream compile progress from the Python core.

`liner list --recursive` searches one level below the directory you pass with `--dir`.

<!-- End proposed route content -->

### https://www.liner.sh/docs/tui/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/tui.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Terminal UI
description: Use keyboard shortcuts to create Projects, review sources and synthesis, compile research, and write AI instructions.
```


Open the terminal UI.

<CommandSnippet command="npx linersh" />

The terminal UI is keyboard-first. Install and authenticate Codex CLI or Claude Code for AI-assisted research. The runner uses your provider account and processes the material it reads under that provider's terms. First-run setup connects your AI runner; Project creation then asks for a task and its desired outcome. **Clarify Job** helps define the situation and boundaries. The remaining steps guide source selection, synthesis review, compile, and AI instruction writing.

## Home

| Key | Action |
| --- | --- |
| `up` / `down` | Choose a command. |
| `enter` | Run the selected command. |
| `/` | Filter commands. |
| `g` | Open Settings. |
| `q` | Quit. |
| `?` | Toggle expanded help. |

Home is the command hub. It shows folder actions only when a folder is open and ready for that action.

## Projects

| Key | Action |
| --- | --- |
| `up` / `down` | Move selection. |
| `enter` | Open the selected project. |
| `/` | Filter projects. |
| `n` | Create a new project. |
| `i` | Import a `.mixtape` archive. |
| `g` | Open Settings. |
| `r` | Refresh the project list. |
| `esc` | Return to Home. |
| `ctrl+c` | Quit. |

## Project

| Key | Action |
| --- | --- |
| `up` / `down` | Move between project sections. |
| `enter` | Run the recommended next action. |
| `a` | Add sources when the project is ready for source entry. |
| `c` | Compile when manual compile is available. |
| `p` | Preview `mixtape/MIXTAPE.md` after compile. |
| `l` | Preview or create `LINER.md`. |
| `m` | Open Core-backed Project and Source maintenance. |
| `u` | Retry excluded local sources from the Sources pane when available. |
| `o` | Open the saved folder. |
| `h` | Return Home. |
| `esc` | Return to Projects. |
| `ctrl+c` | Quit. |

The Project screen shows the state of one saved folder in six sections. Health, Flow, Sources, Artifacts, Operating Layer, and Usage summarize its research, instructions, and current progress.

## Add sources

| Key | Action |
| --- | --- |
| `enter` | Add source text or save staged sources, depending on the current field. |
| `tab` / `up` / `down` | Move between fields and staged rows. |
| `f` or `ctrl+d` | Finish source entry. |
| `d` | Remove a staged source. |
| `a` | Edit more after reviewing staged sources. |
| `ctrl+o` | Open `local-sources/` after it exists. |
| `esc` | Go back. |

New projects ask whether you already have sources right after the folder is created. Choose paste sources to open the inbox, open the source editor for manual entry, or skip and let Build Corpus discover sources.

The source inbox accepts web URLs, YouTube URLs, GitHub skill URLs, installed skill names, local file paths, existing `local-sources/...` paths, and pasted website/article text. In current Projects, absolute local files are copied into `mixtape/local-sources/` automatically. Pasted article text is saved under `mixtape/local-sources/captured/` and added as a local file source. Recipe paths in `mixtape/tape.yaml` remain relative to `mixtape/`.

For multiple sources, paste URLs together separated by spaces, commas, or new lines. For multiple full article bodies, put `--- source ---` on its own line between articles; each article block becomes its own captured local source. The inbox creates and shows the Project's `mixtape/local-sources/` folder so you can also drop `.md`, `.txt`, `.html`, `.htm`, or `.pdf` files there.

For authenticated or paid sources, open the page in your browser, copy the useful article text, and paste it into the source inbox. The terminal UI saves that text under `mixtape/local-sources/captured/`. Automatic browser capture is not available.

Finishing entry opens **Review User-Provided Sources**. Toggle any row you do not want to save, then press Enter to prepare the save. The confirmation names how many Sources are new or already present, states that the change is additive, and shows the next step. Press Enter again to save the active Sources through Liner Core and continue to Clarify Job. Existing Sources stay unchanged.

## Build Corpus

**Build Corpus** runs the research workflow through your configured Claude Code or Codex runner. A corpus is the body of research assembled for the task. The TUI streams progress, writes working files, validates the framing, and prepares draft Sources and Synthesis for review before Compile.

When research finishes, **Review Draft Sources** shows the proposed canonical Source set. Toggle the checked rows, inspect their notes and roles, then accept the selection. Liner saves that Source batch atomically, keeps already-present Sources unchanged, refreshes the Project Snapshot, and opens Synthesis approval.

**Review Synthesis** is a separate Curator decision. Approve `synthesis.md` unchanged when it accurately represents the accepted Sources, or edit it and review the exact proposed replacement. Recording Synthesis approval creates a durable receipt; Liner then verifies the approved Project state and starts Compile automatically.

## Compile

| Key | Action |
| --- | --- |
| `enter` | Continue with the recommended action. This opens source review, repairs sources, rebuilds the corpus after recovery, or creates the Operating Layer as needed. |
| `r` | Repair unavailable custom sources when possible, otherwise retry compile. |
| `p` | Preview `mixtape/MIXTAPE.md`. |
| `v` | Preview compile logs when available. |
| `y` | Copy compiled `mixtape/MIXTAPE.md`. |
| `a` | Add sources when compile output shows source gaps. |
| `i` | Install JS rendering support when the compile warning needs it. |
| `up` / `down` | Move through warning rows or source rows. |
| `o` / `d` | Open or drop the selected warning/source row when that action is available. |
| `esc` | Go back. |

Compile opens with a compact result summary. Press Enter to review individual sources. Rows with errors, custom sources you can retry, and recovered content needing Build Corpus appear first. Press `r` to retry
unavailable custom YouTube or web sources. If a source needs browser rendering,
the repair flow installs JS support first, then retries recovery. Recovered
content is saved under `mixtape/local-sources/recovered/` and the TUI asks for Build
Corpus only when the AI needs to reconsider new local material.

Compile treats a written `mixtape/MIXTAPE.md` as usable even when warnings
remain. Reviewed warnings stay visible, but a usable partial result can continue
to Operating Layer creation.

## Maintain project

Choose **Maintain project** from Home when an existing Project or Source needs a
supported change. The TUI asks Liner Core for a read-only Project Snapshot and a
write-free Change Set, then shows immutable IDs, exact file effects, lifecycle
consequences, risk, and any required approval.

Additive and metadata-only plans may continue after validation when Core says
approval is not required. Remove, purge, semantic replacement, Project moves,
and other higher-risk operations stop on the exact preview. Apply revalidates
that Change Set and finishes with a durable Change Receipt plus the refreshed
Project state and next actions.

Removal retains Source content in the Retention Vault. Purge is a separate,
destructive operation. The TUI never reconstructs Core's risk or writes
canonical Project files directly for supported maintenance.

**Delete Project** moves the whole folder to recoverable Liner Trash. Type the exact Project name, review the destination, and approve the move. The folder leaves the active Projects list while its Project ID, files, and Change Receipt remain intact. To restore it, inspect the trash path and use the reviewed Project move workflow to move it back to an unoccupied destination in your Project library.

## AI instruction files

**Create Operating Layer** writes the instruction files. After the research packet is compiled, the TUI can create `LINER.md`, root `SKILL.md`, and `liner.yaml`. This step tells future AI sessions how to discover and use the saved work.

## Import

The terminal UI imports `.mixtape` archives with a file picker. Press `i` from Projects or choose Import Project from Home, select an archive, and import it into your Liner workspace.

TUI import does not refetch sources. After import, open the folder and compile to produce or update extracted source files. Compile normally reuses valid cached content. Use `liner compile <folder> --no-cache` when you need a new fetch.

<!-- End proposed route content -->

### https://www.liner.sh/docs/sharing/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/sharing.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Sharing
description: Export and import .mixtape archives, choose their contents, and check private material before sharing.
```


Export a Project as a `.mixtape` archive to move it between machines or share it. Check its contents first, since local source files are included by default.

## Export

<CommandSnippet command="liner share mobile-design-foundations" />

This writes an archive in the current directory.

```mdx
<FileTree
  root="."
  label="Archive output"
  items={[
    { name: "mobile-design-foundations.mixtape", kind: "file", accent: true },
  ]}
/>
```

The archive is a zip file with the saved folder contents.

## Import

<CommandSnippet command="liner import mobile-design-foundations.mixtape" />

The CLI extracts the archive and refetches uncached web and YouTube sources. To skip refetching, use this command.

<CommandSnippet command="liner import mobile-design-foundations.mixtape --no-refetch" />

The terminal UI import flow skips refetching. It extracts the archive into your Liner workspace. Open that folder and compile to produce or update extracted source files. Compile normally reuses valid cached content. Use `liner compile <folder> --no-cache` when you need a new fetch.

## Export options

Choose the option for the archive you want. These are separate exports; running another command for the same output path replaces the previous archive.

Export only the tape recipe.

<CommandSnippet command="liner share mobile-design-foundations --minimal" />

Exclude working-note directories.

<CommandSnippet command="liner share mobile-design-foundations --no-working-notes" />

Exclude extracted source directories.

<CommandSnippet command="liner share mobile-design-foundations --no-source-content" />

Exclude raw local-source directories.

<CommandSnippet command="liner share mobile-design-foundations --no-personal" />

Choose a destination for a full export.

<CommandSnippet command="liner share mobile-design-foundations --out ~/Desktop/mobile.mixtape" />

## Private material

The archive includes local source files by default, along with hidden files and run logs. In current Projects, local sources live under `mixtape/local-sources/`; legacy `personal/` folders can be included too.

The export flags filter specific directories. They do not remove private material from every saved artifact.

- `--no-personal` excludes raw `local-sources/` and legacy `personal/` directories. It can leave copies in extracted sources, synthesis, compiled output, and logs.
- `--no-source-content` excludes `sources/`. It can leave raw local files and derived content elsewhere.
- `--no-working-notes` excludes `working/`. It can leave hidden run logs and other saved files.
- `--minimal` includes only the tape recipe, which can still contain private URLs, notes, and metadata.

Inspect the full archive before sharing it, including derived files, hidden files, and raw runner logs. Choose only material you are allowed to share.

<!-- End proposed route content -->

### https://www.liner.sh/docs/tape-format/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/tape-format.md`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Tape Format
description: Reference for tape.yaml metadata, source types, local-file requirements, and source fields.
```

`mixtape/tape.yaml` is the recipe for a Project's research packet. It lists the sources and the metadata used during compile.

This page is a format reference. Use the terminal UI or `liner sources` and
`liner project` commands for supported post-creation maintenance so immutable
IDs, lineage, lifecycle invalidation, approvals, and Change Receipts remain
intact. An optional Maintenance Adapter delegates to those commands; it does not
edit the recipe directly.

## Required fields

```yaml
title: Mobile Design Foundations
description: Core touch and gesture references for native mobile UX.
version: 1
curator: cmdux-sh
mode: quick
jtbd: When I am designing native mobile interactions, I want grounded touch guidance, so I can avoid brittle gesture patterns.

sources:
  - type: web
    url: https://developer.apple.com/design/human-interface-guidelines/gestures
    note: Apple's canonical gesture reference.
    section: foundations
    kind: reference
```

The following top-level fields are required.

- `title`
- `description`
- `version: 1`
- `curator`
- `sources`

`sources` can be empty while you are drafting a tape. The terminal UI requires at least one source before compiling.

## Optional top-level fields

- `mode`, either `quick` or `methodology`
- `tags`
- `created`
- `updated`
- `license`
- `homepage`
- `jtbd`
- `jtbd_clarifications`
- `methodology_version`
- `parent`

The `jtbd` field records the task using Job Story form.

```text
When [circumstance], I want [motivation], so I can [outcome].
```

## Source types

The tape format supports four source types.

### Web

```yaml
- type: web
  url: https://example.com/article
  note: Why this source belongs.
  section: foundations
  priority: required
  kind: reference
```

The optional `render` field controls web fetching.

- With no `render` field, try server HTML first and use headless Chromium if needed.
- Set `render: js` to use Chromium immediately.
- Set `render: server` to use server-rendered HTML only.

### YouTube

```yaml
- type: youtube
  url: https://www.youtube.com/watch?v=XXXX
  note: Watch for the section on source evaluation.
  section: examples
```

The fetcher tries transcript fetching first and falls back through `yt-dlp` where possible.

### Local file

```yaml
- type: local_file
  path: local-sources/touch-target-paper.pdf
  citation: "Parhi et al., Target Size for One-Handed Thumb Use, Mobile HCI 2006."
  note: Canonical empirical study behind small-screen touch targets.
  section: research
```

In current Projects, put files under `mixtape/local-sources/`. The recipe uses a path relative to `mixtape/`, such as `path: local-sources/touch-target-paper.pdf`. Legacy root-level Projects resolve that recipe path from their root and also support `personal/`. Supported formats are Markdown, text, HTML, and PDF, up to 10 MiB per file.

### Skill

```yaml
- type: skill
  path: terminal-ui
  note: Extract terminal interaction guidance as reference material.
  section: craft
```

Skill sources can use an installed skill name, a local skill folder or `SKILL.md` path, a project snapshot under `local-sources/skills/`, or a GitHub skill URL. Skill files are read as reference material and are not installed or executed.

## Source fields

These optional fields apply across source types.

- `note` explains why the source belongs.
- `section` groups sources for compile filters.
- `priority` accepts `required` or `optional` and defaults to `required`.
- `kind` accepts `reference`, `principle`, `prescription`, or `example`.

`local_file` sources require `path` and `citation`. Web and YouTube sources require `url`. Skill sources require either `path` or a GitHub `url`.

<!-- End proposed route content -->

### https://www.liner.sh/docs/troubleshooting/

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/content/docs/docs/troubleshooting.mdx`. Proposed complete content follows. The full text below includes the factual corrections to commands, reference tables, and key bindings. Other command syntax and file-tree entries remain exact.

<!-- Begin proposed route content -->

```yaml
title: Troubleshooting
description: Fix common Liner install, compile, rendering, and transcript issues.
```


## Check the installed version

<CommandSnippet command="npx linersh@latest --version" />

For version 1.1.1, the output is the following.

```text
liner 1.1.1 (tui) · 1.1.1 (core)
```

If the platform core binary cannot be resolved, the version command prints only the TUI package version.

## Enable JavaScript rendering

If compile says a page needs headless Chromium, choose the command for your installation.

For one-shot use, run this command.

<CommandSnippet command="npx --yes linersh@latest setup-js" />

For a global install, run this command.

<CommandSnippet command="liner setup-js" />

The terminal UI also offers setup from the compile screen when it shows the JavaScript rendering notice.

## Recover YouTube transcripts

YouTube can block transcript fetching for age-restricted, region-locked, or rate-limited videos. A Netscape-format cookies file can help when your browser session can access the video.

<CommandSnippet command="liner compile mobile-design-foundations --cookies ~/cookies.txt" />

You can also set the Core fetcher's default in `~/.liner/config.toml`. The AI runner uses the separate `~/.liner/config.yaml` file.

## Help the terminal UI find the CLI

For local development, point the npm launcher at a specific CLI core binary.

<CommandSnippet command="LINER_BIN=/path/to/liner liner" />

The resolver tries `LINER_BIN`, the repo-local virtualenv, the bundled platform package, then `liner` on `PATH`.

## Reset local state

See [Uninstall](/docs/install/#uninstall) to review and remove Liner's local state and shared execution/browser caches. Saved Projects outside the removed paths remain in place.

<!-- End proposed route content -->

## Changelog

Source `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/src/pages/changelog.astro`. The version number comes from `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/VERSION`. Curated public entries below are derived from `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/docs/project/CHANGELOG.md`. Use explicit release state for the current/candidate badge. At review time, npm latest is 1.1.1; a numbered source heading alone is insufficient evidence of publication.

| Element | Proposed copy |
| --- | --- |
| Route title | Liner Changelog |
| Description | Changes to Liner's local CLI, terminal UI, source handling, and Project maintenance. Candidate work is labeled separately from shipped releases. |
| Hero eyebrow | Changelog |
| Hero heading | Changes to the local research toolkit. |
| Hero body | Follow changes to installation, source fetching, research review, and Project maintenance. Candidate entries stay labeled until the release ships. |
| Summary accessible name | `{Current or Candidate} release summary` unchanged |
| Summary labels | Current / Candidate / Install / Output |
| Summary values | `{releaseVersion}` / `npx linersh` / Research + AI instructions |
| Spotlight accessible name | `{Current or Candidate} {releaseVersion} release spotlight` unchanged |
| Spotlight eyebrow | Current release / Release candidate unchanged, with version |
| Spotlight heading | Pick up interrupted research from its saved checkpoint. |
| Spotlight paragraph 1 | Version {releaseVersion} preserves accepted work when a research step stops or a generated file fails validation. Reopen the Project to return to the right review or build step. Decisions from Improve Corpus survive restarts, and applied changes remain visible in the audit history. |
| Spotlight paragraph 2 | The optional Maintenance Adapter continues to route Project changes through the installed CLI. You can review the planned effects, approve changes when required, and inspect the saved Change Receipt. |
| Highlight 01 | Sources you already approved keep their IDs when Assembly adds evaluation details. |
| Highlight 02 | Interrupted research returns to the last saved review or build step. |
| Highlight 03 | Improve Corpus decisions survive restarts, with applied recommendations recorded in audit history. |
| Install aside accessible name | Install command unchanged |
| Install aside eyebrow | Open Liner |
| Install command | `npx linersh` unchanged |
| Install note | Runs locally. AI-assisted research uses your configured Codex CLI or Claude Code runner and its account limits or charges. Compile and instruction-file creation make no model calls. The installed tool has no telemetry. |
| Entry-list eyebrow | Shipped changes / Candidate and shipped changes unchanged |
| Entry-list heading | Release by release. |
| Entry-list body | Candidate entries describe work awaiting publication. Historical entries describe the behavior shipped in each version. |
| Entry status labels | Candidate / Current / Version unchanged |
| Version labels | Preserve version numbers and historical notes unchanged. Display the candidate note as `candidate` without an authored dash in the visible label. |

### Website release introductions

Replace the `releaseIntros` values. All twelve keys remain the same. These introductions pair with the curated entry data below.

| Version | Proposed introduction |
| --- | --- |
| 1.1.1 | Interrupted research resumes from its saved checkpoint. Approved sources keep their IDs through Assembly, and Improve Corpus decisions survive restarts with a visible audit record. |
| 1.1.0 | Source selection and synthesis now receive separate approval before compile. Maintenance previews Project and Source changes, requests approval when required, and saves a receipt, including for recoverable Project deletion. |
| 1.0.2 | Compile summarizes the result before showing source details. It guides recovery for blocked web and YouTube sources and lets reviewed, usable partial results continue to AI instruction writing. |
| 1.0.0 | The first complete Project-builder release uses the Go terminal UI by default. The npm package includes the CLI and terminal UI, and completed folders carry AI instructions. |
| 0.5.7 | Browser fallback can capture PDF downloads and extract their text. |
| 0.5.6 | Source fetching checks remote PDFs, empty JavaScript pages, cookie-only output, and stale cache entries. Partial compiles report usable and unavailable sources separately. |
| 0.5.5 | This release improves terminal scrolling, JavaScript rendering setup, partial compiles, and clean reinstall testing. |
| 0.5.4 | Fresh npm installs find the bundled research skill more reliably. Windows x64 gains a platform install path alongside macOS and Linux. |
| 0.5.3 | Version output reports both the installed terminal UI package and the CLI core. |
| 0.5.2 | The terminal hub gains one status view, clearer navigation, and more accurate run summaries. |
| 0.5.1 | The first update after the npm terminal UI release improves startup, colors, navigation, and folder opening. |
| 0.5.0 | The interactive terminal UI becomes available through npm on clean machines. |

### Curated public release entries

Use this complete site-owned public dataset in place of rendering every internal changelog bullet. Version numbers and historical behavior are preserved; internal test receipts, release packaging procedures, implementation-helper changes, and obsolete model-cost estimates are omitted. Public history stays historical, including version-specific interfaces and milestones.

The `changes` values below are the complete proposed visitor-facing title and body strings. Pair them with the twelve release introductions above. All twelve versions are published; `npm view linersh versions --json` confirmed their presence on 2026-10-17. Published versions 1.0.1 and 1.0.3 have no source changelog entries and remain omitted; do not invent their changes. Future releases need explicit candidate/shipped state.

```json
[
  {
    "version": "1.1.1",
    "changes": [
      {
        "title": "Resume interrupted research",
        "body": "Reopen a Project after an interrupted agent run or a validation failure to return to the last saved review or build step. Partial work stays marked as incomplete."
      },
      {
        "title": "Preserve approved sources",
        "body": "Sources you already accepted keep their IDs when Assembly adds evaluation details. Pending Assembly is shown as the next action when needed."
      },
      {
        "title": "Save improvement decisions",
        "body": "Improve Corpus decisions survive restarts. Applied recommendations stay resolved, and the audit keeps the original gap as history."
      },
      {
        "title": "Clarify confirmations",
        "body": "Source and synthesis transitions show ordinary confirmation messages. Recoverable Project deletion keeps focus on the exact confirmation step."
      }
    ]
  },
  {
    "version": "1.1.0",
    "changes": [
      {
        "title": "Review maintenance through Liner Core",
        "body": "Project and Source changes use the same CLI-owned snapshots, plans, risk checks, and receipts. Higher-risk operations wait for approval of the exact preview."
      },
      {
        "title": "Approve sources and synthesis separately",
        "body": "Source review shows what will be added and what stays unchanged. A separate synthesis approval records the decision and starts compile."
      },
      {
        "title": "Recover deleted Projects",
        "body": "Delete Project asks for the exact name and previews a move to Liner Trash. The folder leaves the active library while its files and Project ID remain recoverable."
      },
      {
        "title": "Maintain with Codex or Claude Code",
        "body": "Optional Maintenance Adapters route changes through the installed CLI. Inspect, install, update, and remove commands preserve content outside Liner-managed files and blocks."
      },
      {
        "title": "Load clearer Project instructions",
        "body": "Generated root SKILL.md files give compatible AI assistants a concise entry point, load order, and completion criteria. Detailed operating rules remain in LINER.md."
      },
      {
        "title": "Recover blocked sources",
        "body": "Browser fallback can recover some security-checkpoint responses. Recovered material refreshes the affected evaluation steps, and ordinary curator notes no longer appear as compile warnings."
      },
      {
        "title": "Simplify browser setup",
        "body": "JavaScript rendering setup offers one install action and shows a loader while Playwright Chromium is downloading."
      }
    ]
  },
  {
    "version": "1.0.2",
    "changes": [
      {
        "title": "Read compile results first",
        "body": "Compile opens with a compact summary. Source review brings errors, retryable custom sources, and recovered content to the top."
      },
      {
        "title": "Repair custom sources",
        "body": "Retry unavailable custom web or YouTube sources, install browser rendering when needed, and review recovered material before continuing."
      },
      {
        "title": "Keep repair work in context",
        "body": "Adding replacement material returns to compile. Recovered local sources remain available for the next research build."
      },
      {
        "title": "Continue with usable partial results",
        "body": "After reviewing warnings, a usable MIXTAPE.md can proceed to AI instruction creation. Completed Projects open LINER.md as their next action."
      }
    ]
  },
  {
    "version": "1.0.0",
    "changes": [
      {
        "title": "Open the Go terminal UI",
        "body": "Running liner or npx linersh opens the terminal UI for creating, opening, and managing local Projects."
      },
      {
        "title": "Install through npm",
        "body": "The linersh package includes the terminal UI and matching platform CLI binaries. Normal use needs no separate Python setup."
      },
      {
        "title": "Create AI instructions",
        "body": "After the research is ready, Create Operating Layer writes LINER.md, root SKILL.md, and liner.yaml from the saved artifacts. The files describe how to use the evidence and respect its boundaries."
      },
      {
        "title": "Carry research into the method",
        "body": "Generated instructions use the saved synthesis, quality checks, source roles, and capability patterns to guide work with the Project."
      }
    ]
  },
  {
    "version": "0.5.7",
    "changes": [
      {
        "title": "Extract browser-downloaded PDFs",
        "body": "When browser fallback receives a PDF download, Liner captures the file and runs PDF text extraction."
      }
    ]
  },
  {
    "version": "0.5.6",
    "changes": [
      {
        "title": "Recognize remote PDFs",
        "body": "Web responses identified as PDFs use PDF extraction, so raw PDF bytes no longer become source text."
      },
      {
        "title": "Refetch unusable cache entries",
        "body": "Cached content that resembles a bad PDF extraction, an empty fallback, or cookie-only output is discarded and fetched again."
      },
      {
        "title": "Detect empty pages",
        "body": "Empty JavaScript shells trigger browser fallback. Cookie-consent-only output is marked unavailable."
      },
      {
        "title": "Explain partial compiles",
        "body": "Compile reports the count of usable sources separately from unavailable source files."
      }
    ]
  },
  {
    "version": "0.5.5",
    "changes": [
      {
        "title": "Improve terminal scrolling",
        "body": "Final responses and long compile source lists use internal scrolling areas that fit the terminal."
      },
      {
        "title": "Set up browser rendering",
        "body": "First-run setup offers optional Playwright Chromium installation. Compile also surfaces missing browser support for JavaScript-only pages."
      },
      {
        "title": "Keep usable compile results accessible",
        "body": "Partial compiles remain available with warnings. Opening a current result avoids rerunning compile; r explicitly retries it."
      },
      {
        "title": "Recover from HTML extraction errors",
        "body": "A text fallback can retain a source with a warning when the primary HTML extractor fails."
      },
      {
        "title": "Remove local state for a clean reinstall",
        "body": "The new uninstall command removes Liner configuration and source cache, shared Playwright browsers, and npm npx execution cache."
      }
    ]
  },
  {
    "version": "0.5.4",
    "changes": [
      {
        "title": "Find the bundled research skill",
        "body": "npm installs locate the included skill bundle more reliably and report the checked paths when it is missing."
      },
      {
        "title": "Add a Windows install path",
        "body": "Windows x64 receives a native platform package alongside macOS and Linux."
      },
      {
        "title": "Configure models by phase",
        "body": "Discovery and evaluation can use a cheaper configured model, while other research phases use the runner default. A rejected model setting warns and retries once with the default."
      },
      {
        "title": "Detect an unresponsive runner",
        "body": "A runner that emits no output within 15 seconds produces a clear failure with retry and switch-runner options."
      },
      {
        "title": "Report usage accurately",
        "body": "Run summaries keep token counts and remove the unsupported dollar-cost estimate."
      }
    ]
  },
  {
    "version": "0.5.3",
    "changes": [
      {
        "title": "Report both installed versions",
        "body": "The version command reports the terminal UI package and bundled CLI core together, or only the terminal UI version when the core cannot be resolved."
      },
      {
        "title": "Keep version labels consistent",
        "body": "The header and startup screen read the package version directly."
      }
    ]
  },
  {
    "version": "0.5.2",
    "changes": [
      {
        "title": "Simplify the research hub",
        "body": "The hub shows one status summary with a phase checklist. In this release, compile is the completion milestone."
      },
      {
        "title": "Streamline import and navigation",
        "body": "Import detects .mixtape files in the current folder and opens the imported research hub. Screen actions and back navigation become more consistent."
      },
      {
        "title": "Fit lists and responses to the terminal",
        "body": "Project lists and final responses use bounded viewports to reduce wrapping and repeated scrollback."
      },
      {
        "title": "Improve runner reliability",
        "body": "Claude runs avoid a known compaction error and use an isolated MCP configuration for the research workflow."
      },
      {
        "title": "Remove unsupported cost estimates",
        "body": "Run summaries show token counts without an invented dollar figure."
      }
    ]
  },
  {
    "version": "0.5.1",
    "changes": [
      {
        "title": "Bundle the research skill",
        "body": "The npm package includes the research skill needed for AI-assisted task clarification."
      },
      {
        "title": "Improve terminal readability",
        "body": "Updated colors, explicit text contrast, and a cleaner startup logo improve legibility."
      },
      {
        "title": "Refine navigation and feedback",
        "body": "Back navigation avoids a loop, result notes stay visible, and the o shortcut opens the research folder."
      },
      {
        "title": "Reduce scrollback artifacts",
        "body": "The terminal app uses the alternate-screen buffer and clears the screen after a run completes."
      }
    ]
  },
  {
    "version": "0.5.0",
    "changes": [
      {
        "title": "Run the terminal UI from npm",
        "body": "npx linersh opens the terminal UI on clean machines. After a global install, liner opens the same interface."
      },
      {
        "title": "Keep CLI commands available",
        "body": "The installed command supports setup-js, compile, share, and status alongside the terminal UI."
      }
    ]
  }
]
```

The source sections `0.3.0`, `0.2.0`, and `0.1.0` are explicitly marked unreleased and stay excluded. Candidate display must follow explicit release state. Do not use a new numbered source heading alone as evidence that a release shipped.

## Public assets

| Source | Text coverage |
| --- | --- |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/public/og-image.svg` | Keep the brand text `Liner`. Replace the headline with `Build reusable` / `AI context.` and the support line with `choose sources · save research · write AI instructions`. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/public/og-image.png` | Re-export from the revised social card so the actual referenced PNG carries the same words. The source SVG alone does not update the image used by metadata. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/public/liner-logo.svg` | Path-only logo artwork. No visible text node, title, or description to revise; all referencing logo alts remain `Liner`. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/public/favicon.svg` | Icon artwork. No public prose or description to revise. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/public/site.webmanifest` | `name` and `short_name` both remain `Liner`. All remaining values are technical configuration. |
| `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/public/robots.txt` | Technical crawl directives unchanged. |
| Remaining PNG/ICO icons and WOFF2 font assets under `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/marketing/site/public/` | No authored prose to revise. |

## Integration notes and checks

- First-use commands contain no uninstall or hard-reset operation. Independent command intentions have separate copy controls, including export options and optional adapter preview/apply steps.
- The current public install page's “Hard Reset Before Release Testing” section should move to internal developer guidance with its test-machine restriction. It is not a first-use install requirement. Do not run any reset as part of this copy task.
- Keep source and synthesis approval separate in every docs route. An accepted source list does not approve its interpretation.
- Define “corpus” at its first explanatory use in each standalone docs route that depends on it. Exact interface action names remain unchanged.
- Preserve immutable IDs, retention before purge, approval requirements, and the recoverable Project deletion distinction.
- Keep telemetry claims scoped to the installed tool. The website layout has optional PostHog code.
- Verify actual clipboard completion and production search strings during implementation. Current components announce success without waiting for the clipboard promise; the proposed failure text needs corresponding behavior.
- The technical reference blocks deliberately retain exact YAML keys, including `mode`, and URL colons. Those are required syntax and are outside authored-prose diction rules.

The final language stage is complete. See `/Users/arturo/Documents/Projects/Liner-site-copy-20261016/docs/liner-website-language-notes.md` for its scope and fidelity checks.

## Final validation receipt

Coverage includes the homepage's seven narrative sections, shared navigation/footer and metadata, all ten docs routes, twelve changelog introductions and 47 curated public release entries, social-card words, manifest names, search/404/default docs chrome, and copy/motion interface states. No website source was edited by the writer.

The final Humanizer pass covered every proposed public text surface. Its fidelity check preserved routes, frontmatter, exact interface action names, technical syntax, filenames, version numbers, and claim limits. The prose linter retains one structural finding for short procedural paragraphs across standalone docs pages. Each introduces a command, prerequisite, or separate action; joining them across copy controls or routes would obscure the instructions. Required YAML keys and code/URL punctuation remain exact.
