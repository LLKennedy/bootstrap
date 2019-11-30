<#
	.Description
	Get-HashiStack gets hashicorp tools
#>

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
		$OutFile
	)
	Write-Output "Downloading ${Product}..."
	$hashData = $(Invoke-WebRequest -Uri "https://releases.hashicorp.com/${Product}/${Version}/${Product}_${Version}_SHA256SUMS" -ErrorAction Stop).Content
	# TODO: Get signature file for checksum file and verify before bothering with the main zip download
	$mainFileResponse = $(Invoke-WebRequest -Uri "https://releases.hashicorp.com/${Product}/${Version}/${Product}_${Version}_${OSWithArch}.zip" -ErrorAction Stop)
	$matchFound = $false
	$fileHash = $(Get-FileHash -InputStream $mainFileResponse.RawContentStream -Algorithm SHA256).Hash
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

	# TODO: Unzip to specified file location

	Write-Progress -Activity "Saving ${Product} to file system"
	Set-Content -Path $OutFile -Value $mainFileResponse.Content -AsByteStream
	Write-Progress -Activity "Saving ${Product} to file system" -Completed $true
	Write-Output "${Product} downloaded!"
}

Write-Output "Downloading HashiCorp stack..."

# Consul
$ConsulVersion = "1.6.2"
$OSWithArch = "windows_amd64"
Get-HashiCorpBinary -Product "consul" -Version $ConsulVersion -OSWithArch ${OSWithArch} -OutFile "consul.zip" # TODO: Change to .exe when unzipping is done

# Vault
$ConsulVersion = "1.3.0"
$OSWithArch = "windows_amd64"
Get-HashiCorpBinary -Product "vault" -Version $ConsulVersion -OSWithArch ${OSWithArch} -OutFile "vault.zip" # TODO: Change to .exe when unzipping is done