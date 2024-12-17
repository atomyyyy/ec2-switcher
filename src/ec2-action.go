package main

import (
	"fmt"
	"os"
	"time"

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
	sess, err := session.NewSession(&aws.Config{Region: aws.String(os.Getenv("REGION"))})
	if err != nil {
		return &EC2StatusActionResult{}, fmt.Errorf("SESSION_CREATION_FAILED")
	}

	svc := ec2.New(sess)

	response, err := svc.StartInstances(
		&ec2.StartInstancesInput{
			InstanceIds: []*string{aws.String(instanceID)},
		},
	)
	if err != nil {
		fmt.Printf(err.Error())
		return &EC2StatusActionResult{}, fmt.Errorf("EC2_STARTUP_FAILED")
	}

	count := 0
	for count < 10 {
		// Sleep before next poll
		count = count + 1
		time.Sleep(1 * time.Second)

		// Get Public IP
		publicIp, err := GetPublicIpFromEC2Instance(instanceID)
		if err != nil || publicIp == "" {
			fmt.Println("PUBLIC_IP_NOT_READY")
			continue
		}

		// Bind to DNS
		err = AssociateWithDNS(publicIp)
		if err != nil {
			return &EC2StatusActionResult{}, fmt.Errorf("DNS_BINDING_FAILURE")
		}

		return &EC2StatusActionResult{
			PrevState: *response.StartingInstances[0].PreviousState.Name,
			CurState:  *response.StartingInstances[0].CurrentState.Name,
			Ip:        publicIp,
		}, nil
	}

	return &EC2StatusActionResult{
		PrevState: *response.StartingInstances[0].PreviousState.Name,
		CurState:  *response.StartingInstances[0].CurrentState.Name,
	}, nil
}

func StopEC2Instance(instanceID string) (*EC2StatusActionResult, error) {
	// Get Public IP
	publicIp, err := GetPublicIpFromEC2Instance(instanceID)
	if err != nil || publicIp == "" {
		return &EC2StatusActionResult{}, fmt.Errorf("INSTANCE_WITHOUT_PUBLIC_IP")
	}

	err = DisassociateWithDNS(publicIp)
	if err != nil {
		fmt.Errorf("UNABLE_TO_REMOVE_DNS_RECORD")
	}

	sess, err := session.NewSession(&aws.Config{Region: aws.String(os.Getenv("REGION"))})
	if err != nil {
		return &EC2StatusActionResult{}, fmt.Errorf("SESSION_CREATION_FAILED")
	}

	svc := ec2.New(sess)

	response, err := svc.StopInstances(&ec2.StopInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return &EC2StatusActionResult{}, fmt.Errorf("UNABLE_TO_STOP_INSTANCE")
	}
	return &EC2StatusActionResult{
		PrevState: *response.StoppingInstances[0].PreviousState.Name,
		CurState:  *response.StoppingInstances[0].CurrentState.Name,
		Ip:        publicIp,
	}, nil
}

func GetPublicIpFromEC2Instance(instanceID string) (string, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(os.Getenv("REGION"))})
	if err != nil {
		return "", fmt.Errorf("SESSION_CREATION_FAILED")
	}

	svc := ec2.New(sess)
	result, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
		InstanceIds: []*string{aws.String(instanceID)},
	})
	if err != nil {
		return "", fmt.Errorf("INSTANCE_NOT_FOUND")
	}

	if len(result.Reservations) > 0 && len(result.Reservations[0].Instances) > 0 {
		instance := result.Reservations[0].Instances[0]
		if instance.PublicIpAddress != nil {
			return *instance.PublicIpAddress, nil
		}
	}

	return "", fmt.Errorf("NO_PUBLIC_IP_FOUND")
}
