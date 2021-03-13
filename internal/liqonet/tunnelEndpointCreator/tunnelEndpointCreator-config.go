package tunnelEndpointCreator

import (
	"context"
	"fmt"
	"net"
	"os"
	"reflect"

	configv1alpha1 "github.com/liqotech/liqo/apis/config/v1alpha1"
	netv1alpha1 "github.com/liqotech/liqo/apis/net/v1alpha1"
	"github.com/liqotech/liqo/pkg/clusterConfig"
	"github.com/liqotech/liqo/pkg/crdClient"
	liqonetOperator "github.com/liqotech/liqo/pkg/liqonet"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/klog"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (tec *TunnelEndpointCreator) GetConfiguration(config *configv1alpha1.ClusterConfig) (map[string]string, error) {
	correctlyParsed := true
	reservedSubnets := make(map[string]string, 0)
	liqonetConfig := config.Spec.LiqonetConfig
	_, _, err := net.ParseCIDR(config.Spec.LiqonetConfig.PodCIDR)
	if err != nil {
		klog.Errorf("an error occurred while parsing the podCIDR: %s", err)
		return nil, err
	} else {
		reservedSubnets[config.Spec.LiqonetConfig.PodCIDR] = config.Spec.LiqonetConfig.PodCIDR
	}
	_, _, err = net.ParseCIDR(config.Spec.LiqonetConfig.ServiceCIDR)
	if err != nil {
		klog.Errorf("an error occurred while parsing the serviceCIDR: %s", err)
		return nil, err
	} else {
		reservedSubnets[config.Spec.LiqonetConfig.ServiceCIDR] = config.Spec.LiqonetConfig.ServiceCIDR
	}
	//check that the reserved subnets are in the right format
	for _, subnet := range liqonetConfig.ReservedSubnets {
		_, _, err := net.ParseCIDR(subnet)
		if err != nil {
			klog.Errorf("an error occurred while parsing configuration: %s", err)
			correctlyParsed = false
		} else {
			//klog.Infof("subnet %s correctly added to the reserved subnets", sn.String())
			reservedSubnets[subnet] = subnet
		}
	}
	if !correctlyParsed {
		return nil, fmt.Errorf("the reserved subnets list is not in the correct format")
	}
	return reservedSubnets, nil
}

func (tec *TunnelEndpointCreator) SetNetParameters(config *configv1alpha1.ClusterConfig) {
	podCIDR := config.Spec.LiqonetConfig.PodCIDR
	serviceCIDR := config.Spec.LiqonetConfig.ServiceCIDR
	if tec.PodCIDR != podCIDR {
		klog.Infof("setting podCIDR to %s", podCIDR)
		tec.PodCIDR = podCIDR
	}
	if tec.ServiceCIDR != serviceCIDR {
		klog.Infof("setting serviceCIDR to %s", serviceCIDR)
		tec.ServiceCIDR = serviceCIDR
	}
}

//it returns the subnets used by the foreign clusters
//get the list of all tunnelEndpoint CR and saves the address space assigned to the
//foreign cluster.
func (tec *TunnelEndpointCreator) GetClustersSubnets() (map[string]string, error) {
	ctx := context.Background()
	var err error
	var tunEndList netv1alpha1.TunnelEndpointList
	subnets := make(map[string]string)

	//if the error is ErrCacheNotStarted we retry until the chaches are ready
	chacheChan := make(chan struct{})
	started := tec.Manager.GetCache().WaitForCacheSync(chacheChan)
	if !started {
		return nil, fmt.Errorf("unable to sync caches")
	}

	err = tec.Client.List(ctx, &tunEndList, &client.ListOptions{})
	if err != nil {
		klog.Errorf("unable to get the list of tunnelEndpoint custom resources -> %s", err)
		return nil, err
	}
	//if the list is empty return a nil slice and nil error
	if tunEndList.Items == nil {
		return nil, nil
	}
	for _, tunEnd := range tunEndList.Items {
		if tunEnd.Status.LocalRemappedPodCIDR != "" && tunEnd.Status.LocalRemappedPodCIDR != DefaultPodCIDRValue {
			subnets[tunEnd.Status.LocalRemappedPodCIDR] = tunEnd.Status.LocalRemappedPodCIDR
			klog.Infof("subnet %s already reserved for cluster %s", tunEnd.Status.LocalRemappedPodCIDR, tunEnd.Spec.ClusterID)
		} else if tunEnd.Status.LocalRemappedPodCIDR == DefaultPodCIDRValue {
			subnets[tunEnd.Spec.PodCIDR] = tunEnd.Spec.PodCIDR
			klog.Infof("subnet %s already reserved for cluster %s", tunEnd.Spec.PodCIDR, tunEnd.Spec.ClusterID)
		}
	}
	return subnets, nil
}

