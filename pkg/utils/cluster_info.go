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

package utils

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/liqotech/liqo/pkg/consts"
	"github.com/liqotech/liqo/pkg/virtualKubelet"
)

// GetClusterIDWithNativeClient returns clusterID using a kubernetes.Interface client.
func GetClusterIDWithNativeClient(ctx context.Context, nativeClient kubernetes.Interface, namespace string) (string, error) {
	cmClient := nativeClient.CoreV1().ConfigMaps(namespace)
	configMapList, err := cmClient.List(ctx, metav1.ListOptions{
		LabelSelector: consts.ClusterIDConfigMapSelector().String(),
	})
	if err != nil {
		return "", err
	}

	return getClusterIDFromConfigMapList(configMapList)
}

// GetClusterIDWithControllerClient returns clusterID using a client.Client client.
func GetClusterIDWithControllerClient(ctx context.Context, controllerClient client.Client, namespace string) (string, error) {
	var configMapList corev1.ConfigMapList
	if err := controllerClient.List(ctx, &configMapList,
		client.MatchingLabelsSelector{Selector: consts.ClusterIDConfigMapSelector()},
		client.InNamespace(namespace)); err != nil {
		klog.Errorf("%s, unable to get the ClusterID ConfigMap in namespace '%s'", err, namespace)
		return "", err
	}

	return getClusterIDFromConfigMapList(&configMapList)
}

func getClusterIDFromConfigMapList(configMapList *corev1.ConfigMapList) (string, error) {
	switch len(configMapList.Items) {
	case 0:
		return "", apierrors.NewNotFound(schema.GroupResource{
			Group:    "v1",
			Resource: "configmaps",
		}, "clusterid-configmap")
	case 1:
		clusterID := configMapList.Items[0].Data[consts.ClusterIDConfigMapKey]
		klog.Infof("ClusterID is '%s'", clusterID)
		return clusterID, nil
	default:
		return "", fmt.Errorf("multiple clusterID configmaps found")
	}
}

// GetClusterIDFromNodeName returns the clusterID from a node name.
func GetClusterIDFromNodeName(nodeName string) string {
	return strings.TrimPrefix(nodeName, virtualKubelet.VirtualNodePrefix)
}

// RetrieveNamespace tries to retrieve the name of the namespace where the process is executed.
// It tries to get the namespace:
// - Firstly, using the POD_NAMESPACE variable
// - Secondly, by looking for the namespace value contained in a mounted ServiceAccount (if any)
// Otherwise, it returns an empty string and an error.
func RetrieveNamespace() (string, error) {
	namespace, found := os.LookupEnv("POD_NAMESPACE")
	if !found {
		klog.Info("POD_NAMESPACE not set")
		data, err := ioutil.ReadFile(consts.ServiceAccountNamespacePath)
		if err != nil {
			return "", fmt.Errorf("unable to get namespace")
		}
		if namespace = strings.TrimSpace(string(data)); namespace == "" {
			return "", fmt.Errorf("unable to get namespace")
		}
	}
	return namespace, nil
}

// GetRestConfig returns a rest.Config object to initialize a client to the target cluster.
func GetRestConfig(configPath string) (config *rest.Config, err error) {
	if _, err = os.Stat(configPath); err == nil {
		// Get the kubeconfig from the filepath.
		config, err = clientcmd.BuildConfigFromFlags("", configPath)
	} else {
		// Set to in-cluster config.
		config, err = rest.InClusterConfig()
	}
	return config, err
}
