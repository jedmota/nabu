#!/bin/bash
# install-ca.sh - Install Nabu CA certificate

set -e

CA_DIR="$HOME/.nabu"
CA_CERT="$CA_DIR/ca.crt"

if [ ! -f "$CA_CERT" ]; then
    echo "Error: CA certificate not found at $CA_CERT"
    echo "Please run nabu first to generate the CA certificate."
    exit 1
fi

echo "Installing Nabu CA certificate..."
echo "CA fingerprint: $(openssl x509 -in "$CA_CERT" -noout -fingerprint -sha256 2>/dev/null | cut -d= -f2)"

# Detect OS and install accordingly
case "$(uname -s)" in
    Linux*)
        if [ -d /etc/pki/ca-trust/source/anchors ]; then
            # Fedora/RHEL/CentOS
            sudo cp "$CA_CERT" /etc/pki/ca-trust/source/anchors/nabu-ca.crt
            sudo update-ca-trust
            echo "CA installed (Fedora/RHEL method)"
        elif [ -d /usr/local/share/ca-certificates ]; then
            # Debian/Ubuntu
            sudo cp "$CA_CERT" /usr/local/share/ca-certificates/nabu-ca.crt
            sudo update-ca-certificates
            echo "CA installed (Debian/Ubuntu method)"
        elif [ -d /etc/ca-certificates/trust-source/anchors ]; then
            # Arch Linux
            sudo cp "$CA_CERT" /etc/ca-certificates/trust-source/anchors/nabu-ca.crt
            sudo trust extract-compat
            echo "CA installed (Arch Linux method)"
        else
            echo "Error: Could not detect Linux distribution"
            echo "Please manually install: $CA_CERT"
            exit 1
        fi
        ;;
    Darwin*)
        # macOS
        sudo security add-trusted-cert -d -r trustRoot -k /Library/Keychains/System.keychain "$CA_CERT"
        echo "CA installed (macOS method)"
        ;;
    MINGW*|CYGWIN*|MSYS*)
        # Windows (Git Bash/Cygwin/MSYS)
        certutil -addstore -f "ROOT" "$CA_CERT"
        echo "CA installed (Windows method)"
        ;;
    *)
        echo "Error: Unsupported operating system: $(uname -s)"
        echo "Please manually install: $CA_CERT"
        exit 1
        ;;
esac

echo ""
echo "CA certificate installed successfully!"
echo ""
echo "Note for browsers:"
echo "  - Firefox: Import manually at about:preferences#privacy -> Certificates"
echo "  - Chrome/Edge: Should use system certificates automatically"
echo ""
echo "To uninstall, remove the certificate from your system's trust store."
