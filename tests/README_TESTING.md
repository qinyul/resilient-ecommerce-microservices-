# Testing the Microservices

## Postman Testing
1. Import `tests/postman/order_service.postman_collection.json`.
2. For gRPC, it's recommended to:
   - Create a new **gRPC Request** in Postman.
   - Server: `localhost:50051`.
   - **Service Definition**: Import `proto/order/v1/order.proto`.
   - This provides full autocomplete for `CreateOrder` and `GetOrder`.

## CLI Testing (grpcurl)

### 1. Create Order
```bash
grpcurl -plaintext -d '{
  "user_id": "c6a7822f-d86d-4c37-a16a-790176378e99",
  "items": [
    {
      "product_id": "prod-1",
      "quantity": 2
    }
  ]
}' localhost:50051 order.v1.OrderService/CreateOrder
```

### 2. Get Order
```bash
grpcurl -plaintext -d '{"order_id": "<ID_FROM_ABOVE>"}' localhost:50051 order.v1.OrderService/GetOrder
```

## Infrastructure Validation
Make sure RabbitMQ and Postgres are running:
```bash
make run-infra
```
Then check the `payment-service` logs to see the event processing:
```bash
docker-compose logs -f payment-service
```
