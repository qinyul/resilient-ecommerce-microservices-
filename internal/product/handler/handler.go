package handler

import (
	"context"
	"errors"

	"github.com/qinyul/resilient-ecommerce-microservices/internal/product/domain"
	pb "github.com/qinyul/resilient-ecommerce-microservices/pb/product/v1"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type ProductHandler struct {
	pb.UnimplementedProductServiceServer
	service domain.ProductService
}

func NewProductHandler(service domain.ProductService) *ProductHandler {
	return &ProductHandler{
		service: service,
	}
}

func (h *ProductHandler) CreateProduct(ctx context.Context, req *pb.CreateProductRequest) (*pb.CreateProductResponse, error) {
	if req.Price == nil {
		return nil,status.Error(codes.InvalidArgument,"price is required")
	}
	input := domain.CreateProductInput{
		Name:        req.Name,
		Description: req.Description,
		Price: domain.Money{
			Currency: req.Price.CurrencyCode,
			Units:    req.Price.Units,
			Nanos:    req.Price.Nanos,
		},
	}

	if err := domain.Validate.Struct(input); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "validation failed: %v", err)
	}

	product, err := h.service.CreateProduct(ctx, input)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create product: %v", err)
	}

	return &pb.CreateProductResponse{
		Product: mapDomainProductToPb(product),
	}, nil
}

func (h *ProductHandler) GetProduct(ctx context.Context, req *pb.GetProductRequest) (*pb.GetProductResponse, error) {
	if req.Id == "" {
		return nil, status.Error(codes.InvalidArgument, "product id is required")
	}

	product, err := h.service.GetProduct(ctx, req.Id)
	if err != nil {
		if errors.Is(err, domain.ErrProductNotFound) {
			return nil , status.Errorf(codes.NotFound,"product with id %s not found",req.Id)
		}
		return nil, status.Errorf(codes.Internal, "failed to get product: %v", err)
	}

	return &pb.GetProductResponse{
		Product: mapDomainProductToPb(product),
	}, nil
}

func mapDomainProductToPb(p *domain.Product) *pb.Product {
	return &pb.Product{
		Id:          p.ID,
		Name:        p.Name,
		Description: p.Description,
		Price: &pb.Money{
			CurrencyCode: p.Price.Currency,
			Units:        p.Price.Units,
			Nanos:        p.Price.Nanos,
		},
		IsActive:  p.IsActive,
		CreatedAt: timestamppb.New(p.CreatedAt),
		UpdatedAt: timestamppb.New(p.UpdatedAt),
	}
}
