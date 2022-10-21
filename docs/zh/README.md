# Alibaba Load Balancer Controller

Alibaba Load Balancer Controller 是用来在云上自建Kubernets集群中使用阿里云负载均衡器产品的云原生控制器。

- 通过CRD创建AlbConfig资源，管理ALB实例和监听资源；
- 通过监听Ingress自动创建监听和相关转发规则
- 关联后端服务器组与Ingress Backend Service，节点变化同步到云端



## 开始

- [快速开始]( )
- [使用指南]()



## 开发



- 对已有用例执行e2e测试，保证历史功能正常 `make test` 
- 构建应用镜像以供Kubernetes集群部署使用 ` make image` 



## 交流



- 关于ALB产品特性可以参考[官网文档](https://help.aliyun.com/document_detail/196881.html)

- 使用中产生问题可以提issue
