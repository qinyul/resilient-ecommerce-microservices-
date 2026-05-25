package handler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	orderpb "github.com/qinyul/resilient-ecommerce-microservices/pb/order/v1"
	productpb "github.com/qinyul/resilient-ecommerce-microservices/pb/product/v1"
)

func HandleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"OK"}`))
}

// Product Handlers
func HandleCreateProduct(client productpb.ProductServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name        string `json:"name"`
			Description string `json:"description"`
			Price       struct {
				CurrencyCode string `json:"currency_code"`
				Units        int64  `json:"units"`
				Nanos        int32  `json:"nanos"`
			} `json:"price"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		gRPCReq := &productpb.CreateProductRequest{
			Name:        req.Name,
			Description: req.Description,
			Price: &productpb.Money{
				CurrencyCode: req.Price.CurrencyCode,
				Units:        req.Price.Units,
				Nanos:        req.Price.Nanos,
			},
		}

		resp, err := client.CreateProduct(r.Context(), gRPCReq)
		if err != nil {
			slog.Error("failed to create product via gRPC", "error", err)
			http.Error(w, "internal server error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func HandleGetProduct(client productpb.ProductServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing product id", http.StatusBadRequest)
			return
		}

		resp, err := client.GetProduct(r.Context(), &productpb.GetProductRequest{Id: id})
		if err != nil {
			slog.Error("failed to get product via gRPC", "error", err, "id", id)
			http.Error(w, "product not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

// Order Handlers
func HandleCreateOrder(client orderpb.OrderServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserID         string `json:"user_id"`
			IdempotencyKey string `json:"idempotency_key"`
			Items          []struct {
				ProductID string `json:"product_id"`
				Quantity  uint32 `json:"quantity"`
			} `json:"items"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		items := make([]*orderpb.CreateOrderItem, len(req.Items))
		for i, it := range req.Items {
			items[i] = &orderpb.CreateOrderItem{
				ProductId: it.ProductID,
				Quantity:  it.Quantity,
			}
		}

		gRPCReq := &orderpb.CreateOrderRequest{
			UserId:         req.UserID,
			IdempotencyKey: req.IdempotencyKey,
			Items:          items,
		}

		resp, err := client.CreateOrder(r.Context(), gRPCReq)
		if err != nil {
			slog.Error("failed to create order via gRPC", "error", err)
			http.Error(w, "internal server error: "+err.Error(), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(resp)
	}
}

func HandleGetOrder(client orderpb.OrderServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := r.PathValue("id")
		if id == "" {
			http.Error(w, "missing order id", http.StatusBadRequest)
			return
		}

		resp, err := client.GetOrder(r.Context(), &orderpb.GetOrderRequest{OrderId: id})
		if err != nil {
			slog.Error("failed to get order via gRPC", "error", err, "id", id)
			http.Error(w, "order not found", http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}

func HandleListOrders(client orderpb.OrderServiceClient) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			http.Error(w, "missing user_id query parameter", http.StatusBadRequest)
			return
		}

		resp, err := client.ListOrders(r.Context(), &orderpb.ListOrdersRequest{UserId: userID})
		if err != nil {
			slog.Error("failed to list orders via gRPC", "error", err, "user_id", userID)
			http.Error(w, "failed to list orders", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}
}
