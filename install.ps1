[CmdletBinding()]
param(
  [string]$Version = "",
  [string]$InstallDir = "",
  [ValidateSet("github", "gitee")]
  [string]$Source = "",
  [switch]$CliOnly,
  [switch]$Bootstrap,
  [switch]$WithDefaultSkills,
  [switch]$NoAgentConfig,
  [string[]]$Agent = @(),
  [string[]]$Completion = @(),
  [string]$Key = "",
  [string]$Endpoint = "",
  [string]$Project = ""
)

$ErrorActionPreference = "Stop"
Set-StrictMode -Version Latest

$Source = if ($Source) { $Source.ToLowerInvariant() } elseif ($env:POPIART_SOURCE) { $env:POPIART_SOURCE.ToLowerInvariant() } else { "github" }
$Repo = if ($env:POPIART_REPO) { $env:POPIART_REPO } else { "" }
$Binary = "popiart.exe"
$InferredTag = ""

function Get-DefaultRepo {
  param([string]$RepoSource)

  switch ($RepoSource) {
    "gitee" { return "wattx/popiartcli" }
    default { return "wtgoku-create/popiartcli" }
  }
}

function Write-Log {
  param([string]$Message)
  Write-Host $Message
}

function Get-DefaultInstallDir {
  if ($InstallDir) {
    return $InstallDir
  }
  return Join-Path $env:LOCALAPPDATA "Programs\popiart\bin"
}

function Get-OsArch {
  $arch = $null

  try {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
  }
  catch {
    $arch = $null
  }

  if (-not $arch) {
    $legacyArch = if ($env:PROCESSOR_ARCHITEW6432) {
      $env:PROCESSOR_ARCHITEW6432
    }
    else {
      $env:PROCESSOR_ARCHITECTURE
    }

    if ($legacyArch) {
      $arch = $legacyArch.ToLowerInvariant()
    }
  }

  switch ($arch) {
    "x64" { return "amd64" }
    "amd64" { return "amd64" }
    "arm64" { return "arm64" }
    "aarch64" { return "arm64" }
    default { throw "unsupported architecture: $arch" }
  }
}

function Get-LatestTag {
  $uri = if ($Source -eq "gitee") {
    "https://gitee.com/api/v5/repos/$Repo/releases/latest"
  }
  else {
    "https://api.github.com/repos/$Repo/releases/latest"
  }
  $release = Invoke-RestMethod -Uri $uri
  if (-not $release.tag_name) {
    throw "failed to resolve latest release tag"
  }
  return [string]$release.tag_name
}

function Trim-GitHubArchiveTag {
  param([string]$Value)

  $trimmed = $Value.Trim()
  foreach ($suffix in @(".tar.gz", ".tgz", ".zip")) {
    if ($trimmed.EndsWith($suffix, [System.StringComparison]::OrdinalIgnoreCase)) {
      return $trimmed.Substring(0, $trimmed.Length - $suffix.Length)
    }
  }
  return $trimmed
}

function Resolve-RepoInput {
  param(
    [string]$InputValue,
    [string]$DefaultSource
  )

  $value = $InputValue.Trim().TrimEnd("/")
  if (-not $value) {
    return [pscustomobject]@{
      Repo   = Get-DefaultRepo $DefaultSource
      Tag    = ""
      Source = $DefaultSource
    }
  }

  if ($value -notmatch "://" -and -not $value.ToLowerInvariant().StartsWith("github.com/") -and -not $value.ToLowerInvariant().StartsWith("gitee.com/")) {
    $parts = $value.TrimEnd("/").TrimEnd(".git").Split("/", [System.StringSplitOptions]::RemoveEmptyEntries)
    if ($parts.Length -ne 2) {
      throw "expected repository in owner/name format"
    }
    return [pscustomobject]@{
      Repo   = "$($parts[0])/$($parts[1])"
      Tag    = ""
      Source = $DefaultSource
    }
  }

  if ($value.ToLowerInvariant().StartsWith("github.com/") -or $value.ToLowerInvariant().StartsWith("gitee.com/")) {
    $value = "https://$value"
  }

  $uri = [System.Uri]$value
  $resolvedSource = switch ($uri.Host.ToLowerInvariant()) {
    "github.com" { "github" }
    "www.github.com" { "github" }
    "gitee.com" { "gitee" }
    "www.gitee.com" { "gitee" }
    default { $null }
  }
  if (-not $resolvedSource) {
    throw "unsupported host: $($uri.Host)"
  }

  $parts = $uri.AbsolutePath.Trim("/").Split("/", [System.StringSplitOptions]::RemoveEmptyEntries)
  if ($parts.Length -lt 2) {
    throw "expected owner/repo path"
  }

  $name = [string]$parts[1]
  if ($name.EndsWith(".git", [System.StringComparison]::OrdinalIgnoreCase)) {
    $name = $name.Substring(0, $name.Length - 4)
  }
  $repo = "$($parts[0])/$name"
  $tag = ""

  if ($parts.Length -ge 5 -and $parts[2] -eq "releases" -and $parts[3] -eq "tag") {
    $tag = [string]$parts[4]
  }
  elseif ($parts.Length -ge 6 -and $parts[2] -eq "archive" -and $parts[3] -eq "refs" -and $parts[4] -eq "tags") {
    $tag = Trim-GitHubArchiveTag ([string]$parts[5])
  }
  elseif ($parts.Length -ge 4 -and $parts[2] -eq "archive") {
    $tag = Trim-GitHubArchiveTag ([string]$parts[3])
  }

  return [pscustomobject]@{
    Repo   = $repo
    Tag    = $tag
    Source = $resolvedSource
  }
}

