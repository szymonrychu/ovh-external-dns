package main

import (
	"fmt"
	"strconv"
	"github.com/ovh/go-ovh/ovh"
	"github.com/kelseyhightower/envconfig"
	log "github.com/sirupsen/logrus"
)

type Specification struct {
	OVHApplicationKey      string `envconfig:"OVH_APPLICATION_KEY" default:""`
	OVHApplicationSecret   string `envconfig:"OVH_APPLICATION_SECRET" default:""`
	OVHConsumerKey         string `envconfig:"OVH_CONSUMER_KEY" default:""`
	OVHApplicationEndpoint string `envconfig:"OVH_ENDPOINT" default:"ovh-eu"`
	OVHDNSDomain           string `envconfig:"OVH_DNS_DOMAIN" default:""`
	OVHDNSTTL              string `envconfig:"OVH_DNS_TTL" default:"60"`
	SleepTime              int `envconfig:"SLEEP_TIME" default:"60"`
}

// Instantiate an OVH client and get the firstname of the currently logged-in user.
// Visit https://api.ovh.com/createToken/index.cgi?GET=/me to get your credentials.
func main() {
	log.SetLevel(log.DebugLevel)
	log.SetFormatter(&log.JSONFormatter{})
	var s Specification
	err := envconfig.Process("main", &s)
	if err != nil {
		log.Fatal(err.Error())
	}else{
		log.Debugf("OVH_APPLICATION_KEY=%s", s.OVHApplicationKey)
		log.Debugf("OVH_APPLICATION_SECRET=%s", s.OVHApplicationSecret)
		log.Debugf("OVH_CONSUMER_KEY=%s", s.OVHApplicationSecret)
		log.Debugf("OVH_ENDPOINT=%s", s.OVHApplicationEndpoint)
		log.Debugf("OVH_DOMAIN=%s", s.OVHDNSDomain)
		log.Debugf("OVH_DNS_TTL=%s", s.OVHDNSTTL)
		log.Debugf("SLEEP_TIME=%v", strconv.Itoa(s.SleepTime))
	}

	client, _ := ovh.NewClient(
		s.OVHApplicationEndpoint,
		s.OVHApplicationKey,
		s.OVHApplicationSecret,
		s.OVHConsumerKey,
	)
	recordIds := []int{}
	recordUrl := fmt.Sprintf("/domain/zone/%s/record", s.OVHDNSDomain)
	log.Debugf("recordMainPath='%s'", recordUrl)
	if err1 := client.Get(recordUrl, &recordIds); err1 != nil {
		log.Fatalf("Error: %q\n", err1)
	}
	for _, recordId := range recordIds {
		type OVHRecord struct {
			Ttl       int    `json:"ttl"`
			Id        int    `json:"id"`
			SubDomain string `json:"subDomain"`
			Zone      string `json:"zone"`
			Target    string `json:"target"`
			FieldType string `json:"fieldType"`
		}
		ovhRecord := OVHRecord{}
		recordIdUrl := fmt.Sprintf("%s/%s", recordUrl, strconv.Itoa(recordId))
		if err2 := client.Get(recordIdUrl, &ovhRecord); err2 != nil {
			log.Fatalf("Error: %q\n", err2)
		}
		log.Debugf("ovhRecord.FieldType='%s'", ovhRecord.FieldType)
		log.Debugf("ovhRecord.SubDomain='%s'", ovhRecord.SubDomain)
	}
}