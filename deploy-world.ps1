# Deploys openworld map scripts to the local game and (if attached) the phone.
param(
    [string]$GameMaps = "C:\GOG Games\Nox\maps",
    [string]$Adb = "C:\Users\scott\Android\sdk\platform-tools\adb.exe"
)
$ErrorActionPreference = "Stop"
$world = Join-Path $PSScriptRoot "world\maps"

Get-ChildItem $world -Directory | ForEach-Object {
    $dst = Join-Path $GameMaps $_.Name
    if (Test-Path $dst) {
        Copy-Item (Join-Path $_.FullName "*") $dst -Force
        Write-Host "PC:    $($_.Name) <- scripts deployed"
    } else {
        Write-Host "PC:    $($_.Name) skipped (no such map dir)"
    }
}

$dev = & $Adb devices 2>$null | Select-String -Pattern "device$"
if ($dev) {
    Get-ChildItem $world -Directory | ForEach-Object {
        Get-ChildItem $_.FullName -File | ForEach-Object {
            $rel = "$($_.Directory.Name)/$($_.Name)"
            & $Adb push $_.FullName "/sdcard/Download/Nox/maps/$rel" | Out-Null
            Write-Host "phone: maps/$rel pushed"
        }
    }
    Write-Host "Note: restart the app so the data mirror picks up new files."
} else {
    Write-Host "phone: not attached, skipped"
}
