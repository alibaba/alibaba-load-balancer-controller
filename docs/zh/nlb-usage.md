# 配置网络型负载均衡NLB

通过Service YAML文件中的Annotation（注解），可以实现丰富的负载均衡功能。网络型负载均衡NLB（Network Load Balancer）是阿里云面向万物互联时代推出的新一代四层负载均衡，支持超高性能和自动弹性能力。本文从NLB、监听和服务器组三种资源维度介绍通过注解可以对NLB进行的常见配置操作。

## 背景信息

关于NLB的更多信息，请参见[什么是网络型负载均衡NLB](https://help.aliyun.com/document_detail/439121.htm#concept-2223473)。

## 注意事项

- Kubernetes版本不低于v1.24
- Service中`spec.loadBalancerClass`需要指定为`alibabacloud.com/nlb`,
- Service一旦创建后`spec.loadBalancerClass`不支持更改

## NLB实例

### 创建公网类型的NLB

- NLB支持的地域及可用区可以登录[NLB控制台](https://slbnew.console.aliyun.com/nlb/cn-hangzhou/nlbs)查看，至少需要两个可用区。
- 多个可用区间用逗号分隔，如`cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321` 。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 创建私网类型的NLB

- NLB支持的地域及可用区可以登录[NLB控制台](https://slbnew.console.aliyun.com/nlb/cn-hangzhou/nlbs)查看，至少需要两个可用区。
- 多个可用区间用逗号分隔，如`cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321` 。
- 可以更改`service.beta.kubernetes.io/alibaba-cloud-loadbalancer-address-type`取值，实现NLB的公私网转变。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-address-type: "intranet"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 指定负载均衡名称

NLB名称长度为2~128个英文或中文字符，必须以大小写字母或中文开头，可包含数字、点号（.）、下划线（_）和短横线（-）。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-name: "${your-nlb-name}" #NLB名称。
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 指定负载均衡所属的资源组

登录[阿里云资源管理平台](https://resourcemanager.console.aliyun.com/?spm=a2c4g.11186623.0.0.47923b14vO9gN7)查询资源组ID，然后使用以下Annotation为负载均衡实例指定资源组。

> **说明** 资源组ID创建后不可被修改。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-resource-group-id:  "${your-resource-group-id}" #资源组ID。
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 创建双栈类型的NLB

- 集群的kube-proxy代理模式需要是IPVS。
- `service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps`中指定的两个vSwitch均需开启IPv6。
- 创建后IP类型不可更改。
- 生成的IPv6地址仅可在支持IPv6的环境中访问。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-ip-version: "DualStack"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  sessionAffinity: None
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 为负载均衡添加额外标签

多个Tag以逗号分隔，例如`k1=v1,k2=v2`。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-additional-resource-tags: "Key1=Value1,Key2=Value2"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  sessionAffinity: None
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 使用已有的负载均衡

Annotation `service.beta.kubernetes.io/alibaba-cloud-loadbalancer-force-override-listeners`决定是否根据Service同步NLB监听配置。取值：

- true：CCM会根据Service配置，创建、更新、删除NLB监听。
- false：CCM不会对NLB监听做任何处理。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-id: "${your-nlb-id}" #NLB的ID。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-force-override-listeners: "true"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  sessionAffinity: None
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

## 监听

### 为监听同时配置TCP及UDP协议

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: udp
    port: 90
    protocol: UDP
    targetPort: 90
  selector:
    app: nginx
  sessionAffinity: None
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 创建TCP类型监听

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  sessionAffinity: None
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 创建UDP类型监听

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: udp
    port: 80
    protocol: UDP
    targetPort: 80
  selector:
    app: nginx
  sessionAffinity: None
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 创建TCPSSL类型监听

登录[数字证书管理服务](https://yundunnext.console.aliyun.com/)控制台创建并记录证书ID，然后使用如下Annotation创建一个TCPSSL类型监听。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:443"
    # 如集群为中国内地Region时，组合后的证书ID为${your-cert-id}-cn-hangzhou。
    # 如集群为除中国内地以外的其他Region时，组合后的证书ID为${your-cert-id}-ap-southeast-1。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${组合后的证书ID}"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 开启双向认证

登录[数字证书管理服务](https://yundunnext.console.aliyun.com/)控制台查看证书ID及CA证书ID，然后使用如下Annotation为监听开启双向认证。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:443"   
    # 如集群为中国内地Region时，组合后的证书ID为${your-cert-id}-cn-hangzhou。
    # 如集群为除中国内地以外的其他Region时，组合后的证书ID为${your-cert-id}-ap-southeast-1。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${组合后的证书ID}" 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cacert-id: "${your-cacert-id}"  # CA证书ID。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cacert: "on"
name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 设置TLS安全策略

登录[数字证书管理服务](https://yundunnext.console.aliyun.com/)控制台查看证书ID，然后使用如下Annotation设置TLS安全策略。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:443"
    # 如集群为中国内地Region时，组合后的证书ID为${your-cert-id}-cn-hangzhou。
    # 如集群为除中国内地以外的其他Region时，组合后的证书ID为${your-cert-id}-ap-southeast-1。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${组合后的证书ID}" 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-tls-cipher-policy: "tls_cipher_policy_1_2"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 设置ProxyProtocol

**重要** 启用ProxyProtocol之前，请检查后端服务是否已开启proxyprotocolV2。如后端未开启proxyprotocolV2会导致访问不通， 请谨慎配置。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-proxy-protocol: "on"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 设置监听每秒新建连接限速值

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cps: "100"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 设置监听连接空闲超时时间

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-idle-timeout: "60"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

## 服务器组

### 设置调度策略

调度算法。取值：

- wrr（默认值）：加权轮询，权重值越高的服务器，被轮询到的概率也越高。
- rr：轮询，按照访问顺序依次将外部请求分发到服务器。
- sch：源IP哈希，相同的源地址会调度到相同的服务器。
- tch：四元组哈希，基于四元组（源IP、目的IP、源端口和目的端口）的一致性哈希，相同的流会调度到相同的服务器。

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-scheduler: "sch"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 设置连接优雅中断

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-connection-drain: "on"
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-connection-drain-timeout: "30"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 设置客户端地址保持

> TCPSSL 没有这个功能

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-preserve-client-ip: "on"
  name: nginx
  namespace: default
spec:
  externalTrafficPolicy: Local
  ports:
  - name: tcp
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 443
  selector:
    app: nginx
  loadBalancerClass: "alibabacloud.com/nlb"
  type: LoadBalancer
```

### 设置健康检查

- 设置TCP类型的健康检查，以下所有Annotation必选。TCP端口默认开启健康检查。

  ```yaml
  apiVersion: v1
  kind: Service
  metadata:
    annotations:
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-flag: "on"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-type: "tcp"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-connect-timeout: "8"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-healthy-threshold: "4"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-unhealthy-threshold: "4"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-interval: "5"
    name: nginx
    namespace: default
  spec:
    externalTrafficPolicy: Local
    ports:
    - name: tcp
      port: 80
      protocol: TCP
      targetPort: 80
    - name: https
      port: 443
      protocol: TCP
      targetPort: 443
    selector:
      app: nginx
    loadBalancerClass: "alibabacloud.com/nlb"
    type: LoadBalancer
  ```

- 设置HTTP类型的健康检查。

  ```yaml
  apiVersion: v1
  kind: Service
  metadata:
    annotations:
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #例如：cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321。
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-flag: "on"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-type: "http"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-uri: "/test/index.html"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-domain: "www.test.com"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-healthy-threshold: "4"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-unhealthy-threshold: "4"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-connect-timeout: "10"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-interval: "5"
      # 设置健康检查方法，该Annotation可选。
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-method: "head"
    name: nginx
    namespace: default
  spec:
    externalTrafficPolicy: Local
    ports:
    - name: tcp
      port: 80
      protocol: TCP
      targetPort: 80
    - name: https
      port: 443
      protocol: TCP
      targetPort: 443
    selector:
      app: nginx
    loadBalancerClass: "alibabacloud.com/nlb"
    type: LoadBalancer
  ```

## 常用注解

### NLB常用注解

| 注解                                                         | 类型   | 描述                                                         | 默认值   |
| :----------------------------------------------------------- | :----- | :----------------------------------------------------------- | :------- |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps | string | 指定NLB的可用区。NLB支持的地域及可用区可以登录[NLB控制台](https://slbnew.console.aliyun.com/nlb/cn-hangzhou/nlbs)查看，至少需要两个可用区。 | 无       |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-address-type | string | 取值：internet：公网NLB。intranet：私网NLB。                 | internet |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-name   | string | 负载均衡实例名称。                                           | 无       |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-resource-group-id | string | 负载均衡所属资源组ID。                                       | 无       |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-ip-version | string | 协议版本。取值：ipv4：IPv4类型。DualStack：双栈类型。        | ipv4     |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-additional-resource-tags | string | 需要添加的Tag列表，多个标签用逗号分隔。例如：`k1=v1,k2=v2`。 | 无       |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-id     | string | 负载均衡实例的ID。                                           | 无       |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-force-override-listeners | string | 是否根据Service同步NLB监听。取值：truefalse                  | false    |

### 监听常用注解

| 注解                                                         | 类型   | 描述                                                         | 默认值                |
| :----------------------------------------------------------- | :----- | :----------------------------------------------------------- | :-------------------- |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port | string | 设置监听的协议类型。多个值之间由逗号分隔，例如：`TCP:80,TCPSSL:443`。 | 无                    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id | string | 服务器证书ID，可以登录[数字证书管理服务](https://yundunnext.console.aliyun.com/)控制台查看。 | 无                    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cacert-id | string | CA证书ID，可以登录[数字证书管理服务](https://yundunnext.console.aliyun.com/)控制台查看。 | 无                    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cacert | string | 是否启动双向认证。取值：on：启动。off：关闭。                | 无                    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-tls-cipher-policy | string | 安全策略ID。支持系统安全策略和自定义安全策略。取值：<br />tls_cipher_policy_1_0<br />tls_cipher_policy_1_1<br />tls_cipher_policy_1_2<br />tls_cipher_policy_1_2_strict<br />tls_cipher_policy_1_2_strict_with_1_3 | tls_cipher_policy_1_0 |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-proxy-protocol | string | 是否开启通过Proxy Protocol协议携带客户端源地址到服务器。取值：on：开启。off：关闭。 | off                   |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cps    | string | 网络型负载均衡实例每秒新建连接限速值。取值范围：0~1000000。0表示不限速。 | 无                    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-idle-timeout | string | 连接空闲超时时间。单位：秒。取值范围：10~900。               | 900                   |

### 服务器组常用注解

| 注解                                                         | 类型   | 描述                                                         | 默认值 |
| :----------------------------------------------------------- | :----- | :----------------------------------------------------------- | :----- |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-scheduler | string | 调度算法。取值：wrr：加权轮询，权重值越高的服务器，被轮询到的概率也越高。rr：轮询，按照访问顺序依次将外部请求分发到服务器。sch：源IP哈希：相同的源地址会调度到相同的服务器。tch：四元组哈希，基于四元组（源IP、目的IP、源端口和目的端口）的一致性哈希，相同的流会调度到相同的服务器。 | wrr    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-connection-drain | string | 是否开启连接优雅中断。取值：on：开启。off：关闭。            | off    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-connection-drain-timeout | string | 设置连接优雅中断超时时间。单位：秒。取值范围：10~900。       | 无     |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-preserve-client-ip | string | 是否开启客户端地址保持功能。取值：on：开启。off：关闭。      | off    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-flag | string | 是否开启健康检查，取值：on：开启。off：关闭。                | on     |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-type | string | 健康检查协议。取值：tcphttp                                  | tcp    |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-connect-port | string | 健康检查的服务器的端口。范围：0~65535。默认值：0，表示使用服务器的端口进行健康检查。 | 0      |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-connect-timeout | string | 健康检查响应的最大超时时间。单位：秒。取值范围：1~300。      | 5      |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-healthy-threshold | string | 健康检查连续成功多少次后，将服务器的健康检查状态由失败判定为成功。取值范围：2~10。 | 2      |