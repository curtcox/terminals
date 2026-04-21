#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SKILLS_DIR="${ROOT_DIR}/.claude/skills"

if [[ ! -d "${SKILLS_DIR}" ]]; then
  echo "missing skills directory: ${SKILLS_DIR}"
  exit 1
fi

errors=0
checked=0

while IFS= read -r -d '' dir; do
  checked=$((checked + 1))
  skill_name="$(basename "${dir}")"
  skill_file="${dir}/SKILL.md"

  if [[ ! -f "${skill_file}" ]]; then
    echo "ERROR: ${skill_name}: missing SKILL.md"
    errors=$((errors + 1))
    continue
  fi

  first_line="$(head -n 1 "${skill_file}" || true)"
  if [[ "${first_line}" != "---" ]]; then
    echo "ERROR: ${skill_name}: SKILL.md must start with YAML frontmatter ('---')"
    errors=$((errors + 1))
    continue
  fi

  frontmatter="$(
    awk '
      NR == 1 && $0 == "---" { in_fm = 1; next }
      in_fm && $0 == "---" { exit }
      in_fm { print }
    ' "${skill_file}"
  )"

  declared_name="$(
    printf '%s\n' "${frontmatter}" \
      | sed -nE 's/^name:[[:space:]]*"?([^"]+)"?[[:space:]]*$/\1/p' \
      | head -n 1
  )"
  description="$(
    printf '%s\n' "${frontmatter}" \
      | sed -nE 's/^description:[[:space:]]*(.+)$/\1/p' \
      | head -n 1
  )"

  if [[ -z "${declared_name}" ]]; then
    echo "ERROR: ${skill_name}: frontmatter is missing 'name'"
    errors=$((errors + 1))
  elif [[ "${declared_name}" != "${skill_name}" ]]; then
    echo "ERROR: ${skill_name}: frontmatter name '${declared_name}' does not match directory '${skill_name}'"
    errors=$((errors + 1))
  fi

  if [[ -z "${description}" ]]; then
    echo "ERROR: ${skill_name}: frontmatter is missing 'description'"
    errors=$((errors + 1))
  fi
done < <(find "${SKILLS_DIR}" -mindepth 1 -maxdepth 1 -type d -print0 | sort -z)

if [[ "${checked}" -eq 0 ]]; then
  echo "ERROR: no skill directories found in ${SKILLS_DIR}"
  exit 1
fi

if [[ "${errors}" -gt 0 ]]; then
  echo "skill validation failed with ${errors} error(s)"
  exit 1
fi

echo "skills validated (${checked} skill directories)"
