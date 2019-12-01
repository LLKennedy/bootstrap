<#
	.Description
	Get-HashiStack gets hashicorp tools
#>
Param(
	[Parameter(Position = 0, Mandatory = $true)]
	[String]
	$OSWithArch,

	[Parameter(Position = 1, Mandatory = $true)]
	[String]
	$ConsulVersion,

	[Parameter(Position = 2, Mandatory = $true)]
	[String]
	$VaultVersion
)

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
	ForEach ($line in $($hashData -split "\n")) { # this isn't os-specific, hashicorp always uses simple \n to separate
		$parts = $line -split "  "
		if ($parts[1] -eq "${Product}_${Version}_${OSWithArch}.zip" -and $parts[0] -eq $fileHash) {
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

$tools = "./HashiCorp Tools"
if (!$(Test-Path $tools)) {
	New-Item -Path $tools -ItemType "directory"
}
$absoluteToolPath = $(Resolve-Path -Path $tools).Path

# Consul
Get-HashiCorpBinary -Product "consul" -Version $ConsulVersion -OSWithArch ${OSWithArch} -OutDirectory $tools

# Vault
Get-HashiCorpBinary -Product "vault" -Version $VaultVersion -OSWithArch ${OSWithArch} -OutDirectory $tools

Write-Output "Temporarily adding tools to PATH"
$pathWithTools = "$([Environment]::GetEnvironmentVariable('Path'));${absoluteToolPath}"
[Environment]::SetEnvironmentVariable('Path', $pathWithTools)
