/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package controller

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"sync"
	"time"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/metrics"

	corev1alpha1 "github.com/paaloeye/uks-controller/api/v1alpha1"

	"github.com/UpCloudLtd/upcloud-go-api/v8/upcloud"
	upcloudclient "github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/client"
	upcloudrequest "github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/request"
	upcloudservice "github.com/UpCloudLtd/upcloud-go-api/v8/upcloud/service"

	"github.com/bytedance/sonic"

	o11y "github.com/paaloeye/uks-controller/internal/observability"
)

type ContextKey string

const (
	ctxKeyCRD                 ContextKey = `crd`                   //
	ctxKeyConnection          ContextKey = `connection`            // type: upcloud.ServerDetails
	ctxKeyConnectionStatus    ContextKey = `connection-status`     // type: ConnectionStatus
	ctxKeyConnectionLastError ContextKey = `connection-last-error` // type: str, optional
	ctxKeyVMUUID              ContextKey = `vm-uuid`               // type: str, required
	ctxKeyExponentialBackOff  ContextKey = `exponential-back-off`  // type: bool, default: false
)

// VirtualMachineReconciler reconciles a VirtualMachine object
type VirtualMachineReconciler struct {
	client.Client
	Scheme *runtime.Scheme

	UpCloudClient *upcloudclient.Client
	UpCloudSVC    *upcloudservice.Service

	ConfigSyncInterval time.Duration

	CurrentlySyncingVirtualMachines sync.Map
}

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.MustRegister(o11y.GaugeVMSyncing)
}

//+kubebuilder:rbac:groups=core.infra.upcloud.com,resources=virtualmachines,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.infra.upcloud.com,resources=virtualmachines/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.infra.upcloud.com,resources=virtualmachines/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.16.3/pkg/reconcile
func (r *VirtualMachineReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	vmUUID := req.Name
	crd := corev1alpha1.VirtualMachine{}

	ctx = context.WithValue(ctx, ctxKeyVMUUID, vmUUID)

	// Fetch CRD
	if err := r.Client.Get(ctx, req.NamespacedName, &crd); err != nil {
		if kerrors.IsNotFound(err) {
			// CRD is not found, hence, stopping reconciliation loop for that VM

			return r.customResourceLifeCycleWithdraw(ctx)
		}

		// Kubernetes API error, try again soon
		return ctrl.Result{}, err
	}
	ctx = context.WithValue(ctx, ctxKeyCRD, &crd)

	// Fetch VM state from UpCloud API
	connectedVM, err := r.UpCloudSVC.GetServerDetails(ctx, &upcloudrequest.GetServerDetailsRequest{
		UUID: vmUUID,
	})

	// Handle when VM is not found using UUID provided
	if r.isNotFound(err) {
		logger.V(3).Info("VM doesn't exist", "uuid", vmUUID)

		// Update connection status and make sure `connection` is zeroed (could have been set previously)
		ctx = context.WithValue(ctx, ctxKeyConnection, &upcloud.ServerDetails{})
		ctx = context.WithValue(ctx, ctxKeyConnectionStatus, corev1alpha1.NotFound)

		return r.customResourceLifeCycleSync(ctx)
	}

	if err != nil {
		logger.Error(err, "UpCloud API Error occurred")

		ctx = context.WithValue(ctx, ctxKeyConnectionStatus, corev1alpha1.UpCloudAPIError)
		ctx = context.WithValue(ctx, ctxKeyConnectionLastError, err.Error())
		ctx = context.WithValue(ctx, ctxKeyExponentialBackOff, true)

		return r.customResourceLifeCycleSync(ctx)
	}

	logger.V(3).Info(
		"VM Info (base64 encoded)",
		"object",
		base64.StdEncoding.EncodeToString([]byte(r.encodeJSON(connectedVM))),
	)

	// Update CRD and send it to API Server
	ctx = context.WithValue(ctx, ctxKeyConnection, connectedVM)
	ctx = context.WithValue(ctx, ctxKeyConnectionStatus, corev1alpha1.Synced)

	return r.customResourceLifeCycleSync(ctx)
}

