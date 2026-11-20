param(
  [string]$Config = "mini-electron.json",
  [switch]$SkipFrontend,
  [switch]$SkipSign,
  [switch]$SkipGen,
  [switch]$Offline
)

$ErrorActionPreference = "Stop"
$root = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $root

function Write-Step($msg) { Write-Host "==> $msg" -ForegroundColor Cyan }
function Write-Warn2($msg) { Write-Warning $msg }

# ---- read mini-electron.json ----
$json = $null
if (Test-Path $Config) {
  $json = Get-Content $Config -Raw | ConvertFrom-Json
} else {
  Write-Warn2 "config not found: $Config (using defaults)"
}

$outputFilename = "app"
$icon = ""
if ($json) {
  if ($json.outputFilename) { $outputFilename = $json.outputFilename }
  if ($json.icon) { $icon = $json.icon }
}
$exePath = Join-Path $root "$outputFilename.exe"
$signCfg = if ($json -and $json.sign -isnot [bool]) { $json.sign } else { $null }
$force = $signCfg -and $signCfg.forceCodeSigning
if ($force -and $SkipSign) { throw '-SkipSign conflicts with forceCodeSigning' }
Write-Host "config: $Config -> output $outputFilename.exe"

# ---- generate frontend bindings (Wails-style) ----
# Reflects mbvip.Options.Bind and writes JS/TS wrappers into frontend/src/bindings,
# so it must run BEFORE the frontend build.
if (-not $SkipGen) {
  if (-not (Test-Path "frontend/dist")) {
    New-Item -ItemType Directory -Force -Path "frontend/dist" | Out-Null
    Set-Content "frontend/dist/.keep" ""
  }
  Write-Step "generate bindings -> frontend/src/bindings"
  go run . -gen "frontend/src/bindings"
  if ($LASTEXITCODE -ne 0) { throw "bindgen failed ($LASTEXITCODE)" }
}

# ---- frontend build ----
if (-not $SkipFrontend -and (Test-Path "frontend")) {
  Write-Step "build frontend"
  Push-Location frontend
  try {
    & npm.cmd ci
    if ($LASTEXITCODE -ne 0) { throw "npm ci failed ($LASTEXITCODE)" }
    & npm.cmd run build
    if ($LASTEXITCODE -ne 0) { throw "npm run build failed ($LASTEXITCODE)" }
  } finally {
    Pop-Location
  }
}
if (-not (Test-Path 'frontend/dist/index.html' -PathType Leaf)) { throw 'frontend/dist/index.html missing; build the frontend first' }
$dist = (Resolve-Path 'frontend/dist').Path
$assetRefs = [regex]::Matches((Get-Content "$dist/index.html" -Raw), '(?i)\b(?:src|href)=["''](?:\./|/)?(assets/[^"''?#]+)')
foreach ($ref in $assetRefs) {
  if (-not (Test-Path -LiteralPath (Join-Path $dist $ref.Groups[1].Value) -PathType Leaf)) {
    throw "frontend asset missing: $($ref.Groups[1].Value)"
  }
}

# ---- embed exe icon (rsrc preferred, goversioninfo fallback) ----
# go build auto-links any *.syso in the package dir, so the icon lands in the
# exe's resource section (Explorer/taskbar icon), same as electron's rcedit/icon step.
if ($icon -and (Test-Path $icon)) {
  $syso = "rsrc.syso"
  if (Get-Command rsrc -ErrorAction SilentlyContinue) {
    Write-Step "embed icon via rsrc: $icon"
    & rsrc -ico "$icon" -arch amd64 -o "$syso"
    if ($LASTEXITCODE -ne 0) { throw "rsrc failed ($LASTEXITCODE)" }
  } elseif (Get-Command goversioninfo -ErrorAction SilentlyContinue) {
    Write-Step "embed icon via goversioninfo: $icon"
    goversioninfo -64 -icon="$icon" -o="$syso"
  } else {
    Write-Warn2 "neither rsrc nor goversioninfo found; exe icon NOT embedded."
    Write-Warn2 "install rsrc: go install github.com/akavel/rsrc@latest"
  }
} elseif ($icon) {
  Write-Warn2 "icon file not found: $icon"
}

