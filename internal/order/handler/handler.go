package handler

import (
	"context"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/order/domain"
	pb "github.com/qinyul/resilient-ecommerce-microservices/pb/order/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type OrderHandler struct {
	pb.UnimplementedOrderServiceServer
	service domain.OrderService
}

func NewOrderHandler(service domain.OrderService) *OrderHandler {
	return &OrderHandler{
		service: service,
	}
}

func (h *OrderHandler) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.CreateOrderResponse, error) {
	if len(req.Items) == 0 {
		return nil, status.Error(codes.InvalidArgument, "at least one item is required")
	}

	items := make([]domain.CreateOrderItem, len(req.Items))
	for i, item := range req.Items {
		items[i] = domain.CreateOrderItem{
			ProductID: item.ProductId,
			Quantity:  int(item.Quantity),
		}
	}

	input := domain.CreateOrderInput{
		UserID: req.UserId,
		Items:  items,
	}

	order, err := h.service.PlaceOrder(ctx, input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to place order: %v", err)
	}

	return &pb.CreateOrderResponse{
		OrderId: order.ID,
		Status:  mapDomainStatusToPb(order.Status),
	}, nil
}

func (h *OrderHandler) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.GetOrderResponse, error) {
	if req.OrderId == "" {
		return nil, status.Error(codes.InvalidArgument, "order id is required")
	}

	order, err := h.service.GetOrder(ctx, req.OrderId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get order: %v", err)
	}

	items := make([]*pb.OrderItem, len(order.Items))
	for i, item := range order.Items {
		items[i] = &pb.OrderItem{
			ProductId: item.ProductID,
			Quantity:  uint32(item.Quantity),
			Price: &pb.Money{
				CurrencyCode: item.UnitPrice.Currency,
				Units:        item.UnitPrice.Units,
				Nanos:        item.UnitPrice.Nanos,
			},
		}
	}

	return &pb.GetOrderResponse{
		OrderId: order.ID,
		UserId:  order.UserID,
		Items:   items,
		TotalAmount: &pb.Money{
			CurrencyCode: order.TotalPrice.Currency,
			Units:        order.TotalPrice.Units,
			Nanos:        order.TotalPrice.Nanos,
		},
		Status:    mapDomainStatusToPb(order.Status),
		CreatedAt: timestamppb.New(order.CreatedAt),
	}, nil
}

func mapDomainStatusToPb(s domain.OrderStatus) pb.OrderStatus {
	switch s {
	case domain.StatusPending:
		return pb.OrderStatus_ORDER_STATUS_PENDING
	case domain.StatusPaid:
		return pb.OrderStatus_ORDER_STATUS_PAID
	case domain.StatusCancelled:
		return pb.OrderStatus_ORDER_STATUS_CANCELED
	case domain.StatusCompleted:
		return pb.OrderStatus_ORDER_STATUS_PAID
	default:
		return pb.OrderStatus_ORDER_STATUS_UNSPECIFIED
	}
}
