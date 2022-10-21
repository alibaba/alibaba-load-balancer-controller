# Alibaba Load Balancer Controller

Alibaba Load Balancer Controller is a cloud-native controller used to use Alibaba Cloud Load Balancer products in self-built Kubernetes clusters on the cloud.

- Create AlbConfig resources through CRD, manage ALB instances and watch resources;
- Automatically create Listeners and related Forwarding Rules by watching Ingress
- Associate the backend server group with the Ingress Backend Service, and the node changes are synchronized to the cloud

## start

- [Getting Start]( )
- [Usage]()

## development

- Execute e2e tests on existing use cases to ensure normal historical functions `make test`
- Build images for Kubernetes cluster deployment ` make image`

## Comminicate

- For ALB product features, please refer to [official website document](https://help.aliyun.com/document_detail/196881.html)

- If you have problems during use, you can raise an issue