// Headless integration check: lists project folders and runs a compile on
// the first one. Not an Ink render — verifies IPC + YAML + bin-resolver work
// end-to-end against the new folder-model CLI.
import { resolveBinary } from "./bin-resolver.js";
import { listProjects, streamCompile } from "./ipc.js";
import { projectFolder, readSynthesisStatus, readTape } from "./yaml-io.js";

async function main(): Promise<void> {
  const bin = resolveBinary();
  console.log(`[smoke] binary: ${bin.command} (${bin.source})`);

  const dir = process.argv[2] || process.cwd();
  const projects = await listProjects(dir);
  console.log(`[smoke] found ${projects.length} project(s) in ${dir}:`);
  for (const p of projects) {
    console.log(
      `  • ${p.name}  "${p.title}"  (${p.source_count} sources, mode=${p.mode ?? "—"})`,
    );
  }

  if (projects.length === 0) {
    console.log("[smoke] no project folders to compile; done.");
    return;
  }

  const first = projects[0]!;
  const project = projectFolder(first.path);
  const synthesis = readSynthesisStatus(project);
  console.log(
    `[smoke] synthesis: exists=${synthesis.exists} ready=${synthesis.isReady} chars=${synthesis.charCount}`,
  );
  if (!synthesis.isReady) {
    console.log("[smoke] synthesis is not ready — compile would fail. Stopping.");
    return;
  }
  const { tape } = readTape(project.tapePath);
  console.log(`[smoke] re-read tape "${tape.title}" — ${tape.sources.length} source(s)`);

  console.log(`[smoke] streaming compile of ${project.path}…`);
  const handle = streamCompile({ folder: project.path }, (e) => {
    if (e.type === "source_start") {
      console.log(`  → starting ${e.spec.type} ${e.spec.url}`);
    } else if (e.type === "source_done" || e.type === "source_cached") {
      console.log(`    ${e.type === "source_cached" ? "cached" : "done"}: ${e.title}`);
    } else if (e.type === "source_failed") {
      console.log(`    failed: ${e.message}`);
    } else if (e.type === "finish") {
      console.log("  → finish");
    } else if (e.type === "result") {
      console.log(
        `[smoke] result: ${e.payload.summary.succeeded}/${e.payload.summary.total} succeeded ` +
          `→ ${e.payload.mixtape_path}`,
      );
    }
  });

  const { code } = await handle.done;
  console.log(`[smoke] compile exit ${code}`);
}

main().catch((e) => {
  console.error("[smoke] FAILED:", e);
  process.exit(1);
});
