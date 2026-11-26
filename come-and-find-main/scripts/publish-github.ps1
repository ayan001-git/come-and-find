# Run from PowerShell after extracting this full project.
# Requires Git for Windows and GitHub sign-in configured with write permission.
# Publishes to a review branch and does NOT overwrite the main branch.
$ErrorActionPreference = 'Stop'
$source = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$branch = 'upgrade/enhanced-mvp-2026-11'
$remote = 'https://github.com/ayan001-git/come-and-find.git'
$target = Join-Path (Split-Path $source -Parent) 'come-and-find-github-upload'

if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
    throw 'Git for Windows is not installed or not available in PATH. Install Git and open a new PowerShell terminal.'
}
if (Test-Path $target) {
    throw "Upload folder already exists: $target. Rename it first, or use that repo to resume."
}
Write-Host "Cloning review branch: $branch"
git clone --branch $branch $remote $target
if ($LASTEXITCODE -ne 0) { throw 'Clone failed. Check GitHub authentication and connection.' }

$files = Get-ChildItem -LiteralPath $source -Recurse -Force -File
foreach ($file in $files) {
    $relative = $file.FullName.Substring($source.Length).TrimStart([char]'\', [char]'/')
    $segments = $relative -split '[\\/]'
    if ($segments | Where-Object { $_ -in @('.git','node_modules','__pycache__','.venv','.vscode') }) { continue }
    if ($file.Name -eq '.env' -or $file.Name -like '*.pyc' -or $file.Extension -eq '.exe') { continue }
    $dst = Join-Path $target $relative
    New-Item -ItemType Directory -Force -Path (Split-Path $dst -Parent) | Out-Null
    Copy-Item -LiteralPath $file.FullName -Destination $dst -Force
}

Push-Location $target
try {
    git add -A
    Write-Host 'Files staged for review:'
    git diff --cached --stat
    git diff --cached --quiet
    if ($LASTEXITCODE -eq 0) {
        Write-Host 'Nothing changed; review branch already has these files.'
        exit 0
    }
    git commit -m 'feat: upload enhanced Come and Find MVP and project documentation'
    if ($LASTEXITCODE -ne 0) { throw 'Commit failed. Configure git user.name / user.email and try again in the upload folder.' }
    git push -u origin $branch
    if ($LASTEXITCODE -ne 0) { throw 'Push failed. Check GitHub sign-in or permissions.' }
    Write-Host "Published review branch: https://github.com/ayan001-git/come-and-find/tree/$branch"
} finally { Pop-Location }
