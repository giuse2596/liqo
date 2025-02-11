// Copyright 2019-2021 The Liqo Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package install

import (
	"context"
	"fmt"

	helm "github.com/mittwald/go-helm-client"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/action"
	"k8s.io/client-go/rest"
	"k8s.io/klog/v2"

	"github.com/liqotech/liqo/pkg/liqoctl/install/aks"
	"github.com/liqotech/liqo/pkg/liqoctl/install/eks"
	"github.com/liqotech/liqo/pkg/liqoctl/install/gke"
	"github.com/liqotech/liqo/pkg/liqoctl/install/k3s"
	"github.com/liqotech/liqo/pkg/liqoctl/install/kind"
	"github.com/liqotech/liqo/pkg/liqoctl/install/kubeadm"
	"github.com/liqotech/liqo/pkg/liqoctl/install/openshift"
	"github.com/liqotech/liqo/pkg/liqoctl/install/provider"
	installutils "github.com/liqotech/liqo/pkg/liqoctl/install/utils"
	"github.com/liqotech/liqo/pkg/utils"
)

func getProviderInstance(providerType string) provider.InstallProviderInterface {
	switch providerType {
	case "kubeadm":
		return kubeadm.NewProvider()
	case "kind":
		return kind.NewProvider()
	case "k3s":
		return k3s.NewProvider()
	case "eks":
		return eks.NewProvider()
	case "gke":
		return gke.NewProvider()
	case "aks":
		return aks.NewProvider()
	case "openshift":
		return openshift.NewProvider()
	default:
		return nil
	}
}

func initHelmClient(config *rest.Config, arguments *provider.CommonArguments) (*helm.HelmClient, error) {
	helmClient, err := InitializeHelmClientWithRepo(config, arguments)
	if err != nil {
		fmt.Printf("Unable to create helmClient: %s", err)
		return nil, err
	}
	return helmClient, nil
}

func installOrUpdate(ctx context.Context, helmClient *helm.HelmClient, k provider.InstallProviderInterface, cArgs *provider.CommonArguments) error {
	output, _, err := helmClient.GetChart(cArgs.ChartPath, &action.ChartPathOptions{Version: cArgs.Version})

	if err != nil {
		return err
	}

	providerValues := make(map[string]interface{})
	k.UpdateChartValues(providerValues)
	values, err := generateValues(output.Values, cArgs.CommonValues, providerValues)
	if err != nil {
		return err
	}

	raw, err := yaml.Marshal(values)
	if err != nil {
		return err
	}

	if cArgs.DumpValues {
		if err := utils.WriteFile(cArgs.DumpValuesPath, raw); err != nil {
			klog.Errorf("Unable to write the Values file in location: %s", cArgs.DumpValuesPath)
			return err
		}
	} else {
		chartSpec := helm.ChartSpec{
			// todo (palexster): Check if it ReleaseName and LiqoNamespace are really configurable
			ReleaseName:      installutils.LiqoReleaseName,
			ChartName:        cArgs.ChartPath,
			Namespace:        installutils.LiqoNamespace,
			ValuesYaml:       string(raw),
			DependencyUpdate: true,
			Timeout:          cArgs.Timeout,
			GenerateName:     false,
			CreateNamespace:  true,
			DryRun:           cArgs.DryRun,
			Devel:            cArgs.Devel,
			Wait:             true,
		}

		// provide the possibility to exit installation on context cancellation
		errCh := make(chan error)
		defer close(errCh)
		go func() {
			_, err = helmClient.InstallOrUpgradeChart(ctx, &chartSpec)
			errCh <- err
		}()

		select {
		case err = <-errCh:
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return nil
		}
	}
	return nil
}

func generateValues(chartValues, commonValues, providerValues map[string]interface{}) (map[string]interface{}, error) {
	intermediateValues, err := installutils.FusionMap(chartValues, commonValues)
	if err != nil {
		return nil, err
	}
	finalValues, err := installutils.FusionMap(intermediateValues, providerValues)
	if err != nil {
		return nil, err
	}
	return finalValues, nil
}
