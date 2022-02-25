package controllers

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"time"

	"sigs.k8s.io/controller-runtime/pkg/log"
)

type IP struct {
	Ip                 string `json:"query"`
	lastCheckTimestamp time.Time
}

func (ip *IP) GetIP(ctx context.Context, ipTimeoutS int64) (string, error) {
	l := log.FromContext(ctx)
	now := time.Now()

	if now.Unix()-ip.lastCheckTimestamp.Unix() > ipTimeoutS {
		l.Info("Checking if IP have changed!", "TimePassed", ipTimeoutS)

		ipClient := http.Client{
			Timeout: time.Second * 10, // Timeout after 2 seconds
		}

		req, err := http.NewRequest(http.MethodGet, "http://ip-api.com/json/", nil)
		if err != nil {
			return "", err
		}

		res, getErr := ipClient.Do(req)
		if getErr != nil {
			return "", getErr
		}
		if res.Body != nil {
			defer res.Body.Close()
		}

		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			return "", readErr
		}

		jsonErr := json.Unmarshal(body, ip)
		if jsonErr != nil {
			return "", jsonErr
		}

		ip.lastCheckTimestamp = now
	}
	return ip.Ip, nil
}
