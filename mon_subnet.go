/*
The MIT License (MIT)

Copyright (c) Stefan Riembauer

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in all
copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
SOFTWARE.
*/

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

// The function gets the AWS subnet data and exposes the corresponding metrics
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

// main function
// Gets all Env variables, starts the corresponding functions
// to expose the metrics and the prom http server
func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("Configuration error, " + err.Error())
	}

	// Handle AWS subnets
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
