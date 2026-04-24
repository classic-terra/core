#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" && pwd)"
OUTPUT_PATH="${1:-$SCRIPT_DIR/pq-roadmap-rfc.pdf}"
PDF_ENGINE="${PDF_ENGINE:-xelatex}"
DOC_TITLE="${DOC_TITLE:-PQ Roadmap RFC fuer Terra Classic}"
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
  "$SCRIPT_DIR/01-executive-summary.md"
  "$SCRIPT_DIR/02-cryptographic-signatures-explained.md"
  "$SCRIPT_DIR/03-current-state-and-terminology.md"
  "$SCRIPT_DIR/04-target-states-and-options.md"
  "$SCRIPT_DIR/05-roadmap-phases.md"
  "$SCRIPT_DIR/06-audit-gates.md"
  "$SCRIPT_DIR/07-live-chain-migration-paths.md"
  "$SCRIPT_DIR/08-governance-communication-rfc-process.md"
  "$SCRIPT_DIR/09-open-decisions-decision-log.md"
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

pandoc \
  "${INPUT_FILES[@]}" \
  --from=gfm+raw_attribute \
  --toc \
  --number-sections \
  --metadata "title=$DOC_TITLE" \
  --include-in-header "$HEADER_FILE" \
  --pdf-engine="$PDF_ENGINE" \
  -o "$OUTPUT_PATH"

echo "PDF generated: $OUTPUT_PATH"
