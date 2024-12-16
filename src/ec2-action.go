package main

import (
	"fmt"
	"os"
	"time"

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
	err = svc.WaitUntilInstanceStatusOk(&ec2.DescribeInstanceStatusInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})

	// Wait until the instance has a public IP address
	var publicIP *string
	var count = 0
	for count < 3 {

		time.Sleep(3 * time.Second) // Wait before checking again

		// Describe the instance to get its public IPv4 address
		result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
			InstanceIds: []*string{aws.String(instanceID)},
		})
		if err != nil {
			fmt.Println("Error describing instance:", err)
			continue
		}

		// Extract the public IPv4 address
		if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
			instance := result.Reservations[0].Instances[0]
			if instance.State != nil && *instance.State.Name == "running" {
				publicIP = instance.PublicIpAddress
				if publicIP != nil {
					fmt.Println("Public IPv4 Address:", *publicIP)
					break // Exit the loop if the public IP is found
				}
			}
		}

		fmt.Println("Waiting for public IPv4 address...")
		count = count + 1
	}

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
	fmt.Println(dns)
	if err != nil {
		fmt.Println("Error associating ec2 with DNS:", err)
		return &EC2StatusActionResult{}, err
	}

	return &EC2StatusActionResult{
		PrevState: *response.StartingInstances[0].PreviousState.Name,
		CurState:  *response.StartingInstances[0].CurrentState.Name,
		Ip:        ip,
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
	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})

	if err != nil {
		fmt.Println("Error describing instance:", err)
		return "", err
	}

	fmt.Println(result)

	// Extract the public IPv4 address
	if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
		instance := result.Reservations[0].Instances[0]
		if instance.PublicIpAddress != nil {
			fmt.Println("Public IPv4 Address:", *instance.PublicIpAddress)
			return *instance.PublicIpAddress, nil
		} else {
			fmt.Println("No Public IPv4 Address assigned to this instance.")
		}
	} else {
		fmt.Println("No instance found.")
	}

	return "", fmt.Errorf("NO_INSTANCE_FOUND")
}
