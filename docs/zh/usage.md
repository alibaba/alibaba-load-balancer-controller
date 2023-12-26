

# Load Balancer Controller 用户文档(中文版)

[toc]

## ALB概述

## Ingress基本概念

在Kubernetes集群中，Ingress作为集群内服务对外暴露的访问接入点，其几乎承载着集群内服务访问的所有流量。Ingress是Kubernetes中的一个资源对象，用来管理集群外部访问集群内部服务的方式。您可以通过Ingress资源来配置不同的转发规则，从而达到根据不同的规则设置访问集群内不同的Service后端Pod。

## ALB Ingress Controller工作原理

ALB Ingress Controller通过API Server获取Ingress资源的变化，动态地生成AlbConfig，然后依次创建ALB实例、监听、路由转发规则以及后端服务器组。Kubernetes中Service、Ingress与AlbConfig有着以下关系：

- Service是后端真实服务的抽象，一个Service可以代表多个相同的后端服务。
- Ingress是反向代理规则，用来规定HTTP/HTTPS请求应该被转发到哪个Service上。例如：根据请求中不同的Host和URL路径，让请求转发到不同的Service上。
- AlbConfig是在ALB Ingress Controller提供的CRD资源，使用AlbConfig CRD来配置ALB实例和监听。一个AlbConfig对应一个ALB实例。

