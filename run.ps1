go mod init
go build .\cmd\kubectl-edit_deploy.go
Copy-Item "./kubectl-edit_deploy.exe" -Destination "../FalconCoreServices.Kubernetes/bin"