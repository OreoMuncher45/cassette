#!/usr/bin/env bash
set -e

echo "=== Installing Cassette Spotify Client ==="

# Locate Go
GO_BIN=""
if command -v go >/dev/null 2>&1; then
    GO_BIN="go"
elif [ -x "$HOME/.local/go/bin/go" ]; then
    GO_BIN="$HOME/.local/go/bin/go"
elif [ -x "/usr/local/go/bin/go" ]; then
    GO_BIN="/usr/local/go/bin/go"
else
    echo "Error: Go compiler not found. Please install Go."
    exit 1
fi

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "Building cassette binary with $GO_BIN..."
$GO_BIN build -o cassette ./cmd/cassette

mkdir -p "$HOME/.local/bin"
cp -f cassette "$HOME/.local/bin/cassette"
chmod +x "$HOME/.local/bin/cassette"

echo "Installed binary to: $HOME/.local/bin/cassette"

# Check profile
PROFILE="minimal"
for arg in "$@"; do
    if [ "$arg" == "--full" ] || [ "$arg" == "-f" ]; then
        PROFILE="full"
    fi
done

# Check dependencies
if ! command -v librespot >/dev/null 2>&1 && [ ! -x "/usr/bin/librespot" ]; then
    echo "librespot not found. Attempting to install..."
    if command -v pacman >/dev/null 2>&1; then
        sudo pacman -S --noconfirm librespot || true
    elif command -v paru >/dev/null 2>&1; then
        paru -S --noconfirm librespot || true
    elif command -v yay >/dev/null 2>&1; then
        yay -S --noconfirm librespot || true
    fi
fi

if [ "$PROFILE" == "full" ]; then
    echo "Full profile selected: checking for chafa..."
    if ! command -v chafa >/dev/null 2>&1; then
        if command -v pacman >/dev/null 2>&1; then
            sudo pacman -S --noconfirm chafa || true
        elif command -v paru >/dev/null 2>&1; then
            paru -S --noconfirm chafa || true
        elif command -v yay >/dev/null 2>&1; then
            yay -S --noconfirm chafa || true
        fi
    fi
    if command -v chafa >/dev/null 2>&1; then
        echo "✓ chafa is installed ($(which chafa))"
    fi
fi

if command -v librespot >/dev/null 2>&1 || [ -x "/usr/bin/librespot" ]; then
    echo "✓ librespot is installed ($(which librespot 2>/dev/null || echo /usr/bin/librespot))"
else
    echo "⚠ Notice: librespot could not be automatically installed. Audio will play via Spotify Connect on other devices."
fi

echo ""
echo "=== Installation Complete! ==="
echo "You can now run:"
echo "  cassette setup    # (Optional) One-time Spotify Connect authorization for terminal audio"
echo "  cassette          # Launch the Cassette TUI player"
