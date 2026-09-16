param(
    [ValidateSet('amd64', 'arm64')]
    [string]$Architecture = 'amd64',
    [ValidatePattern('^[a-zA-Z0-9._-]+$')]
    [string]$Version = 'dev',
    [string]$GoBinary = 'go',
    [string]$OutputDirectory
)

$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
if (-not $OutputDirectory) {
    $OutputDirectory = Join-Path $projectRoot 'dist'
}
$OutputDirectory = [IO.Path]::GetFullPath($OutputDirectory)
$packageName = "confab_${Version}_windows_${Architecture}"
$packageDir = Join-Path $OutputDirectory $packageName
$archivePath = Join-Path $OutputDirectory "$packageName.zip"
if ((Test-Path -LiteralPath $packageDir) -or (Test-Path -LiteralPath $archivePath)) {
    throw 'Output already exists. Choose a new OutputDirectory or Version; existing artifacts are never overwritten.'
}

# Only this build process receives the target settings; restore them afterwards.
$previousGOOS = $env:GOOS
$previousGOARCH = $env:GOARCH
$previousCGO = $env:CGO_ENABLED
Push-Location $projectRoot
try {
    $env:GOOS = 'windows'
    $env:GOARCH = $Architecture
    $env:CGO_ENABLED = '0'
    New-Item -ItemType Directory -Path $packageDir -Force | Out-Null
    $binaryPath = Join-Path $packageDir 'confab.exe'
    $buildDate = [DateTime]::UtcNow.ToString('yyyy-MM-ddTHH:mm:ssZ')
    & $GoBinary build -trimpath -ldflags "-s -w -X main.version=$Version -X main.date=$buildDate" -o $binaryPath .
    if ($LASTEXITCODE -ne 0) { throw "Go build failed: $LASTEXITCODE" }
    Copy-Item -LiteralPath (Join-Path $projectRoot 'docs/windows-portable.md') -Destination (Join-Path $packageDir 'README.md')
    Copy-Item -LiteralPath (Join-Path $projectRoot 'LICENSE') -Destination $packageDir
    Compress-Archive -LiteralPath @($binaryPath, (Join-Path $packageDir 'README.md'), (Join-Path $packageDir 'LICENSE')) -DestinationPath $archivePath
    $hash = (Get-FileHash -LiteralPath $archivePath -Algorithm SHA256).Hash.ToLowerInvariant()
    "$hash  $packageName.zip" | Set-Content -LiteralPath "$archivePath.sha256" -Encoding ASCII
    Write-Output $archivePath
    Write-Output "$archivePath.sha256"
}
finally {
    $env:GOOS = $previousGOOS
    $env:GOARCH = $previousGOARCH
    $env:CGO_ENABLED = $previousCGO
    Pop-Location
}
