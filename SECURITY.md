# Security

Liner runs on your machine, makes no AI calls, and reads text from strangers on the
internet. This page says what it protects against and what it doesn't. Report a
problem by opening an issue at https://github.com/cmdux-sh/liner/issues, and leave
out exploit details.

## What Liner protects against

**Pages and sources**

- `liner fetch` saves a page to a file and returns its path, so its text stays out of
  the agent's session until the agent opens it.
- Saved pages and research prompts tell the agent that sources are evidence, not
  instructions. On Claude Code, the `liner-researcher` subagent, which comes with the
  `liner` skill unless you opt out, reads saved pages with only `Read` and `Grep`.
- Fetches refuse this machine, private networks, and cloud metadata addresses,
  including through redirects, unless you set `allow_private_network = true`.
- Web pages and PDFs stop at 50 MB, local files at 10 MB, remote skill files at 2 MB,
  and repository documents at 512 KB each and 2 MB in total.

**Shared liners and tapes**

- `liner import` refuses unsafe paths and oversized archives, won't replace a folder
  without `--overwrite`, and drops agent transcripts.
- Local files must sit in the liner's `local-sources/` or `personal/` folder. Local
  skills must sit inside the liner, in an installed skills folder (which includes
  `.agents/skills` in the current folder), or in a folder listed in `LINER_SKILL_PATHS`.

**Your liners**

- Each approval is tied to the exact content reviewed, so an edit reopens it.
- Finished liners change through Change Sets that are checked against the current
  content, applied atomically, and recorded in a receipt.
- `liner mcp` has no tool for approving research, deciding a finding, recording a
  verdict, or applying a change. It isn't read-only: it can fetch pages into a liner,
  start a liner, submit research steps, and add gaps and findings.
- The terminal navigator can't change a liner.

## What Liner doesn't protect against

- **A determined agent.** Liner can't tell a person from an agent, and its receipts
  don't record who approved. An agent with a shell can run an approval command.
  On Claude Code, the `liner` skill install adds rules that ask you first, and a hook that
  reads the whole command, so `bash -c '...'`, `env`, or a full path to `liner` asks too.
  An agent can still get past it, for example by writing a script that runs the command,
  and Codex has no equivalent.
  Agents can also write the answers and verdicts they report as yours.
- **An agent that wants to see a blind test's answer key.** Blind tests keep you from
  seeing which answer used the liner. The key is in a file in the liner, so an agent
  with a shell can read it. `liner mcp` refuses that file.
- **Prompt injection.** Liner doesn't detect it or strip hidden text. Instructions in a
  page can reach the agent if it opens that page, and an agent can use its own web tools
  instead of `liner fetch`. Compiled sources are only labeled as reference material
  when they come from a skill. Fetch errors can include up to 500 bytes of the page.
- **Reading files through the subagent.** `liner-researcher` has no shell, network, or
  write tools, but it can read local files, and its report goes to the main agent.
- **False claims in a synthesis.** Checks confirm that each citation points to a kept
  source. They don't confirm that the source says what the claim says, and they trust
  what the agent reports about its evaluation.
- **Every route to a private address.** A hostname can change its address after Liner
  checks it, a proxy resolves names itself, and in the browser a redirect that only
  appears for the browser, or one on a subresource, can reach a private address once.
- **Slow servers.** The size limits don't limit time, so a slow server can hold a fetch
  open for as long as it keeps sending.
- **What your agent does.** Liner doesn't sandbox your agent. Its access to your files,
  shell, and network comes from the agent's own settings.
