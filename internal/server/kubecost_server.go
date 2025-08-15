package server

import (
	"context"
	"fmt"
	"strings"
	"time"

	kubecost "github.com/rshade/pulumicost-plugin-kubecost/internal/kubecost"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/timestamppb"
)

// TODO: Replace these stubs when pulumicost-spec protobuf definitions are available
type UnimplementedCostSourceServer struct{}
type Empty struct{}
type PluginName struct{ Name string }
type ResourceDescriptor struct{ ResourceType string }
type SupportsResponse struct{ Supported bool }
type ActualCostQuery struct{ ResourceId, Start, End string }
type ActualCostResult struct{ Timestamp *timestamppb.Timestamp; Cost float64; UsageAmount float64; UsageUnit, Source string }
type ActualCostResultList struct{ Results []*ActualCostResult }
type PriceInfo struct{ UnitPrice, CostPerMonth float64; Currency, BillingDetail string }
type PricingSpec struct{ Provider, ResourceType, Sku, Region, BillingMode, Currency, Description string; RatePerUnit float64; PluginMetadata map[string]string }

type KubecostServer struct {
	UnimplementedCostSourceServer
	cli *kubecost.Client
}

func NewKubecostServer(cli *kubecost.Client) *KubecostServer {
	return &KubecostServer{cli: cli}
}

func (s *KubecostServer) RegisterService(grpcServer *grpc.Server) {
	// TODO: Uncomment when pulumicost-spec is available
	// pbc.RegisterCostSourceServer(grpcServer, s)
}

func (s *KubecostServer) Name(ctx context.Context, _ *Empty) (*PluginName, error) {
	return &PluginName{Name: "kubecost"}, nil
}

func (s *KubecostServer) Supports(ctx context.Context, r *ResourceDescriptor) (*SupportsResponse, error) {
	supported := r.ResourceType == "k8s-namespace" || r.ResourceType == "k8s-pod" ||
		r.ResourceType == "k8s-controller" || r.ResourceType == "k8s-node"
	return &SupportsResponse{Supported: supported}, nil
}

func (s *KubecostServer) GetActualCost(ctx context.Context, q *ActualCostQuery) (*ActualCostResultList, error) {
	// Map ResourceID like "namespace/default" -> Kubecost filter
	window := windowFromTimes(q.Start, q.End)
	filter := map[string]string{}
	parts := strings.Split(q.ResourceId, "/")
	if len(parts) > 0 {
		switch parts[0] {
		case "namespace":
			if len(parts) >= 2 {
				filter["namespace"] = parts[1]
			}
		case "pod":
			if len(parts) >= 3 {
				filter["namespace"] = parts[1]
				filter["pod"] = parts[2]
			}
		case "controller":
			if len(parts) >= 3 {
				filter["namespace"] = parts[1]
				filter["controller"] = parts[2]
			}
		case "node":
			if len(parts) >= 2 {
				filter["node"] = parts[1]
			}
		}
	}

	resp, err := s.cli.EnhancedAllocation(ctx, kubecost.AllocationQuery{
		Window: window,
		Filter: filter,
	})
	if err != nil {
		return nil, err
	}

	out := &ActualCostResultList{}
	for _, it := range resp.Items {
		// Map Kubecost point → ActualCostResult
		start, _ := time.Parse(time.RFC3339, it.Start)
		acr := &ActualCostResult{
			Timestamp:   timestamppb.New(start),
			Cost:        it.Cost,
			UsageAmount: 0,  // Optional: populate from CPU/RAM hours if needed
			UsageUnit:   "", // Optional
			Source:      "kubecost",
		}
		out.Results = append(out.Results, acr)
	}
	return out, nil
}

func (s *KubecostServer) GetProjectedCost(ctx context.Context, r *ResourceDescriptor) (*PriceInfo, error) {
	// For MVP, ask Kubecost indirectly by extrapolating last N days average
	end := time.Now().UTC()
	start := end.Add(-30 * 24 * time.Hour)
	acr, err := s.GetActualCost(ctx, &ActualCostQuery{
		ResourceId: "", // TODO: map from ResourceDescriptor
		Start:      start.Format(time.RFC3339),
		End:        end.Format(time.RFC3339),
	})
	if err != nil {
		return nil, err
	}
	if len(acr.Results) == 0 {
		return &PriceInfo{Currency: "USD"}, nil
	}

	var sum float64
	for _, p := range acr.Results {
		sum += p.Cost
	}
	daily := sum / float64(len(acr.Results))
	monthly := daily * 30.0

	return &PriceInfo{
		UnitPrice:     daily, // loosely "per-day average"
		Currency:      "USD",
		CostPerMonth:  monthly,
		BillingDetail: "kubecost-avg-daily",
	}, nil
}

func (s *KubecostServer) GetPricingSpec(ctx context.Context, r *ResourceDescriptor) (*PricingSpec, error) {
	// Optional: return a synthetic spec expressing CPU/RAM per-hour costs if available
	return &PricingSpec{
		Provider:       "kubernetes",
		ResourceType:   r.ResourceType,
		Sku:            "", // TODO: map from ResourceDescriptor
		Region:         "", // TODO: map from ResourceDescriptor
		BillingMode:    "per_day",
		RatePerUnit:    0, // unknown; can be derived if desired
		Currency:       "USD",
		Description:    fmt.Sprintf("Kubecost-derived projection for %s", r.ResourceType),
		PluginMetadata: map[string]string{"source": "kubecost"},
	}, nil
}

func windowFromTimes(start, end string) string {
	if start == "" || end == "" {
		return "30d"
	}
	return fmt.Sprintf("%s,%s", start, end)
}

