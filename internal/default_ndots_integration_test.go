package internal_test

import (
	"bytes"
	"default-ndots-admission-controller/internal"
	"encoding/json"
	"fmt"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"io"
	admissionv1 "k8s.io/api/admission/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/uuid"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

const defaultNdotsVal = 2

// TestDefaultNdots simulates serving /webhook endpoint in a real server then run series of tests against it
func TestDefaultNdots(t *testing.T) {
	assertEqual := func(actualAllowed bool, actualPatchType *admissionv1.PatchType, expectedPatch, actualPatch string) {
		assert.Equal(t, true, actualAllowed)
		if expectedPatch != "" && assert.NotNil(t, actualPatchType) {
			assert.Equal(t, admissionv1.PatchTypeJSONPatch, *actualPatchType)
		}
		assert.Equal(t, expectedPatch, actualPatch)
	}
	expectedNdotsVal := "1"
	tests := []struct {
		description    string
		resourceType   string
		podDnsConfig   *corev1.PodDNSConfig
		assertResponse func(response admissionv1.AdmissionReview)
	}{
		{
			description:  "Test deployment with no dnsConfig",
			resourceType: internal.DeploymentsResource,
			assertResponse: func(admissionReview admissionv1.AdmissionReview) {
				expectedPatch := fmt.Sprintf(`[{"op":"replace","path":"/spec/template/spec/dnsConfig","value":{"options":[{"name":"%s","value":"%d"}]}}]`, internal.NDotsKey, defaultNdotsVal)
				assertEqual(admissionReview.Response.Allowed, admissionReview.Response.PatchType, expectedPatch, string(admissionReview.Response.Patch))
			},
		},
		{
			description:  "Test daemonset with no dnsConfig",
			resourceType: internal.DaemonsetsResource,
			assertResponse: func(admissionReview admissionv1.AdmissionReview) {
				expectedPatch := fmt.Sprintf(`[{"op":"replace","path":"/spec/template/spec/dnsConfig","value":{"options":[{"name":"%s","value":"%d"}]}}]`, internal.NDotsKey, defaultNdotsVal)
				assertEqual(admissionReview.Response.Allowed, admissionReview.Response.PatchType, expectedPatch, string(admissionReview.Response.Patch))
			},
		},
		{
			description:  "Test deployment with nameservers but no " + internal.NDotsKey,
			resourceType: internal.DeploymentsResource,
			podDnsConfig: &corev1.PodDNSConfig{
				Nameservers: []string{"8.8.8.8"},
			},
			assertResponse: func(admissionReview admissionv1.AdmissionReview) {
				expectedPatch := fmt.Sprintf(`[{"op":"replace","path":"/spec/template/spec/dnsConfig","value":{"nameservers":["8.8.8.8"],"options":[{"name":"%s","value":"%d"}]}}]`, internal.NDotsKey, defaultNdotsVal)
				assertEqual(admissionReview.Response.Allowed, admissionReview.Response.PatchType, expectedPatch, string(admissionReview.Response.Patch))
			},
		},
		{
			description:  "Test daemonset with nameservers but no " + internal.NDotsKey,
			resourceType: internal.DaemonsetsResource,
			podDnsConfig: &corev1.PodDNSConfig{
				Nameservers: []string{"8.8.4.4"},
			},
			assertResponse: func(admissionReview admissionv1.AdmissionReview) {
				expectedPatch := fmt.Sprintf(`[{"op":"replace","path":"/spec/template/spec/dnsConfig","value":{"nameservers":["8.8.4.4"],"options":[{"name":"%s","value":"%d"}]}}]`, internal.NDotsKey, defaultNdotsVal)
				assertEqual(admissionReview.Response.Allowed, admissionReview.Response.PatchType, expectedPatch, string(admissionReview.Response.Patch))
			},
		},
		{
			description:  "Test deployment with " + internal.NDotsKey,
			resourceType: internal.DeploymentsResource,
			podDnsConfig: &corev1.PodDNSConfig{
				Options: []corev1.PodDNSConfigOption{{Name: internal.NDotsKey, Value: &expectedNdotsVal}},
			},
			assertResponse: func(admissionReview admissionv1.AdmissionReview) {
				assertEqual(admissionReview.Response.Allowed, admissionReview.Response.PatchType, "", string(admissionReview.Response.Patch))
			},
		},
		{
			description:  "Test daemonset with " + internal.NDotsKey,
			resourceType: internal.DaemonsetsResource,
			podDnsConfig: &corev1.PodDNSConfig{
				Options: []corev1.PodDNSConfigOption{{Name: internal.NDotsKey, Value: &expectedNdotsVal}},
			},
			assertResponse: func(admissionReview admissionv1.AdmissionReview) {
				assertEqual(admissionReview.Response.Allowed, admissionReview.Response.PatchType, "", string(admissionReview.Response.Patch))
			},
		},
	}
	for _, test := range tests {
		requestAdmissionReview, err := createAdmissionReview(test.resourceType, test.podDnsConfig)
		if err != nil {
			t.Error(test.description, " failed to create request admission review. Reason: ", err)
			// Bug in the test itself. There's no point to continue
			break
		}
		body, _ := json.Marshal(requestAdmissionReview)
		req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		handlers := internal.NewHandlers(defaultNdotsVal, time.Second*5)
		handlers.ServeHTTP(w, req)
		res := w.Result()
		if res.Body == nil {
			t.Error(test.description, "failed. Webhook returned an empty response")
			continue
		}
		bodyBytes, err := io.ReadAll(res.Body)
		if err != nil {
			t.Error(test.description, "failed. Unable to read response body")
			continue
		}
		str := string(bodyBytes)
		log.Info(str)
		var responseAdmissionReview admissionv1.AdmissionReview
		err = json.Unmarshal(bodyBytes, &responseAdmissionReview)
		if err != nil {
			t.Error(test.description, "failed. Unable to unmarshall response body")
			continue
		}
		assert.Equal(t, requestAdmissionReview.Request.UID, responseAdmissionReview.Response.UID)
		test.assertResponse(responseAdmissionReview)
	}
}

func createAdmissionReview(resourceType string, podDnsConfig *corev1.PodDNSConfig) (admissionv1.AdmissionReview, error) {
	var (
		raw []byte
		err error
	)
	if resourceType == internal.DaemonsetsResource {
		raw, err = json.Marshal(appsv1.DaemonSet{
			Spec: appsv1.DaemonSetSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						DNSConfig: podDnsConfig,
					},
				},
			},
		})
		if err != nil {
			return admissionv1.AdmissionReview{}, err
		}
	} else {
		raw, err = json.Marshal(appsv1.Deployment{
			Spec: appsv1.DeploymentSpec{
				Template: corev1.PodTemplateSpec{
					Spec: corev1.PodSpec{
						DNSConfig: podDnsConfig,
					},
				},
			},
		})
		if err != nil {
			return admissionv1.AdmissionReview{}, err
		}
	}

	// Webhooks are sent as POST requests, with Content-Type: application/json, with an AdmissionReview API object in the admission.k8s.io API group serialized to JSON as the body.
	// see
	// - https://kubernetes.io/docs/reference/access-authn-authz/extensible-admission-controllers/#request
	// - https://www.baeldung.com/java-kubernetes-admission-controller#1-admission-review-requests
	return admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			Kind:       "AdmissionReview",
			APIVersion: "admission.k8s.io/v1",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: uuid.NewUUID(),
			Kind: metav1.GroupVersionKind{
				Group:   "apps",
				Version: "v1",
				Kind:    resourceType,
			},
			Resource: metav1.GroupVersionResource{
				Group:    "apps",
				Version:  "v1",
				Resource: resourceType,
			},
			Name:      "test-" + resourceType,
			Namespace: "test-namespace",
			Operation: admissionv1.Create,
			Object:    runtime.RawExtension{Raw: raw},
		},
		Response: nil,
	}, nil
}

