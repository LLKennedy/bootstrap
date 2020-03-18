<#
    .SYNOPSIS
    

    .Description
    

    It downloads the necessary 
#>
<#
.SYNOPSIS

Sets up a new environment from scratch.

.DESCRIPTION

New-ServiceEnvironment sets up a full microservice environment 
from scratch, with no pre-requisites other than powershell core.

.PARAMETER Name
Specifies the file name.

.PARAMETER Extension
Specifies the extension. "Txt" is the default.

.INPUTS

None. You cannot pipe objects to Add-Extension.

.OUTPUTS

System.String. Add-Extension returns a string with the extension
or file name.

.EXAMPLE

PS> extension -name "File"
File.txt

.EXAMPLE

PS> extension -name "File" -extension "doc"
File.doc

.EXAMPLE

PS> extension "File" "doc"
File.doc

.LINK

http://www.fabrikam.com/extension.html

.LINK

Set-Item
#>
Write-Output "Starting boostrap process!"
./HashiCorp/Get-HashiStack.ps1 -OSWithArch "windows_amd64" -TerraformVersion "0.12.18"