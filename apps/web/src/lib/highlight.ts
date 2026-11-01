// Syntax colours for the code fences in rendered markdown. Pure string in, string out, so the pre-render and the
// browser produce the same HTML and hydration matches.
import { type LanguageName, highlight } from "sugar-high";

const ALIAS: Record<string, LanguageName> = {
  js: "javascript",
  jsx: "javascript",
  javascript: "javascript",
  ts: "typescript",
  tsx: "typescript",
  typescript: "typescript",
  go: "go",
  golang: "go",
  py: "python",
  python: "python",
  c: "c",
  cpp: "cpp",
  "c++": "cpp",
  cc: "cpp",
  java: "java",
  rust: "rust",
  rs: "rust",
  json: "json",
  sh: "shell",
  bash: "shell",
  shell: "shell",
  console: "shell",
  sql: "sql",
  yaml: "yaml",
  yml: "yaml",
  css: "css",
  html: "html",
  kotlin: "kotlin",
  swift: "swift",
  php: "php",
  diff: "diff",
  toml: "toml",
  dockerfile: "dockerfile",
};

const NAMED: Record<string, string> = { lt: "<", gt: ">", quot: '"', apos: "'", amp: "&", nbsp: "\u00a0" };
/** Undo the renderer's HTML escaping (named, decimal and hex entities); sugar-high escapes its own output. */
const unescape = (s: string) =>
  s.replace(/&(#x[0-9a-f]+|#\d+|[a-z]+);/gi, (m, e: string) => {
    if (e[0] === "#") return String.fromCodePoint(e[1] === "x" || e[1] === "X" ? parseInt(e.slice(2), 16) : Number(e.slice(1)));
    return NAMED[e.toLowerCase()] ?? m;
  });

const FENCE = /<pre><code(?: class="language-([\w+#-]+)")?>([\s\S]*?)<\/code><\/pre>/g;

export function highlightHtml(html: string): string {
  return html.replace(FENCE, (_m, lang: string | undefined, body: string) => {
    const name = lang ? ALIAS[lang.toLowerCase()] : undefined;
    const out = highlight(unescape(body).replace(/\n$/, ""), name ? { lang: name } : undefined);
    const cls = lang ? ` class="language-${lang}"` : "";
    return `<pre><code${cls}>${out}</code></pre>`;
  });
}
