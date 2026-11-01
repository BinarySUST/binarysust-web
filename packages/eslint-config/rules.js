// The repository's boundary rules (docs/adr/0001-monorepo-structure.md, "Rules"), in one place.
// `base.js` and `react.js` compose them; a package that owns a concern (ui owns primitives,
// api-client owns fetch) re-composes them with that one allowance via `restrictedImports(...)`.
import { fileURLToPath } from "node:url";

/** The monorepo root, so path zones work the same from every package. */
export const repoRoot = fileURLToPath(new URL("../..", import.meta.url));

const ADR = "See docs/adr/0001-monorepo-structure.md.";

/** Rule 2: apps -> packages only; apps/web and apps/api talk only through @binarysust/api-client. */
export const pathZones = {
  basePath: repoRoot,
  zones: [
    { target: "./packages", from: "./apps", message: `Packages never import apps. ${ADR}` },
    {
      target: "./apps/web",
      from: "./apps/api",
      message: `apps/web reaches the Go API only through @binarysust/api-client (generated from openapi.json). ${ADR}`,
    },
  ],
};

const appPackages = {
  group: ["@binarysust/web", "@binarysust/web/*"],
  message: `Nothing imports an app. Move shared code into packages/*. ${ADR}`,
};

const apiClientInternals = {
  group: ["openapi-fetch", "openapi-fetch/*"],
  message: `Use the one client from @binarysust/api-client. ${ADR}`,
};

/** Rule 3: Base UI / shadcn primitives live in @binarysust/ui; apps compose them. */
const uiPrimitives = {
  group: ["@base-ui/react", "@base-ui/react/*", "@radix-ui/*", "radix-ui", "sonner", "vaul", "cmdk", "~/components/ui/*"],
  message: `UI primitives come from @binarysust/ui (add one with \`pnpm dlx shadcn@latest add <name>\` in apps/web). ${ADR}`,
};

const NO_SERVER_CODE = "The site is static and Go is the only backend. See apps/web/AGENTS.md.";

/** The site builds to static files: server functions and server routes would have nothing to run on. */
const serverCodePaths = [
  {
    name: "@tanstack/react-start",
    importNames: ["createServerFn", "createServerFileRoute", "useServerFn"],
    message: NO_SERVER_CODE,
  },
  { name: "@tanstack/react-start/server", message: NO_SERVER_CODE },
  { name: "@tanstack/react-router", importNames: ["createServerFileRoute"], message: NO_SERVER_CODE },
];

/**
 * `no-restricted-imports` can be configured only once per file, so every restriction is composed here.
 * @param {{ uiPrimitives?: boolean, apiClient?: boolean, staticSite?: boolean }} allow what this package owns
 */
export function restrictedImports({ uiPrimitives: ownsUi = false, apiClient = false, staticSite = false } = {}) {
  const patterns = [appPackages];
  if (!apiClient) patterns.push(apiClientInternals);
  if (!ownsUi) patterns.push(uiPrimitives);
  return ["error", { paths: staticSite ? serverCodePaths : [], patterns }];
}

const API_FETCH = `Call the Go API through @binarysust/api-client, never fetch('/api/...'). ${ADR}`;

/** Rule 4 (fetch('/api/...'), window.fetch(`/api/${x}`)) plus, for the static site, server routes. */
export function restrictedSyntax({ apiClient = false, staticSite = false } = {}) {
  const selectors = [];
  if (!apiClient) {
    for (const callee of ["[callee.name='fetch']", "[callee.property.name='fetch']"]) {
      selectors.push(
        { selector: `CallExpression${callee}[arguments.0.value=/^\\x2Fapi(\\x2F|$)/]`, message: API_FETCH },
        { selector: `CallExpression${callee}[arguments.0.quasis.0.value.raw=/^\\x2Fapi(\\x2F|$)/]`, message: API_FETCH },
      );
    }
  }
  if (staticSite) {
    selectors.push(
      { selector: "Identifier[name=/^(createServerFn|createServerFileRoute)$/]", message: NO_SERVER_CODE },
      {
        selector: "Property[key.name='server'][value.type='ObjectExpression'] > ObjectExpression > Property[key.name='handlers']",
        message: NO_SERVER_CODE,
      },
    );
  }
  return ["error", ...selectors];
}
