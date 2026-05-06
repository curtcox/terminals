import { cp, mkdir, rm } from "node:fs/promises";
import { resolve } from "node:path";

const root = resolve(import.meta.dirname, "..");
const dist = resolve(root, "dist");
await rm(dist, { recursive: true, force: true });
await mkdir(dist, { recursive: true });
await cp(resolve(root, "index.html"), resolve(dist, "index.html"));
await cp(resolve(root, "src"), resolve(dist, "src"), { recursive: true });
await cp(
  resolve(root, "node_modules/@bufbuild/protobuf/dist/esm"),
  resolve(dist, "node_modules/@bufbuild/protobuf/dist/esm"),
  { recursive: true }
);
console.log(`Built static web client in ${dist}`);
