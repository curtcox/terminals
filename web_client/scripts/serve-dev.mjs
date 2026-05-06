import { createServer } from "node:http";
import { createReadStream } from "node:fs";
import { stat } from "node:fs/promises";
import { extname, join, normalize, resolve } from "node:path";

const root = resolve(import.meta.dirname, "..");
const host = process.env.WEB_CLIENT_HOST || "0.0.0.0";
const port = Number(process.env.WEB_CLIENT_PORT || 60740);
const types = new Map([[".html", "text/html"], [".js", "text/javascript"], [".css", "text/css"]]);

createServer(async (req, res) => {
  const url = new URL(req.url, `http://${req.headers.host}`);
  const rel = normalize(url.pathname === "/" ? "/index.html" : url.pathname).replace(/^\/+/, "");
  const file = join(root, rel);
  if (!file.startsWith(root)) {
    res.writeHead(403).end();
    return;
  }
  try {
    const info = await stat(file);
    if (!info.isFile()) throw new Error("not file");
    res.writeHead(200, { "content-type": types.get(extname(file)) || "application/octet-stream" });
    createReadStream(file).pipe(res);
  } catch {
    res.writeHead(404).end("not found");
  }
}).listen(port, host, () => {
  console.log(`Terminals web client: http://${host}:${port}`);
});
