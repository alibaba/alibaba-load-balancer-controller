# Alibaba Load Balancer Controller

Alibaba Load Balancer Controller is a cloud-native controller that allows you to use ALB (Application Load Balancer) and NLB (Network Load Balancer) in self-managed Kubernetes clusters.

- An AlbConfig object is a CustomResourceDefinition (CRD) used to configure ALB instances and listeners. An AlbConfig object corresponds to one ALB instance.
- An Ingress contains reverse proxy rules. It controls to which Services HTTP or HTTPS requests are routed. For example, an Ingress routes requests to different Services based on the hosts and URLs in the requests.
- An AlbConfig object is used to configure an ALB instance. The ALB instance can be specified in forwarding rules of multiple Ingresses. Therefore, an AlbConfig object can be associated with multiple Ingresses.
- Use Service annotations config NLB instance, listeners, and backend server groups. provides ultrahigh performance and auto scaling.



## Start

- [Deployment](docs/dev.md)
- [Quick start](docs/getting-started.md)
- [ALB Usage](docs/usage.md)
- [NLB Usage](docs/nlb-usage.md)

## Development

- Perform e2e tests on existing staging scenarios to make sure that the features are working as expected. `make test`
- Build application images for Kubernetes clusters. ` make image`


## Communication

- For more information about ALB features, see [Official documentation](https://www.alibabacloud.com/help/zh/server-load-balancer/latest/application-load-balancer).

- For more information about NLB features, see [Official documentation](https://www.alibabacloud.com/help/zh/server-load-balancer/latest/network-load-balancer).

- If you have questions, submit issues.
