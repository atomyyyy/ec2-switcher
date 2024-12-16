package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/route53"
)

// EC2 Action Resupt Type
type EC2StatusActionResult struct {
	PrevState string `json:"prev"`
	CurState  string `json:"cur"`
	Ip        string `json:"ip"`
}

func StartEC2Instance(instanceID string) (*EC2StatusActionResult, error) {
	svc := ec2.New(session.New(&aws.Config{Region: aws.String(os.Getenv("REGION"))}))
	input := &ec2.StartInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	}
	// Start Instance
	response, err := svc.StartInstances(input)
	if err != nil {
		fmt.Println(err)
		return &EC2StatusActionResult{}, err
	}

	// Wait until instance running
	err = svc.WaitUntilInstanceRunning(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})

	if err != nil {
		fmt.Println("Error waiting for instance to run:", err)
		return &EC2StatusActionResult{}, err
	}

	// Describe the instance to get its public IPv4 address
	ip, err := DescribeEC2Instance(instanceID)
	if err != nil {
		fmt.Println("Error describing ec2:", err)
		return &EC2StatusActionResult{}, err
	}

	// Bind EC2 to DNS
	dns, err := AssociateWithDNS(ip)
	if err != nil {
		fmt.Println("Error associating ec2 with DNS:", err)
		return &EC2StatusActionResult{}, err
	}

	return &EC2StatusActionResult{
		PrevState: *response.StartingInstances[0].PreviousState.Name,
		CurState:  *response.StartingInstances[0].CurrentState.Name,
		Ip:        dns,
	}, nil
}

func StopEC2Instance(instanceID string) (*EC2StatusActionResult, error) {
	svc := ec2.New(session.New(&aws.Config{Region: aws.String(os.Getenv("REGION"))}))
	input := &ec2.StopInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	}
	response, err := svc.StopInstances(input)
	if err != nil {
		fmt.Println(err)
		return &EC2StatusActionResult{}, err
	}
	return &EC2StatusActionResult{
		PrevState: *response.StoppingInstances[0].PreviousState.Name,
		CurState:  *response.StoppingInstances[0].CurrentState.Name,
	}, nil
}

func AssociateEipToEC2(instanceID string) {
	svc := ec2.New(session.New(&aws.Config{Region: aws.String(os.Getenv("REGION"))}))
	input := &ec2.AssociateAddressInput{
		InstanceId:   aws.String(instanceID),
		AllocationId: aws.String(os.Getenv("ELASTIC_IP_ID")),
	}
	_, err := svc.AssociateAddress(input)
	if err != nil {
		fmt.Println(err)
	}
}

func AssociateWithDNS(publicEc2Url string) (string, error) {
	sess, err := session.NewSession()
	if err != nil {
		return "", err
	}

	svc := route53.New(sess)

	// Define the parameters for the change
	change := &route53.Change{
		Action: aws.String("UPSERT"),
		ResourceRecordSet: &route53.ResourceRecordSet{
			Name:            aws.String(os.Getenv("CUSTOM_DNS")),
			Type:            aws.String("A"),
			TTL:             aws.Int64(300), // Change TTL as needed
			ResourceRecords: []*route53.ResourceRecord{{Value: aws.String(publicEc2Url)}},
		},
	}

	// Specify the hosted zone ID
	hostedZoneID := os.Getenv("HOSTED_ZONE_ID")

	// Create the change batch
	changeBatch := &route53.ChangeBatch{
		Changes: []*route53.Change{change},
	}

	// Update the record set
	_, err = svc.ChangeResourceRecordSets(&route53.ChangeResourceRecordSetsInput{
		HostedZoneId: aws.String(hostedZoneID),
		ChangeBatch:  changeBatch,
	})

	if err != nil {
		return "", err
	}

	return os.Getenv("CUSTOM_DNS"), nil
}

func DescribeEC2Instance(instanceID string) (string, error) {
	fmt.Println("Describe EC2 Instance")
	svc := ec2.New(session.New(&aws.Config{Region: aws.String(os.Getenv("REGION"))}))
	resp, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		fmt.Println("Error describing instance", instanceID, err)
		return "", err
	}

	if len(resp.Reservations) == 0 {
		return "", nil
	}

	if len(resp.Reservations[0].Instances) == 0 {
		return "", nil
	}

	ipAddress := *resp.Reservations[0].Instances[0].PublicIpAddress
	return ipAddress, nil
}
