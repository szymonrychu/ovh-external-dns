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

// PartialMe holds the first name of the currently logged-in user.
// Visit https://api.ovh.com/console/#/me#GET for the full definition
type PartialMe struct {
	Id string `json:"id"`
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

	var me PartialMe

	client, _ := ovh.NewClient(
		s.OVHApplicationEndpoint,
		s.OVHApplicationKey,
		s.OVHApplicationSecret,
		s.OVHConsumerKey,
	)
	client.Get(fmt.Sprintf("/domain/zone/%s/record", s.OVHDNSDomain), &me)
	fmt.Printf("Welcome %s!\n", me.Firstname)
}