func (tec *TunnelEndpointCreator) InitConfiguration(reservedSubnets, clusterSubnets map[string]string) error {
	//here we acquire the lock of the mutex
	tec.Mutex.Lock()
	defer tec.Mutex.Unlock()
	// Reserved networks will be marked as used by ipam
	reserved := make([]string, 0)
	for _, network := range reservedSubnets {
		reserved = append(reserved, network)
	}
	for _, network := range clusterSubnets {
		reserved = append(reserved, network)
	}
	if err := tec.IPManager.Init(reserved, liqonetOperator.Pools); err != nil {
		klog.Errorf("an error occurred while initializing the IP manager -> err")
		return err
	}

	tec.ReservedSubnets = reservedSubnets
	return nil
}

func (tec *TunnelEndpointCreator) UpdateConfiguration(reservedSubnets map[string]string) error {
	//If the configuration is the same return
	if reflect.DeepEqual(reservedSubnets, tec.ReservedSubnets) {
		//klog.Infof("no changes were made at the configuration")
		return nil
	}
	tec.Mutex.Lock()
	defer tec.Mutex.Unlock()
	//save the newly added subnets in the configuration
	for _, values := range reservedSubnets {
		if _, ok := tec.ReservedSubnets[values]; !ok {
			if err := tec.IPManager.AcquireReservedSubnet(values); err != nil {
				return err
			}
			klog.Infof("new subnet to be reserved is added to the configuration file: %s", values)
		}
	}
	//save the removed subnets from the configuration
	for _, values := range tec.ReservedSubnets {
		if _, ok := reservedSubnets[values]; !ok {
			if err := tec.IPManager.FreeReservedSubnet(values); err != nil {
				klog.Errorf("cannot free network %s", values)
			}
			klog.Infof("a reserved subnet is removed from the configuration file: %s", values)
		}
	}
	return nil
}

func (tec *TunnelEndpointCreator) WatchConfiguration(config *rest.Config, gv *schema.GroupVersion) {
	config.ContentConfig.GroupVersion = gv
	config.APIPath = "/apis"
	config.NegotiatedSerializer = scheme.Codecs.WithoutConversion()
	config.UserAgent = rest.DefaultKubernetesUserAgent()
	CRDclient, err := crdClient.NewFromConfig(config)
	if err != nil {
		klog.Error(err, err.Error())
		os.Exit(1)
	}

	go clusterConfig.WatchConfiguration(func(configuration *configv1alpha1.ClusterConfig) {

		//this section is executed at start-up time
		if !tec.IpamConfigured {
			//get the reserved subnets from che configuration CRD
			reservedSubnets, err := tec.GetConfiguration(configuration)
			if err != nil {
				klog.Error(err)
				return
			}
			//get subnets used by foreign clusters
			clusterSubnets, err := tec.GetClustersSubnets()
			if err != nil {
				klog.Error(err)
				return
			}
			if err := tec.InitConfiguration(reservedSubnets, clusterSubnets); err != nil {
				klog.Error(err)
				return
			}
			tec.IpamConfigured = true
		} else {
			//get the reserved subnets from che configuration CRD
			reservedSubnets, err := tec.GetConfiguration(configuration)
			if err != nil {
				klog.Error(err)
				return
			}
			if err := tec.UpdateConfiguration(reservedSubnets); err != nil {
				klog.Error(err)
				return
			}
		}
		tec.SetNetParameters(configuration)
		if !tec.cfgConfigured {
			tec.WaitConfig.Done()
			klog.Infof("called done on waitgroup")
			tec.cfgConfigured = true
		}
		/*if !tec.RunningWatchers {
			tec.ForeignClusterStartWatcher <- true
		}*/

	}, CRDclient, "")
}
