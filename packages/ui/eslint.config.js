import react from "@binarysust/eslint-config/react";
import { restrictedImports } from "@binarysust/eslint-config/rules";

export default [
  ...react,
  {
    // This package owns the UI primitives (Base UI via shadcn/ui).
    files: ["**/*.{ts,tsx}"],
    rules: { "no-restricted-imports": restrictedImports({ uiPrimitives: true }) },
  },
];
