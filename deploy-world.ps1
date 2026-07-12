# Deploys the openworld map set to the local game and (if attached) the phone.
#
# 1. Regenerates the ow_ maps (campaign clones minus story scripts) into the
#    local game install via tools/owgen.
# 2. Copies the openworld Lua scripts next to them.
# 3. Pushes maps + scripts to the phone's Download/Nox/maps mirror.
#
# The ow_*.map files contain original game data and are never committed;
# they are always regenerated from the local install.
param(
    [string]$GameMaps = "C:\GOG Games\Nox\maps",
    [string]$Adb = "C:\Users\scott\Android\sdk\platform-tools\adb.exe"
)
$ErrorActionPreference = "Stop"
$world = Join-Path $PSScriptRoot "world\maps"

# Regenerate ow_ maps for every ow_ zone in world/maps (plus the class starts).
$owNames = @(Get-ChildItem $world -Directory | Where-Object { $_.Name -like 'ow_*' } | ForEach-Object { $_.Name.Substring(3) })
$owNames += @('war01a', 'con01a') # class starting zones, scripted or not
$owNames = $owNames | Sort-Object -Unique
Push-Location $PSScriptRoot
try {
    go run ./tools/owgen -maps $GameMaps -out $GameMaps -dump data\ow_waypoints.json @owNames
    if ($LASTEXITCODE -ne 0) { throw "owgen failed" }
} finally {
    Pop-Location
}

# Scripts next to the maps (PC install).
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
    # adb writes progress to stderr, which PS 5.1 + ErrorActionPreference
    # Stop escalates into a terminating error — route pushes through cmd.
    $ErrorActionPreference = "Continue"
    # ow_ maps: push the generated .map files.
    foreach ($n in $owNames) {
        $ow = "ow_$n"
        $map = Join-Path $GameMaps "$ow\$ow.map"
        if (Test-Path $map) {
            cmd /c "`"$Adb`" push `"$map`" `"/sdcard/Download/Nox/maps/$ow/$ow.map`" 2>nul" | Out-Null
            if ($LASTEXITCODE -ne 0) { Write-Host "phone: PUSH FAILED maps/$ow/$ow.map" }
            else { Write-Host "phone: maps/$ow/$ow.map pushed" }
        }
    }
    # Scripts from the repo.
    Get-ChildItem $world -Directory | ForEach-Object {
        Get-ChildItem $_.FullName -File | ForEach-Object {
            $rel = "$($_.Directory.Name)/$($_.Name)"
            cmd /c "`"$Adb`" push `"$($_.FullName)`" `"/sdcard/Download/Nox/maps/$rel`" 2>nul" | Out-Null
            if ($LASTEXITCODE -ne 0) { Write-Host "phone: PUSH FAILED maps/$rel" }
            else { Write-Host "phone: maps/$rel pushed" }
        }
    }
    $ErrorActionPreference = "Stop"
    Write-Host "Note: restart the app so the data mirror picks up new files."
} else {
    Write-Host "phone: not attached, skipped"
}
