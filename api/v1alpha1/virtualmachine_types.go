/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package v1alpha1

import (
	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type ConnectionStatus string

const (
	NotFound        ConnectionStatus = "NotFound"
	Synced          ConnectionStatus = "Synced"
	UpCloudAPIError ConnectionStatus = "UpCloudAPIError"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// VirtualMachineSpec defines the desired state of VirtualMachine
type VirtualMachineSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// VirtualMachineStatus defines the observed state of VirtualMachine
type VirtualMachineStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Connection          upcloud.ServerDetails `json:"connection,omitempty"`
	ConnectionStatus    ConnectionStatus      `json:"connection_status,omitempty"`
	ConnectionLastError string                `json:"connection_last_error,omitempty"`
	ConnectionSyncedAt  metav1.Time           `json:"connection_synced_at,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Connection Status",type=string,JSONPath=`.status.connection_status`
// +kubebuilder:printcolumn:name="Hostname",type=string,JSONPath=`.status.connection.hostname`
// +kubebuilder:printcolumn:name="Zone",type=string,JSONPath=`.status.connection.zone`
// +kubebuilder:printcolumn:name="Power state",type=string,JSONPath=`.status.connection.state`
// +kubebuilder:printcolumn:name="Synced at",type="string",JSONPath=".status.connection_synced_at"
// +kubebuilder:printcolumn:name="Sync age",type="date",JSONPath=".status.connection_synced_at"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

type VirtualMachine struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   VirtualMachineSpec   `json:"spec,omitempty"`
	Status VirtualMachineStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

type VirtualMachineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []VirtualMachine `json:"items"`
}

func init() {
	SchemeBuilder.Register(&VirtualMachine{}, &VirtualMachineList{})
}
