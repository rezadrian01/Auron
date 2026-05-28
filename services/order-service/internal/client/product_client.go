package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"auron/order-service/internal/domain"

	"github.com/google/uuid"
)

type httpProductClient struct {
	baseURL    string
	httpClient *http.Client
}

func NewProductClient(baseURL string) domain.ProductClient {
	return &httpProductClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
	}
}

func (c *httpProductClient) GetProduct(ctx context.Context, id uuid.UUID) (*domain.ProductSnapshot, error) {
	url := fmt.Sprintf("%s/products/%s", c.baseURL, id.String())

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("product client: create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("product client: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, domain.ErrProductNotFound
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("product client: unexpected status %d", resp.StatusCode)
	}

	// product-service response envelope: {"success": true, "data": {...}}
	var envelope struct {
		Data struct {
			ID       string  `json:"id"`
			Name     string  `json:"name"`
			Price    float64 `json:"price"`
			IsActive bool    `json:"is_active"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("product client: decode response: %w", err)
	}

	productID, err := uuid.Parse(envelope.Data.ID)
	if err != nil {
		return nil, fmt.Errorf("product client: parse product id: %w", err)
	}

	if !envelope.Data.IsActive {
		return nil, domain.ErrProductInactive
	}

	return &domain.ProductSnapshot{
		ID:       productID,
		Name:     envelope.Data.Name,
		Price:    envelope.Data.Price,
		IsActive: envelope.Data.IsActive,
	}, nil
}
