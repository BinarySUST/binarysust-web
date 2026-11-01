import base from "@binarysust/eslint-config/base";
import { restrictedImports, restrictedSyntax } from "@binarysust/eslint-config/rules";

export default [
  ...base,
  {
    // This package owns the API client: openapi-fetch and fetch('/api/...') are allowed only here.
    files: ["**/*.ts"],
    rules: {
      "no-restricted-imports": restrictedImports({ apiClient: true }),
      "no-restricted-syntax": restrictedSyntax({ apiClient: true }),
    },
  },
];
