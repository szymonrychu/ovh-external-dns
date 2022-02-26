package controllers

import (
	"context"

	"github.com/kelseyhightower/envconfig"

	log "github.com/sirupsen/logrus"
)

type Config struct {
	OVHApplicationKey      string `envconfig:"OVH_APPLICATION_KEY" default:""`
	OVHApplicationSecret   string `envconfig:"OVH_APPLICATION_SECRET" default:""`
	OVHConsumerKey         string `envconfig:"OVH_CONSUMER_KEY" default:""`
	OVHApplicationEndpoint string `envconfig:"OVH_ENDPOINT" default:"ovh-eu"`
	OVHDNSDomain           string `envconfig:"OVH_DNS_DOMAIN" default:""`
	OVHDNSTTL              int64  `envconfig:"OVH_DNS_TTL" default:"60"`
	Debug                  bool   `envconfig:"DEBUG" default:"false"`
}

func (conf *Config) Load(ctx context.Context) error {
	if conf.OVHDNSDomain == "" {
		confErr := envconfig.Process("main", conf)
		if confErr != nil {
			return confErr
		}
		log.Infof("Loaded config: {OVH_ENDPOINT:%s, OVH_DNS_DOMAIN:%s}", conf.OVHApplicationEndpoint, conf.OVHDNSDomain)
	}
	return nil
}
