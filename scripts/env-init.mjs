// Creates each app's local env file from its example on first run (`pnpm dev`, `pnpm db:*` call this).
// apps/api/.env feeds the Go API through dotenv-cli; apps/web/.env is read by Vite. Both are git-ignored.
import { copyFileSync, existsSync, mkdirSync, renameSync } from "node:fs";

// Before the per-app split there was a single root .env holding the API's values: move it, don't lose it.
if (existsSync(".env") && !existsSync("apps/api/.env")) {
  renameSync(".env", "apps/api/.env");
  console.log("env:init: moved the root .env to apps/api/.env");
}

for (const app of ["api", "web"]) {
  const file = `apps/${app}/.env`;
  if (!existsSync(file)) {
    copyFileSync(`${file}.example`, file);
    console.log(`env:init: created ${file} from ${file}.example`);
  }
}

mkdirSync(".data/media", { recursive: true });
