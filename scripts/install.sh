#!/bin/bash
set -e

# Jarvis Installation Script
# This script builds and installs Jarvis components

JARVIS_DIR="$HOME/.jarvis"
BIN_DIR="$JARVIS_DIR/bin"
CONFIG_FILE="$JARVIS_DIR/config.yaml"
LAUNCHD_PLIST="$HOME/Library/LaunchAgents/com.jarvis.daemon.plist"

echo "=== Jarvis Installation ==="
echo ""

# Get the directory where this script is located
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

cd "$PROJECT_DIR"

# Check for required tools
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go first."
    exit 1
fi

if ! command -v swift &> /dev/null; then
    echo "Error: Swift is not installed. Please install Xcode Command Line Tools."
    exit 1
fi

# Create directories
echo "Creating directories..."
mkdir -p "$BIN_DIR"
mkdir -p "$JARVIS_DIR"

# Build Go binaries
echo "Building jarvisd..."
go build -o "$BIN_DIR/jarvisd" ./cmd/jarvisd

echo "Building jarvis-notify..."
go build -o "$BIN_DIR/jarvis-notify" ./cmd/jarvis-notify

# Build Swift binary
echo "Building JarvisListen (Swift)..."
cd "$PROJECT_DIR/swift/JarvisListen"
swift build -c release
cp .build/release/JarvisListen "$BIN_DIR/"
cd "$PROJECT_DIR"

# Create sample config if it doesn't exist
if [ ! -f "$CONFIG_FILE" ]; then
    echo "Creating sample configuration..."
    cat > "$CONFIG_FILE" << 'EOF'
# Jarvis Configuration
# Get your Claude API key from https://console.anthropic.com/
# Get your Porcupine access key from https://console.picovoice.ai/

claude_api_key: ""
porcupine_access_key: ""

# Wake word options: jarvis, alexa, computer, hey siri, ok google, etc.
wake_word: "jarvis"

# Voice for text-to-speech (run 'say -v ?' to see available voices)
voice: "Samantha"

# Maximum words for summary
summary_max_words: 100

# Speech-to-text timeout in seconds
stt_timeout_seconds: 30
EOF
    echo ""
    echo "IMPORTANT: Edit $CONFIG_FILE and add your API keys!"
fi

# Create launchd plist
echo "Creating launchd configuration..."
cat > "$LAUNCHD_PLIST" << EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.jarvis.daemon</string>
    <key>ProgramArguments</key>
    <array>
        <string>$BIN_DIR/jarvisd</string>
    </array>
    <key>EnvironmentVariables</key>
    <dict>
        <key>PATH</key>
        <string>/usr/local/bin:/usr/bin:/bin:$BIN_DIR</string>
        <key>JARVIS_STT_BIN</key>
        <string>$BIN_DIR/JarvisListen</string>
    </dict>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
    <key>StandardOutPath</key>
    <string>$JARVIS_DIR/jarvisd.log</string>
    <key>StandardErrorPath</key>
    <string>$JARVIS_DIR/jarvisd.err</string>
</dict>
</plist>
EOF

echo ""
echo "=== Installation Complete ==="
echo ""
echo "Binaries installed to: $BIN_DIR"
echo "Configuration file: $CONFIG_FILE"
echo "LaunchAgent plist: $LAUNCHD_PLIST"
echo ""
echo "Next steps:"
echo "1. Edit $CONFIG_FILE and add your API keys"
echo "2. Run: launchctl load $LAUNCHD_PLIST"
echo "3. Configure Claude Code hooks (see below)"
echo ""
echo "Claude Code hook configuration (~/.claude/settings.json):"
echo '{'
echo '  "hooks": {'
echo '    "postToolUse": ['
echo '      {'
echo '        "matcher": ".*",'
echo "        \"command\": \"$BIN_DIR/jarvis-notify \\\"\$CLAUDE_TRANSCRIPT_PATH\\\"\""
echo '      }'
echo '    ]'
echo '  }'
echo '}'
echo ""
echo "Or for stop hooks only:"
echo '{'
echo '  "hooks": {'
echo '    "stop": ['
echo '      {'
echo "        \"command\": \"$BIN_DIR/jarvis-notify \\\"\$CLAUDE_TRANSCRIPT_PATH\\\"\""
echo '      }'
echo '    ]'
echo '  }'
echo '}'
