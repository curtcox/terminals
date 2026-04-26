# Artifact Service

Typed operations for shared documents, canvases, templates, annotations, and lesson content.

Current typed operations are exposed via admin API endpoints used by REPL commands:

- `POST /admin/api/artifact/create`
- `POST /admin/api/artifact/patch`
- `POST /admin/api/artifact/replace`
- `GET /admin/api/artifact/get?artifact_id=<id>`
- `GET /admin/api/artifact/history?artifact_id=<id>`
- `POST /admin/api/artifact/template/save`
- `POST /admin/api/artifact/template/apply`
