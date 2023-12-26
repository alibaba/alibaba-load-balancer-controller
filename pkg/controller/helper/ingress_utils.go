package helper

import (
	"strings"

	alibabacloudv1 "k8s.io/alibaba-load-balancer-controller/pkg/apis/alibabacloud/v1"
	"k8s.io/alibaba-load-balancer-controller/pkg/util/hash"
	networkingv1 "k8s.io/api/networking/v1"
)

const (
	LabelAlbHash = "alb.ingress.kubernetes.io/hash"
)

func GetIngressHash(ing *networkingv1.Ingress) string {
	var op []interface{}
	op = append(op, ing.Spec, ing.Annotations, ing.DeletionTimestamp)
	return hash.HashObject(op)
}

func IsIngressHashChanged(ing *networkingv1.Ingress) bool {
	if oldHash, ok := ing.Labels[LabelAlbHash]; ok {
		newHash := GetIngressHash(ing)
		return !strings.EqualFold(oldHash, newHash)
	}
	return true
}

func GetAlbConfigHash(albConfig *alibabacloudv1.AlbConfig) string {
	var op []interface{}
	op = append(op, albConfig.Spec, albConfig.Annotations, albConfig.DeletionTimestamp)
	return hash.HashObject(op)
}

func IsAlbConfigHashChanged(cfg *alibabacloudv1.AlbConfig) bool {
	if oldHash, ok := cfg.Labels[LabelAlbHash]; ok {
		newHash := GetAlbConfigHash(cfg)
		return !strings.EqualFold(oldHash, newHash)
	}
	return true
}
