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

package consts

import "k8s.io/apimachinery/pkg/labels"

const (
	// ServiceAccountNamespacePath contains the path where the namespace is stored in the serviceaccount volume mount.
	ServiceAccountNamespacePath = "/var/run/secrets/kubernetes.io/serviceaccount/namespace"
	// ClusterIDLabelName is the name of the label key to use with Cluster ID.
	ClusterIDLabelName = "clusterID"
	// ClusterIDConfigMapKey is the key of the configmap where the cluster-id is stored.
	ClusterIDConfigMapKey = "CLUSTER_ID"
)

// ClusterIDConfigMapSelector returns the selector for the configmap where the cluster-id is stored.
func ClusterIDConfigMapSelector() labels.Selector {
	return labels.SelectorFromSet(labels.Set{
		"app.kubernetes.io/component": "clusterid-configmap",
		"app.kubernetes.io/name":      "clusterid-configmap",
	})
}
