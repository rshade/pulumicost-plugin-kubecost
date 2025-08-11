package kubecost

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

type Client struct {
	cfg  Config
	http *http.Client
}

func NewClient(ctx context.Context, cfg Config) (*Client, error) {
	return &Client{
		cfg:  cfg,
		http: &http.Client{},
	}, nil
}

type AllocationQuery struct {
	Window      string            // "2025-07-01T00:00:00Z,2025-07-31T23:59:59Z" or "30d"
	Filter      map[string]string // namespace, controller, pod, cluster, label:app, node, etc.
	AggregateBy []string          // e.g., ["namespace", "controller"]
}

type AllocationPoint struct {
	Start       string  `json:"start"`
	End         string  `json:"end"`
	Cost        float64 `json:"cost"`
	CPUCost     float64 `json:"cpuCost"`
	RAMCost     float64 `json:"ramCost"`
	GPUCost     float64 `json:"gpuCost"`
	PVCCost     float64 `json:"pvcCost"`
	NetworkCost float64 `json:"networkCost"`
	// ... add fields as needed
}

type AllocationResponse struct {
	Items []AllocationPoint `json:"items"`
}

func (c *Client) Allocation(ctx context.Context, q AllocationQuery) (AllocationResponse, error) {
	url := fmt.Sprintf("%s/model/allocation?window=%s", c.cfg.BaseURL, q.Window)
	// TODO: add filters & aggregate params to URL
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	if c.cfg.APIToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.cfg.APIToken)
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return AllocationResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return AllocationResponse{}, fmt.Errorf("kubecost %d", resp.StatusCode)
	}
	var out AllocationResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}
