# Running
```
$ go build cmd/kubectl-edit_deploy.go
# Place the built binary somewhere in the PATH

# Now we can use the plugin as regular kubectl command:
# Update number of replicas in the current set namespace

$ kubectl edit-deploy <deployment_name> --replicas=<number> 

# Update number of replicas in specified namespace 

$ kubectl edit-deploy <deployment_name> --replicas=<number> -n <specified namespace>

```

# Cleanup
To uninstall the plugin from kubectl by simply removing it from the PATH