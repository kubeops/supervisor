# supervisor

### Project Generator Commands

```bash
> kubebuilder init --domain appscode.com --skip-go-version-check
> kubebuilder edit --multigroup=true
> kubebuilder create api --group supervisor --version v1alpha1 --kind Recommendation
> kubebuilder create api --group supervisor --version v1alpha1 --kind MaintenanceWindow
> kubebuilder create api --group supervisor --version v1alpha1 --kind ClusterMaintenanceWindow --namespaced=false
```
