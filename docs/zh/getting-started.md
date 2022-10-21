# 通过ALB Ingress访问服务

ALB Ingress基于阿里云应用型负载均衡ALB（Application Load Balancer） 实现的Ingress服务，适用于有明显波峰波谷的业务场景。

## 背景信息

Ingress是允许访问集群内Service的规则集合，您可以通过配置转发规则，实现不同URL访问集群内不同的Service。但传统的Nginx Ingress或者四层SLB Ingress，已无法满足云原生应用服务对复杂业务路由、多种应用层协议（例如：QUIC等）、大规模七层流量能力的需求。

## 前提条件

- 已通过kubectl工具连接集群。
- 安装ALB Ingress Controller组件。关于如何安装ALB Ingress Controller组件，请参见[编译部署]()
## 注意事项

- Kubernetes版本1.18及以上版本。
- 如果您使用的是Flannel网络插件，则ALB Ingress后端Service服务仅支持NodePort和LoadBalance类型。

## 步骤一：创建AlbConfig

1. 创建并拷贝以下内容到alb-test.yaml文件中，用于创建AlbConfig。

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
   | **spec.config.name**         | （可选）表示Alb实例的名称。                                  |
   | **spec.config.addressType**  | （必选）表示负载均衡的地址类型。取值如下：Internet（默认值）：负载均衡具有公网IP地址，DNS域名被解析到公网IP，因此可以在公网环境访问。Intranet：负载均衡只有私网IP地址，DNS域名被解析到私网IP，因此只能被负载均衡所在VPC的内网环境访问。 |
   | **spec.config.zoneMappings** | （必选）用于设置ALB Ingress交换机ID，您需要至少指定两个不同可用区交换机ID，指定的交换机必须在ALB当前所支持的可用区内。关于ALB Ingress支持的地域与可用区，请参见[支持的地域与可用区](https://help.aliyun.com/document_detail/258300.htm#task-2087008)。 |

2. 执行以下命令，创建AlbConfig。

   ```
   kubectl apply -f alb-test.yaml
   ```

   预期输出：

   ```
   AlbConfig.alibabacloud.com/alb-demo created
   ```

3. 创建并拷贝以下内容到alb.yaml文件中，用于创建IngressClass。

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
       name: alb-demo
   ```

4. 执行以下命令，创建IngressClass。

   ```
   kubectl apply -f alb.yaml
   ```

   预期输出：

   ```
   ingressclass.networking.k8s.io/alb created
   ```

## 步骤二：部署服务

1. 创建并拷贝以下内容到cafe-service.yaml文件中，用于部署两个名称分别为`coffee`和`tea`的Deployment，以及两个名称分别为`coffee`和`tea`的Service。

   ```yaml
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: coffee
   spec:
     replicas: 2
     selector:
       matchLabels:
         app: coffee
     template:
       metadata:
         labels:
           app: coffee
       spec:
         containers:
         - name: coffee
           image: registry.cn-hangzhou.aliyuncs.com/acs-sample/nginxdemos:latest
           ports:
           - containerPort: 80
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: coffee-svc
   spec:
     ports:
     - port: 80
       targetPort: 80
       protocol: TCP
     selector:
       app: coffee
     type: NodePort
   ---
   apiVersion: apps/v1
   kind: Deployment
   metadata:
     name: tea
   spec:
     replicas: 1
     selector:
       matchLabels:
         app: tea
     template:
       metadata:
         labels:
           app: tea
       spec:
         containers:
         - name: tea
           image: registry.cn-hangzhou.aliyuncs.com/acs-sample/nginxdemos:latest
           ports:
           - containerPort: 80
   ---
   apiVersion: v1
   kind: Service
   metadata:
     name: tea-svc
   spec:
     ports:
     - port: 80
       targetPort: 80
       protocol: TCP
     selector:
       app: tea
     type: NodePort
   ```

2. 执行以下命令，部署两个Deployment和两个Service。

   ```
   kubectl apply -f cafe-service.yaml
   ```

   预期输出

   ```
   deployment "coffee" created
   service "coffee-svc" created
   deployment "tea" created
   service "tea-svc" created
   ```

3. 执行以下命令，查看服务状态。

   ```
   kubectl get svc,deploy
   ```

   预期输出：

   ```
   NAME             TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
   coffee-svc   NodePort    172.16.231.169   <none>        80:31124/TCP   6s
   tea-svc      NodePort    172.16.38.182    <none>        80:32174/TCP   5s
   NAME            DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
   deploy/coffee   2         2         2            2           1m
   deploy/tea      1         1         1            1           1m
   ```

## 步骤三：配置Ingress

1. 创建并拷贝以下内容到cafe-ingress.yaml文件中。

   ```yaml
   apiVersion: networking.k8s.io/v1
   kind: Ingress
   metadata:
     name: cafe-ingress 
   spec:
     ingressClassName: alb
     rules:
      - host: demo.domain.ingress.top
        http:
         paths:
         # 配置Context Path
         - path: /tea
           pathType: Prefix
           backend:
             service:
               name: tea-svc
               port:
                 number: 80
         # 配置Context Path
         - path: /coffee
           pathType: Prefix
           backend:
             service:
               name: coffee-svc
               port: 
                 number: 80
   ```

2. 执行以下命令，配置`coffee`和`tea`服务对外暴露的域名和`path`路径。

   ```
   kubectl apply -f cafe-ingress.yaml
   ```

   预期输出：

   ```
   ingress "cafe-ingress" created
   ```

3. 执行以下命令获取ALB实例IP地址。

   ```
   kubectl get ing
   ```

   预期输出：

   ```
   NAME           CLASS    HOSTS                         ADDRESS                                               PORTS   AGE
   cafe-ingress   alb      demo.domain.ingress.top       alb-m551oo2zn63yov****.cn-hangzhou.alb.aliyuncs.com   80      50s
   ```

## 步骤四：访问服务

- 利用获取的ALB实例IP地址，通过命令行方式访问`coffee`服务：

  ```
  curl -H Host:demo.domain.ingress.top http://alb-lhwdm5c9h8lrcm****.cn-hangzhou.alb.aliyuncs.com/coffee
  ```

- 利用获取的ALB实例IP地址，通过命令行方式访问`tea`服务：

  ```
  curl -H Host:demo.domain.ingress.top http://alb-lhwdm5c9h8lrcm****.cn-hangzhou.alb.aliyuncs.com/tea
  ```

  

