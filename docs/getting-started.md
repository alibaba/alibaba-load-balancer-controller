# Access Services by using an ALB Ingress

Application Load Balancer (ALB) provides Ingresses that allow you to access Services. ALB Ingresses are suitable for handling traffic fluctuations.

## Background information

An Ingress provides a collection of rules that manage external access to Services in a Kubernetes cluster. You can configure forwarding rules to assign different externally-accessible URLs to different Services. However, NGINX Ingresses and Layer 4 Server Load Balancer (SLB) Ingresses cannot meet the requirements of cloud-native applications, such as complex routing, multiple application layer protocols support (such as QUIC), and balancing of heavy traffic loads at Layer 7.

## Prerequisites

- A kubectl client is connected to your cluster.
- The ALB Ingress controller is installed. For more information, see [Compilation and deployment]().
## Precautions

- The version of Kubernetes is 1.18 or later.
- If you use the Flannel network plug-in, the backend Services of the ALB Ingress must be of the NodePort or LoadBalancer type.

## Step 1: Create an AlbConfig object

1. Create a file named alb-test.yaml and copy the following content to the file. The file is used to create an AlbConfig Object.

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

   | Parameter | Description |
   | :--------------------------- | :----------------------------------------------------------- |
   | **spec.config.name** | The name of the ALB instance. This parameter is optional.  |
   | **spec.config.addressType** | The type of IP address that the ALB instance uses to provide services. This parameter is required. Internet: The ALB instance uses a public IP address. The domain name of the Ingress is resolved to the public IP address of the ALB instance. Therefore, the ALB instance is accessible over the Internet. This is the default value. Intranet: The ALB instance uses a private IP address. The domain name of the Ingress is resolved to the private IP address of the ALB instance. Therefore, the ALB instance is accessible only within the virtual private cloud (VPC) where the ALB instance is deployed.  |
   | **spec.config.zoneMappings** | The IDs of the vSwitches that are used by the ALB Ingress. You must specify at least two vSwitch IDs and the vSwitches must be deployed in different zones. The zones of the vSwitches must be supported by ALB Ingresses. This parameter is required. For more information about the regions and zones that are supported by ALB Ingresses, see [Supported regions and zones](https://help.aliyun.com/document_detail/258300.htm#task-2087008).  |

2. Run the following command to create an AlbConfig object:

   ```
   kubectl apply -f alb-test.yaml
   ```

   Expected output:

   ```
   AlbConfig.alibabacloud.com/alb-demo created
   ```

3. Create an alb.yaml file that contains the following content:

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

4. Run the following command to create an IngressClass:

   ```
   kubectl apply -f alb.yaml
   ```

   Expected output

   ```
   ingressclass.networking.k8s.io/alb created
   ```

## Step 2: Deploy applications

1. Create a cafe-service.yaml file and copy the following content to the file. The file is used to deploy two Deployments named `coffee` and `tea` and two Services named `coffee` and `tea`.

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

2. Run the following command to deploy the Deployments and Services:

   ```
   kubectl apply -f cafe-service.yaml
   ```

   Expected output:

   ```
   deployment "coffee" created
   service "coffee-svc" created
   deployment "tea" created
   service "tea-svc" created
   ```

3. Run the following command to query the status of the Services that you created:

   ```
   kubectl get svc,deploy
   ```

   Expected output

   ```
   NAME             TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
   coffee-svc   NodePort    172.16.231.169   <none>        80:31124/TCP   6s
   tea-svc      NodePort    172.16.38.182    <none>        80:32174/TCP   5s
   NAME            DESIRED   CURRENT   UP-TO-DATE   AVAILABLE   AGE
   deploy/coffee   2         2         2            2           1m
   deploy/tea      1         1         1            1           1m
   ```

## Step 3: Configure an Ingress

1. Create a cafe-ingress.yaml file and copy the following content to the file:

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
         # Configure a context path.
         - path: /tea
           pathType: Prefix
           backend:
             service:
               name: tea-svc
               port:
                 number: 80
         # Configure a context path.
         - path: /coffee
           pathType: Prefix
           backend:
             service:
               name: coffee-svc
               port:
                 number: 80
   ```

2. Run the following command to configure an externally-accessible domain name and a `path` for the `coffee` and `tea` Services separately:

   ```
   kubectl apply -f cafe-ingress.yaml
   ```

   Expected output

   ```
   ingress "cafe-ingress" created
   ```

3. Run the following command to query the IP address of the ALB instance:

   ```
   kubectl get ing
   ```

   Expected output

   ```
   NAME           CLASS    HOSTS                         ADDRESS                                               PORTS   AGE
   cafe-ingress   alb      demo.domain.ingress.top       alb-m551oo2zn63yov****.cn-hangzhou.alb.aliyuncs.com   80      50s
   ```

## Step 4: Access the Services

- After you obtain the IP address of the ALB instance, use the CLI to access the `coffee` Service:

   ```
   curl -H Host:demo.domain.ingress.top http://alb-lhwdm5c9h8lrcm****.cn-hangzhou.alb.aliyuncs.com/coffee
   ```

- After you obtain the IP address of the ALB instance, use the CLI to access the `tea` Service:

   ```
   curl -H Host:demo.domain.ingress.top http://alb-lhwdm5c9h8lrcm****.cn-hangzhou.alb.aliyuncs.com/tea
   ```



