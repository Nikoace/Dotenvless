$ErrorActionPreference = 'Stop'
$projectRoot = Split-Path -Parent $PSScriptRoot
$goTool = Join-Path $projectRoot '.tools\go\bin\go.exe'
if (!(Test-Path -LiteralPath $goTool)) { $goTool = (Get-Command go -ErrorAction Stop).Source }
$env:GOCACHE = Join-Path $projectRoot '.cache\go-build'
$env:GOMODCACHE = Join-Path $projectRoot '.cache\go-mod'
$env:GOTOOLCHAIN = 'local'
$compiler = Join-Path $projectRoot '.tools\llvm-mingw\bin\clang.exe'
if (Test-Path -LiteralPath $compiler) {
 $env:CC = $compiler
 $env:CXX = Join-Path $projectRoot '.tools\llvm-mingw\bin\clang++.exe'
 $env:PATH = (Split-Path -Parent $compiler) + ';' + $env:PATH
}
Push-Location (Join-Path $projectRoot 'desktop')
try { & $goTool @args; exit $LASTEXITCODE } finally { Pop-Location }
