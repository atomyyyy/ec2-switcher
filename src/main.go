package main

import (
	"context"
	"encoding/json"
	"os"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
)

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
