# SPDX-License-Identifier: AGPL-3.0
# Register gitli as a Windows Service.
# Run from an elevated PowerShell prompt.

$ErrorActionPreference = "Stop"

$BinPath  = "C:\gitli\gitli.exe"
$Config   = "C:\gitli\config.toml"
$Service  = "gitli"

if (-not (Test-Path $BinPath)) {
    throw "gitli binary not found at $BinPath"
}

New-Service -Name $Service `
    -BinaryPathName "`"$BinPath`" serve --config `"$Config`"" `
    -DisplayName "gitli Git service" `
    -Description "Lightweight self-hosted Git service" `
    -StartupType Automatic

Write-Host "Service '$Service' installed. Start it with: Start-Service $Service"
