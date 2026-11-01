// Base flat config for every TypeScript workspace package: recommended rules + the import boundaries.
import js from "@eslint/js";
import { createTypeScriptImportResolver } from "eslint-import-resolver-typescript";
import importX from "eslint-plugin-import-x";
import globals from "globals";
import tseslint from "typescript-eslint";
import { pathZones, restrictedImports, restrictedSyntax } from "./rules.js";

export const ignores = {
  ignores: [
    "**/dist/**",
    "**/.output/**",
    "**/.tanstack/**",
    "**/.wrangler/**",
    "**/.turbo/**",
    "**/coverage/**",
    "**/playwright-report/**",
    "**/test-results/**",
    "**/*.gen.ts",
    "**/*.d.ts",
  ],
};

/** @type {import("typescript-eslint").ConfigArray} */
export default tseslint.config(
  ignores,
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{js,mjs,ts,tsx}"],
    languageOptions: { globals: { ...globals.node } },
    plugins: { "import-x": importX },
    settings: {
      "import-x/resolver-next": [createTypeScriptImportResolver({ alwaysTryTypes: true })],
    },
    rules: {
      "@typescript-eslint/no-unused-vars": [
        "error",
        { argsIgnorePattern: "^_", varsIgnorePattern: "^_", destructuredArrayIgnorePattern: "^_" },
      ],
      "import-x/no-restricted-paths": ["error", pathZones],
      "no-restricted-imports": restrictedImports(),
      "no-restricted-syntax": restrictedSyntax(),
    },
  },
);