// SetupWithManager sets up the controller with the Manager.
func (r *VirtualMachineReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.VirtualMachine{}).
		Complete(r)
}

// Private API

// UpCloud API Error Handling routines
// Return true if the server with the given UUID does not exist in the UpCloud API.
// This function checks whether the provided error object represents a problem
// related to the UpCloud API. If it does, and its error code 404, then it returns `true`.
func (r *VirtualMachineReconciler) isNotFound(err error) bool {
	if err == nil {
		return false
	}

	var problem *upcloud.Problem

	if errors.As(err, &problem) {
		if problem.ErrorCode() == upcloud.ErrCodeServerNotFound {
			return true
		}
	}

	return false
}

// JSON encode using `sonic`
func (r *VirtualMachineReconciler) encodeJSON(object any) string {
	buf := bytes.NewBuffer(nil)
	enc := sonic.ConfigDefault.NewEncoder(buf)
	enc.Encode(object)
	return buf.String()
}

// Update CRD
func (r *VirtualMachineReconciler) updateCRD(ctx context.Context) error {
	crd := ctx.Value(ctxKeyCRD).(*corev1alpha1.VirtualMachine)

	newStatus := corev1alpha1.VirtualMachineStatus{}

	// Interrogate context
	newConnectionStatus := ctx.Value(ctxKeyConnectionStatus).(corev1alpha1.ConnectionStatus)
	if newConnection := ctx.Value(ctxKeyConnection); newConnection != nil {
		newConnection := newConnection.(*upcloud.ServerDetails)
		newStatus.Connection = *newConnection
	}
	if lastError := ctx.Value(ctxKeyConnectionLastError); lastError != nil {
		newStatus.ConnectionLastError = lastError.(string)
	}

	newStatus.ConnectionStatus = newConnectionStatus
	newStatus.ConnectionSyncedAt = metav1.Now()

	crd.Status = newStatus

	return r.Status().Update(ctx, crd)
}

//
// CRD Life Cycle Methods Overview
//
// 	Claim  -> 	Sync  ... 	Sync  -> 	Withdraw
// 0 ----------------------------------------------------------------> t (axis)
//   	t_{1} 	 	t_{2}  				t_{n}
//	 |		 |		 		 |
//	 |		 |		 		 |---------------| Retract interest wrt the VM and decrement the gauge
//	 | 		 |
//	 | 		 |-----------------------------------------------| Drive CRD state towards current VM state in UpCloud API
//	 |
//   	 |---------------------------------------------------------------| Initiate syncing and increment the gauge

func (r *VirtualMachineReconciler) customResourceLifeCycleClaim(ctx context.Context) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Start polling VM state from UpCloud API")

	o11y.GaugeVMSyncing.Inc()

	return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
}

func (r *VirtualMachineReconciler) customResourceLifeCycleSync(ctx context.Context) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	_, loaded := r.CurrentlySyncingVirtualMachines.LoadOrStore(ctx.Value(ctxKeyVMUUID), true)
	if !loaded {
		// Initial sync for the resource
		r.customResourceLifeCycleClaim(ctx)
	}

	if err := r.updateCRD(ctx); err != nil {
		log.Info("Failed to update CRD")

		return ctrl.Result{}, err
	}

	log.V(3).Info("Synced")

	if backoff := ctx.Value(ctxKeyExponentialBackOff); backoff != nil && backoff.(bool) {
		return ctrl.Result{Requeue: true}, nil
	}

	return ctrl.Result{RequeueAfter: r.ConfigSyncInterval}, nil
}

func (r *VirtualMachineReconciler) customResourceLifeCycleWithdraw(ctx context.Context) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Stop polling VM state from UpCloud API")
	log.Info("Reconciled")

	if deleted := r.CurrentlySyncingVirtualMachines.CompareAndDelete(ctx.Value(ctxKeyVMUUID), true); deleted {
		o11y.GaugeVMSyncing.Dec()
	}

	return ctrl.Result{}, nil
}
