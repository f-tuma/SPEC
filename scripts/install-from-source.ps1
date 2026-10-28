$ErrorActionPreference = "Stop"
$SPEC_SOURCE_ROOT = Split-Path $PSScriptRoot -Parent
Push-Location $SPEC_SOURCE_ROOT
try {
    if (-not (Get-Command go -ErrorAction SilentlyContinue)) {
        throw "Source setup requires Go 1.26+. Alternatively download the native SPEC binary from GitHub Releases."
    }
    New-Item -ItemType Directory -Force -Path bin | Out-Null
    & go build -trimpath -o bin/spec.exe .
    if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
} finally {
    Pop-Location
}
if ($args.Count -eq 0) { & "$SPEC_SOURCE_ROOT/bin/spec.exe" } else { & "$SPEC_SOURCE_ROOT/bin/spec.exe" install @args }
exit $LASTEXITCODE
