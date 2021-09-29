package main

import (
	"context"
	"flag"
	"fmt"
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

func recordMetrics(subnet string, client *ec2.Client) {
	var (
		ipsFree = promauto.NewGauge(prometheus.GaugeOpts{
			Name: "aws_subnet_ips_free",
			Help: "Number of available Ips in subnet",
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
				fmt.Println(*s.AvailableIpAddressCount)
				ipsFree.Set(float64(*s.AvailableIpAddressCount))
			}
			time.Sleep(30 * time.Second)
		}
	}()
}

func main() {
	subnetArgs := flag.String("s", "", "Subnets to monitor")
	flag.Parse()

	if *subnetArgs == "" {
		fmt.Printf("%s subnet1,subnet2,...", os.Args[0])
		os.Exit(1)
	}

	fmt.Println(*subnetArgs)
	subnets := strings.Split(*subnetArgs, ",")
	fmt.Println(subnets)

	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		panic("configuration error, " + err.Error())
	}
	client := ec2.NewFromConfig(cfg)

	for _, s := range subnets {
		recordMetrics(s, client)
	}
	http.Handle("/metrics", promhttp.Handler())
	http.ListenAndServe(":2112", nil)

}
