[CmdletBinding()]
param(
    [string]$Version = "",
    [string]$InstallDir = "",
    [string]$FromDirectory = "",
    [ValidateSet("auto", "gh", "https")][string]$Transport = "auto",
    [switch]$Help
)
$ErrorActionPreference = "Stop"
Set-StrictMode -Version 2.0

function Invoke-SpecGh {
    param([string[]]$Arguments)
    $Output = & gh @Arguments
    if ($LASTEXITCODE -ne 0) { throw "GitHub CLI request failed. Check repository access and whether a release exists." }
    return $Output
}

function Resolve-SpecFileSystemPath {
    param([string]$Path)
    $Provider = $null
    $Drive = $null
    $Result = $ExecutionContext.SessionState.Path.GetUnresolvedProviderPathFromPSPath($Path, [ref]$Provider, [ref]$Drive)
    if ($Provider.Name -ne "FileSystem") { throw "Choose a filesystem installation directory." }
    return $Result
}

function Install-SpecBinary {
    param([string]$SelectedVersion, [string]$BinDirectory, [string]$LocalDirectory, [string]$DownloadTransport)
    if ([Environment]::OSVersion.Platform -ne [PlatformID]::Win32NT) { throw "Use install.sh on Linux/macOS." }
    $Repository = "f-tuma/SPEC"
    if (-not $BinDirectory) {
        if (-not $env:LOCALAPPDATA) { throw "Set -InstallDir because LOCALAPPDATA is unavailable." }
        $BinDirectory = Join-Path $env:LOCALAPPDATA "SPEC\bin"
    }
    $BinDirectory = Resolve-SpecFileSystemPath $BinDirectory
    $Machine = $env:PROCESSOR_ARCHITEW6432
    if (-not $Machine) { $Machine = $env:PROCESSOR_ARCHITECTURE }
    switch ($Machine) {
        "AMD64" { $Architecture = "amd64" }
        "ARM64" { $Architecture = "arm64" }
        default { throw "Supported Windows architectures: AMD64 and ARM64." }
    }
    if ($SelectedVersion -and -not $SelectedVersion.StartsWith("v")) { $SelectedVersion = "v$SelectedVersion" }
    if ($SelectedVersion -and $SelectedVersion -cnotmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') { throw "Use a stable version such as v0.3.0." }
    if ($LocalDirectory) {
        $LocalDirectory = (Resolve-Path -LiteralPath $LocalDirectory).ProviderPath
        if (-not (Test-Path -LiteralPath $LocalDirectory -PathType Container)) { throw "Local package directory is missing." }
        if (-not $SelectedVersion) {
            $MatchesForPlatform = @(Get-ChildItem -LiteralPath $LocalDirectory -File | Where-Object { $_.Name -cmatch "^SPEC_([0-9]+\.[0-9]+\.[0-9]+)_windows_$Architecture\.zip$" })
            if ($MatchesForPlatform.Count -ne 1) { throw "Local mode needs exactly one matching version; otherwise use -Version." }
            $null = $MatchesForPlatform[0].Name -cmatch "^SPEC_([0-9]+\.[0-9]+\.[0-9]+)_windows_$Architecture\.zip$"
            $SelectedVersion = "v" + $Matches[1]
        }
    } else {
        if ($DownloadTransport -eq "auto") {
            $DownloadTransport = "https"
            if (Get-Command gh -CommandType Application -ErrorAction SilentlyContinue) {
                try {
                    & gh auth status --active --hostname github.com *> $null
                    if ($LASTEXITCODE -eq 0) { $DownloadTransport = "gh" }
                } catch { $DownloadTransport = "https" }
            }
        }
        if ($DownloadTransport -eq "gh" -and -not (Get-Command gh -CommandType Application -ErrorAction SilentlyContinue)) { throw "Install and authenticate GitHub CLI first." }
    }
    $Temporary = Join-Path ([IO.Path]::GetTempPath()) ("spec-download-" + [Guid]::NewGuid().ToString("N"))
    $Stage = $null
    $Lock = $null
    $LockOwned = $false
    $KeepBackups = $false
    $Failure = $null
    $CleanupErrors = New-Object 'System.Collections.Generic.List[string]'
    $Applied = New-Object 'System.Collections.Generic.List[string]'
    [IO.Directory]::CreateDirectory($Temporary) | Out-Null
    try {
        if (-not $SelectedVersion) {
            if ($DownloadTransport -eq "gh") {
                $SelectedVersion = (Invoke-SpecGh -Arguments @("release", "view", "--repo", $Repository, "--json", "tagName", "--jq", ".tagName") | Out-String).Trim()
            } else {
                try { $SelectedVersion = (Invoke-RestMethod -Uri "https://api.github.com/repos/$Repository/releases/latest").tag_name }
                catch { throw "No accessible release. Private repositories need authenticated gh or -FromDirectory." }
            }
        }
        if ($SelectedVersion -cnotmatch '^v[0-9]+\.[0-9]+\.[0-9]+$') { throw "Release did not identify one stable version." }
        $Asset = "SPEC_" + $SelectedVersion.Substring(1) + "_windows_$Architecture.zip"
        $Archive = Join-Path $Temporary $Asset
        $Checksums = Join-Path $Temporary "SHA256SUMS"
        if ($LocalDirectory) {
            Copy-Item -LiteralPath (Join-Path $LocalDirectory $Asset) -Destination $Archive
            Copy-Item -LiteralPath (Join-Path $LocalDirectory "SHA256SUMS") -Destination $Checksums
        } elseif ($DownloadTransport -eq "gh") {
            Invoke-SpecGh -Arguments @("release", "download", $SelectedVersion, "--repo", $Repository, "--dir", $Temporary, "--pattern", $Asset, "--pattern", "SHA256SUMS") | Out-Null
        } else {
            $ReleaseUrl = "https://github.com/$Repository/releases/download/$SelectedVersion"
            Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseUrl/SHA256SUMS" -OutFile $Checksums
            Invoke-WebRequest -UseBasicParsing -Uri "$ReleaseUrl/$Asset" -OutFile $Archive
        }
        $ExpectedHashes = @()
        foreach ($Line in Get-Content -LiteralPath $Checksums) {
            if ($Line -cmatch '^([^\s]+)\s+\*?([^\s]+)\s*$' -and $Matches[2] -ceq $Asset) {
                $ParsedHash = $Matches[1]
                if ($ParsedHash -cnotmatch '^[0-9a-f]{64}$') { throw "Invalid checksum entry." }
                $ExpectedHashes += $ParsedHash
            }
        }
        if ($ExpectedHashes.Count -ne 1) { throw "Checksum entry missing or duplicated." }
        if ((Get-FileHash -LiteralPath $Archive -Algorithm SHA256).Hash.ToLowerInvariant() -cne $ExpectedHashes[0]) { throw "SHA-256 mismatch; existing installation was not changed." }
        Add-Type -AssemblyName System.IO.Compression.FileSystem
        $Zip = [IO.Compression.ZipFile]::OpenRead($Archive)
        try {
            $Members = @("spec.exe", "LICENSE", "THIRD_PARTY_NOTICES.md", "README.md")
            if ($Zip.Entries.Count -ne 4) { throw "Unexpected archive layout." }
            foreach ($Name in $Members) {
                $Entries = @($Zip.Entries | Where-Object { $_.FullName -ceq $Name })
                if ($Entries.Count -ne 1) { throw "Missing or duplicated archive member: $Name" }
                $ArchiveInputStream = $Entries[0].Open()
                try {
                    $ArchiveOutputStream = [IO.File]::Open((Join-Path $Temporary $Name), [IO.FileMode]::CreateNew, [IO.FileAccess]::Write)
                    try { $ArchiveInputStream.CopyTo($ArchiveOutputStream) } finally { $ArchiveOutputStream.Dispose() }
                } finally { $ArchiveInputStream.Dispose() }
            }
        } finally { $Zip.Dispose() }
        $ExecutableVersion = & (Join-Path $Temporary "spec.exe") --version
        if ($LASTEXITCODE -ne 0 -or ($ExecutableVersion | Out-String).Trim() -cne $SelectedVersion.Substring(1)) { throw "Executable cannot run or disagrees with the requested version." }
        [IO.Directory]::CreateDirectory($BinDirectory) | Out-Null
        $LockPath = Join-Path $BinDirectory ".spec-install.lock"
        $Lock = [IO.File]::Open($LockPath, [IO.FileMode]::CreateNew, [IO.FileAccess]::Write, [IO.FileShare]::None)
        $LockOwned = $true
        $Stage = Join-Path $BinDirectory (".spec-install-" + [Guid]::NewGuid().ToString("N"))
        [IO.Directory]::CreateDirectory((Join-Path $Stage "old")) | Out-Null
        [IO.Directory]::CreateDirectory((Join-Path $Stage "new")) | Out-Null
        $Payload = [ordered]@{"spec.LICENSE" = "LICENSE"; "spec.THIRD_PARTY_NOTICES.md" = "THIRD_PARTY_NOTICES.md"; "spec.exe" = "spec.exe"}
        foreach ($Name in $Payload.Keys) {
            $Target = Join-Path $BinDirectory $Name
            if (Test-Path -LiteralPath $Target) {
                $Item = Get-Item -LiteralPath $Target -Force
                if ($Item -isnot [IO.FileInfo] -or ($Item.Attributes -band [IO.FileAttributes]::ReparsePoint)) { throw "Refusing to replace a nonfile or link: $Target" }
                [IO.File]::Copy($Target, (Join-Path (Join-Path $Stage "old") $Name))
            }
            [IO.File]::Copy((Join-Path $Temporary $Payload[$Name]), (Join-Path (Join-Path $Stage "new") $Name))
        }
        foreach ($Name in $Payload.Keys) {
            $Target = Join-Path $BinDirectory $Name
            $New = Join-Path (Join-Path $Stage "new") $Name
            $Applied.Add($Name)
            if ([IO.File]::Exists($Target)) { [IO.File]::Replace($New, $Target, $null) }
            else { [IO.File]::Move($New, $Target) }
        }
        $Applied.Clear()
        Write-Host "Installed SPEC $($SelectedVersion.Substring(1)) at $(Join-Path $BinDirectory 'spec.exe')"
        Write-Host ("Run: & '" + (Join-Path $BinDirectory "spec.exe").Replace("'", "''") + "'")
        if (-not (($env:PATH -split ";") -contains $BinDirectory)) { Write-Host "Add this directory to PATH if desired: $BinDirectory" }
    } catch {
        $OriginalError = $_
        for ($Index = $Applied.Count - 1; $Index -ge 0; $Index--) {
            $Name = $Applied[$Index]
            $Backup = Join-Path (Join-Path $Stage "old") $Name
            $Target = Join-Path $BinDirectory $Name
            try {
                if ([IO.File]::Exists($Backup)) {
                    if ([IO.File]::Exists($Target)) { [IO.File]::Replace($Backup, $Target, $null) }
                    else { [IO.File]::Move($Backup, $Target) }
                } elseif ([IO.File]::Exists($Target)) { [IO.File]::Delete($Target) }
            } catch { $KeepBackups = $true; Write-Warning "Recovery incomplete for $Target; backups retained at $Stage" }
        }
        $Failure = $OriginalError
    } finally {
        try { if ($Lock) { $Lock.Dispose() } } catch { $CleanupErrors.Add($_.Exception.Message) }
        try { if ($LockOwned) { [IO.File]::Delete($LockPath) } } catch { $CleanupErrors.Add($_.Exception.Message) }
        try { if ($Stage -and -not $KeepBackups) { Remove-Item -LiteralPath $Stage -Recurse -Force } } catch { $CleanupErrors.Add($_.Exception.Message) }
        try { Remove-Item -LiteralPath $Temporary -Recurse -Force } catch { $CleanupErrors.Add($_.Exception.Message) }
    }
    if ($Failure) {
        foreach ($Message in $CleanupErrors) { Write-Warning -WarningAction Continue "Cleanup incomplete: $Message" }
        throw $Failure
    }
    if ($CleanupErrors.Count) { throw ("Installation completed, but cleanup was incomplete: " + ($CleanupErrors -join "; ")) }
}

if ($Help) {
    Write-Output "Install prebuilt SPEC: .\install.ps1 [-Version vX.Y.Z] [-InstallDir DIR] [-Transport auto|gh|https]"
    Write-Output "Offline: .\install.ps1 -FromDirectory DIR [-Version vX.Y.Z]. No PATH or profile changes."
} else {
    Install-SpecBinary -SelectedVersion $Version -BinDirectory $InstallDir -LocalDirectory $FromDirectory -DownloadTransport $Transport
}
