import { existsSync, readFileSync } from "node:fs";

const URL_RE = /https?:\/\/[^\s"'<>]+/g;

export type LonglistCandidate = {
  url: string;
  title?: string;
  section?: string;
  reason?: string;
};

export type LonglistCandidateGroup = {
  index: number;
  total: number;
  section: string;
  slug: string;
  fragmentPath: string;
  candidates: LonglistCandidate[];
};

export function countCandidateUrlsFromLonglist(path: string): number | undefined {
  const candidates = readCandidateLonglist(path);
  return candidates.length > 0 ? candidates.length : undefined;
}

export function readCandidateLonglist(path: string): LonglistCandidate[] {
  if (!existsSync(path)) return [];
  const candidates: LonglistCandidate[] = [];
  const seen = new Set<string>();
  let section = "";
  let candidateTitle = "";
  let current: LonglistCandidate | undefined;

  for (const line of readFileSync(path, "utf8").split("\n")) {
    const sectionHeading = line.match(/^##\s+(.+?)\s*$/);
    if (sectionHeading) {
      section = cleanSection(sectionHeading[1] ?? "");
      candidateTitle = "";
      current = undefined;
      continue;
    }

    const candidateHeading = line.match(/^###\s+(.+?)\s*$/);
    if (candidateHeading) {
      candidateTitle = candidateHeading[1]?.trim() ?? "";
      current = undefined;
      continue;
    }

    const url = firstUrl(line);
    if (url) {
      if (seen.has(url)) {
        current = candidates.find((candidate) => candidate.url === url);
        continue;
      }
      current = {
        url,
        title: candidateTitle || titleFromCandidateLine(line, url),
        section: section || undefined,
      };
      candidates.push(current);
      seen.add(url);
      candidateTitle = "";
      continue;
    }

    const reason = line.match(/^\s*[-*]\s*(?:Candidate\s+reason|Reason|Rationale):\s*(.+?)\s*$/i);
    if (reason && current) {
      current.reason = reason[1]?.trim();
    }
  }

  return candidates;
}

export function groupCandidateLonglist(
  candidates: LonglistCandidate[],
  maxCandidatesPerGroup = 10,
): LonglistCandidateGroup[] {
  const groupSize = Math.max(1, Math.floor(maxCandidatesPerGroup));
  const bySection = new Map<string, LonglistCandidate[]>();
  for (const candidate of candidates) {
    const section = candidate.section || "Unsectioned";
    const sectionCandidates = bySection.get(section) ?? [];
    sectionCandidates.push(candidate);
    bySection.set(section, sectionCandidates);
  }

  const pending: Array<Omit<LonglistCandidateGroup, "index" | "total" | "fragmentPath">> = [];
  for (const [section, sectionCandidates] of bySection) {
    for (let i = 0; i < sectionCandidates.length; i += groupSize) {
      const chunk = sectionCandidates.slice(i, i + groupSize);
      const chunkNumber = Math.floor(i / groupSize) + 1;
      const chunkCount = Math.ceil(sectionCandidates.length / groupSize);
      const suffix = chunkCount > 1 ? `-${chunkNumber}` : "";
      pending.push({
        section,
        slug: `${slugify(section)}${suffix}`,
        candidates: chunk,
      });
    }
  }

  const total = pending.length;
  return pending.map((group, i) => ({
    ...group,
    index: i + 1,
    total,
    fragmentPath: `working/evaluation-decisions/${String(i + 1).padStart(2, "0")}-${group.slug}.yaml`,
  }));
}

export function normalizeCandidateUrl(url: string): string {
  return url.trim().replace(/[)\],.;:]+$/g, "");
}

function firstUrl(line: string): string {
  const match = line.match(URL_RE);
  return match?.[0] ? normalizeCandidateUrl(match[0]) : "";
}

function cleanSection(section: string): string {
  return section.trim().replace(/^\d+\.\s+/, "");
}

function titleFromCandidateLine(line: string, url: string): string | undefined {
  const beforeUrl = line.slice(0, line.indexOf(url));
  const cleaned = beforeUrl
    .replace(/^\s*[-*]\s*/, "")
    .replace(/\*\*/g, "")
    .replace(/[-—:|\s]+$/g, "")
    .trim();
  return cleaned || undefined;
}

function slugify(value: string): string {
  const slug = value
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");
  return slug || "section";
}
