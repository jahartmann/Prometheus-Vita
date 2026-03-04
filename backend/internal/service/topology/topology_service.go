package topology

import (
	"context"
	"fmt"
	"sync"

	"github.com/antigravity/prometheus/internal/model"
	"github.com/antigravity/prometheus/internal/proxmox"
	"github.com/antigravity/prometheus/internal/repository"
)

type Service struct {
	nodeRepo      repository.NodeRepository
	clientFactory proxmox.ClientFactory
}

func NewService(nodeRepo repository.NodeRepository, clientFactory proxmox.ClientFactory) *Service {
	return &Service{
		nodeRepo:      nodeRepo,
		clientFactory: clientFactory,
	}
}

func (s *Service) BuildTopology(ctx context.Context) (*model.TopologyGraph, error) {
	nodes, err := s.nodeRepo.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}

	graph := &model.TopologyGraph{
		Nodes: []model.TopologyNode{},
		Edges: []model.TopologyEdge{},
	}

	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, n := range nodes {
		wg.Add(1)
		go func(node model.Node) {
			defer wg.Done()

			status := "offline"
			if node.IsOnline {
				status = "online"
			}

			topoNode := model.TopologyNode{
				ID:     node.ID.String(),
				Type:   "host",
				Label:  node.Name,
				Status: status,
				Metadata: map[string]interface{}{
					"hostname": node.Hostname,
					"type":     string(node.Type),
				},
			}

			mu.Lock()
			graph.Nodes = append(graph.Nodes, topoNode)
			mu.Unlock()

			if node.Type != model.NodeTypePVE {
				return
			}

			client, err := s.clientFactory.CreateClient(&node)
			if err != nil || client == nil {
				return
			}

			// Get Proxmox node names
			pveNodes, err := client.GetNodes(ctx)
			if err != nil || len(pveNodes) == 0 {
				return
			}
			pveNode := pveNodes[0]

			// Get VMs
			vms, err := client.GetVMs(ctx, pveNode)
			if err == nil {
				mu.Lock()
				for _, vm := range vms {
					vmID := fmt.Sprintf("%s-vm-%d", node.ID.String(), vm.VMID)
					vmStatus := "stopped"
					if vm.Status == "running" {
						vmStatus = "running"
					}
					vmType := "vm"
					if vm.Type == "lxc" {
						vmType = "ct"
					}

					graph.Nodes = append(graph.Nodes, model.TopologyNode{
						ID:     vmID,
						Type:   vmType,
						Label:  vm.Name,
						Status: vmStatus,
						Metadata: map[string]interface{}{
							"vmid":   vm.VMID,
							"cpu":    vm.CPUs,
							"memory": vm.MaxMem,
						},
					})

					graph.Edges = append(graph.Edges, model.TopologyEdge{
						Source: node.ID.String(),
						Target: vmID,
						Label:  "runs",
					})
				}
				mu.Unlock()
			}

			// Get Storage
			storages, err := client.GetStorage(ctx, pveNode)
			if err == nil {
				mu.Lock()
				for _, st := range storages {
					stID := fmt.Sprintf("%s-storage-%s", node.ID.String(), st.Storage)
					graph.Nodes = append(graph.Nodes, model.TopologyNode{
						ID:     stID,
						Type:   "storage",
						Label:  st.Storage,
						Status: "online",
						Metadata: map[string]interface{}{
							"type":  st.Type,
							"total": st.Total,
							"used":  st.Used,
						},
					})
					graph.Edges = append(graph.Edges, model.TopologyEdge{
						Source: node.ID.String(),
						Target: stID,
						Label:  "storage",
					})
				}
				mu.Unlock()
			}

			// Get Network Bridges
			netIfs, err := client.GetNetworkInterfaces(ctx, pveNode)
			if err == nil {
				mu.Lock()
				for _, ni := range netIfs {
					if ni.Type != "bridge" {
						continue
					}
					niID := fmt.Sprintf("%s-net-%s", node.ID.String(), ni.Iface)
					graph.Nodes = append(graph.Nodes, model.TopologyNode{
						ID:     niID,
						Type:   "network",
						Label:  ni.Iface,
						Status: "online",
						Metadata: map[string]interface{}{
							"cidr": ni.CIDR,
						},
					})
					graph.Edges = append(graph.Edges, model.TopologyEdge{
						Source: node.ID.String(),
						Target: niID,
						Label:  "bridge",
					})
				}
				mu.Unlock()
			}
		}(n)
	}

	wg.Wait()
	return graph, nil
}
