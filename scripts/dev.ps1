$GoArgs = @($args)
$ErrorActionPreference = 'Stop'
$root = Split-Path -Parent $PSScriptRoot
$localGo = Join-Path $root '.tools\go\bin\go.exe'
$go = if (Test-Path -LiteralPath $localGo) { $localGo } else { (Get-Command go -ErrorAction Stop).Source }
$env:GOCACHE = Join-Path $root '.cache\go-build'
$env:GOMODCACHE = Join-Path $root '.cache\go-mod'
$env:GOTOOLCHAIN = 'local'
Push-Location $root
try {
    & $go @GoArgs
    exit $LASTEXITCODE
} finally {
    Pop-Location
}
