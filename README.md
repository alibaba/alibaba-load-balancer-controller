# Alibaba Load Balancer Controller

Alibaba Load Balancer Controller is a cloud-native controller that allows you to use the ALB service in self-managed Kubernetes clusters.

- You can manage ALB instances and listener resources by using CRD to create AlbConfig resources.
- Use listener Ingresses to automatically create listeners and forwarding rules.
- Associate backend server groups with the Ingress Backend Service. Node changes are synchronized to the cloud.



## Start

- [Quick start](docs/getting-started.md)
- [User guide](docs/usage.md)



## Development



- Perform e2e tests on existing staging scenarios to make sure that the features are working as expected. `make test`
- Build application images for Kubernetes clusters. ` make image`



## Communication



- For more information about ALB features, see [Official documentation](https://help.aliyun.com/document_detail/196881.html).

- If you have questions, submit issues.
