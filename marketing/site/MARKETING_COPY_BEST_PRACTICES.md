# Marketing Copy Best Practices

Date: 2026-06-27

Scope: public marketing copy for Liner, especially the homepage, docs entry points, changelog intros, social cards, route metadata, and any visible product explanation. Use this before editing `marketing/site/src/pages/index.astro` or public docs copy.

This is the active Liner-specific guide. Historical audit notes and generic copy guidance were archived outside Git so this file can remain the source of truth for site copy.

This guide captures the copy lessons from the June 2026 rewrite. It is written for future agents who need to keep the site clear for a visitor who has never seen the product.

## The Standard

Good marketing copy explains the product before it names the product model.

A new reader should understand four things without learning our internal vocabulary first:

1. What this is.
2. What problem it solves.
3. What the user gives it.
4. What the user gets back.

For Liner, the plain version is:

> Liner is a local CLI and terminal UI for turning focused research sources into files your AI assistant can use later.

Everything else earns its place after that.

## The Reader Lens

Write for someone who knows nothing about Liner.

They may know ChatGPT, Claude, Codex, local CLIs, docs, PDFs, transcripts, and pasted notes. They do not know what we mean by project, corpus, mixtape, Operating Layer, Project Complete, source roles, or Capability Brief.

Assume the reader is asking:

1. What is this tool?
2. Why would I use it instead of asking an AI directly?
3. What do I put into it?
4. What files does it create?
5. How does another AI session use those files?
6. When is this worth the extra work?
7. What command starts it?

The page can answer those questions without turning them into literal headings.

## Explain By Showing The Work

Do not build sections around questions like "How is this different?" or "What gets saved?"

Answer through concrete states:

- A plain task becomes a narrow brief.
- Sources get chosen because each one has a role.
- The useful source material is saved as files.
- `MIXTAPE.md` collects the compiled research.
- `LINER.md` and `SKILL.md` tell a later AI session how to use the evidence.
- A clean session can reopen the folder without relying on chat history.

The reader should feel the answer forming from the example, not from a product argument.

## The Comparison To Direct AI Answers

The site needs to answer this objection:

> Why not just ask Codex, Claude, or ChatGPT to research the topic?

Do not make that the section title. Do not write a debate against generic AI tools.

Use the practical contrast:

> A direct AI answer can solve one request. The saved folder keeps the setup that made the answer possible: source choices, limits, notes, and task instructions that another session can inspect before writing.

That is the whole difference:

- A chat answer gives you a response.
- A saved folder gives you the response path: sources, notes, limits, and instructions.
- A later AI session can inspect the same evidence before it writes.
- The work can be diffed, archived, shared, or reopened from a clean session.

Keep the tone calm. The direct answer path is valid when the user only needs the response.

## Product Nouns Need An On-ramp

Public copy should introduce nouns in this order:

1. Local CLI and terminal UI.
2. Focused research for one task.
3. Sources: URLs, transcripts, PDFs, local files, pasted text.
4. Local folder.
5. Markdown and YAML files.
6. Compiled research packet.
7. AI instructions.
8. File names: `MIXTAPE.md`, `LINER.md`, `SKILL.md`, `liner.yaml`.
9. Project, corpus, mixtape, and Operating Layer only after the plain objects are clear.

Internal milestone labels should not be public marketing labels.

Use these translations:

| Internal term | Public copy |
| --- | --- |
| Corpus Ready | Research compiled |
| Project Complete | Instructions written |
| Operating Layer | AI instructions, instruction files, rules for the next session |
| Project Skill | `SKILL.md`, an agent entry point |
| Capability Brief | Narrow brief, specific job, task instructions |
| Source roles | The reason each source belongs |
| JTBD | Task, job, desired output |

Docs can use the internal nouns, but they still need definitions before use.

## Truth Comes Before Drama

Only claim what the product can do now.

Do not imply:

- A public library exists if it is only planned.
- There are five shipped mixtapes if there are not.
- The tool audits everything automatically if the feature is not ready.
- Hosted services, accounts, sync, or cloud features exist.
- The CLI calls AI models during compile.
- A folder is safe to share without checking contents.

Good copy has enough confidence to say the smaller true thing:

> The core CLI skips model calls during compile. The TUI can use your configured Claude Code or Codex runner for methodology work.

Truthful constraints build trust when they are written plainly.

## Section Copy Should Do A Job

Every section should have one job. If a section needs two jobs, split it or simplify it.

Useful section jobs:

- Define the tool.
- Name the pain.
- Show one concrete task.
- Show the files created.
- Explain how a later AI session uses the folder.
- Show when the tool is worth using.
- Show when a quick chat answer is enough.
- Give the install command.

Weak section jobs:

- Announce a framework.
- Describe internal architecture.
- Ask a question and answer it literally.
- Repeat the brand name.
- Sound like a status file.
- Defend the product against a straw man.

If a line sounds like it belongs in `liner.yaml`, rewrite it for a person.

## Headings Should Carry Meaning

A good heading lets the reader understand the next section without decoding our process.

Use headings like:

- From ask to files
- The work leaves the chat
- Files on disk
- Evidence, then instructions
- Where the folder becomes reusable
- The saved research keeps working after the first answer
- The setup stops being temporary
- The right size of work
- Details before install

Avoid headings like:

- What gets saved
- What Liner makes
- How they stack
- Use cases
- Questions
- How is this different?
- Project layers
- Corpus Ready
- Project Complete

The bad versions are not always false. They are too literal, too internal, or too dependent on the reader already knowing the product.

## Examples Beat Abstractions

The best copy in the rewrite used one concrete task:

> "Help my AI write App Store review-safe copy for a mental-health app."

That works because the reader can immediately see:

