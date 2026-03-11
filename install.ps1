$ErrorActionPreference = "Stop"

$Repo = "alanisme/adb-tui"
$Binary = "adb-tui.exe"

$Arch = if ($env:PROCESSOR_ARCHITECTURE -eq "ARM64") { "arm64" } else { "amd64" }

if ($args.Count -gt 0) {
    $Version = $args[0]
} else {
    $Release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $Release.tag_name
}

$Archive = "adb-tui_$($Version.TrimStart('v'))_windows_$Arch.zip"
$Url = "https://github.com/$Repo/releases/download/$Version/$Archive"

$TmpDir = Join-Path $env:TEMP "adb-tui-install"
New-Item -ItemType Directory -Force -Path $TmpDir | Out-Null

$ZipPath = Join-Path $TmpDir $Archive

Write-Host "Downloading adb-tui $Version (windows/$Arch)..."
Invoke-WebRequest -Uri $Url -OutFile $ZipPath

Expand-Archive -Path $ZipPath -DestinationPath $TmpDir -Force

$InstallDir = Join-Path $env:LOCALAPPDATA "adb-tui"
New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
Move-Item -Force (Join-Path $TmpDir $Binary) (Join-Path $InstallDir $Binary)

Remove-Item -Recurse -Force $TmpDir

$Path = [Environment]::GetEnvironmentVariable("Path", "User")
if ($Path -notlike "*$InstallDir*") {
    [Environment]::SetEnvironmentVariable("Path", "$Path;$InstallDir", "User")
    Write-Host "Added $InstallDir to PATH (restart terminal to take effect)."
}

Write-Host "Installed adb-tui $Version to $InstallDir\$Binary"
