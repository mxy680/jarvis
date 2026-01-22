#!/bin/bash
set -e

# Jarvis Uninstallation Script

JARVIS_DIR="$HOME/.jarvis"
LAUNCHD_PLIST="$HOME/Library/LaunchAgents/com.jarvis.daemon.plist"

echo "=== Jarvis Uninstallation ==="
echo ""

# Stop the daemon if running
if launchctl list | grep -q "com.jarvis.daemon"; then
    echo "Stopping Jarvis daemon..."
    launchctl unload "$LAUNCHD_PLIST" 2>/dev/null || true
fi

# Remove launchd plist
if [ -f "$LAUNCHD_PLIST" ]; then
    echo "Removing launchd configuration..."
    rm -f "$LAUNCHD_PLIST"
fi

# Ask about config preservation
read -p "Remove configuration and data? (y/N) " -n 1 -r
echo
if [[ $REPLY =~ ^[Yy]$ ]]; then
    echo "Removing $JARVIS_DIR..."
    rm -rf "$JARVIS_DIR"
else
    echo "Keeping configuration at $JARVIS_DIR"
    echo "Removing only binaries..."
    rm -rf "$JARVIS_DIR/bin"
    rm -f "$JARVIS_DIR/jarvis.sock"
    rm -f "$JARVIS_DIR/jarvisd.log"
    rm -f "$JARVIS_DIR/jarvisd.err"
fi

echo ""
echo "=== Uninstallation Complete ==="
echo ""
echo "Don't forget to remove the Jarvis hook from your Claude Code settings."
