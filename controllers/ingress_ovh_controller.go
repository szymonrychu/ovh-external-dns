/*
Copyright 2022.

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
	"strings"

	v1networking "k8s.io/api/networking/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// OvhDomainReconciler reconciles a OvhDomain object
type IngressOVHReconciller struct {
	client.Client
	Scheme *runtime.Scheme
}

var ipCache IP
var conf Config

//+kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch

func (r *IngressOVHReconciller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	ingressList := &v1networking.IngressList{}
	getErr := r.List(ctx, ingressList)
	if getErr != nil {
		l.Error(getErr, "Error getting ingress!")
		return ctrl.Result{}, nil
	}

	confErr := conf.Load(ctx)
	if confErr != nil {
		l.Error(confErr, "Error loading config!")
		return ctrl.Result{}, nil
	}

	ip, ipErr := ipCache.GetIP(ctx, conf.OVHDNSTTL)
	if ipErr != nil {
		l.Error(ipErr, "Error getting IP address!!")
		return ctrl.Result{}, nil
	}

	hosts := []string{}
	for _, ingress := range ingressList.Items {
		for _, rule := range ingress.Spec.Rules {
			hosts = append(hosts, rule.DeepCopy().Host)
		}
	}

	manager := OVHManager{}
	manager.Init(conf)

	getRecordsErr := manager.LoadRemoteRecords(ctx)
	if getRecordsErr != nil {
		l.Error(getRecordsErr, "Couldn't get OVH records!")
		return ctrl.Result{}, nil
	}

	aRecordFound := false
	for _, aRecord := range manager.RemoteARecords {
		if aRecord.SubDomain == "" {
			aRecordFound = true
			if aRecord.Target != ip {
				l.Info("Updating 'A' record with new ip", "A", conf.OVHDNSDomain, "target", ip)
				aRecord.FieldType = "A"
				aRecord.SubDomain = ""
				aRecord.Target = ip
				aRecord.Ttl = conf.OVHDNSTTL
				if updErr := aRecord.UpdateRecord(manager); updErr != nil {
					l.Error(updErr, "Couldn't update OVH record!")
					return ctrl.Result{}, nil
				}
			}
		}
	}
	if !aRecordFound {
		l.Info("Adding 'A' record with new ip", "A", conf.OVHDNSDomain, "target", ip)
		mainARecord := OVHRecord{}
		mainARecord.FieldType = "A"
		mainARecord.SubDomain = ""
		mainARecord.Target = ip
		mainARecord.Ttl = conf.OVHDNSTTL
		if updErr := mainARecord.AddRecord(manager); updErr != nil {
			l.Error(updErr, "Couldn't add OVH record!")
			return ctrl.Result{}, nil
		}
	}

	for _, host := range hosts {
		subdomain := strings.ReplaceAll(host, "."+conf.OVHDNSDomain, "")
		record, err := manager.GetRecordBySubDomain(subdomain)
		if err != nil {
			l.Info("Adding missing 'CNAME' record with new target", "CNAME", subdomain, "target", conf.OVHDNSDomain+".")
			record.InitWithConfig(subdomain, conf)
			if addErr := record.AddRecord(manager); addErr != nil {
				l.Error(addErr, "Couldn't add OVH record!", "CNAME", subdomain, "target", conf.OVHDNSDomain+".")
				return ctrl.Result{}, nil
			}
		} else if record.Target != conf.OVHDNSDomain+"." {
			l.Info("Updaing 'CNAME' record with new target", "CNAME", subdomain, "target", conf.OVHDNSDomain+".")
			record.InitWithConfig(subdomain, conf)
			if updErr := record.UpdateRecord(manager); updErr != nil {
				l.Error(updErr, "Couldn't update OVH record!", "CNAME", subdomain, "target", conf.OVHDNSDomain+".")
				return ctrl.Result{}, nil
			}
		}
	}
	for _, record := range manager.RemoteCNAMERecords {
		domain := record.SubDomain + "." + conf.OVHDNSDomain
		domainStillNecessary := false
		for _, host := range hosts {
			if host == domain {
				domainStillNecessary = true
				break
			}
		}
		if !domainStillNecessary {
			l.Info("Deleting 'CNAME' record", "CNAME", record.SubDomain+"."+conf.OVHDNSDomain)
			if delErr := record.DeleteRecord(manager); delErr != nil {
				l.Error(delErr, "Couldn't delete OVH record!")
				return ctrl.Result{}, nil
			}
		}
	}
	return ctrl.Result{Requeue: true}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *IngressOVHReconciller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1networking.Ingress{}).
		Complete(r)
}
