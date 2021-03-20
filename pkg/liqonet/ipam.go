package liqonet

import (
	"fmt"
	goipam "github.com/davidefalcone1/go-ipam"
	"inet.af/netaddr"
	"k8s.io/klog"
	"strconv"
	"strings"
)

type Ipam interface {
	Init(reservedNetworks map[string]string, networkPool []string, subnetPerCluster map[string]string) error
	GetSubnetPerCluster(network, clusterID string) (string, error)
	FreeSubnetPerCluster(clusterID string) error
	AcquireReservedSubnet(network string) error
	FreeReservedSubnet(network string) error
}

type IPAM struct {
	/* Map that store the network allocated for a given remote cluster */
	SubnetPerCluster map[string]string
	/* Set of networks from which IPAM takes new networks. */
	pools []string
	ipam  goipam.Ipamer
}

func NewIPAM() *IPAM {
	liqoIPAM := IPAM{
		SubnetPerCluster: make(map[string]string),
		pools:            make([]string, 0),
		ipam:             goipam.New(),
	}
	return &liqoIPAM
}

var Pools = []string{
	"10.0.0.0/8",
	"192.168.0.0/16",
	"172.16.0.0/12",
}

/* Init receives a set of networks that will be marked as used, and a slice of pools from which it will allocate subnets for remote clusters */
func (liqoIPAM *IPAM) Init(reservedNetworks map[string]string, networkPool []string, subnetPerCluster map[string]string) error {

	/* Set network pools */
	for _, network := range networkPool {
		if _, err := liqoIPAM.ipam.NewPrefix(network); err != nil {
			return fmt.Errorf("failed to create a new prefix for network %s", network)
		}
		liqoIPAM.pools = append(liqoIPAM.pools, network)
	}
	/* Acquire reserved networks */
	for _, network := range reservedNetworks {
		if err := liqoIPAM.AcquireReservedSubnet(network); err != nil {
			return err
		}
	}
	/* Store networks per cluster */
	liqoIPAM.SubnetPerCluster = subnetPerCluster
	return nil
}

/* AcquireReservedNetwork marks as used the network received as parameter */
func (liqoIPAM *IPAM) AcquireReservedSubnet(reservedNetwork string) error {
	klog.Infof("Request to reserve network %s has been received", reservedNetwork)
	klog.Infof("Trying to acquire network %s from one pool", reservedNetwork)
	pool, ok, err := liqoIPAM.getPoolFromNetwork(reservedNetwork)
	if err != nil {
		return err
	}
	if ok {
		klog.Infof("Network %s is contained in pool %s", reservedNetwork, pool)
		if _, err := liqoIPAM.ipam.AcquireSpecificChildPrefix(pool, reservedNetwork); err != nil {
			klog.Infof("Network %s has already been reserved", reservedNetwork)
			return nil
		}
		klog.Infof("Network %s has just been reserved", reservedNetwork)
		return nil
	}
	klog.Infof("Network %s is not contained in any pool", reservedNetwork)
	if _, err := liqoIPAM.ipam.NewPrefix(reservedNetwork); err != nil {
		klog.Infof("Network %s has already been reserved", reservedNetwork)
		return nil
	}
	klog.Infof("Network %s has just been reserved.", reservedNetwork)
	return nil
}

/* Function that receives a network as parameter and returns the pool to which this network belongs to. The second return parameter is a boolean: it is false if the network does not belong to any pool */
func (liqoIPAM *IPAM) getPoolFromNetwork(network string) (string, bool, error) {
	var poolIPset netaddr.IPSetBuilder
	// Build IPSet for new network
	ipprefix, err := netaddr.ParseIPPrefix(network)
	if err != nil {
		return "", false, err
	}
	for _, pool := range liqoIPAM.pools {
		// Build IPSet for pool
		c, err := netaddr.ParseIPPrefix(pool)
		if err != nil {
			return "", false, err
		}
		poolIPset.AddPrefix(c)
		// Check if the pool contains network
		if poolIPset.IPSet().ContainsPrefix(ipprefix) {
			return pool, true, nil
		}
	}
	return "", false, nil
}

