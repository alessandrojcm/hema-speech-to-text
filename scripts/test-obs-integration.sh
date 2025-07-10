#!/bin/bash

# OBS Integration Test Script
# This script tests the OBS WebSocket integration with a real OBS instance

set -e

echo "🔍 OBS Integration Test Script"
echo "=============================="

# Check if OBS is running
echo "Checking if OBS Studio is running..."
if ! lsof -i :4455 >/dev/null 2>&1; then
    echo "❌ OBS Studio is not running or WebSocket server is not enabled"
    echo ""
    echo "To run this test:"
    echo "1. Start OBS Studio"
    echo "2. Go to Tools → WebSocket Server Settings"
    echo "3. Enable WebSocket server (default port 4455)"
    echo "4. Run this script again"
    exit 1
fi

echo "✅ OBS Studio WebSocket server detected on port 4455"

# Test basic connection
echo ""
echo "Testing basic OBS connection..."
if go test -v -tags=integration ./internal/obs/ -run TestRealOBSConnection; then
    echo "✅ Basic OBS connection test passed"
else
    echo "❌ Basic OBS connection test failed"
    exit 1
fi

# Test scene operations
echo ""
echo "Testing OBS scene operations..."
if go test -v -tags=integration ./internal/obs/ -run TestRealOBSSceneOperations; then
    echo "✅ OBS scene operations test passed"
else
    echo "❌ OBS scene operations test failed"
    echo "Note: This test requires at least 2 scenes in OBS"
    exit 1
fi

# Test text source operations
echo ""
echo "Testing OBS text source operations..."
if go test -v -tags=integration ./internal/obs/ -run TestRealOBSTextSource; then
    echo "✅ OBS text source operations test passed"
else
    echo "⚠️  OBS text source operations test completed with warnings"
    echo "Note: This test requires a text source named 'TestText' in OBS"
fi

echo ""
echo "🎉 All OBS integration tests completed!"
echo ""
echo "Next steps:"
echo "- Create a text source named 'TestText' in OBS to test text updates"
echo "- Ensure you have multiple scenes for scene switching tests"
echo "- Test with different OBS configurations (password-protected, etc.)"