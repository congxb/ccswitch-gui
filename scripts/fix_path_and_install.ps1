$npmPrefix = npm config get prefix
Write-Host "npm prefix: $npmPrefix"

$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')

if ($userPath -notlike "*$npmPrefix*") {
    $newPath = $userPath + ";" + $npmPrefix
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    Write-Host "PATH added: $npmPrefix"
} else {
    Write-Host "PATH already has npm prefix"
}

$env:Path += ";$npmPrefix"
Write-Host "Session PATH updated"

Write-Host ""
Write-Host "Installing Claude Code..."
npm install -g @anthropic-ai/claude-code

Write-Host ""
Write-Host "Verifying..."
$env:Path += ";$npmPrefix"

$found = Get-Command claude -ErrorAction SilentlyContinue
if ($found) {
    Write-Host "Found: $($found.Source)"
    claude --version
} else {
    Write-Host "Not found via PATH, checking direct..."
    $paths = @(
        (Join-Path $npmPrefix "claude.cmd"),
        (Join-Path $npmPrefix "claude")
    )
    foreach ($p in $paths) {
        if (Test-Path $p) {
            Write-Host "Found at: $p"
            & $p --version
            break
        }
    }
}

Write-Host ""
Write-Host "=== DONE ==="
