# Configure NLB instances

You can add annotations to the YAML file of a Service to configure load balancing. Network Load Balancer (NLB) instances are next-generation Layer 4 load balancers developed by Alibaba Cloud for the Internet of Everything (IoE). NLB provides ultrahigh performance and auto scaling. This topic describes how to use annotations to configure load balancing. You can configure NLB instances, listeners, and backend server groups.

## Background information

For more information, see [What is NLB?](https://www.alibabacloud.com/help/en/server-load-balancer/latest/what-is-nlb#concept-2223473).

## Precautions

- The Kubernetes version of your cluster must be V1.24 or later
- To configure an NLB instance for a Service, set the `spec.loadBalancerClass` parameter of the Service to `alibabacloud.com/nlb`. 
- You cannot modify the `spec.loadBalancerClass` parameter of a Service 

## NLB

### Create an Internet-facing NLB instance

- You can log on to the [NLB](https://slbnew.console.aliyun.com/nlb/cn-hangzhou/nlbs) console to view the regions and zones that support NLB. Select at least two zones for each NLB instance.
- Separate multiple zones with commas (,). Example: `cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321`.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Create an internal-facing NLB instance

- You can log on to the [NLB](https://slbnew.console.aliyun.com/nlb/cn-hangzhou/nlbs) console to view the regions and zones that support NLB. Select at least two zones for each NLB instance.
- Separate multiple zones with commas (,). Example: `cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321`.
- You can modify the `service.beta.kubernetes.io/alibaba-cloud-loadbalancer-address-type` parameter to change the type of NLB instance between internal-facing and Internet-facing.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Specify the name of the NLB instance

The name must be 2 to 128 characters in length, and can contain letters, digits, periods (.), underscores (_), and hyphens (-). The name must start with a letter.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-name: "${your-nlb-name}" #The name of the NLB instance. 
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

### Specify the resource group to which the NLB instance belongs

Log on to the [Resource Management console](https://resourcemanager.console.aliyun.com/) to obtain the ID of a resource group. Then, use the annotation in the following template to specify the resource group to which the NLB instance belongs.

**Note** You cannot change the resource group ID after the NLB instance is created.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-resource-group-id:  "${your-resource-group-id}" #The ID of the resource group. 
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

### Create a dual-stack NLB instance

- The kube-proxy mode of the cluster must be set to IPVS.
- IPv6 must be enabled for the vSwitches specified in the `service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps` parameter.
- You cannot change the IP address type after the NLB instance is created.
- The assigned IPv6 address can be used only in an IPv6 network.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Add additional tags to the NLB instance

Separate multiple tags with commas (,). Example: `k1=v1,k2=v2`.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Use an existing NLB instance

The `service.beta.kubernetes.io/alibaba-cloud-loadbalancer-force-override-listeners` annotation specifies whether to modify the listener configurations of the NLB instance based on the configurations of the Service. Valid values:

- true: The CCM creates, updates, or deletes listeners for the NLB instance based on the configurations of the Service.
- false: The CCM does not make any changes to the listeners of the NLB instance.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-id: "${your-nlb-id}" #The ID of the NLB instance. 
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

## Listeners

### Configure a listener to use both TCP and UDP

**Note** Only clusters whose Kubernetes version is 1.24 and later support this feature. For more information about how to update the Kubernetes version of a cluster, see [Update the Kubernetes version of an ACK cluster](https://www.alibabacloud.com/help/en/container-service-for-kubernetes/latest/upgrade-the-kubernetes-version-of-an-ack-cluster#task-1664343).

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Create a TCP listener

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Create a UDP listener

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Create a listener that uses SSL over TCP

Log on to the [Certificate Management Service console](https://yundunnext.console.aliyun.com/), create an SSL certificate, and record the certificate ID. Then, use the following annotation to create a listener that uses SSL over TCP:

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:443"
    # If the cluster is deployed in a region in the Chinese mainland, the SSL certificate ID appended with region information is ${your-cert-id}-cn-hangzhou. 
    # If the cluster is deployed in a region outside the Chinese mainland, the SSL certificate ID appended with region information is ${your-cert-id}-ap-southeast-1. 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${The appended certificate ID}"
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

### Enable mutual authentication

Log on to the [Certificate Management Service console](https://yundunnext.console.aliyun.com/) and record the IDs of the SSL certificate and CA certificate. Then, use the following annotation to enable mutual authentication for the listener:

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:443"   
    # If the cluster is deployed in a region in the Chinese mainland, the SSL certificate ID appended with region information is ${your-cert-id}-cn-hangzhou. 
    # If the cluster is deployed in a region outside the Chinese mainland, the SSL certificate ID appended with region information is ${your-cert-id}-ap-southeast-1. 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${The appended certificate ID}" 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cacert-id: "${your-cacert-id}"  #The CA certificate ID. 
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

### Specify a TLS security policy

Log on to the [Certificate Management Service console](https://yundunnext.console.aliyun.com/) and record the ID of the SSL certificate. Then, use the following annotation to specify a TLS security policy:

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port: "tcpssl:443"
    # If the cluster is deployed in a region in the Chinese mainland, the SSL certificate ID appended with region information is ${your-cert-id}-cn-hangzhou. 
    # If the cluster is deployed in a region outside the Chinese mainland, the SSL certificate ID appended with region information is ${your-cert-id}-ap-southeast-1. 
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id: "${The appended certificate ID}" 
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

### Configure Proxy Protocol

**Important** Before you enable Proxy Protocol, check whether the backend application has Proxy Protocol V2 enabled. If Proxy Protocol V2 is disabled for the backend application, requests cannot be forwarded to the backend application.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Set the maximum number of connections that can be created per second.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Set the timeout period of idle connections.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

## Server groups

### Configure scheduling policies

The scheduling algorithm. Valid values:

- wrr (default): The Weighted Round Robin algorithm is used. Backend servers with higher weights receive more requests than backend servers with lower weights.
- rr: Requests are forwarded to backend servers in sequence.
- sch: Requests from the same source IP address are forwarded to the same backend server.
- tch: Consistent hashing based on the following factors is used: source IP address, destination IP address, source port, and destination port. Requests that contain the same information based on the four factors are forwarded to the same backend server.

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Configure connection draining

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Configure client IP preservation

> not support TCPSSL listener

```yaml
apiVersion: v1
kind: Service
metadata:
  annotations:
    service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

### Configure health checks

- To enable TCP health checks, all annotations in the following template are required. By default, health checks are enabled for TCP ports.

  ```yaml
  apiVersion: v1
  kind: Service
  metadata:
    annotations:
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
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

- Enable HTTP health checks.

  ```yaml
  apiVersion: v1
  kind: Service
  metadata:
    annotations:
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps: "${zone-A}:${vsw-A},${zone-B}:${vsw-B}" #Example: cn-hangzhou-k:vsw-i123456,cn-hangzhou-j:vsw-j654321. 
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-flag: "on"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-type: "http"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-uri: "/test/index.html"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-domain: "www.test.com"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-healthy-threshold: "4"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-unhealthy-threshold: "4"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-connect-timeout: "10"
      service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-interval: "5"
      # Specify the health check method. This annotation is optional. 
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

## Commonly used annotations

### Commonly used NLB annotations

| Annotation                                                   | Type   | Description                                                  | Default value |
| :----------------------------------------------------------- | :----- | :----------------------------------------------------------- | :------------ |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-zone-maps | string | The zones of the NLB instance. You can log on to the [NLB](https://slbnew.console.aliyun.com/nlb/cn-hangzhou/nlbs) console to view the regions and zones that support NLB. Select at least two zones for each NLB instance. | None          |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-address-type | string | Valid values:internet: Internet-facing NLB instanceintranet: internal-facing NLB instance | internet      |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-name   | string | The name of the NLB instance.                                | None          |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-resource-group-id | string | The resource group to which the NLB instance belongs.        | None          |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-ip-version | string | The IP version. Valid values:ipv4: IPv4DualStack: dual stack | ipv4          |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-additional-resource-tags | string | The tags that you want to add to the NLB instance. Separate multiple tags with commas (,). Example: `k1=v1,k2=v2`. | None          |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-id     | string | The ID of the NLB instance.                                  | None          |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-force-override-listeners | string | Specifies whether to modify the listener configurations of the NLB instance based on the configurations of the Service. Valid values:truefalse | false         |

### Commonly used listener annotations

| Annotation                                                   | Type   | Description                                                  | Default value         |
| :----------------------------------------------------------- | :----- | :----------------------------------------------------------- | :-------------------- |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-protocol-port | string | The type of listener. Separate multiple listener types with commas (,). Example: `TCP:80,TCPSSL:443`. | None                  |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cert-id | string | The SSL certificate ID. You can log on to the [Certificate Management Service console](https://yundunnext.console.aliyun.com/) to view SSL certificate IDs. | None                  |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cacert-id | string | The CA certificate ID. You can log on to the [Certificate Management Service console](https://yundunnext.console.aliyun.com/) to view CA certificate IDs. | None                  |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cacert | string | Specifies whether to enable mutual authentication. Valid values:true: enablefalse: disable | None                  |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-tls-cipher-policy | string | The ID of the security policy. System security policies and custom security policies are supported. Valid values:<br />tls_cipher_policy_1_0<br />tls_cipher_policy_1_1<br />tls_cipher_policy_1_2<br />tls_cipher_policy_1_2_strict<br />tls_cipher_policy_1_2_strict_with_1_3 | tls_cipher_policy_1_0 |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-proxy-protocol | string | Specifies whether to enable Proxy Protocol to pass client IP addresses to backend servers. Valid values:true: enablefalse: disable | false                 |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-cps    | string | The maximum number of connections that can be created per second on the NLB instance. Valid values: 0 to 1000000. 0 indicates that the number of connections is unlimited. | None                  |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-idle-timeout | string | The timeout period of idle connections. Unit: seconds. Valid values: 10 to 900. | 900                   |

### Commonly used server group annotations

| Annotation                                                   | Type   | Description                                                  | Default value |
| :----------------------------------------------------------- | :----- | :----------------------------------------------------------- | :------------ |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-scheduler | string | The scheduling algorithm. Valid values:wrr: Backend servers with higher weights receive more requests than backend servers with lower weights.rr: Requests are forwarded to backend servers in sequence.sch: Requests from the same source IP address are forwarded to the same backend server.tch: Consistent hashing based on the following factors is used: source IP address, destination IP address, source port, and destination port. Requests that contain the same information based on the four factors are forwarded to the same backend server. | wrr           |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-connection-drain | string | Specifies whether to enable connection draining. Valid values:true: enablefalse: disable | false         |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-connection-drain-timeout | string | The timeout period of connection draining. Unit: seconds. Valid values: 10 to 900. | None          |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-preserve-client-ip | string | Specifies whether to enable client IP preservation. Valid values:true: enablefalse: disable | false         |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-flag | string | Specifies whether to enable health checks. Valid values:true: enablefalse: disable | true          |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-type | string | The protocol that is used for health checks. Valid values:tcphttp | tcp           |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-connect-port | string | The backend port that is used for health checks.Valid values: 0 to 65535.Default value: 0. This value indicates that the health check port specified on a backend server is used. | 0             |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-health-check-connect-timeout | string | The timeout period of health checks.Unit: seconds. Valid values: 1 to 300. | 5             |
| service.beta.kubernetes.io/alibaba-cloud-loadbalancer-healthy-threshold | string | The number of consecutive successful health checks that must occur before an unhealthy backend server can be declared healthy. Valid values: 2 to 10. | 2             |