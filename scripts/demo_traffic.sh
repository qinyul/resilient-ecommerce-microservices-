#!/bin/bash

# Base URL for the Gateway
GATEWAY_URL="http://localhost:8080/api/v1"

echo "Starting Demo Traffic Simulation (Success & Errors). Press Ctrl+C to stop."

while true; do
  echo "=========================================="
  echo " [SCENARIO 1] SUCCESS PATH"
  echo "=========================================="
  
  PRODUCT_NAME="DemoProduct-$(date +%s)"
  echo "[1.1] Creating Product: $PRODUCT_NAME (Expected: 201 Created)"
  PRODUCT_RES=$(curl -s -w "\nHTTP_STATUS:%{http_code}\n" -X POST "$GATEWAY_URL/products" -H "Content-Type: application/json" -d "{
    \"name\": \"$PRODUCT_NAME\",
    \"description\": \"Awesome product for demo\",
    \"price\": {
      \"currency_code\": \"USD\",
      \"units\": 49,
      \"nanos\": 990000000
    }
  }")
  
  PRODUCT_ID=$(echo "$PRODUCT_RES" | grep -o '"id":"[^"]*' | grep -o '[^"]*$')
  echo "      -> Response Status: $(echo "$PRODUCT_RES" | grep HTTP_STATUS)"
  
  if [ -n "$PRODUCT_ID" ]; then
    echo "[1.2] Fetching Product ID: $PRODUCT_ID (Expected: 200 OK)"
    curl -s -o /dev/null -w "      -> Response Status: HTTP_STATUS:%{http_code}\n" -X GET "$GATEWAY_URL/products/$PRODUCT_ID"
    
    USER_ID=$(printf '%04x%04x-%04x-%04x-%04x-%04x%04x%04x\n' $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM)
    IDEMPOTENCY_KEY="demo-key-$(date +%s)-$RANDOM"
    
    echo "[1.3] Creating Order for Product ID: $PRODUCT_ID (Expected: 200/201)"
    ORDER_RES=$(curl -s -w "\nHTTP_STATUS:%{http_code}\n" -X POST "$GATEWAY_URL/orders" -H "Content-Type: application/json" -d "{
      \"user_id\": \"$USER_ID\",
      \"idempotency_key\": \"$IDEMPOTENCY_KEY\",
      \"items\": [
        {
          \"product_id\": \"$PRODUCT_ID\",
          \"quantity\": 1
        }
      ]
    }")
    echo "      -> Response Status: $(echo "$ORDER_RES" | grep HTTP_STATUS)"
  fi

  sleep 2

  echo ""
  echo "=========================================="
  echo " [SCENARIO 2] ERROR PATHS"
  echo "=========================================="

  echo "[2.1] Fetching Non-Existent Product (Expected: 404 Not Found)"
  curl -s -o /dev/null -w "      -> Response Status: HTTP_STATUS:%{http_code}\n" -X GET "$GATEWAY_URL/products/invalid-id-9999"

  sleep 1

  echo "[2.2] Creating Product with Invalid Payload (Expected: 400 Bad Request)"
  curl -s -o /dev/null -w "      -> Response Status: HTTP_STATUS:%{http_code}\n" -X POST "$GATEWAY_URL/products" -H "Content-Type: application/json" -d "{
    \"name\": \"Broken Product\"
    // Missing closing brackets and invalid JSON
  "
  
  sleep 1

  USER_ID=$(printf '%04x%04x-%04x-%04x-%04x-%04x%04x%04x\n' $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM)
  echo "[2.3] Creating Order with Non-Existent Product (Expected: 400/500 depending on service logic)"
  curl -s -o /dev/null -w "      -> Response Status: HTTP_STATUS:%{http_code}\n" -X POST "$GATEWAY_URL/orders" -H "Content-Type: application/json" -d "{
    \"user_id\": \"$USER_ID\",
    \"idempotency_key\": \"fail-key-$RANDOM\",
    \"items\": [
      {
        \"product_id\": \"fake-product-123\",
        \"quantity\": 5
      }
    ]
  }"

  sleep 3
done