![ALB Ingress](https://help-static-aliyun-doc.aliyuncs.com/assets/img/zh-CN/2397826461/p339356.png)

## ALB Ingress Controller使用说明

 **注意** 为Ingress服务的ALB是由Controller完全托管的，不能自行在ALB控制台上进行配置，否则可能造成Ingress服务的异常。关于ALB额度的更多信息，请参见[使用限制](https://help.aliyun.com/document_detail/197204.htm#section-5ra-dwn-lwx)。

ALB Ingress基于阿里云应用型负载均衡ALB（Application Load Balancer）之上提供更为强大的Ingress流量管理方式，兼容Nginx Ingress，具备处理复杂业务路由和证书自动发现的能力，支持HTTP、HTTPS和QUIC协议，完全满足在云原生应用场景下对超强弹性和大规模七层流量处理能力的需求。



# ALB Ingress 常用配置

​	前提条件

- 已创建Kubernetes集群并安装了Load Balancer Controller组件，详见[安装文档]()

- 已通过Kubectl工具连接集群，能够执行集群的get、apply等操作

- 通过AlbConfig创建了 Alb实例 [安装文档]()

- 创建了基础的部署集与服务，示例如下：

  ```yaml
  apiVersion: v1
  kind: Service
  metadata:
    name: demo-service
    namespace: default
  spec:
    ports:
      - name: port1
        port: 80
        protocol: TCP
        targetPort: 8080
    selector:
      app: demo
    sessionAffinity: None
    type: ClusterIP
  ---
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: demo
    namespace: default
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: demo
    template:
      metadata:
        labels:
          app: demo
      spec:
        containers:
          - image: registry.cn-hangzhou.aliyuncs.com/alb-sample/cafe:v1
            imagePullPolicy: IfNotPresent
            name: demo
            ports:
              - containerPort: 8080
                protocol: TCP
  ```

  

## 基于域名转发请求

通过以下命令创建一个简单的Ingress，根据指定的正常域名或空域名转发请求。

- 基于正常域名转发请求的示例如下：

  i. 部署以下模板，创建Ingress，将访问请求通过Ingress的域名转发至Service

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: demo
    namespace: default
  spec:
    ingressClassName: alb
    rules:
      - host: demo.domain.ingress.top
        http:
          paths:
            - backend:
                service:
                	name: demo-service
                  port: 
                    number: 80
              path: /hello
              pathType: Prefix
  ```

  ii. 执行以下命令，通过指定的正常域名访问服务。

  替换**ADDRESS**为ALB实例对应的域名地址，可通过`kubectl get ing`获取

  ```shell
  curl -H "host: demo.domain.ingress.top" <ADDRESS>/hello
  ```

  预期输出

  ```
  {"hello":"coffee"}
  ```

- 基于空域名转发请求的示例如下：

  i. 部署以下模板，创建Ingress。

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: demo
    namespace: default
  spec:
    ingressClassName: alb
    rules:
      - host: ""
        http:
          paths:
            - backend:
                service:
                  name: demo-service
                  port: 
                    number: 80
              path: /hello
              pathType: Prefix
  ```

  ii. 执行以下命令，通过空域名访问服务。

  替换**ADDRESS**为ALB实例对应的域名地址，可通过`kubectl get ing`获取。

  ```
  curl <ADDRESS>/hello
  ```

  预期输出：

  ```
  {"hello":"coffee"}
  ```

## 基于URL路径转发请求

ALB Ingress支持按照URL转发请求，可以通过`pathType`字段设置不同的URL匹配策略。`pathType`支持Exact、ImplementationSpecific和Prefix三种匹配方式。

三种匹配方式的示例如下：

- Exact：以区分大小写的方式精确匹配URL路径，不支持正则符号

  i. 部署以下模板，创建Ingress。

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: demo-path
    namespace: default
  spec:
    ingressClassName: alb
    rules:
      - http:
          paths:
          - path: /hello
            backend:
              service:
                name: demo-service
                port: 
                  number: 80
            pathType: Exact
  ```

  ii. 执行以下命令，访问服务。

  替换**ADDRESS**为ALB实例对应的域名地址，可通过`kubectl get ing`获取。

  ```shell
  curl <ADDRESS>/hello
  ```

  预期输出：

  ```
  {"hello":"coffee"}
  ```

- ImplementationSpecific: 默认。以字符形式配置路径，不写正则符号时行为与`Exact` 相同，可以写 `/*`类的通配正则

  i. 部署以下模板，创建Ingress。

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: demo-path
    namespace: default
  spec:
    ingressClassName: alb
    rules:
      - http:
          paths:
          - path: /hello
            backend:
              service:
                name: demo-service
                port:
                  number: 80
            pathType: ImplementationSpecific
  ```

  ii. 执行以下命令，访问服务。

  替换**ADDRESS**为ALB实例对应的域名地址，可通过`kubectl get ing`获取。

  ```
  curl <ADDRESS>/hello
  ```

  预期输出：

  ```
  {"hello":"coffee"}
  ```

- Prefix：以`/`分隔的URL路径进行前缀匹配。匹配区分大小写，并且对路径中的元素逐个完成匹配。

  i. 部署以下模板，创建Ingress。

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    name: demo-path-prefix
    namespace: default
  spec:
    ingressClassName: alb
    rules:
      - http:
          paths:
          - path: /
            backend:
              service:
                name: demo-service
                port:
                  number: 80
            pathType: Prefix
  ```

  ii. 执行以下命令，访问服务。

  替换**ADDRESS**为ALB实例对应的域名地址，可通过`kubectl get ing`获取。

  ```
  curl <ADDRESS>/hello
  ```

  预期输出：

  ```
  {"hello":"coffee"}
  ```



## 配置健康检查

ALB Ingress支持配置`后端服务器组`的健康检查，可以通过设置以下注解实现。

配置健康检查的YAML示例如下所示：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
    alb.ingress.kubernetes.io/healthcheck-enabled: "true"
    alb.ingress.kubernetes.io/healthcheck-path: "/"
    alb.ingress.kubernetes.io/healthcheck-protocol: "HTTP"
    alb.ingress.kubernetes.io/healthcheck-method: "HEAD"
    alb.ingress.kubernetes.io/healthcheck-httpcode: "http_2xx"
    alb.ingress.kubernetes.io/healthcheck-timeout-seconds: "5"
    alb.ingress.kubernetes.io/healthcheck-interval-seconds: "2"
    alb.ingress.kubernetes.io/healthy-threshold-count: "3"
    alb.ingress.kubernetes.io/unhealthy-threshold-count: "3"
spec:
  ingressClassName: alb
  rules:
  - http:
      paths:
      # 配置Context Path
      - path: /tea
        backend:
          service:
            name: tea-svc
            port:
              number: 80
      # 配置Context Path
      - path: /coffee
        backend:
          service:
            name: coffee-svc
            port:
              number: 80
```

相关参数解释如下表所示

| 参数                                                       | 说明                                                         |
| :--------------------------------------------------------- | :----------------------------------------------------------- |
| **alb.ingress.kubernetes.io/healthcheck-enabled**          | （可选）表示是否开启健康检查。默认开启（**true**）。         |
| **alb.ingress.kubernetes.io/healthcheck-path**             | （可选）表示健康检查路径。默认/。输入健康检查页面的URL，建议对静态页面进行检查。长度限制为1~80个字符，支持使用字母、数字和短划线（-）、正斜线（/）、半角句号（.）、百分号（%）、半角问号（?）、井号（#）和and（&）以及扩展字符集_;~!（)*[]@$^:',+。URL必须以正斜线（/）开头。HTTP健康检查默认由负载均衡系统通过后端ECS内网IP地址向该服务器应用配置的默认首页发起HTTP Head请求。如果您用来进行健康检查的页面并不是应用服务器的默认首页，需要指定具体的检查路径。 |
| **alb.ingress.kubernetes.io/healthcheck-protocol**         | （可选）表示健康检查协议。**HTTP**（默认）：通过发送HEAD或GET请求模拟浏览器的访问行为来检查服务器应用是否健康。**TCP**：通过发送SYN握手报文来检测服务器端口是否存活。**GRPC**：通过发送POST或GET请求来检查服务器应用是否健康。 |
| **alb.ingress.kubernetes.io/healthcheck-method**           | （可选）选择一种健康检查方法。**HEAD**（默认）：HTTP监听健康检查默认采用HEAD方法。请确保您的后端服务器支持HEAD请求。如果您的后端应用服务器不支持HEAD方法或HEAD方法被禁用，则可能会出现健康检查失败，此时可以使用GET方法来进行健康检查。**POST**：GRPC监听健康检查默认采用POST方法。请确保您的后端服务器支持POST请求。如果您的后端应用服务器不支持POST方法或POST方法被禁用，则可能会出现健康检查失败，此时可以使用GET方法来进行健康检查。**GET**：如果响应报文长度超过8 KB，会被截断，但不会影响健康检查结果的判定。 |
| **alb.ingress.kubernetes.io/healthcheck-httpcode**         | 设置健康检查正常的状态码。当健康检查协议为**HTTP**协议时，可以选择**http_2xx**（默认）、**http_3xx**、**http_4xx**和**http_5xx**。当健康检查协议为**GRPC**协议时，状态码范围为0~99。支持范围输入，最多支持20个范围值，多个范围值使用半角逗号（,）隔开。 |
| **alb.ingress.kubernetes.io/healthcheck-timeout-seconds**  | 表示接收健康检查的响应需要等待的时间。如果后端ECS在指定的时间内没有正确响应，则判定为健康检查失败。时间范围为1~300秒，默认值为5秒。 |
| **alb.ingress.kubernetes.io/healthcheck-interval-seconds** | 健康检查的时间间隔。取值范围1~50秒，默认为2秒。              |
| **alb.ingress.kubernetes.io/healthy-threshold-count**      | 表示健康检查连续成功所设置的次数后会将后端服务器的健康检查状态由失败判定为成功。取值范围2~10，默认为3次。 |
| **alb.ingress.kubernetes.io/unhealthy-threshold-count**    | 表示健康检查连续失败所设置的次数后会将后端服务器的健康检查状态由成功判定为失败。取值范围2~10，默认为3次。 |

## 配置自动发现HTTPS证书功能

ALB Ingress Controller提供证书自动发现功能。需要在[数字证书管理服务控制台](https://yundunnext.console.aliyun.com/?p=cas)拥有证书，然后ALB Ingress Controller会根据Ingress中TLS配置的域名自动匹配发现证书。测试阶段如果没有购买证书的打算，可以按照下面的步骤，使用自签证书来进行功能测试

1. 执行以下命令，通过`openssl`创建自签证书。注意，此证书默认不会被系统验证通过

   ```shell
   openssl genrsa -out albtop-key.pem 4096
   openssl req -subj "/CN=demo.alb.ingress.top" -sha256  -new -key albtop-key.pem -out albtop.csr
   echo subjectAltName = DNS:demo.alb.ingress.top > extfile.cnf
   openssl x509 -req -days 3650 -sha256 -in albtop.csr -CA ca.pem -CAkey ca-key.pem -CAcreateserial -out albtop-cert.pem -extfile extfile.cnf
   ```

2. 在[数字证书管理服务控制台](https://yundunnext.console.aliyun.com/?p=cas)上传证书。

   具体操作，请参见[上传证书](https://help.aliyun.com/document_detail/98573.htm#concept-g5c-3xn-yfb)。

3. 在Ingress的YAML中添加以下命令，配置该证书对应的域名。

   ```
   tls:
     - hosts:
       - demo.alb.ingress.top
   ```

   示例如下：

   ```yaml
   apiVersion: networking.k8s.io/v1
    kind: Ingress
    metadata:
      name: demo-https
      namespace: default
    spec:
      ingressClassName: alb
      tls:
      - hosts:
        - demo.alb.ingress.top
      rules:
        - host: demo.alb.ingress.top
          http:
            paths:
              - backend:
                  service:
                    name: demo-service
                    port:  
                      number: 80
                path: /
                pathType: Prefix
   ```

4. 执行以下命令，查看证书。

   ```
   curl -v https://demo.alb.ingress.top/
   ```

   在输出控制台上可以看到tls握手证书

   ```
   * Server certificate:
   *  subject: CN=demo.alb.ingress.top
   ```

## 配置HTTP重定向至HTTPS

ALB Ingress通过设置注解`alb.ingress.kubernetes.io/ssl-redirect: "true"`，可以将HTTP请求重定向到HTTPS 443端口。

配置示例如下：

```
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    alb.ingress.kubernetes.io/ssl-redirect: "true"
  name: demo-ssl
  namespace: default
spec:
  ingressClassName: alb
  tls:
  - hosts:
    - demo.alb.ingress.top
  rules:
    - host: demo.alb.ingress.top
      http:
        paths:
          - backend:
              service:
                name: demo-service
                port: 
                  number: 80
            path: /
            pathType: Prefix
```



## 后端服务支持HTTPS和GRPC协议

当前ALB后端协议支持HTTPS和GRPC协议，通过ALB Ingress只需要在注解中配置`alb.ingress.kubernetes.io/backend-protocol: "grpc" `或`alb.ingress.kubernetes.io/backend-protocol: "https" `即可。使用Ingress转发gRPC服务需要对应域名拥有SSL证书，使用TLS协议进行通信。配置GRPC协议的示例如下：

> 后端协议不支持修改，如果需要变更协议，应当删除重建Ingress。

```
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    alb.ingress.kubernetes.io/backend-protocol: "grpc"
  name: lxd-grpc-ingress
spec:
  ingressClassName: alb
  tls:
  - hosts:
    - demo.alb.ingress.top
  rules:
  - host: demo.alb.ingress.top
    http:
      paths:  
      - path: /
        pathType: Prefix
        backend:
          service:
            name: grpc-demo-svc
            port:
              number: 9080
```

## 支持Rewrite重写

当前ALB支持Rewrite重写，通过ALB Ingress只需要在注解中配置`alb.ingress.kubernetes.io/rewrite-target: /path/${2} `即可。

> - Rewrite-target中的 ${number}捕获组功能目前属于高级特性，[提交工单](https://selfservice.console.aliyun.com/ticket/createIndex)申请正则白名单
> - 在`rewrite-target`注解中，`${number}`类型的捕获组变量需要在路径为Prefix类型的`path`上配置
> - `pathType`为Prefix时默认无法配置正则符号，例如`*`、`?`等，需要通过配置`rewrite-target`注解使用正则符号。
> - `path`必须以 `/` 开头，这是Ingress资源的限制

```
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    alb.ingress.kubernetes.io/rewrite-target: /path/${2}
  name: rewrite-ingress
spec:
  ingressClassName: alb
  rules:
  - host: demo.alb.ingress.top
    http:
      paths:
      - path: /something(/|$)(.*)
        pathType: Prefix
        backend:
          service:
            name: demo-service
            port:
              number: 80
```

## 配置自定义监听端口

默认情况，Ingress仅开启80端口，通过配置tls字段，可以开启443端口；配置重定向可以同时监听80+443端口，但是80会返回301重定向；如果有除此以外的端口暴露需求，需要通过自定义端口注解来完成。

示例将服务同时暴露80端口和443端口，配置如下：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
   alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80},{"HTTPS": 443}]'
spec:
  ingressClassName: alb
  tls:
  - hosts:
    - demo.alb.ingress.top
  rules:
  - host: demo.alb.ingress.top
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: demo-service
            port:
              number: 80
```

## 配置转发规则优先级

在ALB的转发规则模型中，规则匹配是严格按照顺序进行的，如配置的匹配条件有交集，就需要通过配置Ingress注解来定义ALB转发规则优先级。

> 匹配条件交集说明，如需配置两个路径，分别为前缀的path1=/api 和 path2=/api/v1 ，如path1的顺序在path2前，那么会导致 /api/v1的流量进入 path1，path2没有收到流量，这种情况就需要手动设置order来配置优先级

> 同一个监听内规则优先级必须唯一。`alb.ingress.kubernetes.io/order`用于标识Ingress之间的优先级顺序，取值范围为1~1000，值越小表示优先级越高。

```
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
   alb.ingress.kubernetes.io/order: "2"
spec:
  ingressClassName: alb
  rules:
  - host: demo.alb.ingress.top
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: demo-service
            port:
              number: 80
```

## 通过注解实现灰度发布

ALB提供复杂路由处理能力，支持基于Header、Cookie以及权重的灰度发布功能。灰度发布功能可以通过设置注解来实现，为了启用灰度发布功能，需要设置注解`alb.ingress.kubernetes.io/canary: "true"`，通过不同注解可以实现不同的灰度发布功能：

> - 灰度优先级顺序：基于Header > 基于Cookie > 基于权重（从高到低）。
>
> - 灰度过程中不能删除原有的规则，否则会导致服务异常。待灰度验证无误后，将原有Ingress中的后端服务Service更新为新的Service，最后将灰度的Ingress删除。

- `alb.ingress.kubernetes.io/canary-by-header`和`alb.ingress.kubernetes.io/canary-by-header-value`：匹配的Request Header的值，该规则允许您自定义Request Header的值，但必须与`alb.ingress.kubernetes.io/canary-by-header`一起使用。

  - 当请求中的`header`和`header-value`与设置的值匹配时，请求流量会被分配到灰度服务入口。
  - 对于其他`header`值，将会忽略`header`，并通过灰度优先级将请求流量分配到其他规则设置的灰度服务。

  当请求Header为`location: hz`时将访问灰度服务；其它Header将根据灰度权重将流量分配给灰度服务。配置示例如下：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  annotations:
    alb.ingress.kubernetes.io/order: "1"
    alb.ingress.kubernetes.io/canary: "true"
    alb.ingress.kubernetes.io/canary-by-header: "location"
    alb.ingress.kubernetes.io/canary-by-header-value: "hz"
  name: demo-canary
  namespace: default
spec:
  ingressClassName: alb
  rules:
    - http:
        paths:
          - backend:
              service:
                name: demo-service-canary
                port: 
                  number: 80
            path: /hello
            pathType: Prefix
```

- `alb.ingress.kubernetes.io/canary-by-cookie`：基于Cookie的流量切分。

  - 当配置的`cookie`值为`always`时，请求流量将被分配到灰度服务入口。
  - 当配置的`cookie`值为`never`时，请求流量将不会分配到灰度服务入口。

  > 基于Cookie的灰度不支持设置自定义，只有`always`和`never`。

  请求的Cookie为`demo=always`时将访问灰度服务。配置示例如下：

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    annotations:
      alb.ingress.kubernetes.io/order: "2"
      alb.ingress.kubernetes.io/canary: "true"
      alb.ingress.kubernetes.io/canary-by-cookie: "demo"
    name: demo-canary-cookie
    namespace: default
  spec:
    ingressClassName: alb
    rules:
      - http:
          paths:
            - backend:
                service:
                  name: demo-service-hello
                  port: 
                    number: 80
              path: /hello
              pathType: Prefix
  ```

- `lb.ingress.kubernetes.io/canary-weight`：设置请求到指定服务的百分比（值为0~100的整数）。

  配置灰度服务的权重为50%，示例如下：

  ```yaml
  apiVersion: networking.k8s.io/v1
  kind: Ingress
  metadata:
    annotations:
      alb.ingress.kubernetes.io/order: "3"
      alb.ingress.kubernetes.io/canary: "true"
      alb.ingress.kubernetes.io/canary-weight: "50"
    name: demo-canary-weight
    namespace: default
  spec:
    ingressClassName: alb
    rules:
      - http:
          paths:
            - backend:
                service:
                  name: demo-service-hello
                  port: 
                    number: 80
              path: /hello
              pathType: Prefix
  ```

## 通过注解实现会话保持

ALB Ingress支持通过注解实现会话保持：

- `alb.ingress.kubernetes.io/sticky-session`：是否启用会话保持。取值：`true`或`false`；默认值：`false`。

- `alb.ingress.kubernetes.io/sticky-session-type`：Cookie的处理方式。取值：`Insert`或`Server`；默认值：`Insert`。

  - `Insert`：植入Cookie。客户端第一次访问时，负载均衡会在返回请求中植入Cookie（即在HTTP或HTTPS响应报文中插入SERVERID），下次客户端携带此Cookie访问时，负载均衡服务会将请求定向转发给之前记录到的后端服务器。
  - `Server`：重写Cookie。负载均衡发现用户自定义了Cookie，将会对原来的Cookie进行重写，下次客户端携带新的Cookie访问时，负载均衡服务会将请求定向转发给之前记录到的后端服务器。

  > 当前服务器组`StickySessionEnabled`为`true`时，该参数生效。

- `alb.ingress.kubernetes.io/cookie-timeout`：Cookie超时时间。单位：秒；取值：1~86400；默认值：`1000`。

> 注意，会话保持功能是针对挂载到后端服务器组中的节点生效，在使用NodePort类型Service进行挂载时，因为ALB网络和Pod网络不在一个平面，所以不能针对Pod进行会话保持，但是在设置 `Service的externalTrafficPolicy: local`情况下，Node和Pod是一对一映射的，可以实现会话保持，这只适用于同一个Node不会挂载多个Pod的场景。想要对Pod使用完全的会话保持能力，需要为Pod分配ENI来打通ALB网络与Pod网络。

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress-v3
  annotations:
    alb.ingress.kubernetes.io/sticky-session: "true"
    alb.ingress.kubernetes.io/sticky-session-type: "Insert"
    alb.ingress.kubernetes.io/cookie-timeout: "1800"
spec:
  ingressClassName: alb
  rules:
  - http:
      paths:
      #配置Context Path。
      - path: /tea2
        backend:
          service:
            name: tea-svc
            port: 
             number: 80
      #配置Context Path。
       - path: /coffee2
         backend:
           service:
              name: coffee-svc
              port: 
               number: 80
```

## 指定服务器组负载均衡算法

ALB Ingress支持通过设置Ingress注解`alb.ingress.kubernetes.io/backend-scheduler`指定服务器组负载均衡算法。配置示例如下：

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: cafe-ingress
  annotations:
   alb.ingress.kubernetes.io/backend-scheduler: "wlc"
spec:
  ingressClassName: alb
  rules:
  - host: demo.alb.ingress.top
    http:
      paths:
      - path: /tea
        pathType: ImplementationSpecific
        backend:
          service:
            name: tea-svc
            port:
              number: 80
```

# AlbConfig Common Configuration

一个AlbConfig对应一个ALB实例，如果一个ALB实例配置多个转发规则，那么一个AlbConfig则对应多个Ingress，所以AlbConfig与Ingress是一对多的对应关系。

## 创建AlbConfig

一个AlbConfig对应一个ALB实例，如果您需要使用多个ALB实例，可以通过创建多个AlbConfig实现。创建AlbConfig操作如下：

1. 应用配置示例，用于创建AlbConfig

   ```yaml
   apiVersion: alibabacloud.com/v1
   kind: AlbConfig
   metadata:
     name: alb-demo
   spec:
     config:
       name: alb-test
       addressType: Internet
       zoneMappings:
       - vSwitchId: vsw-uf6ccg2a9g71hx8go****
       - vSwitchId: vsw-uf6nun9tql5t8nh15****
   ```

   | 参数                         | 说明                                                         |
   | :--------------------------- | :----------------------------------------------------------- |
   | **spec.config.name**         | （可选）表示ALB实例的名称。                                  |
   | **spec.config.addressType**  | （必选）表示负载均衡的地址类型。取值如下：Internet（默认值）：负载均衡具有公网IP地址，DNS域名被解析到公网IP，因此可以在公网环境访问。Intranet：负载均衡只有私网IP地址，DNS域名被解析到私网IP，因此只能被负载均衡所在VPC的内网环境访问。 |
   | **spec.config.zoneMappings** | （必选）用于设置ALB Ingress交换机ID，您需要至少指定两个不同可用区交换机ID，指定的交换机必须在ALB当前所支持的可用区内。关于ALB Ingress支持的地域与可用区，请参见[支持的地域与可用区](https://help.aliyun.com/document_detail/258300.htm#task-2087008)。 |

2. 应用Albconfig到集群

   ```
   kubectl apply -f alb-test.yaml
   ```

   预期输出

   ```
   AlbConfig.alibabacloud.com/alb-demo created
   ```

3. 执行以下命令，确认AlbConfig创建成功

   ```
   kubectl -n kube-system get AlbConfig
   ```

   预期输出

   ```
   NAME       AGE
   alb-demo   87m
   ```

## 关联Ingress

AlbConfig通过K8s中标准的IngressClass资源与Ingress进行关联。您需要先创建IngressClass，然后关联AlbConfig。

1. 配置示例文件，用于创建IngressClass

   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: IngressClass
   metadata:
     name: alb
   spec:
     controller: ingress.k8s.alibabacloud/alb
     parameters:
       apiGroup: alibabacloud.com
       kind: AlbConfig
       name:alb-demo
   ```

2. 执行以下命令，创建IngressClass。

   ```
   kubectl apply -f alb.yaml
   ```

   预期输出

   ```
   ingressclass.networking.k8s.io/alb created
   ```

3. 在Ingress的YAML中通过`ingressClassName`参数指定名称为alb的IngressClass，关联AlbConfig。

   ```
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: cafe-ingress 
   spec:
     ingressClassName: alb
     rules:
     - http:
         paths:
         # 配置Context Path。
         - path: /tea
           pathType: ImplementationSpecific
           backend:
             service:
               name: tea-svc
               port:
                 number: 80
         # 配置Context Path。
         - path: /coffee
           pathType: ImplementationSpecific
           backend:
             service:
               name: coffee-svc
               port: 
                 number: 80
   ```

## 修改AlbConfig的名称

如果您需要修改AlbConfig的名称，可以执行以下命令。保存之后，新名称自动生效。

```
kubectl -n kube-system edit AlbConfig alb-demo
...
  spec:
    config:
      name: test   #输入修改后的名称。
...
```

## 修改AlbConfig的vSwitch配置

如果您需要修改AlbConfig的vSwitch配置。保存之后，新配置自动生效。

```
kubectl -n kube-system edit AlbConfig alb-demo
...
  zoneMappings:
    - vSwitchId: vsw-wz92lvykqj1siwvif****
    - vSwitchId: vsw-wz9mnucx78c7i6iog****
...
```

## 指定HTTPS证书

通过ALBConfig可以为监听使用证书，通过配置ALBConfig的`listeners`，指定HTTPS的证书ID。具体操作步骤如下：

1. 登录[数字证书管理服务控制台](https://yundunnext.console.aliyun.com/?p=cas)。

2. 在数字证书管理服务控制台左侧导航栏，单击**SSL证书**。

3. 在**SSL证书**页面，单击**上传证书**页签。

4. 在目标证书右侧**操作**列下，选择***\*![更多](https://help-static-aliyun-doc.aliyuncs.com/assets/img/zh-CN/6390999461/p430273.png)\** > \**详情\****，获取证书ID。

   证书详情示例如下：

   ![证书详情示例](https://help-static-aliyun-doc.aliyuncs.com/assets/img/zh-CN/6390999461/p430304.jpg)

   5. 编辑并保存AlbConfig YAML文件。

      ```yaml
      apiVersion: alibabacloud.com/v1
      kind: AlbConfig
      metadata:
        name: alb-demo
      spec:
        config:
          #...
        listeners:
        - caEnabled: false
          certificates:
          - CertificateId: 756****-cn-hangzhou
            IsDefault: true
          port: 443
          protocol: HTTPS
        #...
      ```

      AlbConfig部分参数说明如下：

      | 参数            | 说明                                                         |
      | :-------------- | :----------------------------------------------------------- |
      | `CertificateId` | 表示证书ID。本文配置为`756****-cn-hangzhou`，`756****`为上一步获取的证书ID。`CertificateId`格式示例及说明如下：中国地域：`756****-cn-hangzhou`。`-cn-hangzhou`为固定内容，配置时您只需替换`756****`即可。海外地域：`756****-ap-southeast-1`。`-ap-southeast-1`为固定内容，配置时您只需替换`756****`即可。 |
      | `IsDefault`     | 表示是否为默认证书。本文配置为`true`，表示是默认证书。       |
      | `protocol`      | 表示支持监听的协议类型。本文配置为` HTTPS`，表示支持HTTPS协议的监听。 |

## 支持TLS安全策略

当前ALBConfig配置HTTPS监听时，支持指定TLS安全策略。TLS安全策略包含自定义策略和系统默认策略，更多信息，请参见[TLS安全策略](https://help.aliyun.com/document_detail/198572.htm#task-2020940)。

```yaml
apiVersion: alibabacloud.com/v1
kind: AlbConfig
metadata:
  name: alb-demo
spec:
  config:
    #...
  listeners:
  - port: 443
    protocol: HTTPS
    securityPolicyId: tls_cipher_policy_1_1
  #...
```

## 开启日志服务访问日志

如果您希望ALB Ingress能够收集访问日志Access Log，则只需要在AlbConfig中指定`logProject`和`logStore`。

> 当前Log Project需要您手动创建，不支持自动创建。创建Log Project的具体操作，请参见[管理Project](https://help.aliyun.com/document_detail/48984.htm#concept-mxk-414-vdb)。

```yaml
apiVersion: alibabacloud.com/v1
kind: AlbConfig
metadata:
  name: alb-demo
spec:
  config:
    accessLogConfig:
      logProject: "k8s-log-xz92lvykqj1siwvif****"
      logStore: "alb_****"
    #...
```

> logStore命名需要以`alb_`开头，若指定logStore不存在，系统则会自动创建。(托管版限制，开源版无此限制)

保存命令之后，可以在[日志服务控制台](https://sls.console.aliyun.com/)，单击目标LogStore，查看收集的访问日志。

![ALB Ingress日志.png](https://help-static-aliyun-doc.aliyuncs.com/assets/img/zh-CN/6575438361/p361420.png)

## 复用已有ALB实例

如果您希望复用已有ALB实例，只需要创建AlbConfig时指定ALB实例ID即可。

```
apiVersion: alibabacloud.com/v1
kind: AlbConfig
metadata:
  name: reuse-alb
spec:
  config:
    id: ****
    forceOverride: false   # 表示是否覆盖已有监听: true表示强制覆盖，false表示不覆盖已有监听，但是会管理与已有监听不冲突的监听，如已有80监听，可以通过ingress来管理90等其他端口。
```

## 使用多个ALB实例

同一个ingress不能使用多个alb实例，但是同一个集群可以使用多个ALB实例，在Ingress中通过`spec.ingressClassName`指定不同的IngressClass即可。

1. 创建并拷贝以下内容到alb-demo2.yaml文件中，用于创建AlbConfig。

   ```yaml
   apiVersion: alibabacloud.com/v1
   kind: AlbConfig
   metadata:
     name: demo
   spec:
     config:
       name: alb-demo2        #ALB实例名称。
       addressType: Internet  #负载均衡具有公网IP地址。
       zoneMappings:
       - vSwitchId: vsw-uf6ccg2a9g71hx8go****
       - vSwitchId: vsw-uf6nun9tql5t8nh15****
   ```

2. 执行以下命令，创建AlbConfig。

   ```
   kubectl apply -f alb-demo2.yaml
   ```

   预期输出：

   ```
   AlbConfig.alibabacloud.com/demo created
   ```

3. 创建并拷贝以下内容到alb.yaml文件中，用于创建IngressClass。

   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: IngressClass
   metadata:
     name: alb-demo2
   spec:
     controller: ingress.k8s.alibabacloud/alb
     parameters:
       apiGroup: alibabacloud.com
       kind: AlbConfig
       name: demo
   ```

4. 执行以下命令，创建IngressClass。

   ```
   kubectl apply -f alb.yaml
   ```

   预期输出：

   ```
   ingressclass.networking.k8s.io/alb-demo2 created
   ```

5. 在Ingress的YAML中通过`ingressClassName`指定不同的ALB实例。

   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: demo
     namespace: default
   spec:
     ingressClassName: alb-demo2
     rules:
       - host: ""
         http:
           paths:
             - backend:
                service:
                 name: demo-service
                 port:
                   number: 80
               path: /hello
               pathType: Prefix
   ```

## 删除ALB实例

一个ALB实例对应一个AlbConfig， 因此可以通过删除AlbConfig实现删除ALB实例，但前提是需要**先删除AlbConfig关联的所有Ingress**。

```
kubectl -n kube-system delete AlbConfig alb-demo
```

`alb-demo`可以替换为您实际需要删除的AlbConfig。

# ALB Ingress配置词典
为便于您发现和解决各类配置的格式问题，本文提供一份ALB Ingress的全局配置词典供您使用。全局配置词典包括ALB Ingress支持的Annotation和ALB Ingress的AlbConfig字段两部分内容。

## ALB Ingress支持的Annotation
请根据需求将注解（Annotation）添加到ALB Ingress资源上，以配置与ALB相关的属性。

### 健康检查
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/healthcheck-enabled`   | 是否开启后端服务器组健康检查   | `"true"` or `"false"`                                          | `"false"`|
| `alb.ingress.kubernetes.io/healthcheck-path`      | 健康检查路径                   | string                                                         | `"/"`    |
| `alb.ingress.kubernetes.io/healthcheck-protocol`  | 健康检查协议                   | `"HTTP"`, `"TCP"`                               | `"HTTP"` |
| `alb.ingress.kubernetes.io/healthcheck-method`    | 健康检查方法                   | `"HEAD"`, `"POST"`, `"GET"`                                   | `"HEAD"` |
| `alb.ingress.kubernetes.io/healthcheck-httpcode`  | 健康检查状态码                 | `"http_2xx"`, `"http_3xx"`, `"http_4xx"`, `"http_5xx"`| `"http_2xx"` |
| `alb.ingress.kubernetes.io/healthcheck-timeout-seconds` | 健康检查超时时间，单位秒       | `1~300`                                                        | `5`       |
| `alb.ingress.kubernetes.io/healthcheck-interval-seconds` | 健康检查周期                   | `1~50`                                                         | `2`       |
| `alb.ingress.kubernetes.io/healthy-threshold-count`      | 健康检查成功多少次判定为成功   | `2~10`                                                         | `3`       |
| `alb.ingress.kubernetes.io/unhealthy-threshold-count`    | 健康检查失败多少次判定为失败   | `2~10`                                                         | `3`       |
| `alb.ingress.kubernetes.io/healthcheck-connect-port`     | 健康检查端口，`0`表示使用后端服务器的端口进行健康检查                    | `0~65535`                                                      | `0` |

### 重定向
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/ssl-redirect`      | 是否将HTTP请求重定向到HTTPS| `"true"` or `"false"` | `"false"`|

### 后端服务使用的协议
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/backend-protocol` | 后端服务器组协议   | `"http"`, `"https"`, `"grpc"` | `"http"`|

### 重写
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/rewrite-target`          | 路径重写的地址                   | string                                | 无                          |

### 监听
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/listen-ports`            | 关联监听端口与协议               | jsonObject                            | `'[{"HTTP": 80}]'` 或 `'[{"HTTPS": 443}]'` 或 `'[{"HTTP": 80},{"HTTPS": 443}]'` |

### 优先级
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/order`                   | 转发规则的相对优先级             | `1~1000`                              | `10`                           |

### 灰度
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/canary`                  | 是否启用canary灰度               | `"true"` or `"false"`                 | `"false"`                      |
| `alb.ingress.kubernetes.io/canary-by-header`        | 启用灰度时命中的请求标头         | string                                | 无                            |
| `alb.ingress.kubernetes.io/canary-by-header-value`  | 启用灰度时命中的请求标头对应的标头值 | string                            | 无                            |
| `alb.ingress.kubernetes.io/canary-by-cookie`        | 启用灰度时的cookie标记           | string                                | 无                        |

### 会话保持
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/sticky-session`          | 是否开启后端服务器组会话保持     | `"true"` or `"false"`         | `"false"`|
| `alb.ingress.kubernetes.io/sticky-session-type`     | 开启会话保持的类型               | `"Insert"` or `"Server"`      | `"Insert"`|
| `alb.ingress.kubernetes.io/cookie-timeout`          | 会话保持超时时间，单位秒         | `1~86400`                     | `1000`   |

### 负载均衡
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/backend-scheduler`              | 后端服务器组负载均衡算法                   | `"wrr"`, `"wlc"`, `"sch"`, `"uch"`          | `"wrr"` |
| `alb.ingress.kubernetes.io/backend-scheduler-uch-value`    | 负载均衡算法为uch时的辅助参数               | string                                          | 无     |

### 跨域
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/enable-cors`                | 是否启用跨域配置                      | `"true"` or `"false"`                                                                           | `"false"`                                                                                              |
| `alb.ingress.kubernetes.io/cors-allow-origin`          | 允许跨域的源                          | string                                                                                          | `"*"`                                                                                                  |
| `alb.ingress.kubernetes.io/cors-expose-headers`        | 允许暴露的header列表                  | stringArray                                                                                     | 无                                                                                                    |
| `alb.ingress.kubernetes.io/cors-allow-methods`         | 允许跨域的请求方法                    |  以下选项选择一项或多项：`["GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS", "PATCH"]`                  | `"GET, PUT, POST, DELETE, PATCH, OPTIONS"`                                                             |
| `alb.ingress.kubernetes.io/cors-allow-credentials`     | 跨域是否允许携带凭证信息              | `"true"` or `"false"`                                                                           | `"true"`                                                                                               |
| `alb.ingress.kubernetes.io/cors-max-age`               | 预检请求在浏览器最大的缓存时间        | `-1~172800`                                                                                     | `172800`                                                                                               |
| `alb.ingress.kubernetes.io/cors-allow-headers`         | 允许跨域的header列表                  | stringArray                                                                                     | `"DNT,X-CustomHeader,Keep-Alive,User-Agent,X-Requested-With,If-Modified-Since,Cache-Control,Content-Type,Authorization"` |

### 自定义转发
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/actions.{svcName}`          | 自定义转发动作                        | json                                                                                            | 无                                                                                                    |
| `alb.ingress.kubernetes.io/conditions.{svcName}`       | 自定义转发条件                        | json                                                                                            | 无                                                                                                   |
| `alb.ingress.kubernetes.io/rule-direction.{svcName}`   | 自定义转发方向                        | `"Request"` or `"Response"`                                                                      | `"Request"`                                                                                            |

### 其他
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `alb.ingress.kubernetes.io/backend-keepalive`          | 是否启用后端长链接                    | `"true"` or `"false"`                                                                           | `"false"`                                                                                              |
| `alb.ingress.kubernetes.io/traffic-limit-qps`          | QPS限速配置                           | `1~100000`                                                                                      | 无                                                                                                    |
| `alb.ingress.kubernetes.io/use-regex`                  | 允许Path字段使用正则，仅在Prefix类型下生效 | `"true"` or `"false"`                                                                           | `"false"`                                                                                              |

## ALB Ingress的AlbConfig字段
AlbConfig是用来描述ALB实例及监听的自定义资源CRD，关于字段的详细描述，请参见下文。

### Albconfig
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `apiVersion` | APIVersion 定义了该对象的版本化模式。                | `"alibabacloud.com/v1"`                                         |                     无                            |
| `kind`      | Kind表示该对象所代表的 REST 资源。                     | `"AlbConfig"`                                                   |                           无                        |
| `metadata`  | 标准对象的 metadata。关于metadata的更多信息，请参见[metadata](https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#metadata)。        | [ObjectMeta](https://kubernetes.io/docs/reference/generated/kubernetes-api/v1.23/#objectmeta-v1-meta)                                                    |    无   |
| `spec`      | 用来描述alb实例属性和监听属性的参数列表。              | [AlbConfigSpec](#AlbConfigSpec)                                                 |                               无                    |
| `status`    | 在调和成功后，会将实例状态写入status中，表示实例当前状态。 | [AlbConfigStatus](#AlbConfigStatus)                                                     |                    无                               |

### AlbConfigSpec
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `config`   | ALB实例属性    | [LoadBalancerSpec](#LoadBalancerSpec) | 无    |
| `listeners`| 实例下的监听属性 | [[]ListenerSpec](#ListenerSpec)  | 无    |

### LoadBalancerSpec 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `id`                        | ALB实例ID，填写代表启用复用模式     | `string`                           | `""`                                |
| `name`                      | ALB实例名                         | `string`                           | `k8s-{namespace}-{name}-{hashCode}` |
| `addressAllocatedMode`      | 实例的地址模式                     | `"Dynamic"` or `"Fixed"`           | `"Dynamic"`                         |
| `addressType`               | ALB的IPv4网段地址类型             | `"Internet"` or `"Intranet"`       | `"Internet"`                        |
| `ipv6AddressType`           | ALB的IPv6网段地址类型             | `"Internet"` or `"Intranet"`       | `"Intranet"`                        |
| `addressIpVersion`          | 协议版本                          | `"IPv4"` or `"DualStack"`          | `"IPv4"`                            |
| `resourceGroupId`           | 实例所属的资源组id                | string                           | 默认资源组                          |
| `edition`                   | 实例功能版本                      | `"Standard"` or `"StandardWithWaf"`| `"Standard"`                        |
| `deletionProtectionEnabled` | 保留字段，目前不可调整，强制保留   | `*bool`                            | `null`                              |
| `forceOverride`             | 复用模式下强制覆盖实例属性         | `*bool`                            | `false`                             |
| `listenerForceOverride`     | 复用模式下强制覆盖监听属性         | `*bool`                            | `null`                              |
| `zoneMappings`              | 可用区和EIP配置                    | [[]ZoneMapping](#ZoneMapping)                  |               无                      |
| `accessLogConfig`           | 日志收集                          | [AccessLogConfig](#AccessLogConfig)                  |                无                     |
| `billingConfig`             | 计费方式                          | [BillingConfig](#BillingConfig)                   |                     无                |
| `modificationProtectionConfig`| 配置修改保护                    | [ModificationProtectionConfig](#ModificationProtectionConfig)     |                无                     |
| `tags`                      | 实例标签                          | [[]Tag](#Tag)                          |                    无                 |

### ZoneMapping 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `vSwitchId`   | 必填<br>虚拟交换机的id             | `string`  | `""`   |
| `zoneId`      | 自动装填<br>虚拟交换机的可用区         | `string`  | `""`   | 
| `allocationId`| 弹性公网EIP的ID           | `string`  | `""`   |
| `eipType`     | 保留字段                   | `string`  | `""`   |

### AccessLogConfig
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `logStore`  | SLS日志库的名称    | string | `""`   |
| `logProject`| SLS日志项目的名称  | string | `""`   |

### BillingConfig 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `internetBandwidth`  | 保留字段            | int    | `0`        |
| `internetChargeType` | 保留字段            | string | `""`       |
| `payType`            | 计费方式            | string | `"PostPay"`|
| `bandWidthPackageId` | 绑定共享带宽包ID，绑定后不支持解绑    | string | `""`       |

### ModificationProtectionConfig 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `reason` | 保留字段      | `string` | `""`          |
| `status` | 保留字段      | `string` | `""`          |

### Tag 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `key` | 标签key     | string | `""`          |
| `value` | 标签value | string | `""`          |


### ListenerSpec 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `gzipEnabled`      | 是否开启压缩        | `null` or `true` or `false` | `null`                                  |
| `http2Enabled`     | 是否开启HTTP2协议   | `null` or `true` or `false` | `null`                                  |
| `port`             | 必填<br>监听端口            | Int Or String         | `0`                                         |
| `protocol`         | 必填<br>监听协议            | `"HTTP"` or `"HTTPS"` or `"QUIC"` | `""`                              |
| `securityPolicyId` | TLS安全策略的ID       | string              | `""`                                         |
| `idleTimeout`      | 空闲链接超时时间<br>取值为0，表示使用默认的空闲超时值    | `int`                 | `60`                                        |
| `loadBalancerId`   | 保留字段            | string              | `""`                                         |
| `description`      | 监听名              | string              | `ingress-auto-listener-{port}`              |
| `caEnabled`        | 保留字段            | bool                | `false`                                      |
| `requestTimeout`   | 请求超时时间        | int                | `60`                                         |
| `quicConfig`       | Quic监听配置        | [QuicConfig](#QuicConfig)          |                                               |
| `defaultActions`   | 保留字段            | `[]Action`            | `null`                                       |
| `caCertificates`   | 保留字段            | [Certificate](#Certificate)         | `null`                                       |
| `certificates`     | 监听服务器证书      |[Certificate](#Certificate)           | `null`                                       |
| `xForwardedForConfig` | XForward字段配置信息 | [XForwardedForConfig](#XForwardedForConfig ) |              无                               |
| `logConfig`        | 保留字段            | `LogConfig`           |                     无                          |
| `aclConfig`        | 访问控制            | [AclConfig](#)           |                    无                           |



### QuicConfig 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `quicUpgradeEnabled`| 是否开启Quic 升级  | bool   | `false`       |
| `quicListenerId`    | Quic 的关联监听    | string | `""`          |

### Certificate
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `IsDefault`     | 指示证书是否为默认证书 <br> 一个服务或系统只能指示一个证书为默认证书      | bool   | `false`       |
| `CertificateId` | 证书CertIdentifier的ID| string | `""`          |

### XForwardedForConfig 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `XForwardedForClientCertSubjectDNAlias`         | 自定义头名称，只有当`XForwardedForClientCertSubjectDNEnabled`的值为`true`时，此值才会生效                     | string   | `""`          |
| `XForwardedForClientCertSubjectDNEnabled`       | 是否通过`X-Forwarded-Clientcert-subjectdn`头字段获取访问负载均衡实例客户端证书的所有者信息                 | bool     | `false`       |
| `XForwardedForProtoEnabled`                     | 是否通过`X-Forwarded-Proto`头字段获取负载均衡实例的监听协议                                                | bool     | `false`       |
| `XForwardedForClientCertIssuerDNEnabled`        | 是否通过`X-Forwarded-Clientcert-issuerdn`头字段获取访问负载均衡实例客户端证书的发行者信息                | bool     | `false`       |
| `XForwardedForSLBIdEnabled`                     | 是否通过`X-Forwarded-For-SLB-ID`头字段获取负载均衡实例ID                                                                     | bool     | `false`       |
| `XForwardedForClientSrcPortEnabled`             | 是否通过`X-Forwarded-Client-Port`头字段获取访问负载均衡实例客户端的端口                                       | bool     | `false`       |
| `XForwardedForClientCertFingerprintEnabled`     | 是否通过`X-Forwarded-Clientcert-fingerprint`头字段获取访问负载均衡实例客户端证书的指纹取值                   | bool     | `false`       |
| `XForwardedForEnabled`                          | 是否通过`X-Forwarded-For`头字段获取来访者真实IP                                                         | bool     | `false`       |
| `XForwardedForSLBPortEnabled`                   | 是否通过`X-Forwarded-Port`头字段获取负载均衡实例的监听端口                                                   | bool     | `false`       |
| `XForwardedForClientCertClientVerifyAlias`      | 自定义头名称，只有当`XForwardedForClientCertClientVerifyEnabled`的值为`true`的时候，该值才会生效；否则该值不会生效 | string   | `""`          |
| `XForwardedForClientCertIssuerDNAlias`          | 自定义头名称，只有当`XForwardedForClientCertIssuerDNEnabled`的值为`true`的时候，此值才会生效                     | string   | `""`          |
| `XForwardedForClientCertFingerprintAlias`       | 自定义头名称，只有当`XForwardedForClientCertFingerprintEnabled`的值为`true`时生效                               | string   | `""`          |
| `XForwardedForClientCertClientVerifyEnabled`    | 是否通过`X-Forwarded-Clientcert-clientverify`头字段获取对访问负载均衡实例客户端证书的校验结果                 | bool     | `false`       |


### AclConfig 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `aclName`   | AclEntry模式下关联的ACL策略名  | string    |     无          |
| `aclType`   | 策略类型，黑白名单             | `""` or `Black` or `White` | `""`  |
| `aclEntries`| 直接写访问策略条目的           | []string  | `null`          |
| `aclIds`    | 关联已经存在的策略ID           | []string  | `null`          |

### AlbConfigStatus
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `loadBalancer` | 负载均衡状态实例      | [LoadBalancerStatus](#LoadBalancerStatus) |       无        |

### LoadBalancerStatus 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `dnsname`   | ALB实例DNS地址    | string           |        无        |
| `id`        | ALB实例ID         | string           |      无          |
| `listeners` | ALB监听属性       | [[]ListenerStatus](#ListenerStatus) |         无       |


### ListenerStatus 
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `portAndProtocol` | 监听与协议   | string              | `80/HTTP`     |
| `certificates`    | 关联证书     | [[]AppliedCertificate](#AppliedCertificate) |      无         |

### AppliedCertificate
|**注解项（Annotation）**|**说明**|**取值**|**默认值**|
| :------------ | :------------ | :------------ | :------------ |
| `certificateId` | 证书的标识符（certIdentifier） | string  | `xxxx-cn-hangzhou`  |
| `isDefault`     | 是否为默认证书      | bool    | `true`              |
