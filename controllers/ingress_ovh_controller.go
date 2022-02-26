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

	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
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
	ingressList := &v1networking.IngressList{}
	getErr := r.List(ctx, ingressList)
	if getErr != nil {
		log.Errorf("Error getting ingress! %s", getErr)
		return ctrl.Result{}, nil
	}

	confErr := conf.Load(ctx)
	if confErr != nil {
		log.Errorf("Error loading config! %s", confErr)
		return ctrl.Result{}, nil
	}

	ip, ipErr := ipCache.GetIP(conf.OVHDNSTTL)
	if ipErr != nil {
		log.Errorf("Error getting IP address! %s", ipErr)
		return ctrl.Result{}, nil
	}

	hosts := []string{}
	for _, ingress := range ingressList.Items {
		for _, rule := range ingress.Spec.Rules {
			host := rule.DeepCopy().Host
			hosts = append(hosts, host)
		}
	}

	manager := OVHManager{}
	manager.Init(conf)

	getRecordsErr := manager.LoadRemoteRecords(ctx)
	if getRecordsErr != nil {
		log.Errorf("Couldn't get OVH records! %s", getRecordsErr)
		return ctrl.Result{}, nil
	}

	aRecordFound := false
	for _, aRecord := range manager.RemoteARecords {
		if aRecord.SubDomain == "" {
			aRecordFound = true
			if aRecord.Target != ip {
				log.Infof("Updating A record '%s' with new ip '%s'", conf.OVHDNSDomain, ip)
				aRecord.FieldType = "A"
				aRecord.SubDomain = ""
				aRecord.Target = ip
				aRecord.Ttl = conf.OVHDNSTTL
				if updErr := aRecord.UpdateRecord(manager); updErr != nil {
					log.Errorf("Couldn't update OVH record! %s", updErr)
					return ctrl.Result{}, nil
				}
			}
		}
	}
	if !aRecordFound {
		log.Infof("Adding A record '%s' with new ip '%s'", conf.OVHDNSDomain, ip)
		mainARecord := OVHRecord{}
		mainARecord.FieldType = "A"
		mainARecord.SubDomain = ""
		mainARecord.Target = ip
		mainARecord.Ttl = conf.OVHDNSTTL
		if updErr := mainARecord.AddRecord(manager); updErr != nil {
			log.Errorf("Couldn't add OVH record! %s", updErr)
			return ctrl.Result{}, nil
		}
	}

	for _, host := range hosts {
		subdomain := strings.ReplaceAll(host, "."+conf.OVHDNSDomain, "")
		if subdomain == conf.OVHDNSDomain {
			log.Infof("Ignoring host/subdomain/target: %s/%s/%s", host, subdomain, conf.OVHDNSDomain+".")
			continue
		}
		record, err := manager.GetRecordBySubDomain(subdomain)
		if err != nil {
			log.Infof("Adding missing CNAME subdomain record '%s' with new target '%s'", subdomain, conf.OVHDNSDomain+".")
			record.InitWithConfig(subdomain, conf)
			if addErr := record.AddRecord(manager); addErr != nil {
				log.Errorf("Couldn't add OVH record! %s", addErr)
				return ctrl.Result{}, nil
			}
		} else if record.Target != conf.OVHDNSDomain+"." {
			log.Infof("Updaing missing CNAME subdomain record '%s' with new target '%s'", subdomain, conf.OVHDNSDomain+".")
			record.InitWithConfig(subdomain, conf)
			if updErr := record.UpdateRecord(manager); updErr != nil {
				log.Errorf("Couldn't update OVH record! %s", updErr)
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
			log.Infof("Deleting CNAME subdomain record '%s' with new target '%s'", record.SubDomain, conf.OVHDNSDomain+".")
			if delErr := record.DeleteRecord(manager); delErr != nil {
				log.Errorf("Couldn't delete OVH record! %s", delErr)
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
