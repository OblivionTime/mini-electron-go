param([Parameter(Mandatory=$true)][string]$Exe)
$ErrorActionPreference = 'Stop'
$exePath = (Resolve-Path -LiteralPath $Exe).Path
$root = Join-Path $env:TEMP ('mini-electron-smoke-' + [guid]::NewGuid())
$cache = $env:LOCALAPPDATA
$names = 'MINI_ELECTRON_CONFIG', 'MINI_ELECTRON_DLL', 'MINI_ELECTRON_URL', 'MINI_ELECTRON_SMOKE'
$saved = @{}
foreach ($name in $names) { $saved[$name] = [Environment]::GetEnvironmentVariable($name) }
try {
  New-Item -ItemType Directory -Path "$root/bin", "$root/cache" | Out-Null
  Copy-Item -LiteralPath $exePath -Destination "$root/bin/app.exe"
  foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, $null) }
  $env:MINI_ELECTRON_SMOKE = '1'
  $env:LOCALAPPDATA = "$root/cache"
  $log = "$root/cache/mini-electron-app/app.log"
  foreach ($pass in 'cold', 'warm') {
    if (Test-Path $log) { Remove-Item -LiteralPath $log }
    $p = Start-Process "$root/bin/app.exe" -WorkingDirectory "$root/bin" -PassThru
    try {
      $deadline = [DateTime]::UtcNow.AddSeconds(25)
      do {
        $p.Refresh()
        if ($p.HasExited) { throw "$pass run exited early (code $($p.ExitCode))" }
        $ready = $p.MainWindowHandle -and (Test-Path $log) -and ((Get-Content $log -Raw) -match 'smoke: generated RPC success and rejection verified')
        if (!$ready) { Start-Sleep -Milliseconds 200 }
      } until ($ready -or [DateTime]::UtcNow -ge $deadline)
      if (!$ready) { throw "$pass run did not complete RPC smoke" }
      if (!$p.CloseMainWindow() -or !$p.WaitForExit(8000) -or $p.ExitCode -ne 0) { throw "$pass run did not exit cleanly" }
      Write-Host "$pass run: startup, generated RPC, rejection, and close passed"
    } finally {
      if (!$p.HasExited) { $p.Kill(); $p.WaitForExit() }
      $p.Dispose()
    }
  }
} catch {
  if ($log -and (Test-Path $log)) { Get-Content $log -Tail 20 | Write-Host }
  throw
} finally {
  $env:LOCALAPPDATA = $cache
  foreach ($name in $names) { [Environment]::SetEnvironmentVariable($name, $saved[$name]) }
  if (Test-Path $root) { Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue }
}
