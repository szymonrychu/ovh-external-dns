package main

import (
	"fmt"
	"strconv"
	"time"
	"io/ioutil"
	"encoding/json"
	"context"
    "strings"
	"net/http"
	// "flag"
	"path/filepath"
	"k8s.io/client-go/util/homedir"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kelseyhightower/envconfig"
	"github.com/ovh/go-ovh/ovh"
	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/client-go/kubernetes"
	// rest "k8s.io/client-go/rest"
)

type Specification struct {
	OVHApplicationKey      string `envconfig:"OVH_APPLICATION_KEY" default:""`
	OVHApplicationSecret   string `envconfig:"OVH_APPLICATION_SECRET" default:""`
	OVHConsumerKey         string `envconfig:"OVH_CONSUMER_KEY" default:""`
	OVHApplicationEndpoint string `envconfig:"OVH_ENDPOINT" default:"ovh-eu"`
	OVHDNSDomain           string `envconfig:"OVH_DNS_DOMAIN" default:""`
	OVHDNSTTL              string `envconfig:"OVH_DNS_TTL" default:"60"`
	SleepTime              int    `envconfig:"SLEEP_TIME" default:"60"`
	Debug                  bool   `envconfig:"DEBUG" default:"false"`
}

type OVHRecord struct {
	Ttl       int    `json:"ttl"`
	Id        int    `json:"id"`
	SubDomain string `json:"subDomain"`
	Target    string `json:"target"`
	FieldType string `json:"fieldType"`
}
type K8sIngress struct {
	SubDomain string
}

func getip2() string {
	type IP struct {
		Query string
	}
    req, err := http.Get("http://ip-api.com/json/")
    if err != nil {
        return err.Error()
    }
    defer req.Body.Close()

    body, err := ioutil.ReadAll(req.Body)
    if err != nil {
        return err.Error()
    }

    var ip IP
    json.Unmarshal(body, &ip)

    return ip.Query
}

