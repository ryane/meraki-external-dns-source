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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var (
	DNSEndpointGroupVersion = schema.GroupVersion{
		Group:   "externaldns.k8s.io",
		Version: "v1alpha1",
	}
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// MerakiRef is a reference to a Meraki resource
type MerakiRef struct {
	Name string `json:"name,omitempty"`
	ID   string `json:"id,omitempty"`
}

// MerakiSourceSpec defines the desired state of MerakiSource
type MerakiSourceSpec struct {
	// Organization is a reference to the organization to query (name or id)
	Organization MerakiRef `json:"organization,omitempty"`

	// Network is a reference to the network to query (name or id)
	Network MerakiRef `json:"network,omitempty"`

	// Domain is the DNS suffix to use for the client DNS registration
	Domain string `json:"domain,omitempty"`

	// +kubebuilder:validation:Minimum=0

	// TTL requests the TTL of the record for the client. The actual TTL that is
	// used will depend on the provider
	// https://github.com/kubernetes-sigs/external-dns/blob/master/docs/ttl.md
	TTL *int64 `json:"ttl,omitempty"`
}

// MerakiSourceStatus defines the observed state of MerakiSource
type MerakiSourceStatus struct {
	// Endpoint is a pointer to the managed DNSEndpoint
	// +optional
	Endpoint corev1.ObjectReference `json:"endpoint,omitempty"`

	// SyncedAt is the time the endpoint was last synced from Meraki
	// +optional
	SyncedAt *metav1.Time `json:"syncedAt,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// MerakiSource is the Schema for the merakisources API
type MerakiSource struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   MerakiSourceSpec   `json:"spec,omitempty"`
	Status MerakiSourceStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// MerakiSourceList contains a list of MerakiSource
type MerakiSourceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []MerakiSource `json:"items"`
}

func init() {
	SchemeBuilder.Register(&MerakiSource{}, &MerakiSourceList{})
}
