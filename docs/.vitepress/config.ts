import { defineConfig } from "vitepress";

export default defineConfig({
  title: "skival",
  description:
    "A Go CLI for evaluating AI coding skill performance. Measures time, cost, tokens, and correctness across configurable eval suites.",
  base: "/skival/",
  cleanUrls: true,
  themeConfig: {
    nav: [
      { text: "Getting Started", link: "/getting-started" },
      { text: "Configuration", link: "/configuration" },
      { text: "CLI", link: "/cli" },
      { text: "Verifiers", link: "/verifiers" },
    ],
    sidebar: [
      {
        text: "Guide",
        items: [
          { text: "Introduction", link: "/" },
          { text: "Getting Started", link: "/getting-started" },
        ],
      },
      {
        text: "Reference",
        items: [
          { text: "Configuration", link: "/configuration" },
          { text: "CLI", link: "/cli" },
          { text: "Verifiers", link: "/verifiers" },
        ],
      },
    ],
    socialLinks: [
      { icon: "github", link: "https://github.com/driangle/skival" },
    ],
  },
  srcExclude: ["PLAN.md", "specs/**"],
});
