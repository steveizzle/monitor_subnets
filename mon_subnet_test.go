package main

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/prometheus/client_golang/prometheus"
)

// Mock EC2 client for testing
type mockEC2Client struct{}

func (m *mockEC2Client) DescribeSubnets(ctx context.Context, params *ec2.DescribeSubnetsInput, optFns ...func(*ec2.Options)) (*ec2.DescribeSubnetsOutput, error) {
	// Implement mock behavior here and return test data
	// For example, return some predefined output that you want to test
	return &ec2.DescribeSubnetsOutput{
		Subnets: []types.Subnet{
			{
				CidrBlock:               aws.String("10.0.0.0/24"),
				AvailableIpAddressCount: aws.Int32(50),
			},
		},
	}, nil
}

func TestRecordSubnetMetrics(t *testing.T) {
	subnet := "subnet-1"

	// Use the mock EC2 client for testing
	client := &mockEC2Client{}

	// Record metrics using the function to be tested
	recordSubnetMetrics(subnet, client)

	// Wait for some time to let the metrics goroutine run
	time.Sleep(1 * time.Second)

	// Now, you can check if the metrics are correctly recorded and updated
	// For example, you can use prometheus.DefaultGatherer and prometheus.Gatherer.Gather method to retrieve metrics and validate their values
	// Below is an example of how you can test if the metrics are registered and their values are as expected.
	mfs, err := prometheus.DefaultGatherer.Gather()
	if err != nil {
		t.Fatalf("Failed to gather metrics: %v", err)
	}

	// Assuming you have only one subnet metric registered
	if len(mfs) != 29 {
		t.Log(mfs)
		t.Fatalf("Expected 2 metrics, but got %d", len(mfs))
	}

	expectedMetrics := map[string]float64{
		"aws_subnet_ips_free":  50,
		"aws_subnet_ips_total": 256, // Assuming a /24 subnet
	}

	for _, mf := range mfs {
		if mf == nil || mf.Metric == nil {
			t.Fatalf("Unexpected nil MetricFamily or Metric in MetricFamily")
		}
		for _, m := range mf.Metric {
			if m == nil || m.Label == nil {
				t.Fatalf("Unexpected nil Metric or Label in Metric")
			}
			subnetLabel := ""
			for _, l := range m.Label {
				if l.GetName() == "subnet" {
					subnetLabel = l.GetValue()
					break
				}
			}
			value := m.Gauge.GetValue()

			expectedValue, found := expectedMetrics[mf.GetName()]
			if found && subnetLabel == subnet {
				t.Log(value)
				t.Log(expectedValue)
				if value != expectedValue {
					t.Errorf("Metric %s value mismatch, expected: %f, got: %f", mf.GetName(), expectedValue, value)
				}
				delete(expectedMetrics, mf.GetName())
			}
		}
	}

	// Make sure all expected metrics were found
	if len(expectedMetrics) != 0 {
		t.Errorf("Not all expected metrics were found")
	}
}
