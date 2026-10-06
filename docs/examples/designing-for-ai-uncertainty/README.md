# designing-for-ai-uncertainty

A complete Liner Project from a real research run on 2 and 3 October 2026, kept as an example. It was built for this job:

> When an AI feature in my product can be wrong, I want to show how far to trust it and let people check its work, so they rely on it the right amount.

An AI reads `SKILL.md` first and `LINER.md` second, and opens `mixtape/MIXTAPE.md` and the sources only when an answer needs them. A person can follow the same route down to `mixtape/working/`, where the notes explain why each source was chosen.

| Path | What it holds |
| --- | --- |
| `SKILL.md` | Tells a compatible AI assistant when the project applies and points it to the method. |
| `LINER.md` | The working method, expected output, limits and a map to the research. |
| `liner.yaml` | The project's state for Liner. |
| `mixtape/MIXTAPE.md` | The combined findings and a source index noting why each source matters and where it falls short. |
| `mixtape/synthesis.md` | The synthesis on its own, approved in a separate review before it is compiled into `MIXTAPE.md`. |
| `mixtape/tape.yaml` | The project's manifest: each source's address, priority and kind, plus the job and the answers to Liner's clarifying questions. |
| `mixtape/sources/` | One file for each of the 30 approved sources. |
| `mixtape/working/` | The research record: the plan, the candidate long-list, the keep, trim and drop decisions, and the quality checks behind the selection. |
| `mixtape/local-sources/` | The source list Liner fetched from. |

## Source text is not included

Each file in `mixtape/sources/` keeps its title, address and curator note, but not the text Liner extracted, which belongs to its publishers. Follow the address to read the original. Liner's hidden run logs are also left out.
