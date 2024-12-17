package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/route53"
)

func AssociateWithDNS(publicIp string) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("SESSION_CREATION_FAILED")
	}

	svc := route53.New(sess)
	_, err = svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(os.Getenv("HOSTED_ZONE_ID")),
		ChangeBatch: &route53.ChangeBatch{
			Changes: []*route53.Change{
				{
					Action: aws.String("UPSERT"),
					ResourceRecordSet: &route53.ResourceRecordSet{
						Name:            aws.String(os.Getenv("CUSTOM_DNS")),
						Type:            aws.String("A"),
						TTL:             aws.Int64(300),
						ResourceRecords: []*route53.ResourceRecord{{Value: aws.String(publicIp)}},
					},
				},
			},
		},
	})

	if err != nil {
		return fmt.Errorf("RESOURCE_RECORD_CHANGE_FAILED")
	}

	return nil
}

func DisassociateWithDNS(publicIp string) error {
	sess, err := session.NewSession()
	if err != nil {
		return fmt.Errorf("SESSION_CREATION_FAILED")
	}

	svc := route53.New(sess)

	records, err := svc.ListResourceRecordSets(&route53.ListResourceRecordSetsInput{
		HostedZoneId: aws.String(os.Getenv("HOSTED_ZONE_ID")),
	})
	if err != nil {
		return fmt.Errorf("LIST_RESOURCE_RECORD_FAILED")
	}

	for _, record := range records.ResourceRecordSets {
		if publicIp == *record.ResourceRecords[len(record.ResourceRecords)-1].Value {
			_, err = svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
				HostedZoneId: aws.String(os.Getenv("HOSTED_ZONE_ID")),
				ChangeBatch: &route53.ChangeBatch{
					Changes: []*route53.Change{
						{
							Action: aws.String("DELETE"),
							ResourceRecordSet: &route53.ResourceRecordSet{
								Name:            record.Name,
								Type:            record.Type,
								TTL:             record.TTL,
								ResourceRecords: record.ResourceRecords,
							},
						},
					},
				},
			})
			if err != nil {
				return fmt.Errorf("DELETE_RESOURCE_RECORD_FAILED")
			}
			break
		}
	}
	return nil
}