/* GetSubnetPerCluster tries to reserve the network received as parameter for cluster clusterID. If it cannot allocate the network itself, GetSubnetPerCluster maps it to a new network. The network returned can be the original network, or the mapped network */
func (liqoIPAM *IPAM) GetSubnetPerCluster(network, clusterID string) (string, error) {
	var mappedNetwork string
	if value, ok := liqoIPAM.SubnetPerCluster[clusterID]; ok {
		return value, nil
	}
	klog.Infof("Network %s allocation request for cluster %s", network, clusterID)
	_, err := liqoIPAM.ipam.NewPrefix(network)
	if err != nil && !strings.Contains(err.Error(), "overlaps") {
		/* Overlapping is not considered an error in this context. */
		return "", fmt.Errorf("Cannot reserve network %s:%w", network, err)
	}
	if err == nil {
		klog.Infof("Network %s successfully assigned for cluster %s", network, clusterID)
		liqoIPAM.SubnetPerCluster[clusterID] = network
		return network, nil
	}
	/* Since NewPrefix failed, network belongs to a pool or it has been already reserved */
	klog.Infof("Cannot allocate network %s, checking if it belongs to any pool...", network)
	pool, ok, err := liqoIPAM.getPoolFromNetwork(network)
	if ok {
		klog.Infof("Network %s belongs to pool %s, trying to acquire it...", network, pool)
		_, err := liqoIPAM.ipam.AcquireSpecificChildPrefix(pool, network)
		if err != nil && !strings.Contains(err.Error(), "is not available") {
			/* Uknown error, return */
			return "", fmt.Errorf("cannot acquire prefix %s from prefix %s: %w", network, pool, err)
		}
		if err == nil {
			klog.Infof("Network %s successfully assigned to cluster %s", network, clusterID)
			liqoIPAM.SubnetPerCluster[clusterID] = network
			return network, nil
		}
		/* Network is not available, need a mapping */
		klog.Infof("Cannot acquire network %s from pool %s", network, pool)
	}
	/* Network is already reserved, need a mapping */
	klog.Infof("Looking for a mapping for network %s...", network)
	mappedNetwork, ok = liqoIPAM.mapNetwork(network)
	if !ok {
		return "", fmt.Errorf("Cannot assign any network to cluster %s", clusterID)
	}
	klog.Infof("Network %s successfully mapped to network %s", mappedNetwork, network)
	klog.Infof("Network %s successfully assigned to cluster %s", mappedNetwork, clusterID)
	liqoIPAM.SubnetPerCluster[clusterID] = mappedNetwork
	return mappedNetwork, nil
}

func (liqoIPAM *IPAM) mapNetwork(network string) (string, bool) {
	for _, pool := range liqoIPAM.pools {
		klog.Infof("Trying to acquire a child prefix from prefix %s (mask lenght=%d)", pool, getMask(network))
		if mappedNetwork, err := liqoIPAM.ipam.AcquireChildPrefix(pool, getMask(network)); err == nil {
			klog.Infof("Network %s has been mapped to network %s", network, mappedNetwork)
			return mappedNetwork.String(), true
		}
	}
	return "", false
}

/* Helper function to get a mask from a net.IPNet */
func getMask(network string) uint8 {
	stringMask := network[len(network)-2:]
	mask, _ := strconv.ParseInt(stringMask, 10, 8)
	return uint8(mask)
}

/* FreeReservedSubnet marks as free a reserved subnet */
func (liqoIPAM *IPAM) FreeReservedSubnet(network string) error {
	if p := liqoIPAM.ipam.PrefixFrom(network); p == nil {
		return fmt.Errorf("network %s is already available", network)
	} else {
		//Network exists
		if err := liqoIPAM.ipam.ReleaseChildPrefix(liqoIPAM.ipam.PrefixFrom(network)); err != nil {
			klog.Infof("Cannot release subnet %s previously allocated from the pools", liqoIPAM.ipam.PrefixFrom(network))
			if _, err := liqoIPAM.ipam.DeletePrefix(network); err != nil {
				klog.Errorf("Cannot delete prefix %s", network)
				return fmt.Errorf("cannot remove subnet %s", network)
			}
		}
	}
	return nil
}

/* FreeSubnetPerCluster marks as free the network previously allocated for cluster clusterID */
func (liqoIPAM *IPAM) FreeSubnetPerCluster(clusterID string) error {
	if _, ok := liqoIPAM.SubnetPerCluster[clusterID]; !ok {
		//Network does not exists
		return nil
	}
	if err := liqoIPAM.FreeReservedSubnet(liqoIPAM.SubnetPerCluster[clusterID]); err != nil {
		return err
	}
	delete(liqoIPAM.SubnetPerCluster, clusterID)
	return nil
}