# ---- build ----
Write-Step "go build"
go build -ldflags "-H=windowsgui -s -w" -o "$outputFilename.exe" .
if ($LASTEXITCODE -ne 0) { throw "go build failed ($LASTEXITCODE)" }
Write-Host "built: $outputFilename.exe"

# ============================================================================
# Windows code signing -- mirrors electron-builder's signtool path
# (app-builder-lib windowsCodeSign.ts: computeSignToolArgs + sign()).
#   - dual-signs by default: ["sha1","sha256"], 2nd pass appended with /as
#   - /t legacy timestamp for sha1; /tr + /td sha256 (RFC3161) for sha256
#   - cert from .pfx/.p12 (/f) or Windows cert store (/sha1 /s [/sm])
#   - retries once (15s) on lock / timestamp-server failure
#   - tool from sign.signtoolPath -> $env:SIGNTOOL_PATH -> PATH -> Windows SDK
#   - env: WIN_CSC_LINK/CSC_LINK (path|url|base64), WIN_CSC_KEY_PASSWORD/CSC_KEY_PASSWORD
#   - ELECTRON_BUILDER_OFFLINE=true (or -Offline) skips timestamping
# ============================================================================

function Resolve-SigntoolPath([string]$explicit) {
  if ($explicit -and (Test-Path -LiteralPath $explicit)) { return $explicit }
  if ($env:SIGNTOOL_PATH -and (Test-Path -LiteralPath $env:SIGNTOOL_PATH)) { return $env:SIGNTOOL_PATH }
  $cmd = Get-Command signtool.exe -ErrorAction SilentlyContinue
  if ($cmd) { return $cmd.Source }
  $kits = @("${env:ProgramFiles(x86)}\Windows Kits\10\bin", "$env:ProgramFiles\Windows Kits\10\bin")
  $found = Get-ChildItem $kits -Recurse -Filter signtool.exe -ErrorAction SilentlyContinue |
    Where-Object { $_.FullName -match '\\x64\\' } |
    Sort-Object FullName -Descending | Select-Object -First 1
  if ($found) { return $found.FullName }
  return $null
}

# WIN_CSC_LINK / CSC_LINK may be a file path, an https URL, or a base64 .pfx
# (exactly what electron-builder accepts).
function Resolve-CertLink([string]$link, [string]$tmpDir) {
  if (Test-Path -LiteralPath $link) { return (Resolve-Path -LiteralPath $link).Path }
  if ($link -match '^https://') {
    $out = Join-Path $tmpDir "csc-link.pfx"
    try { Invoke-WebRequest -Uri $link -OutFile $out -ErrorAction Stop }
    catch { throw 'Failed to download the signing certificate over HTTPS' }
    return $out
  }
  try {
    $bytes = [Convert]::FromBase64String(($link -replace '\s', ''))
    $out = Join-Path $tmpDir "csc-link.pfx"
    [IO.File]::WriteAllBytes($out, $bytes)
    return $out
  } catch {
    throw 'CSC_LINK is neither a file, HTTPS URL, nor base64 payload'
  }
}

