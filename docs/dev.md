# Developer guide

Before you deploy an image for Alibaba Load Balancer Controller, you must create an image based on Dockerfile. When you deploy an image for Alibaba Load Balancer Controller, the standard Kubernetes Deployment requirements must be met. You can deploy multiple images to implement disaster recovery. This topic describes how to deploy an image for Alibaba Load Balancer Controller by using the source code. In this topic, the source code is complied on an on-premises machine and is delivered to a remote depository. Then, you can deploy the image in a Kubernetes cluster.

## Prerequisites

- The Golang 1.17 compiler or image is prepared to compile the source code.
- Docker Desktop or a podman tool is prepared to build an image.
- A private image repository that can be connected to an on-premises machine or a Kubernetes cluster is created, such as a [Container Registry](https://cr.console.aliyun.com/cn-beijing/instances) repository.
- The version of the Kubernetes cluster is 1.20 or later.



## Build an image

Alibaba Load Balancer Controller uses Go vendor to manage dependencies. If the dependencies do not match, you can run the vendor command to obtain the dependencies.

1. Optional. Manage dependencies

   ```
   go mod vendor
   ```

2. Compile the binary code.

   ```
   go build -o bin/load-balancer-controller cmd/manager/main.go
   ```

3. Optional. If the on-premises machine and the cluster use different architectures, you can use cross-compiling of Go to build a cross-platform image.

   ```
   GOOS=linux GOARCH=amd64 go build -o bin/load-balancer-controller cmd/manager/main.go
   ```

4. Obtain a base image in advance.

   ```
   docker pull docker.io/library/alpine:3.11.6
   ```

5. Use the following Dockerfile to build an image based on the existing binary code.

   ```
   FROM alpine:3.11.6

   # Do not use docker multiple stage build until we
   # figure a way out how to solve build cache problem under 'go mod'.
   #

   RUN apk add --no-cache --update ca-certificates
   COPY bin/load-balancer-controller /load-balancer-controller
   ENTRYPOINT  ["/load-balancer-controller"]
   ```

6. Run the following command in the root directory of the project.

   ```
   docker build -f /path/to/your/Dockerfile . -t ${region}/${namespce}/alb:${tag}
   ```

7. The private image repository that is delivered

   ```
   docker push ${region}/${namespce}/alb:${tag}
   ```

You can perform the preceding operations in a Dockerfile that is placed in the project. Alternatively, you can build an image by using different dependencies based on your requirements.

## Deployment

When you deploy an image for Alibaba Load Balancer Controller, the standard Kubernetes Deployment requirements must be met. Before you run Alibaba Load Balancer Controller, additional dependencies are required.

1. Alibaba Load Balancer Controller monitors cluster resources based on RBAC. You must create the following resources: ServiceAccount, ClusterRoleBinding, and ClusterRole.

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

2. Before you run Alibaba Load Balancer Controller, you must configure relevant files or run the corresponding commands. In this example, the relevant files are configured by using ConfigMap.

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

3. Refer to the following file to deploy and run the Deployment file.

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

Perform the preceding operations in the deploy/vv1/load-balancer-controller.yaml directory.