package main

import (
	"fmt"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// EC2 Action Resupt Type
type EC2StatusActionResult struct {
	PrevState string `json:"prev"`
	CurState  string `json:"cur"`
	Ip        string `json:"ip"`
}

func StartEC2Instance(instanceID string) (*EC2StatusActionResult, error) {
	svc := ec2.New(session.New())
	input := &ec2.StartInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	}
	response, err := svc.StartInstances(input)
	if err != nil {
		fmt.Println(err)
		return &EC2StatusActionResult{}, err
	}
	// Associate eip to the instance
	AssociateEipToEC2(instanceID)
	return &EC2StatusActionResult{
		PrevState: *response.StartingInstances[0].PreviousState.Name,
		CurState:  *response.StartingInstances[0].CurrentState.Name,
	}, nil
}

func StopEC2Instance(instanceID string) (*EC2StatusActionResult, error) {
	svc := ec2.New(session.New())
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
	svc := ec2.New(session.New())
	input := &ec2.AssociateAddressInput{
		InstanceId:   aws.String(instanceID),
		AllocationId: aws.String(os.Getenv("ELASTIC_IP_ID")),
	}
	_, err := svc.AssociateAddress(input)
	if err != nil {
		fmt.Println(err)
	}
}

func DescribeEC2Instance(instanceID string) (string, error) {
	fmt.Println("Describe EC2 Instance")
	svc := ec2.New(session.New())
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
