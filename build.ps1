param(
  [string]$Config = "mini-electron.json",
  [switch]$SkipFrontend
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

# ---- read mini-electron.json ----
$outputFilename = "app"
$icon = ""
if (Test-Path $Config) {
  $json = Get-Content $Config -Raw | ConvertFrom-Json
  if ($json.outputFilename) { $outputFilename = $json.outputFilename }
  if ($json.icon) { $icon = $json.icon }
} else {
  Write-Warning "config not found: $Config (using defaults)"
}
Write-Host "config: $Config -> output $outputFilename.exe"

# ---- frontend build ----
if (-not $SkipFrontend -and (Test-Path "frontend")) {
  Push-Location frontend
  try {
    if (-not (Test-Path "node_modules")) { npm install }
    npm run build
  } finally {
    Pop-Location
  }
}

# ---- embed exe icon (optional, needs goversioninfo) ----
if ($icon -and (Test-Path $icon)) {
  if (Get-Command goversioninfo -ErrorAction SilentlyContinue) {
    goversioninfo -64 -icon="$icon" -o="resource.syso"
    Write-Host "embedded exe icon: $icon"
  } else {
    Write-Warning "goversioninfo not found; exe icon NOT embedded (runtime window icon still works)."
    Write-Warning "install: go install github.com/josephspurrier/goversioninfo/cmd/goversioninfo@latest"
  }
} elseif ($icon) {
  Write-Warning "icon file not found: $icon"
}

# ---- build ----
go build -ldflags "-H=windowsgui -s -w" -o "$outputFilename.exe" .
Write-Host "done: $outputFilename.exe"
