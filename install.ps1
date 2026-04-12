# Tryve installer for Windows (PowerShell)
# Usage:
#   irm https://raw.githubusercontent.com/liemle3893/e2e-runner/main/install.ps1 | iex
#   .\install.ps1 -Dir C:\custom\path -Version v1.2.3

param(
    [string]$Dir = "$env:LOCALAPPDATA\tryve\bin",
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"
$Repo = "liemle3893/e2e-runner"
$Binary = "tryve"

function Resolve-Version {
    if ($Version -eq "") {
        Write-Host "Fetching latest release..."
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $script:Version = $release.tag_name
        if (-not $script:Version) {
            throw "Could not determine latest version"
        }
    }
}

function Detect-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default { throw "Unsupported architecture: $arch" }
    }
}

function Install-Tryve {
    Resolve-Version

    $arch = Detect-Arch
    $rawVersion = $Version.TrimStart("v")
    $archive = "${Binary}_${rawVersion}_windows_${arch}.zip"
    $url = "https://github.com/$Repo/releases/download/$Version/$archive"

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    try {
        Write-Host "Downloading $Binary $Version for windows/$arch..."
        Invoke-WebRequest -Uri $url -OutFile (Join-Path $tmpDir $archive) -UseBasicParsing

        Write-Host "Extracting..."
        Expand-Archive -Path (Join-Path $tmpDir $archive) -DestinationPath $tmpDir -Force

        # Ensure install directory exists
        if (-not (Test-Path $Dir)) {
            New-Item -ItemType Directory -Path $Dir -Force | Out-Null
        }

        # Copy binary
        Copy-Item -Path (Join-Path $tmpDir "$Binary.exe") -Destination (Join-Path $Dir "$Binary.exe") -Force

        # Add to PATH if not already present
        $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
        if ($currentPath -notlike "*$Dir*") {
            [Environment]::SetEnvironmentVariable("PATH", "$currentPath;$Dir", "User")
            Write-Host "Added $Dir to user PATH (restart your terminal to use)"
        }

        Write-Host ""
        Write-Host "Successfully installed $Binary $Version to $Dir\$Binary.exe"
        Write-Host "Run '$Binary --help' to get started."
    }
    finally {
        Remove-Item -Recurse -Force $tmpDir -ErrorAction SilentlyContinue
    }
}

Install-Tryve
