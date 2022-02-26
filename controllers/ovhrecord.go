package controllers

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/ovh/go-ovh/ovh"
)

type OVHRecord struct {
	Ttl       int64  `json:"ttl"`
	Id        int64  `json:"id"`
	SubDomain string `json:"subDomain"`
	Target    string `json:"target"`
	FieldType string `json:"fieldType"`
	Zone      string `json:"zone"`
}

func (record *OVHRecord) AddRecord(manager OVHManager) error {
	client := manager.GetOVHClient()
	recordIdUrl := fmt.Sprintf("/domain/zone/%s/record/%d", manager.GetZone(), record.Id)
	if err := (*client).Post(recordIdUrl, record, nil); err != nil {
		return err
	}
	return nil
}

func (record *OVHRecord) UpdateRecord(manager OVHManager) error {
	client := manager.GetOVHClient()
	recordIdUrl := fmt.Sprintf("/domain/zone/%s/record/%d", manager.GetZone(), record.Id)
	if err := (*client).Put(recordIdUrl, record, nil); err != nil {
		return err
	}
	return nil
}

func (record *OVHRecord) DeleteRecord(manager OVHManager) error {
	client := manager.GetOVHClient()
	recordIdUrl := fmt.Sprintf("/domain/zone/%s/record/%d", manager.GetZone(), record.Id)
	if err := (*client).Delete(recordIdUrl, nil); err != nil {
		return err
	}
	return nil
}

func (record *OVHRecord) InitWithConfig(subdomain string, conf Config) {
	record.FieldType = "CNAME"
	record.SubDomain = subdomain
	record.Target = conf.OVHDNSDomain + "."
	record.Ttl = conf.OVHDNSTTL
}
func (r1 *OVHRecord) Compare(r2 OVHRecord) bool {
	return r1.FieldType == r2.FieldType && r1.SubDomain == r2.SubDomain && r1.Target == r2.Target && r1.Zone == r2.Zone && r1.Ttl == r2.Ttl
}

type OVHManager struct {
	RemoteCNAMERecords []OVHRecord
	RemoteARecords     []OVHRecord
	ovhClient          *ovh.Client
	config             Config
}

func (ovhManager *OVHManager) GetRecordBySubDomain(subDomain string) (OVHRecord, error) {
	for _, record := range ovhManager.RemoteCNAMERecords {
		if record.SubDomain == subDomain {
			return record, nil
		}
	}
	return OVHRecord{}, errors.New("record not found")
}

func (ovhManager *OVHManager) Init(conf Config) error {
	ovhManager.config = conf
	var clientErr error
	ovhManager.ovhClient, clientErr = ovh.NewClient(
		ovhManager.config.OVHApplicationEndpoint,
		ovhManager.config.OVHApplicationKey,
		ovhManager.config.OVHApplicationSecret,
		ovhManager.config.OVHConsumerKey,
	)
	if clientErr != nil {
		return clientErr
	}
	return nil
}

func (ovhManager *OVHManager) GetOVHClient() *ovh.Client {
	return ovhManager.ovhClient
}

func (ovhManager *OVHManager) GetZone() string {
	return ovhManager.config.OVHDNSDomain
}

func (ovhManager *OVHManager) LoadRemoteRecords(ctx context.Context) error {
	// l := log.FromContext(ctx)

	recordIds := []int{}
	recordUrl := fmt.Sprintf("/domain/zone/%s/record", ovhManager.config.OVHDNSDomain)

	if recordsGetErr := ovhManager.GetOVHClient().Get(recordUrl, &recordIds); recordsGetErr != nil {
		return recordsGetErr
	}

	for _, recordId := range recordIds {
		ovhRecord := OVHRecord{}
		recordIdUrl := fmt.Sprintf("%s/%s", recordUrl, strconv.Itoa(recordId))
		if recordErr := ovhManager.GetOVHClient().Get(recordIdUrl, &ovhRecord); recordErr != nil {
			return recordErr
		}
		if ovhRecord.FieldType == "A" {
			ovhManager.RemoteARecords = append(ovhManager.RemoteARecords, ovhRecord)
		} else if ovhRecord.FieldType == "CNAME" {
			ovhManager.RemoteCNAMERecords = append(ovhManager.RemoteCNAMERecords, ovhRecord)
		}
	}

	return nil
}
