[![Build Status](https://github.com/kubeops/supervisor/workflows/CI/badge.svg)](https://github.com/kubeops/supervisor/actions?workflow=CI)
[![Docker Pulls](https://img.shields.io/docker/pulls/appscode/supervisor.svg)](https://hub.docker.com/r/appscode/supervisor/)
[![Slack](https://shields.io/badge/Join_Slack-salck?color=4A154B&logo=slack)](https://slack.appscode.com)
[![Twitter](https://img.shields.io/twitter/follow/kubeops.svg?style=social&logo=twitter&label=Follow)](https://twitter.com/intent/follow?screen_name=Kubeops)

# supervisor

### Project Generator Commands

```bash
> kubebuilder init --domain appscode.com --skip-go-version-check
> kubebuilder edit --multigroup=true
> kubebuilder create api --group supervisor --version v1alpha1 --kind Recommendation
> kubebuilder create api --group supervisor --version v1alpha1 --kind MaintenanceWindow
> kubebuilder create api --group supervisor --version v1alpha1 --kind ClusterMaintenanceWindow --namespaced=false
```
