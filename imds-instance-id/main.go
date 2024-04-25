package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/ec2metadata"
	"github.com/aws/aws-sdk-go/aws/session"
)

func main() {
	awsConfig := aws.NewConfig()
	sess, err := session.NewSession(awsConfig)
	if err != nil {
		fmt.Println(err)
		return
	}
	imdsClient := ec2metadata.New(sess)

	doc, err := imdsClient.GetInstanceIdentityDocument()
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(doc.InstanceID)
}
