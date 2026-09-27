# Development-only, pinned compiler bootstrap. Never called by the application.
$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$projectRoot = Split-Path -Parent $PSScriptRoot
$toolsRoot = Join-Path $projectRoot '.tools'
$compilerRoot = Join-Path $toolsRoot 'llvm-mingw'
if (Test-Path -LiteralPath (Join-Path $compilerRoot 'bin\clang.exe')) { Write-Output 'Desktop compiler already available'; exit 0 }
[IO.Directory]::CreateDirectory($toolsRoot) | Out-Null
$archive = Join-Path $toolsRoot 'llvm-mingw.zip'
$expected = 'e3ad77d117a4bea19a7a3b333341824d79a5a371004a10e25b8504e7b3047666'
if (!(Test-Path -LiteralPath $archive) -or (Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) {
 Invoke-WebRequest -UseBasicParsing -Uri 'https://github.com/mstorsjo/llvm-mingw/releases/download/20260922/llvm-mingw-20260922-ucrt-x86_64.zip' -OutFile $archive
}
if ((Get-FileHash -LiteralPath $archive -Algorithm SHA256).Hash.ToLowerInvariant() -ne $expected) { throw 'Compiler archive checksum mismatch' }
$expanded = Join-Path $toolsRoot 'llvm-mingw-20260922-ucrt-x86_64'
if (!(Test-Path -LiteralPath $expanded)) { Expand-Archive -LiteralPath $archive -DestinationPath $toolsRoot }
# Both paths are fixed children of this repository's tools directory.
if ([IO.Path]::GetDirectoryName([IO.Path]::GetFullPath($expanded)) -ne [IO.Path]::GetFullPath($toolsRoot)) { throw 'Invalid compiler path' }
Rename-Item -LiteralPath $expanded -NewName 'llvm-mingw'
Write-Output 'Pinned desktop compiler is ready. No system settings changed.'