- The task is narrow.
- The sources matter.
- Unsupported claims are risky.
- A later AI session needs rules, not only notes.

When writing examples, make the job specific enough that source choice matters.

Good example shapes:

- Write App Store review-safe copy for a mental-health app.
- Turn visual references into a concrete web art-direction brief.
- Plan a tricky API migration from framework docs and failure reports.
- Evaluate interview findings against a product thesis while preserving raw evidence.

Weak example shapes:

- Research design.
- Help me with marketing.
- Make better AI context.
- Build a project.

Broad examples make the product sound vague. Narrow examples make the value obvious.

## Reduce Brand Repetition

Use "Liner" when naming or defining the product.

After the reader knows what the sentence is about, use:

- the tool
- the builder
- the TUI
- the saved folder
- the local folder
- the files

Too much brand repetition makes every paragraph sound like an ad. Good copy lets the artifact carry the message.

## Make File Names Proof, Not Jargon

File names are useful because they make the product inspectable.

Introduce them after the plain explanation:

> The folder contains saved sources, a compiled research packet, AI instructions, and status metadata. The main compiled file is `MIXTAPE.md`.

Bad order:

> Liner creates a Mixtape corpus, Operating Layer, root Project Skill, and status snapshot.

Good order:

> The tool saves sources and notes, compiles the research into `MIXTAPE.md`, and writes `LINER.md` and `SKILL.md` so a later AI session knows how to use the evidence.

The file names should prove the claim. They should not replace the explanation.

## Objections Should Become Practical Edges

FAQ-style concerns can stay on the page, but avoid literal question headings when the rest of the page can answer through copy.

Use statement cards:

- Manual pasting still works.
- Quick answers still have a place.
- File-reading assistants can use the whole folder.
- Other tools can use the compiled file.
- Your AI account stays yours.
- Private files stay on disk.
- Archives need a quick contents check.

This keeps the copy useful without making the page feel like a support transcript.

## Use Plain Contrasts

Contrasts are useful when they help the reader choose.

Good:

> A direct AI answer is enough when you only need the response. Use the saved-folder path when source choices, evidence trail, and task instructions need to survive for later work.

Bad:

> Liner is not just a research tool, it is an end-to-end project intelligence layer.

The bad version sounds bigger and explains less.

## Avoid These Patterns

Run a pass for these before shipping.

### Literal scaffolding

- "How is this different?"
- "What gets saved?"
- "What is inside?"
- "Use cases"
- "Questions"

These can be useful in planning notes, but they rarely belong as polished public section titles.

### Internal status copy

- Corpus Ready
- Project Complete
- PROJECT_COMPLETE
- Capability Brief
- SOURCE_ROLES
- JTBD
- Operating Layer, before it has been defined

### Vague prestige copy

- powerful
- seamless
- robust
- innovative
- cutting-edge
- unlock
- level up
- end-to-end
- source-grounded intelligence layer

### False contrast

- not just
- more than just
- rather than
- instead of
- not only

State the positive claim and stop.

### Over-polished AI phrasing

- the whole point
- the secret sauce
- the part that matters
- the path is obvious
- the future of
- your AI deserves

If it reads like a tagline for any AI product, cut it.

## The Rewrite Process

Use this process for homepage copy, docs intros, and route metadata.

1. Read the copy as a cold visitor.
2. List every noun the visitor has not learned yet.
3. Replace early internal nouns with plain objects.
4. Find every line that asks a question or labels the page mechanics.
5. Turn those lines into concrete states, examples, or file behavior.
6. Check every claim against current product reality.
7. Reduce repeated brand mentions.
8. Read the section order: product, pain, inputs, outputs, reuse, fit, install.
9. Run the detector grep.
10. Build the site and scan the rendered output, not only source files.

Rendered output matters because Astro frontmatter, metadata, layout chrome, docs pages, and terminal demos can keep stale copy after the body has been fixed.

## Verification Commands

Run these from the repo root.

```bash
npm --prefix marketing/site run build
git diff --check
rg -n "How is|How it|How does|What is|What gets|What changes|What Liner|Use cases|Questions|Can I|Do I|Does it|When is|Is it|Why this works|Example workflow|Example jobs|Good fit|Where it fits|The simple test|How they stack|Corpus Ready|Project Complete|PROJECT_COMPLETE|Capability Brief|JTBD|SOURCE_ROLES|Operating Layer|public library|planned library|five mixtapes|five projects|score push ready" marketing/site/src/pages/index.astro marketing/site/src/layouts/BaseLayout.astro marketing/site/dist/index.html
```

The grep can return hits inside audit or best-practices documents because those files preserve bad examples. It should not return hits from the live homepage source, layout metadata, or built homepage.

## Before And After Examples

### Product definition

Before:

> Build a project your AI can use.

After:

> Build reusable AI context.

Better body:

> Liner is a local CLI and terminal UI for turning focused research sources into files your AI assistant can use later. Add URLs, transcripts, PDFs, local files, or pasted text; it compiles the research and writes task-specific instructions.

### Saved work

Before:

> What gets saved?

After:

> The work leaves the chat.

### Direct answer comparison

Before:

> How is this different from asking Codex?

After:

> A direct AI answer can solve the immediate request. The saved folder keeps the setup that made the answer possible: which sources mattered, what limits were found, and what a later AI session should follow.

### Fit

Before:

> Good fit.

After:

> Worth the folder.

### FAQ

Before:

> Can I paste my own context into the chat myself?

After:

> Manual pasting still works.

## The Final Test

A future agent should be able to read the page and say:

- I know what this is.
- I know what problem it solves.
- I know what files it creates.
- I know how an AI assistant uses those files later.
- I know when a quick chat answer is enough.
- I did not need to learn internal product nouns before understanding the product.

If the page fails any of those, keep rewriting.
