$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2.0
$Root = Split-Path $PSScriptRoot -Parent
$Binary = Join-Path $Root "bin/SPEC.exe"
if (-not (Test-Path -LiteralPath $Binary)) { throw "Build bin/SPEC.exe before running this native Windows integration test." }
$Version = (& $Binary --version | Out-String).Trim()
if ($LASTEXITCODE -ne 0) { throw "Test executable failed." }
$Architecture = $env:PROCESSOR_ARCHITEW6432
if (-not $Architecture) { $Architecture = $env:PROCESSOR_ARCHITECTURE }
switch ($Architecture) { "AMD64" { $Arch = "amd64" } "ARM64" { $Arch = "arm64" } default { throw "Unsupported test architecture." } }
$Temporary = Join-Path ([IO.Path]::GetTempPath()) ("spec-install-test-" + [Guid]::NewGuid().ToString("N"))
[IO.Directory]::CreateDirectory($Temporary) | Out-Null
$OriginalDotNetDirectory = [Environment]::CurrentDirectory
Push-Location $Temporary
try {
    [Environment]::CurrentDirectory = $Root
    $Packages = Join-Path $Temporary "release packages"
    $Payload = Join-Path $Temporary "payload"
    $BinDirectory = Join-Path $Temporary "bin with spaces"
    [IO.Directory]::CreateDirectory($Packages) | Out-Null
    [IO.Directory]::CreateDirectory($Payload) | Out-Null
    Copy-Item -LiteralPath $Binary -Destination (Join-Path $Payload "spec.exe")
    foreach ($Name in @("README.md", "LICENSE", "THIRD_PARTY_NOTICES.md")) { Copy-Item -LiteralPath (Join-Path $Root $Name) -Destination (Join-Path $Payload $Name) }
    $Asset = "SPEC_" + $Version + "_windows_$Arch.zip"
    $Archive = Join-Path $Packages $Asset
    Compress-Archive -Path (Join-Path $Payload "*") -DestinationPath $Archive
    $Digest = (Get-FileHash -LiteralPath $Archive -Algorithm SHA256).Hash.ToLowerInvariant()
    $Checksums = Join-Path $Packages "SHA256SUMS"
    [IO.File]::WriteAllText($Checksums, "$Digest  $Asset`n")
    & (Join-Path $Root "install.ps1") -FromDirectory "release packages" -InstallDir "bin with spaces"
    if (-not (Test-Path -LiteralPath (Join-Path $BinDirectory "spec.exe"))) { throw "Installer did not resolve paths from the caller's PowerShell location." }
    $Installed = Join-Path $BinDirectory "spec.exe"
    if ((& $Installed --version | Out-String).Trim() -cne $Version) { throw "Installed executable version mismatch." }
    foreach ($Name in @("spec.LICENSE", "spec.THIRD_PARTY_NOTICES.md")) {
        if (-not (Test-Path -LiteralPath (Join-Path $BinDirectory $Name))) { throw "Missing installed license: $Name" }
    }
    $Before = (Get-FileHash -LiteralPath $Installed -Algorithm SHA256).Hash
    [IO.File]::WriteAllText($Checksums, (("0" * 64) + "  $Asset`n"))
    $Failed = $false
    try { & (Join-Path $Root "install.ps1") -FromDirectory $Packages -InstallDir $BinDirectory }
    catch { $Failed = $true }
    if (-not $Failed) { throw "Checksum mismatch was accepted." }
    if ((Get-FileHash -LiteralPath $Installed -Algorithm SHA256).Hash -cne $Before) { throw "Checksum failure changed the existing executable." }
    [IO.File]::WriteAllText($Checksums, "$Digest  $Asset`n$Digest  $Asset`n")
    $Failed = $false
    try { & (Join-Path $Root "install.ps1") -FromDirectory $Packages -InstallDir $BinDirectory }
    catch { $Failed = $true }
    if (-not $Failed) { throw "Duplicate checksums were accepted." }
    [IO.File]::WriteAllText($Checksums, "$Digest  $Asset`n")
    $LockPath = Join-Path $BinDirectory ".spec-install.lock"
    $Lock = [IO.File]::Open($LockPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
    try {
        $Failed = $false
        try { & (Join-Path $Root "install.ps1") -FromDirectory $Packages -InstallDir $BinDirectory }
        catch { $Failed = $true }
        if (-not $Failed) { throw "Concurrent install lock ignored." }
        if (-not (Test-Path -LiteralPath $LockPath)) { throw "Another process's lock was removed." }
    } finally { $Lock.Dispose(); [IO.File]::Delete($LockPath) }
    if ((Get-FileHash -LiteralPath $Installed -Algorithm SHA256).Hash -cne $Before) { throw "Failed install changed existing executable." }
    Write-Output "Native Windows prebuilt installer: install/version/licenses/checksum/duplicate/lock preservation PASS"
} finally {
    Pop-Location
    [Environment]::CurrentDirectory = $OriginalDotNetDirectory
    Remove-Item -LiteralPath $Temporary -Force -Recurse
}
