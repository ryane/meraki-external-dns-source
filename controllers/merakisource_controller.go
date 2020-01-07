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

package controllers

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"github.com/kubernetes-incubator/external-dns/endpoint"
	"github.com/ryane/meraki-external-dns-source/pkg/meraki"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	dnsv1alpha1 "github.com/ryane/meraki-external-dns-source/api/v1alpha1"
)

// MerakiSourceReconciler reconciles a MerakiSource object
type MerakiSourceReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=dns.jossware.com,resources=merakisources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.jossware.com,resources=merakisources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=externaldns.k8s.io,resources=dnsendpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=externaldns.k8s.io,resources=dnsendpoints/status,verbs=get

func (r *MerakiSourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("merakisource", req.NamespacedName)

	var source dnsv1alpha1.MerakiSource
	if err := r.Get(ctx, req.NamespacedName, &source); err != nil {
		if apierrs.IsNotFound(err) {
			// 404, wait for next notification
			log.V(1).Info("not found")
			return ctrl.Result{}, nil
		}
		log.Error(err, "unable to fetch MerakiSource")
		return ctrl.Result{}, err
	}

	var dnsEndpoint endpoint.DNSEndpoint
	// dns endpoint will have the same name as the MerakiSource
	if err := r.Get(ctx, req.NamespacedName, &dnsEndpoint); err != nil {
		if apierrs.IsNotFound(err) {
			log.V(1).Info("dns endpoint not found")
			// create it
			e := &endpoint.DNSEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      source.Name,
					Namespace: source.Namespace,
				},
			}
			if err := ctrl.SetControllerReference(&source, e, r.Scheme); err != nil {
				return ctrl.Result{}, err
			}
			if err := r.Create(ctx, e); err != nil {
				log.Error(err, "unable to create dns endpoint", "dns-endpoint", e)
				return ctrl.Result{}, err
			}
			log.V(1).Info("created dns endpoint")
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "unable to get dns endpoint", "dns-endpoint", req.NamespacedName)
		return ctrl.Result{}, err
	}

	// update the spec from MerakiData
	endpoints, err := r.GetEndpoints(&source)
	if err != nil {
		log.Error(err, "failed to get endpoints")
		return ctrl.Result{}, err
	}

	dnsEndpoint.Spec.Endpoints = endpoints
	if err := r.Update(ctx, &dnsEndpoint); err != nil {
		log.Error(err, "failed to update dns endpoint", "dns-endpoint", dnsEndpoint)
		return ctrl.Result{}, err
	}

	ref, err := ref.GetReference(r.Scheme, &dnsEndpoint)
	if err != nil {
		log.Error(err, "unable to make reference to dns endpoint", "dns-endpoint", dnsEndpoint)
		return ctrl.Result{}, err
	}
	source.Status.Endpoint = *ref
	if err := r.Status().Update(ctx, &source); err != nil {
		if apierrs.IsConflict(err) {
			log.V(1).Info("stale MerakiSource, requeue")
			return ctrl.Result{Requeue: true}, nil
		}
		log.Error(err, "unable to update MerakiSource status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 1 * time.Minute}, nil
}

func (r *MerakiSourceReconciler) GetEndpoints(source *dnsv1alpha1.MerakiSource) ([]*endpoint.Endpoint, error) {
	// TODO: configurable api key
	merakiClient := meraki.New("")

	if source.Spec.Organization.ID == "" {
		// look up organization
	}

	if source.Spec.Network.ID == "" {
		// look up network
	}

	clients, err := merakiClient.Clients(source.Spec.Network.ID)
	if err != nil {
		return nil, err
	}

	var endpoints []*endpoint.Endpoint
	for _, client := range clients {
		e := endpoint.NewEndpoint(client.DNSName()+"."+source.Spec.Domain, "A", client.IP)
		if source.Spec.TTL != nil {
			e.RecordTTL = endpoint.TTL(*source.Spec.TTL)
		}
		r.Log.V(1).Info("created endpoint", "endpoint", e)
		endpoints = append(endpoints, e)
	}

	return endpoints, nil
}

func (r *MerakiSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dnsv1alpha1.MerakiSource{}).
		Owns(&endpoint.DNSEndpoint{}).
		Complete(r)
}
