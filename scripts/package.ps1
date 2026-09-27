param([string]$Version='0.2.0')
$ErrorActionPreference='Stop'
if ($Version -notmatch '^\d+\.\d+\.\d+(-[a-z0-9.-]+)?$') { throw 'Invalid release version' }
$projectRoot=Split-Path -Parent $PSScriptRoot
$goTool=Join-Path $projectRoot '.tools\go\bin\go.exe'
if (!(Test-Path -LiteralPath $goTool)) { $goTool=(Get-Command go -ErrorAction Stop).Source }
$compiler=Join-Path $projectRoot '.tools\llvm-mingw\bin\clang.exe'
if (!(Test-Path -LiteralPath $compiler)) { throw 'Run scripts/setup-desktop.ps1 before packaging' }
$env:GOCACHE=Join-Path $projectRoot '.cache\go-build'
$env:GOMODCACHE=Join-Path $projectRoot '.cache\go-mod'
$env:GOTOOLCHAIN='local'
$env:GOWORK='off'
$env:GOOS='windows'
$env:GOARCH='amd64'
$env:CC=$compiler
$env:CXX=Join-Path $projectRoot '.tools\llvm-mingw\bin\clang++.exe'
$env:PATH=(Split-Path -Parent $compiler)+';'+$env:PATH
function Invoke-GoChecked {
 & $goTool @args
 if ($LASTEXITCODE -ne 0) { throw 'Go verification/build failed' }
}
Push-Location $projectRoot
try {
 $files=@(Get-ChildItem -LiteralPath cmd,internal,desktop -Recurse -Filter '*.go' -File | ForEach-Object { $_.FullName })
 $gofmt=Join-Path (Split-Path -Parent $goTool) 'gofmt.exe'
 $unformatted=& $gofmt -l $files
 if ($LASTEXITCODE -ne 0 -or $unformatted) { throw 'Go formatting check failed' }
 # Treat both local modules as main modules only while verifying the dependency cache.
 # This avoids Go's ziphash check on the non-downloadable local "dotenvless" replacement.
 $verifyRoot=Join-Path $projectRoot ('.cache\verify-'+[Guid]::NewGuid().ToString('N'))
 [IO.Directory]::CreateDirectory($verifyRoot) | Out-Null
 Push-Location $verifyRoot
 try { Invoke-GoChecked work init $projectRoot (Join-Path $projectRoot 'desktop') } finally { Pop-Location }
 $env:GOWORK='off'
 $env:CGO_ENABLED='0'
 Invoke-GoChecked mod verify
 Invoke-GoChecked vet ./...
 Invoke-GoChecked test -count=1 ./...
 Invoke-GoChecked build -trimpath -ldflags "-s -w -X dotenvless/internal/cli.Version=$Version" -o bin/dvl.exe ./cmd/dvl
 $cliModules=& $goTool list -deps -f '{{with .Module}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}' ./cmd/dvl
 if ($LASTEXITCODE -ne 0) { throw 'Cannot collect CLI dependency notices' }
 $env:DVL_E2E_CLI=Join-Path $projectRoot 'bin\dvl.exe'
 Push-Location (Join-Path $projectRoot 'desktop')
 try {
  $env:GOWORK=Join-Path $verifyRoot 'go.work'
  try { Invoke-GoChecked mod verify } finally { $env:GOWORK='off' }
  Invoke-GoChecked vet -tags ci ./...
  Invoke-GoChecked test -tags ci -count=1 ./...
  $env:CGO_ENABLED='1'
  Invoke-GoChecked vet ./...
  Invoke-GoChecked build -trimpath -ldflags '-H=windowsgui -s -w' -o ../bin/dvl-desktop.exe .
  $desktopModules=& $goTool list -deps -f '{{with .Module}}{{.Path}}|{{.Version}}|{{.Dir}}{{end}}' .
  $modules=@($cliModules)+@($desktopModules) | Where-Object { $_ } | Sort-Object -Unique
  if ($LASTEXITCODE -ne 0) { throw 'Cannot collect dependency notices' }
 } finally { Pop-Location }
 $stage=Join-Path $projectRoot ('.cache\package-'+[Guid]::NewGuid().ToString('N'))
 [IO.Directory]::CreateDirectory($stage) | Out-Null
 Copy-Item -LiteralPath bin/dvl.exe,bin/dvl-desktop.exe,README.md,SECURITY.md,CHANGELOG.md,CONTRIBUTING.md,AGENTS.md -Destination $stage
 Copy-Item -LiteralPath docs -Destination $stage -Recurse
 $notices=New-Object Text.StringBuilder
 [void]$notices.AppendLine('Third-party dependency licenses and notices for Dotenvless '+$Version)
 [void]$notices.AppendLine('These licenses apply to their respective dependencies. Dotenvless has not declared a source license.')
 [void]$notices.AppendLine('Go runtime license')
 [void]$notices.AppendLine([IO.File]::ReadAllText((Join-Path (Split-Path -Parent (Split-Path -Parent $goTool)) 'LICENSE')))
 [void]$notices.AppendLine('LLVM-MinGW toolchain license')
 [void]$notices.AppendLine([IO.File]::ReadAllText((Join-Path (Split-Path -Parent (Split-Path -Parent $compiler)) 'LICENSE.TXT')))
 foreach ($line in $modules) {
  $parts=$line -split '\|',3
  if ($parts[0] -eq 'dotenvless' -or $parts[0] -eq 'dotenvless/desktop' -or !$parts[2]) { continue }
  [void]$notices.AppendLine('')
  [void]$notices.AppendLine('Dependency: '+$parts[0]+' '+$parts[1])
  $licenseFiles=Get-ChildItem -LiteralPath $parts[2] -File -Recurse | Where-Object { $_.Name -match '^(LICENSE|LICENCE|COPYING|NOTICE|OFL)([._-].*)?$' }
  if (!$licenseFiles) { throw ('Missing dependency license: '+$parts[0]) }
  foreach ($file in $licenseFiles) {
   [void]$notices.AppendLine('Source: '+$file.FullName.Substring($parts[2].Length+1))
   [void]$notices.AppendLine([IO.File]::ReadAllText($file.FullName))
  }
 }
 [IO.File]::WriteAllText((Join-Path $stage 'THIRD_PARTY_NOTICES.txt'),$notices.ToString(),(New-Object Text.UTF8Encoding($false)))
 $records=@()
 foreach ($name in @('dvl.exe','dvl-desktop.exe')) {
  $records+=@{ name=$name; sha256=(Get-FileHash -LiteralPath (Join-Path $stage $name) -Algorithm SHA256).Hash.ToLowerInvariant() }
 }
 $manifest=@{ version=$Version; os='windows'; arch='amd64'; binaries=$records } | ConvertTo-Json -Depth 4
 [IO.File]::WriteAllText((Join-Path $stage 'manifest.json'),$manifest,(New-Object Text.UTF8Encoding($false)))
 $dist=Join-Path $projectRoot 'dist'
 [IO.Directory]::CreateDirectory($dist) | Out-Null
 $zip=Join-Path $dist ('dotenvless-'+$Version+'-windows-amd64.zip')
 Compress-Archive -Path (Join-Path $stage '*') -DestinationPath $zip -Force
 $hash=(Get-FileHash -LiteralPath $zip -Algorithm SHA256).Hash.ToLowerInvariant()
 [IO.File]::WriteAllText(($zip+'.sha256'),$hash+'  '+[IO.Path]::GetFileName($zip)+[Environment]::NewLine,(New-Object Text.UTF8Encoding($false)))
 & (Join-Path $stage 'dvl.exe') --version
 if ($LASTEXITCODE -ne 0) { throw 'Packaged CLI did not start' }
 Write-Output ('Release package: '+$zip)
} finally { Pop-Location }
