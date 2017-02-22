// +build !ignore_autogenerated

/*
Copyright 2016 The Kubernetes Authors All rights reserved.

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

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package intstr

import (
	conversion "k8s.io/kubernetes/pkg/conversion"
)

func DeepCopy_intstr_IntOrString(in IntOrString, out *IntOrString, c *conversion.Cloner) error {
	out.Type = in.Type
	out.IntVal = in.IntVal
	out.StrVal = in.StrVal
	return nil
}
