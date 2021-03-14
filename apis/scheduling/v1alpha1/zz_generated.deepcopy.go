// +build !ignore_autogenerated

/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1alpha1

import (
	"k8s.io/api/core/v1"
	resource "k8s.io/apimachinery/pkg/api/resource"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SchedulingNode) DeepCopyInto(out *SchedulingNode) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SchedulingNode.
func (in *SchedulingNode) DeepCopy() *SchedulingNode {
	if in == nil {
		return nil
	}
	out := new(SchedulingNode)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SchedulingNode) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SchedulingNodeList) DeepCopyInto(out *SchedulingNodeList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]SchedulingNode, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SchedulingNodeList.
func (in *SchedulingNodeList) DeepCopy() *SchedulingNodeList {
	if in == nil {
		return nil
	}
	out := new(SchedulingNodeList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *SchedulingNodeList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SchedulingNodeSpec) DeepCopyInto(out *SchedulingNodeSpec) {
	*out = *in
	if in.Images != nil {
		in, out := &in.Images, &out.Images
		*out = make([]v1.ContainerImage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
	in.LimitRange.DeepCopyInto(&out.LimitRange)
	in.ResourceQuota.DeepCopyInto(&out.ResourceQuota)
	if in.Neighbors != nil {
		in, out := &in.Neighbors, &out.Neighbors
		*out = make(map[v1.ResourceName]v1.ResourceList, len(*in))
		for key, val := range *in {
			var outVal map[v1.ResourceName]resource.Quantity
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make(v1.ResourceList, len(*in))
				for key, val := range *in {
					(*out)[key] = val.DeepCopy()
				}
			}
			(*out)[key] = outVal
		}
	}
	if in.Properties != nil {
		in, out := &in.Properties, &out.Properties
		*out = make(map[v1.ResourceName]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.Prices != nil {
		in, out := &in.Prices, &out.Prices
		*out = make(v1.ResourceList, len(*in))
		for key, val := range *in {
			(*out)[key] = val.DeepCopy()
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SchedulingNodeSpec.
func (in *SchedulingNodeSpec) DeepCopy() *SchedulingNodeSpec {
	if in == nil {
		return nil
	}
	out := new(SchedulingNodeSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *SchedulingNodeStatus) DeepCopyInto(out *SchedulingNodeStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new SchedulingNodeStatus.
func (in *SchedulingNodeStatus) DeepCopy() *SchedulingNodeStatus {
	if in == nil {
		return nil
	}
	out := new(SchedulingNodeStatus)
	in.DeepCopyInto(out)
	return out
}
