// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import tailwindcss from "@tailwindcss/vite";
import { satteri } from "@astrojs/markdown-satteri";
import keyboardTables from "./src/plugins/keyboard-tables.mjs";

// https://astro.build/config
export default defineConfig({
  site: "https://www.liner.sh",
  server: { port: 4399, host: "127.0.0.1" },
  markdown: { processor: satteri({ hastPlugins: [keyboardTables] }) },
  integrations: [
    starlight({
      title: "Liner Docs",
      disable404Route: true,
      favicon: "/favicon.svg",
      logo: {
        src: "./public/liner-logo.svg",
        alt: "Liner",
      },
      customCss: ["./src/styles/starlight.css"],
      social: [
        {
          icon: "github",
          label: "GitHub",
          href: "https://github.com/cmdux-sh/liner",
        },
      ],
      components: {
        Header: "./src/components/starlight/Header.astro",
      },
      sidebar: [
        {
          label: "Start here",
          items: [
            { label: "Overview", slug: "docs" },
            { label: "Install", slug: "docs/install" },
            { label: "Build a Project", slug: "docs/build-a-mixtape" },
            { label: "Maintain with an Agent", slug: "docs/maintenance" },
          ],
        },
        {
          label: "Concepts",
          items: [
            { label: "Projects and Mixtapes", slug: "docs/mixtapes" },
            { label: "Tape format", slug: "docs/tape-format" },
            { label: "Sharing", slug: "docs/sharing" },
          ],
        },
        {
          label: "Reference",
          items: [
            { label: "CLI commands", slug: "docs/cli" },
            { label: "Terminal UI", slug: "docs/tui" },
            { label: "Troubleshooting", slug: "docs/troubleshooting" },
            { label: "Changelog", link: "/changelog/" },
          ],
        },
      ],
    }),
  ],
  vite: {
    plugins: [tailwindcss()],
  },
});