func TestProcessRequestFromJson(t *testing.T) {
	description := "Test process from json request"
	contents, err := os.ReadFile("test_admission_review_request.json")
	if err != nil {
		t.Error(err)
		return
	}
	req := httptest.NewRequest(http.MethodPost, "/webhook", bytes.NewReader(contents))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	handlers := internal.NewHandlers(defaultNdotsVal, time.Second*5)
	handlers.ServeHTTP(w, req)
	res := w.Result()
	if res.Body == nil {
		t.Error(description, "failed. Webhook returned an empty response")
		return
	}
	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		t.Error(description, "failed. Unable to read response body")
		return
	}
	str := string(bodyBytes)
	log.Info(str)
	var responseAdmissionReview admissionv1.AdmissionReview
	err = json.Unmarshal(bodyBytes, &responseAdmissionReview)
	if err != nil {
		t.Error(description, "failed. Unable to unmarshall response body")
		return
	}
	assert.Equal(t, types.UID("6fab93f8-abdc-4c4d-bc5f-3e1e1f0cac33"), responseAdmissionReview.Response.UID)
	assert.Equal(t, true, responseAdmissionReview.Response.Allowed)
	if assert.NotNil(t, responseAdmissionReview.Response.PatchType) {
		assert.Equal(t, admissionv1.PatchTypeJSONPatch, *responseAdmissionReview.Response.PatchType)
	}
	expectedPatch := fmt.Sprintf(`[{"op":"replace","path":"/spec/template/spec/dnsConfig","value":{"options":[{"name":"%s","value":"%d"}]}}]`, internal.NDotsKey, defaultNdotsVal)
	assert.Equal(t, expectedPatch, string(responseAdmissionReview.Response.Patch))
}
