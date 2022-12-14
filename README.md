# Alibaba Load Balancer Controller

Alibaba Load Balancer Controller is a cloud-native controller that allows you to use ALB (Application Load Balancer) in self-managed Kubernetes clusters.

- An AlbConfig object is a CustomResourceDefinition (CRD) used to configure ALB instances and listeners. An AlbConfig object corresponds to one ALB instance.
- An Ingress contains reverse proxy rules. It controls to which Services HTTP or HTTPS requests are routed. For example, an Ingress routes requests to different Services based on the hosts and URLs in the requests.
- An AlbConfig object is used to configure an ALB instance. The ALB instance can be specified in forwarding rules of multiple Ingresses. Therefore, an AlbConfig object can be associated with multiple Ingresses.



## Start

- [Deployment](docs/dev.md)
- [Quick start](docs/getting-started.md)
- [Usage](docs/usage.md)

## Development

- Perform e2e tests on existing staging scenarios to make sure that the features are working as expected. `make test`
- Build application images for Kubernetes clusters. ` make image`


## Communication

- For more information about ALB features, see [Official documentation](https://help.aliyun.com/document_detail/196881.html).

- If you have questions, submit issues.
