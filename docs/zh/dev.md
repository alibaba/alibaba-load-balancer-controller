# 开发文档

Alibaba Load Balancer Controller项目的使用镜像部署，需要按照Dockerfile构建镜像，并使用部署Kubernetes的标准Deployment进行部署，可以通过多副本进行容灾。接下来介绍如何从源码进行controller的部署；本文采用本机编译源码，推送到远程仓库中，在Kubernetes集群中配置镜像进行部署操作。

## 前提依赖

- golang 1.17 编译器或镜像，用来编译源码
- docker desktop或同样用于构建镜像的 podman工具
- 本机和集群都能链接的私有镜像仓库，如[阿里云容器镜像服务](https://cr.console.aliyun.com/cn-beijing/instances)
- 可以链接使用的kubernetes 1.20版本以上集群



## 构建

项目使用go vendor进行依赖管理，可能存在依赖不匹配问题，在获取到项目后，可以执行vendor指令获取依赖

1. 【可选】依赖管理

   ```
   go mod vendor
   ```

2. 编译二进制

   ```
   go build -o bin/load-balancer-controller cmd/manager/main.go
   ```

3. 【可选】如果本机的架构和集群的架构不同，可以采用go的交叉编译，构建跨平台镜像

   ```
   GOOS=linux GOARCH=amd64 go build -o bin/load-balancer-controller cmd/manager/main.go
   ```

4. 提前获取一个基础镜像

   ```
   docker pull docker.io/library/alpine:3.11.6
   ```

5. 基于已有二进制构建镜像，可以用下面的Dockerfile文件

   ```
   FROM alpine:3.11.6
   
   # Do not use docker multiple stage build until we
   # figure a way out how to solve build cache problem under 'go mod'.
   #
   
   RUN apk add --no-cache --update ca-certificates
   COPY bin/load-balancer-controller /load-balancer-controller
   ENTRYPOINT  ["/load-balancer-controller"]
   ```

6. 执行构建指令，注意要在项目主目录执行下面的指令

   ```
   docker build -f /path/to/your/Dockerfile . -t ${region}/${namespce}/alb:${tag}
   ```

7. 推送的私有镜像仓库

   ```
   docker push ${region}/${namespce}/alb:${tag}
   ```

上述过程可以通过一个Dockerfile进行，已经放在了项目中，因为存在些公司内部环境的依赖，不同开发者可能需要的并不相同，可以按照上述过程构建符合个人需求的构建过程。

## 部署

Alibaba Load Balancer Controller的部署采用标准的Kubernetes Deployment资源进行部署，Controller运行需要其他资源依赖

1. Controller基于RBAC对集群资源进行监控，需要分别创建ServiceAccount、ClusterRoleBinding、ClusterRole创建资源

   ```yaml
  apiVersion: rbac.authorization.k8s.io/v1
  kind: ClusterRole
  metadata:
    name: system:load-balancer-controller
  rules:
  - apiGroups:
    - ""
    resources:
    - events
    verbs:
    - get
    - list
    - create
    - patch
    - update
  - apiGroups:
    - ""
    resources:
    - nodes
    verbs:
    - get
    - list
    - watch
  - apiGroups:
    - ""
    resources:
    - nodes/status
    verbs:
    - patch
    - update
  - apiGroups:
    - ""
    resources:
    - services
    - pods
    verbs:
    - get
    - list
    - watch
    - update
    - patch
  - apiGroups:
    - ""
    resources:
    - configmaps
    verbs:
    - get
    - list
    - watch
    - update
    - patch
  - apiGroups:
    - ""
    resources:
    - services/status
    - pods/status
    verbs:
    - update
    - patch
  - apiGroups:
    - ""
    resources:
    - serviceaccounts
    verbs:
    - create
  - apiGroups:
    - ""
    resources:
    - endpoints
    verbs:
    - get
    - list
    - watch
    - create
    - patch
    - update
  - apiGroups:
    - coordination.k8s.io
    resources:
    - leases
    verbs:
    - get
    - list
    - update
    - create
  - apiGroups:
    - apiextensions.k8s.io
    resources:
    - customresourcedefinitions
    verbs:
    - get
    - update
    - create
    - delete
  - apiGroups:
    - networking.k8s.io
    resources:
    - ingresses
    - ingressclasses
    verbs:
    - get
    - list
    - watch
    - update
    - create
    - patch
    - delete
  - apiGroups:
    - alibabacloud.com
    resources:
    - albconfigs
    verbs:
    - get
    - list
    - watch
    - update
    - create
    - patch
    - delete
  - apiGroups:
    - alibabacloud.com
    resources:
    - albconfigs/status
    verbs:
    - update
    - patch
  - apiGroups:
    - networking.k8s.io
    resources:
    - ingresses/status
    verbs:
    - update
    - patch
  - apiGroups:
    - ""
    resources:
    - namespaces/status
    - namespaces
    - services
    - secrets
    verbs:
    - get
    - list
    - watch
    - update
    - create
    - patch
    - delete
  - apiGroups:
    - extensions
    - apps
    resources:
    - deployments
    verbs:
    - get
    - list
    - watch
    - update
    - create
    - patch
    - delete
  ---
  apiVersion: v1
  kind: ServiceAccount
  metadata:
    name: load-balancer-controller
    namespace: kube-system
  ---
  kind: ClusterRoleBinding
  apiVersion: rbac.authorization.k8s.io/v1
  metadata:
    name: system:load-balancer-controller
  roleRef:
    apiGroup: rbac.authorization.k8s.io
    kind: ClusterRole
    name: system:load-balancer-controller
  subjects:
    - kind: ServiceAccount
      name: load-balancer-controller
      namespace: kube-system
   ```

2. Controller运行需要配置文件，可以采用命令行或配置文件的方式，这里我们采用ConfigMap挂载配置文件的方式

   ```yaml
    apiVersion: v1
    kind: ConfigMap
    metadata:
      name: load-balancer-config
      namespace: kube-system
    data:
      cloud-config.conf: |-
          {
              "Global": {
                  "AccessKeyID": "VndV***", # 需要base64编码
                  "AccessKeySecret": "UWU0NnUyTFdhcG***" # 需要base64编码
              }
          }
   ```

3. Deployment文件的部署和运行指令参考以下文件

   ```yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    labels:
      app: load-balancer-controller
      tier: control-plane
    name: load-balancer-controller
    namespace: kube-system
  spec:
    replicas: 1
    selector:
      matchLabels:
        app: load-balancer-controller
        tier: control-plane
    template:
      metadata:
        labels:
          app: load-balancer-controller
          tier: control-plane
      spec:
        containers:
          - command:
              - /load-balancer-controller
              - --cloud-config=/etc/kubernetes/config/cloud-config.conf
              - --controllers=ingress,service
              - --leader-elect-resource-name=alb
              - --configure-cloud-routes=false
            image: ${path/to/your/image/registry}
            imagePullPolicy: Always
            livenessProbe:
              failureThreshold: 8
              initialDelaySeconds: 15
              periodSeconds: 10
              successThreshold: 1
              tcpSocket:
                port: 10258
              timeoutSeconds: 15
            name: load-balancer-controller
            readinessProbe:
              failureThreshold: 8
              initialDelaySeconds: 15
              periodSeconds: 10
              successThreshold: 1
              tcpSocket:
                port: 10258
              timeoutSeconds: 15
            resources:
              limits:
                cpu: "1"
                memory: 1Gi
              requests:
                cpu: 100m
                memory: 200Mi
            securityContext:
              allowPrivilegeEscalation: false
              readOnlyRootFilesystem: true
              runAsNonRoot: true
              runAsUser: 1200
            terminationMessagePath: /dev/termination-log
            terminationMessagePolicy: File
            volumeMounts:
              - mountPath: /etc/kubernetes/config
                name: cloud-config
        dnsPolicy: ClusterFirst
        imagePullSecrets:
        - name: aliyun-secret
        restartPolicy: Always
        schedulerName: default-scheduler
        securityContext: {}
        serviceAccount: load-balancer-controller
        serviceAccountName: load-balancer-controller
        terminationGracePeriodSeconds: 30
        tolerations:
          - operator: Exists
        volumes:
          - configMap:
              defaultMode: 420
              items:
                - key: cloud-config.conf
                  path: cloud-config.conf
              name: load-balancer-config
            name: cloud-config
   ```

上述过程统一放在 deploy/vv1/load-balancer-controller.yaml 中，方便直接使用