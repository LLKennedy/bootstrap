<#
	.Description
	Get-HashiStack gets hashicorp tools
#>
Param(
	[Parameter(Mandatory = $true)]
	[String]
	$OSWithArch,

	[Parameter(Mandatory = $false)]
	[String]
	$ConsulVersion,

	[Parameter(Mandatory = $false)]
	[String]
	$VaultVersion,

	[Parameter(Mandatory = $false)]
	[String]
	$NomadVersion,

	[Parameter(Mandatory = $false)]
	[String]
	$TerraformVersion,

	[Parameter(Mandatory = $false)]
	[String]
	$VagrantVersion
)

$tools = "./HashiCorp Tools"
if (!$(Test-Path $tools)) {
	New-Item -Path $tools -ItemType "directory"
}
$absoluteToolPath = $(Resolve-Path -Path $tools).Path

Function Get-HashiCorpBinary {
	Param(
		[parameter(Mandatory=$true)]
		[String]
		$Product,

		[parameter(Mandatory=$true)]
		[String]
		$Version,

		[parameter(Mandatory=$true)]
		[String]
		$OSWithArch,

		[parameter(Mandatory=$false)]
		[String]
		$OutDirectory
	)
	Write-Output "Downloading ${Product}..."
	$hashData = $(Invoke-WebRequest -Uri "https://releases.hashicorp.com/${Product}/${Version}/${Product}_${Version}_SHA256SUMS" -ErrorAction Stop).Content
	# We're just going to trust PKI, if they control hashicorp.com we'll trust the checksum file without verifying the sig file
	$fullProductName = "${Product}_${Version}_${OSWithArch}"
	Invoke-WebRequest -Uri "https://releases.hashicorp.com/${Product}/${Version}/${fullProductName}.zip" -ErrorAction Stop -OutFile "${fullProductName}.zip"
	$matchFound = $false
	$fileHash = $(Get-FileHash -Algorithm SHA256 -Path "${fullProductName}.zip").Hash
	# Write-Output "Hash is ${fileHash}"
	ForEach ($line in $($hashData -split "\n")) { # this isn't os-specific, hashicorp always uses simple \n to separate
		$parts = $line -split "  "
		# Write-Output "Line is ${parts}"
		if ($parts[1] -like "*${Product}_${Version}_${OSWithArch}.zip" -and $parts[0] -eq $fileHash) {
			$matchFound = $true
			break
		}
	}
	if (!$matchFound) {
		throw "Downloaded zip for ${Product} at version ${Version} was corrupt or invalid"
	}

	Expand-Archive -Path "${fullProductName}.zip" -DestinationPath $fullProductName -Force

	$file = $(Get-ChildItem -Path $fullProductName)
	Move-Item -Path "$fullProductName/$($file.PSChildName)" -Destination "$OutDirectory/$($file.PSChildName)" -Force
	Remove-Item $fullProductName # Remove the unzipped folder now that it's empty
	Write-Output "${Product} downloaded!"
}

Write-Output "Downloading HashiCorp stack..."

# Consul
if ($null -ne $ConsulVersion -and $ConsulVersion -ne "") {
	Get-HashiCorpBinary -Product "consul" -Version $ConsulVersion -OSWithArch ${OSWithArch} -OutDirectory $tools
}

# Vault
if ($null -ne $VaultVersion -and $VaultVersion -ne "") {
	Get-HashiCorpBinary -Product "vault" -Version $VaultVersion -OSWithArch ${OSWithArch} -OutDirectory $tools
}

# Nomad
if ($null -ne $NomadVersion -and $NomadVersion -ne "") {
	Get-HashiCorpBinary -Product "nomad" -Version $NomadVersion -OSWithArch ${OSWithArch} -OutDirectory $tools
}

# Terraform
if ($null -ne $TerraformVersion -and $TerraformVersion -ne "") {
	Get-HashiCorpBinary -Product "terraform" -Version $TerraformVersion -OSWithArch ${OSWithArch} -OutDirectory $tools
}

# Vagrant
if ($null -ne $VagrantVersion -and $VagrantVersion -ne "") {
	Get-HashiCorpBinary -Product "vagrant" -Version $VagrantVersion -OSWithArch ${OSWithArch} -OutDirectory $tools
}

# Terraform
if ($null -ne $TerraformVersion -and $TerraformVersion -ne "") {
	Get-HashiCorpBinary -Product "terraform" -Version $TerraformVersion -OSWithArch ${OSWithArch} -OutDirectory $tools
}

Write-Output "Temporarily adding tools to PATH"
$pathWithTools = "$([Environment]::GetEnvironmentVariable('Path'));${absoluteToolPath}"
[Environment]::SetEnvironmentVariable('Path', $pathWithTools)