// Instantiate an OVH ovhClient and get the firstname of the currently logged-in user.
// Visit https://api.ovh.com/createToken/index.cgi?GET=/me to get your credentials.
func main() {
	log.SetFormatter(&log.JSONFormatter{})
	var s Specification
	err := envconfig.Process("main", &s)
	if(s.Debug){
		log.SetLevel(log.DebugLevel)
	}else{
		log.SetLevel(log.InfoLevel)
	}
	if err != nil {
		log.Fatal(err.Error())
	} else {
		log.Debugf("OVH_APPLICATION_KEY=%s", s.OVHApplicationKey)
		log.Debugf("OVH_APPLICATION_SECRET=%s", s.OVHApplicationSecret)
		log.Debugf("OVH_CONSUMER_KEY=%s", s.OVHApplicationSecret)
		log.Debugf("OVH_ENDPOINT=%s", s.OVHApplicationEndpoint)
		log.Debugf("OVH_DOMAIN=%s", s.OVHDNSDomain)
		log.Debugf("OVH_DNS_TTL=%s", s.OVHDNSTTL)
		log.Debugf("SLEEP_TIME=%v", strconv.Itoa(s.SleepTime))
	}

	ovhClient, _ := ovh.NewClient(
		s.OVHApplicationEndpoint,
		s.OVHApplicationKey,
		s.OVHApplicationSecret,
		s.OVHConsumerKey,
	)
	var kubeconfig string
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = filepath.Join(home, ".kube", "config")
	}

	// use the current context in kubeconfig
	config, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		log.Fatal(err.Error())
	}

	// create the clientset
	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		log.Fatal(err.Error())
	}
	for {
		ovhARecord := OVHRecord{}
		ovhCNAMERecords := []OVHRecord{}
		k8sIngresses := []K8sIngress{}

		ingressList, err := k8sClient.ExtensionsV1beta1().Ingresses("").List(context.TODO(), metav1.ListOptions{})
		if err != nil {
			log.Fatal(err.Error())
		}
		ingressCtrls := ingressList.Items
		if len(ingressCtrls) > 0 {
			for _, ingress := range ingressCtrls {
				log.Debugf("ingress %s exists in namespace %s\n", ingress.Name, ingress.Namespace)
				for _, rule := range ingress.Spec.Rules {
					if strings.Contains(rule.Host, "." + s.OVHDNSDomain){
						subPath := strings.ReplaceAll(rule.Host, "." + s.OVHDNSDomain, "")
						log.Debugf("subPath: %s", subPath)
						tmp := K8sIngress{subPath}
						k8sIngresses = append(k8sIngresses, tmp)
					}
				}
			}
		} else {
			log.Debug("no ingress found")
		}
		recordIds := []int{}
		recordUrl := fmt.Sprintf("/domain/zone/%s/record", s.OVHDNSDomain)
		log.Debugf("recordMainPath='%s'", recordUrl)
		if err1 := ovhClient.Get(recordUrl, &recordIds); err1 != nil {
			log.Fatalf("Error: %q\n", err1)
		}
		for _, recordId := range recordIds {
			ovhRecord := OVHRecord{}
			recordIdUrl := fmt.Sprintf("%s/%s", recordUrl, strconv.Itoa(recordId))
			if err2 := ovhClient.Get(recordIdUrl, &ovhRecord); err2 != nil {
				log.Fatalf("Error: %q\n", err2)
			}
			if ovhRecord.FieldType == "A" {
				ovhARecord = ovhRecord
			} else if ovhRecord.FieldType == "CNAME" {
				ovhCNAMERecords = append(ovhCNAMERecords, ovhRecord)
			} else {
				log.Debugf("Ingoring: %s %s.%s", ovhRecord.FieldType, ovhRecord.SubDomain, s.OVHDNSDomain)
			}
			log.Debugf("ovhRecord.FieldType='%s'", ovhRecord.FieldType)
			log.Debugf("ovhRecord.SubDomain='%s'", ovhRecord.SubDomain)
		}
		
		currentIp := getip2()

		log.Debugf("Current IP: %s", currentIp)
		if ovhARecord.Id != 0 {
			if ovhARecord.Target != currentIp {
				log.Infof("Found OVH A record %s, but IP doesnt match %s, updating!", ovhARecord.Target, currentIp)
				ovhARecord.Target = currentIp
				recordIdUrl := fmt.Sprintf("%s/%s", recordUrl, strconv.Itoa(ovhARecord.Id))
				if err := ovhClient.Put(recordIdUrl, ovhARecord, nil); err != nil {
					log.Fatal(err.Error())
				}
			} else {
				log.Infof("Found OVH A record for current IP %s", ovhARecord.Target)
			}
		} else {
			log.Infof("Missing OVH A record %s, updating!", ovhARecord.Target)
			type PostOVHRecord struct {
				Ttl       int    `json:"ttl"`
				SubDomain string `json:"subDomain"`
				Target    string `json:"target"`
				FieldType string `json:"fieldType"`
			}
			tmp := PostOVHRecord{}
			i, err := strconv.Atoi(s.OVHDNSTTL)
			if err != nil {
				log.Fatal(err.Error())
			}
			tmp.Ttl = i
			tmp.SubDomain = ""
			tmp.Target = currentIp
			tmp.FieldType = "A"
			if err := ovhClient.Post(recordUrl, tmp, nil); err != nil {
				log.Fatal(err.Error())
			}
		}


		// handle missing ingresses
		for _, k8sIng := range k8sIngresses {
			found := false
			for _, ovhCname := range ovhCNAMERecords {
				if ! found && ovhCname.SubDomain == k8sIng.SubDomain {
					found = true
					if ovhCname.Target != s.OVHDNSDomain + "." {
						log.Infof("Found OVH record for ing %s, but targets doesn't match %s, updating!", k8sIng.SubDomain, s.OVHDNSDomain + ".")	
						recordIdUrl := fmt.Sprintf("%s/%s", recordUrl, strconv.Itoa(ovhCname.Id))
						ovhCname.Target = s.OVHDNSDomain + "."
						if err := ovhClient.Put(recordIdUrl, ovhCname, nil); err != nil {
							log.Fatal(err.Error())
						}
					}else{
						log.Debugf("Found OVH record for ing %s", k8sIng.SubDomain)	
					}
				}
			}
			if ! found {
				log.Infof("Not found OVH record for ing %s, updating!", k8sIng.SubDomain)
				type PostOVHRecord struct {
					Ttl       int    `json:"ttl"`
					SubDomain string `json:"subDomain"`
					Target    string `json:"target"`
					FieldType string `json:"fieldType"`
				}
				tmp := PostOVHRecord{}
				i, err := strconv.Atoi(s.OVHDNSTTL)
				if err != nil {
					log.Fatal(err.Error())
				}
				tmp.Ttl = i
				tmp.SubDomain = k8sIng.SubDomain
				tmp.Target = s.OVHDNSDomain
				tmp.FieldType = "CNAME"
				if err := ovhClient.Post(recordUrl, tmp, nil); err != nil {
					log.Fatal(err.Error())
				}
			}	
		}
		log.Info("Done adding missing OVH Entries")
		
		for _, ovhCname := range ovhCNAMERecords {
			found := false
			for _, k8sIng := range k8sIngresses {
				if ! found && ovhCname.SubDomain == k8sIng.SubDomain {
					found = true
				}
			}
			if ! found {
				log.Infof("OVH Record for %s domain is missing ingress, updating!", ovhCname.SubDomain)
				recordIdUrl := fmt.Sprintf("%s/%s", recordUrl, strconv.Itoa(ovhCname.Id))
				if err := ovhClient.Delete(recordIdUrl, nil); err != nil {
					log.Fatal(err.Error())
				}	
			}
		}
		log.Info("Done deleting missing OVH Entries")



		log.Infof("Sleeping for %ss", strconv.Itoa(s.SleepTime))
		time.Sleep(time.Duration(s.SleepTime) * time.Second)
	}

}
