#!/usr/bin/env bash
# cnav installer — builds the binary into ~/bin/cnav-bin and adds the shell
# function to your zshrc/bashrc if it isn't there yet.
set -euo pipefail

REPO_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
BIN_DIR="${CNAV_BIN_DIR:-$(go env GOPATH)/bin}"
BIN_PATH="$BIN_DIR/cnav-bin"

echo "→ building cnav-bin → $BIN_PATH"
mkdir -p "$BIN_DIR"
( cd "$REPO_DIR" && go build -o "$BIN_PATH" ./cmd/cnav )

# Pick a shell rc file.
RC=""
case "${SHELL:-}" in
  *zsh) RC="$HOME/.zshrc" ;;
  *bash) RC="$HOME/.bashrc" ;;
  *) RC="$HOME/.zshrc" ;;
esac

MARKER='# >>> cnav shell function >>>'
if grep -qF "$MARKER" "$RC" 2>/dev/null; then
  echo "→ wrapper already present in $RC"
else
  echo "→ adding cnav() function to $RC"
  {
    echo ""
    echo "$MARKER"
    echo 'eval "$('"$BIN_PATH"' init)"'
    echo '# <<< cnav shell function <<<'
  } >> "$RC"
fi

echo ""
echo "Done. Open a new shell (or 'source $RC') and run: cnav"
echo ""
echo "Make sure $BIN_DIR is on your PATH."
