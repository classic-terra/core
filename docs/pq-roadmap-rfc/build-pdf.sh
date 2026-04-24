#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_PATH="${1:-$SCRIPT_DIR/pq-roadmap-rfc.pdf}"
PDF_ENGINE="${PDF_ENGINE:-xelatex}"
DOC_TITLE="${DOC_TITLE:-Post-Quantum Roadmap RFC for Terra Classic}"
FONT_MODE="${FONT_MODE:-sans}"

case "$FONT_MODE" in
  sans)
    HEADER_FILE="$SCRIPT_DIR/pandoc-header.tex"
    ;;
  mono)
    HEADER_FILE="$SCRIPT_DIR/pandoc-header-mono.tex"
    ;;
  *)
    echo "Error: unsupported FONT_MODE '$FONT_MODE' (use 'sans' or 'mono')." >&2
    exit 1
    ;;
esac

INPUT_FILES=(
  "$SCRIPT_DIR/00-scope-note.md"
  "$SCRIPT_DIR/01-executive-summary.md"
  "$SCRIPT_DIR/02-cryptographic-signatures-explained.md"
  "$SCRIPT_DIR/03-current-state-and-terminology.md"
  "$SCRIPT_DIR/04-target-states-and-options.md"
  "$SCRIPT_DIR/05-roadmap-phases.md"
  "$SCRIPT_DIR/06-audit-gates.md"
  "$SCRIPT_DIR/07-live-chain-migration-paths.md"
  "$SCRIPT_DIR/08-governance-communication-rfc-process.md"
  "$SCRIPT_DIR/09-open-decisions-decision-log.md"
  "$SCRIPT_DIR/10-comment-log.md"
)

declare -A CHAPTER_KEY_BY_FILE=(
  ["$SCRIPT_DIR/00-scope-note.md"]="scope-note"
  ["$SCRIPT_DIR/01-executive-summary.md"]="executive-summary"
  ["$SCRIPT_DIR/02-cryptographic-signatures-explained.md"]="signatures-explained"
  ["$SCRIPT_DIR/03-current-state-and-terminology.md"]="current-state-terminology"
  ["$SCRIPT_DIR/04-target-states-and-options.md"]="target-states-options"
  ["$SCRIPT_DIR/05-roadmap-phases.md"]="roadmap-phases"
  ["$SCRIPT_DIR/06-audit-gates.md"]="audit-gates"
  ["$SCRIPT_DIR/07-live-chain-migration-paths.md"]="live-chain-migration-paths"
  ["$SCRIPT_DIR/08-governance-communication-rfc-process.md"]="governance-process"
  ["$SCRIPT_DIR/09-open-decisions-decision-log.md"]="decision-log"
  ["$SCRIPT_DIR/10-comment-log.md"]="comment-log"
)

declare -A CHAPTER_ANCHOR_BY_KEY=(
  ["scope-note"]="ch-scope-note"
  ["executive-summary"]="ch-executive-summary"
  ["signatures-explained"]="ch-signatures-explained"
  ["current-state-terminology"]="ch-current-state-terminology"
  ["target-states-options"]="ch-target-states-options"
  ["roadmap-phases"]="ch-roadmap-phases"
  ["audit-gates"]="ch-audit-gates"
  ["live-chain-migration-paths"]="ch-live-chain-migration-paths"
  ["governance-process"]="ch-governance-process"
  ["decision-log"]="ch-decision-log"
  ["comment-log"]="ch-comment-log"
)

if ! command -v pandoc >/dev/null 2>&1; then
  echo "Error: pandoc not found in PATH." >&2
  exit 1
fi

for file in "${INPUT_FILES[@]}"; do
  if [[ ! -f "$file" ]]; then
    echo "Error: missing input file: $file" >&2
    exit 1
  fi
done

if [[ ! -f "$HEADER_FILE" ]]; then
  echo "Error: missing header file: $HEADER_FILE" >&2
  exit 1
fi

mkdir -p "$(dirname "$OUTPUT_PATH")"

declare -A CHAPTER_NUMBER_BY_KEY
chapter_number=1
for file in "${INPUT_FILES[@]}"; do
  key="${CHAPTER_KEY_BY_FILE[$file]:-}"
  if [[ -n "$key" ]]; then
    CHAPTER_NUMBER_BY_KEY["$key"]="$chapter_number"
  fi
  chapter_number=$((chapter_number + 1))
done

TMP_DIR="$(mktemp -d)"
trap 'rm -rf "$TMP_DIR"' EXIT
PROCESSED_FILES=()

for file in "${INPUT_FILES[@]}"; do
  out_file="$TMP_DIR/$(basename "$file")"
  cp "$file" "$out_file"

  for key in "${!CHAPTER_NUMBER_BY_KEY[@]}"; do
    number="${CHAPTER_NUMBER_BY_KEY[$key]}"
    anchor="${CHAPTER_ANCHOR_BY_KEY[$key]:-}"
    if [[ -z "$anchor" ]]; then
      echo "Error: missing anchor mapping for chapter key '$key'" >&2
      exit 1
    fi
    perl -0pi -e "s/\\{\\{chapter:$key\\}\\}/[$number](#$anchor)/g" "$out_file"
  done

  if grep -Eq '\{\{chapter:[a-z0-9-]+\}\}' "$out_file"; then
    echo "Error: unresolved chapter reference token in $file" >&2
    grep -En '\{\{chapter:[a-z0-9-]+\}\}' "$out_file" >&2
    exit 1
  fi

  PROCESSED_FILES+=("$out_file")
done

pandoc \
  "${PROCESSED_FILES[@]}" \
  --from=markdown+raw_attribute \
  --toc \
  --number-sections \
  --metadata "title=$DOC_TITLE" \
  --include-in-header "$HEADER_FILE" \
  --pdf-engine="$PDF_ENGINE" \
  -o "$OUTPUT_PATH"

echo "PDF generated: $OUTPUT_PATH"
