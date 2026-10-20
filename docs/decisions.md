# Decisions

## 2026-10-17 - Improve the complete public website

Arturo asked to complete copy, design, accessibility, and performance improvements across all seven project websites, following the Xipher work. For Liner, cover the homepage, documentation, changelog, metadata, social image, and shared interface feedback. Use Arturo Writer with multiple OpenAI agents, with no Anthropic agents and no max or ultra effort. Keep the existing terminal visual direction, source-backed product vocabulary, and private development remote. Website implementation does not authorize CLI behavior changes, package publication, public mirroring, or marketing deployment. Preserve project acceptance and publication requirements.

## 2026-10-17 - Keep the homepage headline on two lines

Arturo's screenshot identified the mobile-only break between “AI” and “context.” as an unwanted third line. Preserve the approved wording, color roles, and type scale, with “Build reusable” on the first line and “AI context.” on the second. Remove the explicit break and its unused responsive CSS; let the existing responsive typography handle the supported widths.


## 2026-10-18 - Accept the website and authorize merge

Arturo reviewed the project websites, said they all look good, and authorized merging the completed website work where appropriate so the live sites can be checked. This records visual acceptance of the reviewed website outcome at ea0b5e0 and authorizes this PR's merge after its checks pass, including any deployment triggered by the existing hosting integration. The reason is to make the accepted work available for live-site verification. Reuse the recorded functional verification because this follow-up changes documentation only. Physical assistive-technology and field-performance limits remain as documented; visual acceptance does not establish those checks. Manual production deployment, package publication, public mirroring, native/App Store release, and corpus/provider operations are outside this approval. Preserve the running preview and task worktree while Arturo is still using them; cleanup is deferred.

## 2026-10-19 - Agents check, the person decides, at five decision points

From the Liner 1.1.1 case-study run, Arturo set the principle that Liner's agents handle what another agent can check, while the person makes the decisions that depend on them (which Sources to trust, what the synthesis says, what to do about what is missing), with the agents' notes and reasons on screen. Liner never makes one of those decisions for the person and never hides the record when they decide. Applied here: repair never starts the browser download (installing stays a separate offer with its measured size); setup installs only the headless browser Liner launches; Review Draft Sources shows each Source's full agent note and how much of it the agents could read; the synthesis and Project Skill editors accept lines at any length with the cursor visible; continuing without reviewed failed Sources works from the Project screen with every key listed; and the improvement offer follows the latest status, with plain-word gaps. Merge, npm publication, public mirroring, release, and website deployment need separate approval.

## 2026-10-20 - Move Linux CI to the bounded Depot free trial

Arturo approved migrating Linux CI to Depot through 2026-10-27T16:48:00Z, the same cutoff as Akast, to avoid automatic GitHub-hosted Actions usage. Preserve the original checks and manual GitHub fallback, keep macOS and Windows local/manual, and guard every Depot job before checkout. No paid upgrade, package publication, public mirroring, provider operation, or deployment is authorized. See [the trial coverage and rollback record](ci/depot-ci-trial.md).

## 2026-10-20 - Release 1.1.2 in order: merge, npm, mirror, then site

Arturo approved merging PR 3 and set the release order: merge, npm release, public mirror, then site deploy, each approved separately. Because a `master` merge deploys the site to production through Vercel, Arturo chose to pause Vercel's automatic production deploys for `liner-site` first, so the site cannot describe 1.1.2 behavior before npm serves it. The pause is the project's Ignored Build Step, which skips only production builds (`VERCEL_ENV=production`); previews still build. The site deploy step clears that setting (it was empty before) and deploys `master` to production.

## 2026-10-20 - Default to Sol 6.1 and Sonnet 5.5 at Medium effort

Arturo asked to add Sol 6.1 and replace the older Claude models with Sonnet 5.5 and Opus 5.5, with both defaults at Medium effort. OpenAI Auto now runs Sol 6.1 (`gpt-6.1-sol`) at Medium for every task, replacing the Luna High and Sol Medium phase split, so one model is easy to reason about. Settings offers only the new models: Auto, OpenAI default, Sol 6.1, and Custom for OpenAI; Sonnet 5.5, Opus 5.5, and Custom for Claude. Claude gets a selectable Thinking effort (Claude Code's `--effort`, Low through Maximum) that defaults to Medium. A saved GPT-5.6 model keeps working as a custom ID, and saved `sonnet` or `opus` aliases read as the 5.5 models. Built-in defaults that a provider rejects retry once on the provider's own default; choices made in Settings never fall back silently.

## 2026-10-20 - Follow the publish checklist for 1.1.2

The release checklist builds all six npm packages from the public `cmdux-sh/liner` repository's Platform Bundles workflow, so the public mirror has to come before npm. Depot runs only Linux, and this private repository's GitHub-hosted runners are blocked by a billing problem, so macOS Intel and Windows can't be built here. Arturo chose to follow `packages/tui/PUBLISH-CHECKLIST.md`: bump to 1.1.2, mirror the release commit to the public repository, build and publish the six packages, then deploy the site. Each publication step still needs Arturo's approval. The site stays paused until it is the last step.

## 2026-10-20 - Mirror 1.1.2 publicly without internal docs

Arturo approved mirroring the 1.1.2 release to the public `cmdux-sh/liner` repository. The mirror merges private `master` into public `main` through a public pull request, keeps the public-only example project (`docs/examples/designing-for-ai-uncertainty/`), and reconciles the two Depot CI changes. Internal working docs stay private: `AGENTS.md` (it names the private repository), `docs/agents/`, and the website copy deck, language notes, performance audit, review, and truth audit (`docs/liner-website-*.md`). npm publication and the site deploy still need their own approval.
