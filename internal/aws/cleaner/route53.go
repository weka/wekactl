package cleaner

import (
	"github.com/aws/aws-sdk-go/service/route53"
	route53internal "wekactl/internal/aws/route53"
	"wekactl/internal/logging"
)

type Route53 struct {
	DnsAlias  string
	DnsZoneId string
	RecordSet *route53.ResourceRecordSet
}

func (r *Route53) Fetch() error {
	record, err := route53internal.GetRoute53Record(r.DnsAlias, r.DnsZoneId)
	if err != nil {
		return err
	}
	r.RecordSet = record
	return nil
}

func (r *Route53) Delete() error {
	if r.RecordSet != nil {
		return route53internal.DeleteRoute53Record(r.RecordSet, r.DnsAlias, r.DnsZoneId)
	}
	return nil
}

func (r *Route53) Print() {
	logging.UserInfo("Route53 record:")
	if r.RecordSet != nil {
		logging.UserInfo("\t- %s", r.DnsAlias)
	}
}
