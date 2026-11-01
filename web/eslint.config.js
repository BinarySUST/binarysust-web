import js from "@eslint/js";
import reactHooks from "eslint-plugin-react-hooks";
import globals from "globals";
import tseslint from "typescript-eslint";

// Go is the only backend and the site is static: server functions and server routes would have
// nothing to run on. See AGENTS.md.
const NO_SERVER_CODE = "The site is static and Go is the only backend. See AGENTS.md.";

export default tseslint.config(
  { ignores: ["dist", ".output", ".tanstack", ".wrangler", "node_modules", "src/routeTree.gen.ts", "src/lib/api/schema.d.ts", "public"] },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ["**/*.{ts,tsx}"],
    languageOptions: { globals: { ...globals.browser, ...globals.node } },
    plugins: { "react-hooks": reactHooks },
    rules: {
      ...reactHooks.configs.recommended.rules,
      "@typescript-eslint/no-unused-vars": ["error", { argsIgnorePattern: "^_", varsIgnorePattern: "^_", destructuredArrayIgnorePattern: "^_" }],
      "no-restricted-imports": [
        "error",
        {
          paths: [
            { name: "@tanstack/react-start", importNames: ["createServerFn", "createServerFileRoute", "useServerFn"], message: NO_SERVER_CODE },
            { name: "@tanstack/react-start/server", message: NO_SERVER_CODE },
            { name: "@tanstack/react-router", importNames: ["createServerFileRoute"], message: NO_SERVER_CODE },
          ],
        },
      ],
      "no-restricted-syntax": [
        "error",
        { selector: "Identifier[name=/^(createServerFn|createServerFileRoute)$/]", message: NO_SERVER_CODE },
        { selector: "Property[key.name='server'][value.type='ObjectExpression'] > ObjectExpression > Property[key.name='handlers']", message: NO_SERVER_CODE },
      ],
    },
  },
);
