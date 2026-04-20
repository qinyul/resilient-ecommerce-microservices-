.PHONY: proto build test run-infra run-order run-payment docker-up docker-down clean

# 1. Protobuf Generation
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/order/v1/order.proto
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/payment/v1/payment.proto

# 2. Local Build
build:
	go build -o bin/order ./cmd/order/main.go
	go build -o bin/payment ./cmd/payment/main.go

# 3. Running Services Locally
run-order:
	go run cmd/order/main.go

run-payment:
	go run cmd/payment/main.go

# 4. Running Infrastructure Only (RabbitMQ and Postgres)
run-infra:
	docker-compose up -d rabbitmq order-db

# 5. Full Environment (Infrastructure + Applications)
docker-up:
	docker-compose up -d --build

docker-down:
	docker-compose down

# 6. Testing
test:
	go test -v ./...

# 7. Cleanup
clean:
	rm -rf bin/
