// React flat config (apps and React packages): base + hooks rules + browser globals.
// Apps get the static-site bans too (no server functions or server routes: see apps/web/AGENTS.md).
import reactHooks from "eslint-plugin-react-hooks";
import globals from "globals";
import tseslint from "typescript-eslint";
import base from "./base.js";
import { restrictedImports, restrictedSyntax } from "./rules.js";

/** @type {import("typescript-eslint").ConfigArray} */
export default tseslint.config(...base, {
  files: ["**/*.{ts,tsx}"],
  languageOptions: { globals: { ...globals.browser, ...globals.node } },
  plugins: { "react-hooks": reactHooks },
  rules: {
    ...reactHooks.configs.recommended.rules,
    "no-restricted-imports": restrictedImports({ staticSite: true }),
    "no-restricted-syntax": restrictedSyntax({ staticSite: true }),
  },
});
