#!/bin/bash

# Serve the CA certificate over HTTP for easy installation on devices

PORT=${1:-8888}
CA_DIR="$HOME/.proxy-tui"
CA_FILE="$CA_DIR/ca.crt"

# Check if CA exists
if [ ! -f "$CA_FILE" ]; then
    echo "Error: CA certificate not found at $CA_FILE"
    echo "Run the proxy first to generate the CA certificate."
    exit 1
fi

# Get local IP address
get_local_ip() {
    if command -v ip &> /dev/null; then
        ip route get 1 | awk '{print $7; exit}'
    elif command -v ifconfig &> /dev/null; then
        ifconfig | grep -Eo 'inet (addr:)?([0-9]*\.){3}[0-9]*' | grep -Eo '([0-9]*\.){3}[0-9]*' | grep -v '127.0.0.1' | head -1
    else
        echo "localhost"
    fi
}

LOCAL_IP=$(get_local_ip)

echo "============================================"
echo "  Proxy TUI - CA Certificate Server"
echo "============================================"
echo ""
echo "Serving CA certificate on port $PORT"
echo ""
echo "Download URL:"
echo "  http://$LOCAL_IP:$PORT/ca.crt"
echo "  http://localhost:$PORT/ca.crt"
echo ""
echo "Installation instructions:"
echo "  iOS/iPadOS: Download, Settings → Profile Downloaded → Install"
echo "  macOS:      Download, double-click, add to Keychain, trust for SSL"
echo "  Android:    Download, Settings → Security → Install from storage"
echo "  Windows:    Download, double-click, install to Trusted Root CAs"
echo ""
echo "Press Ctrl+C to stop"
echo "============================================"

# Create a temporary directory for serving
TEMP_DIR=$(mktemp -d)
cp "$CA_FILE" "$TEMP_DIR/ca.crt"

# Create an index.html
cat > "$TEMP_DIR/index.html" << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>Proxy TUI - CA Certificate</title>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, sans-serif; max-width: 600px; margin: 50px auto; padding: 20px; }
        h1 { color: #333; }
        a { display: inline-block; background: #007AFF; color: white; padding: 15px 30px; border-radius: 8px; text-decoration: none; font-size: 1.2em; margin: 20px 0; }
        a:hover { background: #0056b3; }
        .instructions { background: #f5f5f5; padding: 20px; border-radius: 8px; margin-top: 20px; }
        .instructions h3 { margin-top: 0; }
        code { background: #e0e0e0; padding: 2px 6px; border-radius: 4px; }
    </style>
</head>
<body>
    <h1>Proxy TUI CA Certificate</h1>
    <a href="/ca.crt">Download CA Certificate</a>
    <div class="instructions">
        <h3>Installation Instructions:</h3>
        <p><strong>iOS/iPadOS:</strong> Download, go to Settings → Profile Downloaded → Install, then Settings → General → About → Certificate Trust Settings → Enable</p>
        <p><strong>macOS:</strong> Download, double-click, add to Keychain, then double-click cert → Trust → Always Trust</p>
        <p><strong>Android:</strong> Download, Settings → Security → Install from storage</p>
        <p><strong>Windows:</strong> Download, double-click, install to Trusted Root CAs</p>
        <p><strong>Linux:</strong> <code>sudo cp ca.crt /usr/local/share/ca-certificates/ && sudo update-ca-certificates</code></p>
    </div>
</body>
</html>
EOF

# Cleanup on exit
cleanup() {
    rm -rf "$TEMP_DIR"
    echo ""
    echo "Server stopped."
}
trap cleanup EXIT

# Start Python HTTP server
cd "$TEMP_DIR"
if command -v python3 &> /dev/null; then
    python3 -m http.server "$PORT"
elif command -v python &> /dev/null; then
    python -m SimpleHTTPServer "$PORT"
else
    echo "Error: Python is required to run this script"
    exit 1
fi
