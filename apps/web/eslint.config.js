import react from "@binarysust/eslint-config/react";

export default [
  ...react,
  { ignores: ["public/**", "src/routeTree.gen.ts"] },
];
