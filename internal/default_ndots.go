package internal

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
)

const (
	NDotsKey = "ndots"
)

type DefaultNDotsAdmitHandler struct {
	ndotsValue int
}

func NewDefaultNDotsAdmitHandler(ndotsValue int) *DefaultNDotsAdmitHandler {
	return &DefaultNDotsAdmitHandler{ndotsValue: ndotsValue}
}

// admitHander updates the [dnsConfig] with ndots. It's like doing kubectl patch - Eg.
//
//	kubectl patchDnsConfig deployment some_deployment --type json -p '[{"op":"replace","path":"/spec/template/spec/dnsConfig","value":{"options":[{"name":"ndots","value":"2"}]}}]'
//
// [dnsConfig]: https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.25/#poddnsconfig-v1-core
func (d *DefaultNDotsAdmitHandler) admitHander(ar admissionv1.AdmissionReview) *admissionv1.AdmissionResponse {
	log.Debug("Default ndots admission controller triggered.")
	if ar.Request.Operation == admissionv1.Connect || ar.Request.Operation == admissionv1.Delete {
		// The accompanying MutatingWebhookConfiguration should be configured to only handle CREATE and UPDATE
		// But just in-case it is misconfigured, we handle it in a non-invasive way
		// see https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#matching-requests-rules
		log.Errorf("Default ndots admission controller is misconfigured. It is unexpectedly handling %s operation.", ar.Request.Operation)
		return &admissionv1.AdmissionResponse{
			UID:     ar.Request.UID,
			Allowed: true,
		}
	}

	daemonsetResource := metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "daemonset"}
	deploymentResource := metav1.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployment"}
	if ar.Request.Resource == daemonsetResource {
		daemonset := appsv1.DaemonSet{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(ar.Request.Object.Raw, nil, &daemonset); err != nil {
			log.Error(err)
			return toV1AdmissionResponse(err)
		}
		return d.mutateNdots(daemonset.Spec.Template.Spec.DNSConfig)
	} else if ar.Request.Resource == deploymentResource {
		deployment := appsv1.Deployment{}
		deserializer := codecs.UniversalDeserializer()
		if _, _, err := deserializer.Decode(ar.Request.Object.Raw, nil, &deployment); err != nil {
			log.Error(err)
			return toV1AdmissionResponse(err)
		}
		return d.mutateNdots(deployment.Spec.Template.Spec.DNSConfig)
	} else {
		// Log the error but admit the resource to make this controller non-invasive
		log.Errorf("Default ndots admission controller is misconfigured. It is unexpectedly handling %s resource.", ar.Request.Resource)
		return &admissionv1.AdmissionResponse{
			UID:     ar.Request.UID,
			Allowed: true,
		}
	}
}

// mutateNdots adds ndots to dnsConfig if it doesn't exist. This function is implemented to be non-invasive. If it produces error, it will always permit the request to proceed.
func (d *DefaultNDotsAdmitHandler) mutateNdots(dnsConfigToCheck *corev1.PodDNSConfig) *admissionv1.AdmissionResponse {
	defaultNDotsValue := strconv.Itoa(d.ndotsValue)
	reviewResponse := admissionv1.AdmissionResponse{}
	reviewResponse.Allowed = true

	var podDnsConfig *corev1.PodDNSConfig
	if dnsConfigToCheck == nil {
		podDnsConfig = &corev1.PodDNSConfig{
			Options: []corev1.PodDNSConfigOption{
				{
					Name:  NDotsKey,
					Value: &defaultNDotsValue,
				},
			},
		}
	} else {
		hasNoNdots := true
		for _, opt := range dnsConfigToCheck.Options {
			if opt.Name == NDotsKey {
				hasNoNdots = false
				break
			}
		}
		if hasNoNdots {
			podDnsConfig = &corev1.PodDNSConfig{
				Nameservers: dnsConfigToCheck.Nameservers,
				Searches:    dnsConfigToCheck.Searches,
			}
			podDnsConfig.Options = append(podDnsConfig.Options, corev1.PodDNSConfigOption{
				Name:  NDotsKey,
				Value: &defaultNDotsValue,
			})
		}
	}
	if podDnsConfig != nil {
		patchBytes, err := json.Marshal([]patchDnsConfig{{
			Op:    "replace",
			Path:  "/spec/template/spec/dnsConfig",
			Value: *podDnsConfig,
		}})
		if err != nil {
			// Log the error but admit the resource to make this controller non-invasive
			log.Error("Failed to marchall patch dns config. Reason:", err)
		} else {
			reviewResponse.Patch = patchBytes
		}
	}

	return &reviewResponse
}

// patchDnsConfig is a format for describing changes to a JSON document - see https://jsonpatch.com/
type patchDnsConfig struct {
	Op    string              `json:"op"`
	Path  string              `json:"path"`
	Value corev1.PodDNSConfig `json:"value"`
}
