package route53

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/elbv2"
	"github.com/aws/aws-sdk-go/service/route53"
	"github.com/rs/zerolog/log"
	"wekactl/internal/connectors"
)

func CreateApplicationLoadBalancerAliasRecord(loadBalancer *elbv2.LoadBalancer, dnsAlias, dnsZoneId string) error {
	svc := connectors.GetAWSSession().Route53

	_, err := svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("CREATE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						AliasTarget: &route53.AliasTarget{
							DNSName:              loadBalancer.DNSName,
							EvaluateTargetHealth: aws.Bool(false),
							HostedZoneId:         loadBalancer.CanonicalHostedZoneId,
						},
						Name: &dnsAlias,
						Type: aws.String("A"),
					},
				},
			},
		},
		HostedZoneId: &dnsZoneId,
	})

	if err != nil {
		return err
	}

	log.Debug().Msgf("route53 alias %s was updated successfully", dnsAlias)
	return nil
}

func DeleteRoute53Record(recordSet *route53.ResourceRecordSet, dnsAlias, dnsZoneId string) error {
	svc := connectors.GetAWSSession().Route53

	_, err := svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("DELETE"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						AliasTarget: &route53.AliasTarget{
							DNSName:              recordSet.AliasTarget.DNSName,
							EvaluateTargetHealth: aws.Bool(false),
							HostedZoneId:         recordSet.AliasTarget.HostedZoneId,
						},
						Name: &dnsAlias,
						Type: aws.String("A"),
					},
				},
			},
		},
		HostedZoneId: &dnsZoneId,
	})

	if err != nil {
		return err
	}

	log.Debug().Msgf("route53 alias %s was deleted successfully!", dnsAlias)
	return nil
}

func GetRoute53Record(dnsAlias, dnsZoneId string) (recordSet *route53.ResourceRecordSet, err error) {
	svc := connectors.GetAWSSession().Route53
	route53Output, err := svc.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId:    &dnsZoneId,
		StartRecordType: aws.String("A"),
		StartRecordName: &dnsAlias,
	})
	if err != nil {
		return
	}
	for _, record := range route53Output.ResourceRecordSets {
		if dnsAlias[len(dnsAlias)-1:] != "." {
			dnsAlias = dnsAlias + "."
		}
		if *record.Name == dnsAlias {
			recordSet = record
			break
		}
	}
	return
}
