.PHONY: proto build test run-infra run-order run-payment docker-up docker-down clean tf-init tf-validate tf-plan tf-apply tf-destroy

# 1. Protobuf Generation
proto:
	protoc --go_out=. --go_opt=module=github.com/qinyul/resilient-ecommerce-microservices \
		--go-grpc_out=. --go-grpc_opt=module=github.com/qinyul/resilient-ecommerce-microservices \
		proto/order/v1/order.proto
	protoc --go_out=. --go_opt=module=github.com/qinyul/resilient-ecommerce-microservices \
		--go-grpc_out=. --go-grpc_opt=module=github.com/qinyul/resilient-ecommerce-microservices \
		proto/payment/v1/payment.proto
	protoc --go_out=. --go_opt=module=github.com/qinyul/resilient-ecommerce-microservices \
		--go-grpc_out=. --go-grpc_opt=module=github.com/qinyul/resilient-ecommerce-microservices \
		proto/product/v1/product.proto

# 2. Local Build
build:
	go build -o bin/gateway ./cmd/gateway/main.go
	go build -o bin/order ./cmd/order/main.go
	go build -o bin/payment ./cmd/payment/main.go
	go build -o bin/product ./cmd/product/main.go

# 3. Running Services Locally
run-gateway:
	go run cmd/gateway/main.go

run-order:
	go run cmd/order/main.go

run-payment:
	go run cmd/payment/main.go

run-product:
	go run cmd/product/main.go

# 4. Running Infrastructure Only (RabbitMQ and Postgres)
run-infra:
	docker-compose up -d rabbitmq order-db product-db

# 5. Full Environment (Infrastructure + Applications)
docker-up:
	docker-compose up -d --build

docker-down:
	docker-compose down

# 6. Testing
test:
	go test -v ./...

# 7. Terraform Observability Provisioning
tf-init:
	docker run --rm --network host -v $(shell pwd)/terraform:/workspace -w /workspace hashicorp/terraform:latest init

tf-validate:
	docker run --rm --network host -v $(shell pwd)/terraform:/workspace -w /workspace hashicorp/terraform:latest validate

tf-plan:
	docker run --rm --network host -v $(shell pwd)/terraform:/workspace -w /workspace hashicorp/terraform:latest plan

tf-apply:
	docker run --rm --network host -v $(shell pwd)/terraform:/workspace -w /workspace hashicorp/terraform:latest apply -auto-approve

tf-destroy:
	docker run --rm --network host -v $(shell pwd)/terraform:/workspace -w /workspace hashicorp/terraform:latest destroy -auto-approve

# 8. Cleanup
clean:
	rm -rf bin/
