<#
    .Description
    New-ServiceEnvironment sets up a full microservice environment from scratch, with no pre-requisites other than powershell core
#>
Write-Output "Starting boostrap process!"
./HashiCorp/Get-HashiStack.ps1 -OSWithArch "windows_amd64" -ConsulVersion "1.6.2" -VaultVersion "1.3.0"