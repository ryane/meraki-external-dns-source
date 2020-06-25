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
	"errors"
	"fmt"
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
	Log                 logr.Logger
	Scheme              *runtime.Scheme
	APIKey              string
	APIThrottleInterval time.Duration
	RequeueInterval     time.Duration
}

// +kubebuilder:rbac:groups=dns.jossware.com,resources=merakisources,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=dns.jossware.com,resources=merakisources/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=externaldns.k8s.io,resources=dnsendpoints,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=externaldns.k8s.io,resources=dnsendpoints/status,verbs=get

func (r *MerakiSourceReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("merakisource", req.NamespacedName)

	// get meraki source resource
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
			dnsEndpoint = endpoint.DNSEndpoint{
				ObjectMeta: metav1.ObjectMeta{
					Name:      source.Name,
					Namespace: source.Namespace,
				},
			}
		} else {
			log.Error(err, "unable to get dns endpoint", "dns-endpoint", req.NamespacedName)
			return ctrl.Result{}, err
		}
	}

	if err := ctrl.SetControllerReference(&source, &dnsEndpoint, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	// update the spec from MerakiData
	// don't query meraki if we already did in the last 1 minute
	if source.Status.SyncedAt == nil || time.Since(source.Status.SyncedAt.Time) > r.APIThrottleInterval {
		endpoints, err := r.GetEndpoints(&source)
		if err != nil {
			log.Error(err, "failed to get endpoints")
			return ctrl.Result{}, err
		}

		dnsEndpoint.Spec.Endpoints = endpoints

		if r.isNew(dnsEndpoint) {
			if err := r.Create(ctx, &dnsEndpoint); err != nil {
				log.Error(err, "failed to create dns endpoint", "dns-endpoint", dnsEndpoint)
				return ctrl.Result{}, err
			}
			log.V(1).Info("created dns endpoint", "dns-endpoint", dnsEndpoint.GetName())
		} else {
			if err := r.Update(ctx, &dnsEndpoint); err != nil {
				log.Error(err, "failed to update dns endpoint", "dns-endpoint")
				return ctrl.Result{}, err
			}
			log.V(1).Info("updated dns endpoint", "dns-endpoint", dnsEndpoint.GetName())
		}

		ts := metav1.Now()
		source.Status.SyncedAt = &ts
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

	return ctrl.Result{RequeueAfter: r.RequeueInterval}, nil
}

func (r *MerakiSourceReconciler) GetEndpoints(source *dnsv1alpha1.MerakiSource) ([]*endpoint.Endpoint, error) {
	merakiClient := meraki.New(r.APIKey)

	networkID := source.Spec.Network.ID
	if networkID == "" {
		networkName := source.Spec.Network.Name
		if networkName == "" {
			return nil, errors.New("network name or ID is required")
		}

		// make sure we have the organization ID
		orgID := source.Spec.Organization.ID
		if orgID == "" {
			orgName := source.Spec.Organization.Name
			if orgName == "" {
				return nil, errors.New("organization name or ID is required")
			}

			// look up organization
			org, err := merakiClient.FindOrganization(orgName)
			if err != nil {
				return nil, err
			}

			if org == nil {
				return nil, fmt.Errorf("%s organization not found. check your name or API key", orgName)
			}

			orgID = org.ID
		}

		// lookup network
		network, err := merakiClient.FindNetwork(orgID, networkName)
		if err != nil {
			return nil, err
		}

		if network == nil {
			return nil, fmt.Errorf("%s network not found. check your name, organization, or API key", networkName)
		}

		networkID = network.ID
	}

	clients, err := merakiClient.Clients(networkID)
	if err != nil {
		return nil, err
	}

	var endpoints []*endpoint.Endpoint
	for _, client := range clients {
		e := endpoint.NewEndpoint(client.DNSName()+"."+source.Spec.Domain, "A", client.IP)
		if source.Spec.TTL != nil {
			e.RecordTTL = endpoint.TTL(*source.Spec.TTL)
		}
		r.Log.V(1).Info("found endpoint", "endpoint", e)
		endpoints = append(endpoints, e)
	}

	return endpoints, nil
}

func (r *MerakiSourceReconciler) isNew(e endpoint.DNSEndpoint) bool {
	return e.GetCreationTimestamp().Time.IsZero()
}

func (r *MerakiSourceReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dnsv1alpha1.MerakiSource{}).
		Owns(&endpoint.DNSEndpoint{}).
		Complete(r)
}
