cd .\edit_cr
go mod tidy
go build .\kubectl-edit_cr.go
Copy-Item "./kubectl-edit_cr.exe" -Destination "../../FalconCoreServices.Kubernetes/bin"

cd ..\edit_deploy
go mod tidy
go build .\kubectl-edit_deploy.go
Copy-Item "./kubectl-edit_deploy.exe" -Destination "../../FalconCoreServices.Kubernetes/bin"
cd ..


