# Cloud exporter

This is a prometheus exporter which should scrape different cloud metrics by setting the corresponding environment variable and exposes it under :2112/metrics

Currently the only available metric is: 
* AWS Subnet Metrics (total ips free and used in subnet) by setting a comma seperated list in env AWS_MONITOR_SUBNETS