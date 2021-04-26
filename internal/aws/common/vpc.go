package common

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"wekactl/internal/connectors"
)

func VpcBySubnet(subnetId string) (string, error) {
	svc := connectors.GetAWSSession().EC2
	subnets, err := svc.DescribeSubnets(&ec2.DescribeSubnetsInput{
		SubnetIds: []*string{aws.String(subnetId)},
	})
	if err != nil {
		return "", err
	}
	vpcId := subnets.Subnets[0].VpcId
	return *vpcId, nil
}

func GetVpcSubnets(vpcId string) (subnets []*ec2.Subnet, err error) {
	svc := connectors.GetAWSSession().EC2

	var nextToken *string
	for {
		var subnetsOutput *ec2.DescribeSubnetsOutput
		subnetsOutput, err = svc.DescribeSubnets(&ec2.DescribeSubnetsInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("vpc-id"),
					Values: []*string{
						&vpcId,
					},
				},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return
		}

		subnets = append(subnets, subnetsOutput.Subnets...)
		nextToken = subnetsOutput.NextToken
		if nextToken == nil {
			break
		}
	}
	return
}

func GetRouteTables(vpcId string) (routeTables []*ec2.RouteTable, err error) {
	svc := connectors.GetAWSSession().EC2

	var nextToken *string
	for {
		var subnetsOutput *ec2.DescribeRouteTablesOutput
		subnetsOutput, err = svc.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
			Filters: []*ec2.Filter{
				{
					Name: aws.String("vpc-id"),
					Values: []*string{
						&vpcId,
					},
				},
			},
			NextToken: nextToken,
		})
		if err != nil {
			return
		}

		routeTables = append(routeTables, subnetsOutput.RouteTables...)
		nextToken = subnetsOutput.NextToken
		if nextToken == nil {
			break
		}
	}
	return
}
