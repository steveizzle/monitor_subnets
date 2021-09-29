package main

import (
	"context"
	"encoding/binary"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func recordSubnetMetrics(subnet string, client *ec2.Client) {
	var (
		ipsFree = promauto.NewGauge(prometheus.GaugeOpts{
			Name:        "aws_subnet_ips_free",
			Help:        "Number of available Ips in subnet",
			ConstLabels: prometheus.Labels{"subnet": subnet},
		})
	)
	var (
		ipsTotal = promauto.NewGauge(prometheus.GaugeOpts{
			Name:        "aws_subnet_ips_total",
			Help:        "Number of total Ips in subnet",
			ConstLabels: prometheus.Labels{"subnet": subnet},
		})
	)
	go func() {
		for {
			param := ec2.DescribeSubnetsInput{
				SubnetIds: []string{subnet},
			}
			output, err := client.DescribeSubnets(context.TODO(), &param)
			if err != nil {
				fmt.Println(err)
			}
			for _, s := range output.Subnets {
				_, ipv4Net, err := net.ParseCIDR(*s.CidrBlock)
				if err != nil {
					log.Fatal(err)
				}
				ipsTotal.Set(float64(binary.BigEndian.Uint32(ipv4Net.Mask) ^ 0xffffffff))
				ipsFree.Set(float64(*s.AvailableIpAddressCount))
			}
			time.Sleep(30 * time.Second)
		}
	}()
}

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("Configuration error, " + err.Error())
	}

	// Handle subnets
	subnetEnv := os.Getenv("AWS_MONITOR_SUBNETS")
	if subnetEnv != "" {
		subnets := strings.Split(subnetEnv, ",")
		client := ec2.NewFromConfig(cfg)
		for _, s := range subnets {
			recordSubnetMetrics(s, client)
		}
	}
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)
}
