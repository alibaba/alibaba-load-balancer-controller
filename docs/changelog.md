# ChangeLog
All notable changes to this project will be documented in this file.

## [v1.2.0] - 2023-12-26
### Added
- Custom tags can be added to ALB instances.
- Services can be reconciled with backend server groups.
- Network access control lists (ACLs) can be associated with ALB instances by specifying the IDs of the network ACLs.
- HTTPS and QUIC services can be deployed on the same port.
- Multiple server groups, rewrites, and uppercase and lowercase letters are supported by custom actions.
- Certificates that are configured by using Secrets have higher priorities than AlbConfigs.
- Error messages are optimized.
- By default, the managed ALB Ingress controller is deployed in multiple replicated pods to ensure high availability.
- Resource groups can be specified when you create ALB instances.
- Multiple status codes are supported by health checks.
- Consistent hashing is supported for distributing traffic to backend server groups.
- The use-regex annotation is supported.
- The ALB Ingress controller can be deployed in a single zone.
- The network types of ALB instances can be changed.
- Internet Shared Bandwidth instances can be associated with ALB instances.
- Asynchronous API operations are optimized.
- Service reconciliation events are exposed.
- The use of the ssl-redirect annotation is optimized.
- ShangMi (SM) certificates can be automatically discovered and filtered.
- Hash values can be added to Ingresses and the AlbConfig to ensure that no unexpected changes occur when the ALB Ingress controller restarts.
- The exposure of abnormal events is optimized.
- The reconciliation process is optimized for scenarios in which reserved fields are used.
- The synchronization logic of server groups is optimized.
- Custom forwarding actions support QPS rate limiting and source IP rate limiting.

### Fixed 
- Event notifications are optimized.
- No finalizers are configured for Ingress deletions. This fixes the issue that Ingresses are stuck when you delete them.
- Issues that occur when you switch the network type of an ALB instance to IPv6 are fixed.
- The issue that Ingress certificates are repeatedly discovered is fixed.
- The issue that the tags of backend server groups become invalid during canary releases is fixed.
- The reconciliation process and rule priorities are optimized to accelerate rule synchronization.
- Configuration errors of Gzip compression are fixed.
- The issue that the default certificates displayed in the console are different from the actual default certificates used by ALB instances and the issue that the default certificates used by ALB instances are repeatedly specified are fixed.
- The issue that forwarding rules may be deleted when the pods of the component are restarted is fixed.
- The issue that server reconciliations are not retried is fixed.
- The issue that the keys in custom forwarding rules do not take effect is fixed.
- API throttling can be avoided when multiple server groups are reconciled with a Service.
- The issue related to the reconciliation of CookieConfig in custom forwarding rules is fixed.
- The following issue is fixed: The ALB Ingress controller crashes if the http field of an Ingress is not configured.
- The following issue is fixed: Configuration updates fail if multiple actions are specified in the configuration of an Ingress.
- The issue that the cache is not synchronized after Ingress resources are deleted is fixed.
- The issue that node event handling is interrupted is fixed.

### Removed 
- Hard-coded timeout values are removed.
- Internet Shared Bandwidth instances are no longer deleted during the reconciliation process.
- The network type of the ALB instance used by the component cannot be changed. This is a temporary change.

## [v1.1.2] - 2023-04-18
### Fixed 
- Bugfix Some Problem About Certificate.

## [v1.1.1] - 2023-04-18
### Fixed 
- Bugfix ServerGroup Creating Problem.

## [v1.1.0] - 2023-04-11
### Added
- Support NLB Controller.

## [v1.0.0] - 2023-04-10
### Added
- Support ALB Ingress Controller.