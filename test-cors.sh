#!/bin/bash

echo "🔍 Testing CORS Configuration..."
echo "================================"

# Test 1: Simple GET request
echo "1. Testing simple GET request:"
curl -s -H "Origin: app://obsidian.md" http://localhost:8081/api/v1/health | head -1

# Test 2: OPTIONS preflight request
echo -e "\n2. Testing OPTIONS preflight request:"
curl -s -H "Origin: app://obsidian.md" \
     -H "Access-Control-Request-Method: GET" \
     -H "Access-Control-Request-Headers: Content-Type, Authorization" \
     -X OPTIONS \
     -I http://localhost:8081/api/v1/health | grep -E "(HTTP/|Access-Control)"

# Test 3: Check if service is running
echo -e "\n3. Service status:"
if curl -s http://localhost:8081/api/v1/health > /dev/null; then
    echo "✅ Service is running on port 8081"
else
    echo "❌ Service is not responding on port 8081"
fi

# Test 4: Check CORS headers in response
echo -e "\n4. CORS headers in response:"
curl -s -I -H "Origin: app://obsidian.md" http://localhost:8081/api/v1/health | grep -i "access-control"

echo -e "\n💡 If CORS is working correctly, you should see:"
echo "   - Access-Control-Allow-Origin: app://obsidian.md (or *)"
echo "   - Access-Control-Allow-Methods: GET, POST, PUT, DELETE, OPTIONS, PATCH"
echo "   - Access-Control-Allow-Headers: Content-Type, Authorization, X-Requested-With, Accept, Origin"