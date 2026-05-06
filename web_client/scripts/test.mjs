#!/usr/bin/env node

import { spawnSync } from "node:child_process";
import { existsSync, readdirSync, statSync } from "node:fs";
import path from "node:path";

const root = process.cwd();
const args = process.argv.slice(2);
const nodeArgs = ["--test"];

function collectTestFiles(directory) {
  return readdirSync(directory, { withFileTypes: true })
    .flatMap((entry) => {
      const entryPath = path.join(directory, entry.name);
      if (entry.isDirectory()) return collectTestFiles(entryPath);
      return entry.name.endsWith(".test.js") ? [entryPath] : [];
    })
    .sort();
}

if (args.length === 0) {
  nodeArgs.push(...collectTestFiles(path.join(root, "test")));
}

for (const arg of args) {
  if (arg.startsWith("-")) {
    nodeArgs.push(arg);
    continue;
  }

  const testPath = path.join(root, "test", arg);
  if (existsSync(testPath) && statSync(testPath).isDirectory()) {
    nodeArgs.push(...collectTestFiles(testPath));
  } else {
    nodeArgs.push(arg);
  }
}

const result = spawnSync(process.execPath, nodeArgs, { stdio: "inherit" });
process.exit(result.status ?? 1);
