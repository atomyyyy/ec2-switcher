package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

// Machine Enum
const (
	DEVELOPMENT = "development"
	GAME        = "game"
)

// Action Enum
const (
	START = "start"
	STOP  = "stop"
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

func RequestHandler(ctx context.Context, request events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	// Initialize Resource Id Mapping
	resourceIdMapping := map[string]string{
		GAME:        os.Getenv("GAME_EC2_RESOURCE_ID"),
		DEVELOPMENT: os.Getenv("DEVELOPMENT_EC2_RESOURCE_ID"),
	}

	// Get Query Parameters
	action := request.QueryStringParameters["action"]
	machine := request.QueryStringParameters["machine"]

	// Define Default Action
	if len(action) == 0 {
		action = STOP
	}

	// Define EC2 Instance Id
	var ec2InstanceId string
	if len(machine) == 0 {
		ec2InstanceId = resourceIdMapping[GAME]
	} else {
		ec2InstanceId = resourceIdMapping[machine]
	}

	var ec2ActionResponse *EC2StatusActionResult
	var eip string = ""
	var err error

	// Start EC2
	switch action {
	case START:
		{
			ec2ActionResponse, err = StartEC2Instance(ec2InstanceId)
			// Get IP of the EC2 instance
			eip, err = DescribeEC2Instance(ec2InstanceId)
		}
	default:
		{
			ec2ActionResponse, err = StopEC2Instance(ec2InstanceId)
		}
	}

	if err != nil {
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       err.Error(),
		}, nil
	}

	// Create Response Object
	response := EC2StatusActionResult{
		PrevState: ec2ActionResponse.PrevState,
		CurState:  ec2ActionResponse.CurState,
		Ip:        eip,
	}

	jsonString, err := json.Marshal(response)

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Body:       string(jsonString),
	}, nil
}

func main() {
	lambda.Start(RequestHandler)
}
