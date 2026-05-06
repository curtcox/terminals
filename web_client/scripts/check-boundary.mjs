import { readFile } from "node:fs/promises";
import { resolve } from "node:path";
import { globSync } from "node:fs";

const root = resolve(import.meta.dirname, "..");
const forbidden = [
  /scenario/i,
  /placement/i,
  /claims/i,
  /repl/i,
  /mcp/i,
  /appruntime/i,
  /transport\/control_socket/
];

let failed = false;
for (const path of globSync("src/ui/**/*.js", { cwd: root })) {
  const text = await readFile(resolve(root, path), "utf8");
  for (const line of text.split("\n")) {
    if (!line.trim().startsWith("import")) continue;
    const bad = forbidden.find((pattern) => pattern.test(line));
    if (bad) {
      console.error(`${path}: forbidden UI import: ${line}`);
      failed = true;
    }
  }
}

if (failed) process.exit(1);
console.log("web client boundary ok");