# Look a code-signing cert up in the Windows store by subject name and/or SHA1
# (electron-builder getCertificateFromStoreInfo).
function Get-CertFromStore([string]$subjectName, [string]$sha1) {
  if ($sha1) { $sha1 = $sha1.ToUpperInvariant() }
  $certs = Get-ChildItem -Recurse Cert: -CodeSigningCert -ErrorAction SilentlyContinue
  foreach ($c in $certs) {
    if ($subjectName -and ($c.Subject -notlike "*$subjectName*")) { continue }
    if ($sha1 -and ($c.Thumbprint.ToUpperInvariant() -ne $sha1)) { continue }
    $parent = $c.PSParentPath
    $store = $parent.Substring($parent.LastIndexOf('\') + 1)
    return [pscustomobject]@{
      Thumbprint     = $c.Thumbprint
      Subject        = $c.Subject
      Store          = $store
      IsLocalMachine = ($parent -match 'LocalMachine')
    }
  }
  throw "Cannot find certificate '$subjectName$sha1' in the Windows certificate store"
}

# Build the signtool argument list exactly like computeSignToolArgs().
function Get-SigntoolArgs {
  param($File, $Hash, [bool]$IsNest, $Cert, [bool]$Offline, $Cfg, [string]$Description, [string]$Website)
  if (-not $Cfg) { $Cfg = [pscustomobject]@{} }
  $a = @("sign")
  if (-not $Offline) {
    $legacyTs = if ($Cfg.timeStampServer) { $Cfg.timeStampServer } else { "http://timestamp.digicert.com" }
    if ($IsNest -or $Hash -eq "sha256") {
      $rfc = if ($Cfg.rfc3161TimeStampServer) { $Cfg.rfc3161TimeStampServer } else { "http://timestamp.digicert.com" }
      $a += @("/tr", $rfc)
    } else {
      $a += @("/t", $legacyTs)
    }
  }
  if ($Cert.File) {
    $ext = [IO.Path]::GetExtension($Cert.File).ToLowerInvariant()
    if ($ext -ne ".pfx" -and $ext -ne ".p12") { throw "Please specify a pkcs12 (.p12/.pfx) file, got $($Cert.File)" }
    $a += @("/f", $Cert.File)
  } else {
    $a += @("/sha1", $Cert.Thumbprint)
    if ($Cert.Store) { $a += @("/s", $Cert.Store) }
    if ($Cert.IsLocalMachine) { $a += @("/sm") }
  }
  if ($Hash -ne "sha1") {
    $a += @("/fd", $Hash)
    if (-not $Offline) { $a += @("/td", "sha256") }
  } else {
    # recent signtool requires an explicit digest flag even for sha1
    $a += @("/fd", "sha1")
  }
  if ($Description) { $a += @("/d", $Description) }
  if ($Website) { $a += @("/du", $Website) }
  if ($IsNest) { $a += @("/as") }
  if ($Cert.Password) { $a += @("/p", $Cert.Password) }
  if ($Cfg.additionalCertificateFile) { $a += @("/ac", $Cfg.additionalCertificateFile) }
  $a += $File
  return $a
}

function Invoke-Signtool([string]$tool, [string[]]$signArgs) {
  for ($attempt = 1; $attempt -le 2; $attempt++) {
    # capture so signtool's own stdout/stderr never leaks into the return value
    & $tool @signArgs 2>&1 | Out-Null
    $code = $LASTEXITCODE
    if ($code -eq 0) { return 0 }
    if ($attempt -eq 1) {
      # electron-builder retries once after 15s on lock / timestamp failure
      Write-Warn2 "signtool exited $code -- retrying once in 15s"
      Start-Sleep -Seconds 15
    }
  }
  return $code
}

$runSigning = $false
if (-not $SkipSign) {
  $offlineBuild = ($env:ELECTRON_BUILDER_OFFLINE -eq "true") -or $Offline

  # credentials: env wins (WIN_* over generic), then config
  $link = if ($env:WIN_CSC_LINK) { $env:WIN_CSC_LINK } elseif ($env:CSC_LINK) { $env:CSC_LINK } else { $null }
  $pass = if ($env:WIN_CSC_KEY_PASSWORD) { $env:WIN_CSC_KEY_PASSWORD } elseif ($env:CSC_KEY_PASSWORD) { $env:CSC_KEY_PASSWORD } else { $null }
  $subjectName = $null
  $sha1 = $null
  if ($signCfg) {
    if (-not $link -and $signCfg.certificateFile) { $link = $signCfg.certificateFile }
    if (-not $pass -and $signCfg.certificatePassword) { $pass = $signCfg.certificatePassword }
    $subjectName = $signCfg.certificateSubjectName
    $sha1 = $signCfg.certificateSha1
  }

  $hasEnvCreds = [bool]$env:WIN_CSC_LINK -or [bool]$env:CSC_LINK
  # sign when explicitly enabled, when credentials are present (electron-builder
  # behavior), or when env credentials exist
  $hasCreds = [bool]$link -or [bool]$sha1 -or [bool]$subjectName
  $enabled = $force -or ($signCfg -and ($signCfg.enabled -or $hasCreds)) -or $hasEnvCreds

  if ($enabled) {
    $err = $null
    $tool = Resolve-SigntoolPath $(if ($signCfg) { $signCfg.signtoolPath } else { $null })
    if (-not $link -and -not $sha1 -and -not $subjectName) {
      $err = "signing enabled but no certificate (certificateFile/SubjectName/Sha1 or CSC_LINK)"
    } elseif (-not $tool) {
      $err = "signtool.exe not found (install the Windows SDK, or set sign.signtoolPath / SIGNTOOL_PATH)"
    }

    if ($err) {
      if ($force) { throw $err }
      Write-Warn2 "$err -- skipping signing"
    } else {
      $runSigning = $true
    }
  }

  if ($runSigning) {
    $tmpDir = [IO.Directory]::CreateDirectory((Join-Path ([IO.Path]::GetTempPath()) ([guid]::NewGuid().ToString()))).FullName
    try {
    $cert = [pscustomobject]@{ File = $null; Password = $null; Thumbprint = $null; Store = $null; IsLocalMachine = $false }
    if ($link) {
      $cert.File = Resolve-CertLink $link $tmpDir
      $cert.Password = $pass
    } else {
      $store = Get-CertFromStore $subjectName $sha1
      $cert.Thumbprint = $store.Thumbprint
      $cert.Store = $store.Store
      $cert.IsLocalMachine = $store.IsLocalMachine
      Write-Host "cert from store: $($store.Subject) [$($store.Thumbprint)] store=$($store.Store) localMachine=$($store.IsLocalMachine)"
    }

    $hashes = @("sha1", "sha256")
    if ($signCfg -and $signCfg.signingHashAlgorithms) { $hashes = @($signCfg.signingHashAlgorithms) }

    $description = $null
    if ($signCfg -and $signCfg.description) { $description = $signCfg.description }
    elseif ($json -and $json.name) { $description = $json.name }
    $website = if ($signCfg) { $signCfg.website } else { $null }

    Write-Step "sign: $outputFilename.exe (algorithms: $($hashes -join ', '))"
    $isNest = $false
    $failed = $false
    foreach ($h in $hashes) {
      $signArgs = Get-SigntoolArgs -File $exePath -Hash $h -IsNest $isNest -Cert $cert -Offline $offlineBuild -Cfg $signCfg -Description $description -Website $website
      if ((Invoke-Signtool $tool $signArgs) -ne 0) { $failed = $true; break }
      $isNest = $true
    }

    if ($failed) {
      if ($force) { throw "code signing failed (forceCodeSigning=true)" }
      Write-Warn2 "code signing failed; do not treat this EXE as signed and verified"
    } else {
      & $tool verify /pa "$exePath"
      if ($LASTEXITCODE -ne 0) {
        if ($force) { throw "signtool verify failed ($LASTEXITCODE)" }
        Write-Warn2 "signtool verify reported an issue (exit $LASTEXITCODE)"
      }
      else { Write-Host "signed + verified: $outputFilename.exe" }
    }
    } finally {
      Remove-Item -LiteralPath $tmpDir -Recurse -Force
    }
  }
}

Write-Step "done: $outputFilename.exe"