$resolvedRepo = Resolve-RepoInput $Repo $Source
$Repo = [string]$resolvedRepo.Repo
$InferredTag = [string]$resolvedRepo.Tag
$Source = [string]$resolvedRepo.Source

function Ensure-UserPathContains {
  param([string]$Dir)

  $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
  $parts = @()
  if ($userPath) {
    $parts = $userPath.Split(';', [System.StringSplitOptions]::RemoveEmptyEntries)
  }

  if ($parts -contains $Dir) {
    return $false
  }

  $newPath = if ($userPath) { "$userPath;$Dir" } else { $Dir }
  [Environment]::SetEnvironmentVariable("Path", $newPath, "User")
  $env:Path = "$env:Path;$Dir"
  return $true
}

function Invoke-Bootstrap {
  param(
    [string]$ExePath,
    [string]$EndpointValue,
    [string]$ProjectValue,
    [string]$KeyValue,
    [string[]]$AgentValues,
    [string[]]$CompletionValues
  )

  $args = @()
  if ($EndpointValue) {
    $args += @("--endpoint", $EndpointValue)
  }
  if ($ProjectValue) {
    $args += @("--project", $ProjectValue)
  }

  $args += @("--plain", "bootstrap")

  if ($KeyValue) {
    $args += @("--key", $KeyValue)
  }
  if ($WithDefaultSkills) {
    $args += "--with-default-skills"
  }
  if ($NoAgentConfig) {
    $args += "--no-agent-config"
  }
  foreach ($item in $AgentValues) {
    if ($item) {
      $args += @("--agent", $item)
    }
  }
  foreach ($item in $CompletionValues) {
    if ($item) {
      $args += @("--completion", $item)
    }
  }

  Write-Log "running bootstrap"
  & $ExePath @args
}

$tag = if ($env:VERSION) { $env:VERSION } elseif ($Version) { $Version } elseif ($InferredTag) { $InferredTag } else { Get-LatestTag }
$versionNoV = $tag.TrimStart("v")
$arch = Get-OsArch
$targetDir = Get-DefaultInstallDir
$archiveName = "popiart_${versionNoV}_windows_${arch}.zip"
$baseUrl = if ($Source -eq "gitee") {
  "https://gitee.com/$Repo/releases/download/v$versionNoV"
}
else {
  "https://github.com/$Repo/releases/download/v$versionNoV"
}

$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ("popiart-" + [System.Guid]::NewGuid().ToString("N"))
New-Item -ItemType Directory -Path $tempDir | Out-Null

try {
  $archivePath = Join-Path $tempDir $archiveName
  $checksumsPath = Join-Path $tempDir "checksums.txt"

  Write-Log "downloading $archiveName"
  Invoke-WebRequest -Uri "$baseUrl/$archiveName" -OutFile $archivePath
  Invoke-WebRequest -Uri "$baseUrl/checksums.txt" -OutFile $checksumsPath

  $expectedSha = $null
  foreach ($line in Get-Content $checksumsPath) {
    if ($line -match "^([a-fA-F0-9]+)\s+(\S+)$" -and $matches[2] -eq $archiveName) {
      $expectedSha = $matches[1].ToLowerInvariant()
      break
    }
  }
  if (-not $expectedSha) {
    throw "checksum entry for $archiveName not found"
  }

  $actualSha = (Get-FileHash -Path $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
  if ($actualSha -ne $expectedSha) {
    throw "checksum mismatch for $archiveName"
  }

  New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
  Expand-Archive -Path $archivePath -DestinationPath $tempDir -Force

  $exeSource = Join-Path $tempDir $Binary
  if (-not (Test-Path $exeSource)) {
    throw "failed to extract $Binary from $archiveName"
  }

  $exeTarget = Join-Path $targetDir $Binary
  Copy-Item -Path $exeSource -Destination $exeTarget -Force
  Write-Log "installed popiart $versionNoV to $exeTarget"

  $pathUpdated = Ensure-UserPathContains -Dir $targetDir
  if ($pathUpdated) {
    Write-Log "added $targetDir to the user PATH"
    Write-Log "open a new terminal after installation to pick up the updated PATH"
  }

  if (-not $Bootstrap -or $CliOnly) {
    exit 0
  }

  Invoke-Bootstrap -ExePath $exeTarget -EndpointValue $Endpoint -ProjectValue $Project -KeyValue $Key -AgentValues $Agent -CompletionValues $Completion
}
finally {
  if (Test-Path $tempDir) {
    Remove-Item -Path $tempDir -Recurse -Force
  }
}
