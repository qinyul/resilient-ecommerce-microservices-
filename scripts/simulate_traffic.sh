#!/bin/bash

# Base URL for the Gateway
GATEWAY_URL="http://localhost:8080/api/v1"

echo "Starting Traffic Simulation. Press Ctrl+C to stop."

while true; do
  echo "-----------------------------------"
  # 1. Create a Product
  PRODUCT_NAME="Product-$(date +%s)"
  echo "[1] Creating Product: $PRODUCT_NAME..."
  PRODUCT_RES=$(curl -s -X POST "$GATEWAY_URL/products" -H "Content-Type: application/json" -d "{
    \"name\": \"$PRODUCT_NAME\",
    \"description\": \"Test product for traffic simulation\",
    \"price\": {
      \"currency_code\": \"USD\",
      \"units\": 10,
      \"nanos\": 990000000
    }
  }")
  
  PRODUCT_ID=$(echo $PRODUCT_RES | grep -o '"id":"[^"]*' | grep -o '[^"]*$')
  if [ -z "$PRODUCT_ID" ]; then
    PRODUCT_ID="dummy-product-id"
    echo "  -> Failed to parse product ID. Response: $PRODUCT_RES"
  else
    echo "  -> Product Created! ID: $PRODUCT_ID"
  fi

  # 2. Get the created Product
  echo "[2] Fetching Product ID: $PRODUCT_ID..."
  curl -s -X GET "$GATEWAY_URL/products/$PRODUCT_ID" > /dev/null
  echo "  -> Fetched!"

  # 3. Create an Order
  if [ -f /proc/sys/kernel/random/uuid ]; then
    USER_ID=$(cat /proc/sys/kernel/random/uuid)
  elif command -v uuidgen >/dev/null 2>&1; then
    USER_ID=$(uuidgen)
  else
    USER_ID=$(printf '%04x%04x-%04x-%04x-%04x-%04x%04x%04x\n' $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM $RANDOM)
  fi
  IDEMPOTENCY_KEY="key-$(date +%s)-$RANDOM"
  echo "[3] Creating Order for User: $USER_ID..."
  ORDER_RES=$(curl -s -X POST "$GATEWAY_URL/orders" -H "Content-Type: application/json" -d "{
    \"user_id\": \"$USER_ID\",
    \"idempotency_key\": \"$IDEMPOTENCY_KEY\",
    \"items\": [
      {
        \"product_id\": \"$PRODUCT_ID\",
        \"quantity\": 2
      }
    ]
  }")
  
  ORDER_ID=$(echo $ORDER_RES | grep -oE '"(order_)?id":"[^"]*' | grep -o '[^"]*$')
  if [ -n "$ORDER_ID" ]; then
    echo "  -> Order Created! ID: $ORDER_ID"
    # 4. Fetch the Order
    echo "[4] Fetching Order ID: $ORDER_ID..."
    curl -s -X GET "$GATEWAY_URL/orders/$ORDER_ID" > /dev/null
  else
    echo "  -> Failed to parse order ID. Response: $ORDER_RES"
  fi

  # Sleep to simulate realistic user traffic and avoid hitting rate limits too fast
  sleep 3
done
