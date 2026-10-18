# Decisions

## 2026-10-17 - Improve the complete public website

Arturo asked to complete copy, design, accessibility, and performance improvements across all seven project websites, following the Xipher work. For Liner, cover the homepage, documentation, changelog, metadata, social image, and shared interface feedback. Use Arturo Writer with multiple OpenAI agents, with no Anthropic agents and no max or ultra effort. Keep the existing terminal visual direction, source-backed product vocabulary, and private development remote. Website implementation does not authorize CLI behavior changes, package publication, public mirroring, or marketing deployment. Preserve project acceptance and publication requirements.

## 2026-10-17 - Keep the homepage headline on two lines

Arturo's screenshot identified the mobile-only break between “AI” and “context.” as an unwanted third line. Preserve the approved wording, color roles, and type scale, with “Build reusable” on the first line and “AI context.” on the second. Remove the explicit break and its unused responsive CSS; let the existing responsive typography handle the supported widths.


## 2026-10-18 - Accept the website and authorize merge

Arturo reviewed the project websites, said they all look good, and authorized merging the completed website work where appropriate so the live sites can be checked. This records visual acceptance of the reviewed website outcome at ea0b5e0 and authorizes this PR's merge after its checks pass, including any deployment triggered by the existing hosting integration. The reason is to make the accepted work available for live-site verification. Reuse the recorded functional verification because this follow-up changes documentation only. Physical assistive-technology and field-performance limits remain as documented; visual acceptance does not establish those checks. Manual production deployment, package publication, public mirroring, native/App Store release, and corpus/provider operations are outside this approval. Preserve the running preview and task worktree while Arturo is still using them; cleanup is deferred.
