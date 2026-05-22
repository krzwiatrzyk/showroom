# ArgoCD Scenario Showcase

The goal of this scenario is to prepare a multi-cluster progressive  deployments that satisfy few requirements:
- for each application first deployment is done on beta then on web clusters
- for each cluster, first infrastructure apps are deployed and only when they are ready and healthy applications are deployed

"Infra-ready" app is used to mark ArgoCD clusters as ready with infra deployed so applications can be deployed on them.