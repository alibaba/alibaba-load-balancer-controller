package apis

import (
	v1 "k8s.io/alibaba-load-balancer-controller/pkg/apis/alibabacloud/v1"
)

func init() {
	// Register the types with the Scheme so the components can map objects to GroupVersionKinds and back
	AddToSchemes = append(AddToSchemes, v1.SchemeBuilder.AddToScheme)
}
