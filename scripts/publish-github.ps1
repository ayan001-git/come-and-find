# Publish a prepared complete project to the review branch from Windows PowerShell.
# Requires Git for Windows and authorization to push to ayan001-git/come-and-find.
$ErrorActionPreference = 'Stop'
$source = (Resolve-Path (Join-Path $PSScriptRoot '..')).Path
$branch = 'upgrade/enhanced-mvp-2026-11'
$target = Join-Path (Split-Path $source -Parent) 'come-and-find-github-upload'
if (-not (Get-Command git -ErrorAction SilentlyContinue)) {
  throw 'Install Git for Windows and restart PowerShell.'
}
if (Test-Path $target) { throw "The folder $target already exists. Rename it or resume there." }
git clone --branch $branch https://github.com/ayan001-git/come-and-find.git $target
if ($LASTEXITCODE -ne 0) { throw 'Clone failed; check GitHub login.' }
Get-ChildItem -LiteralPath $source -Force -Recurse -File | ForEach-Object {
  $relative = $_.FullName.Substring($source.Length).TrimStart([char]'\', [char]'/')
  if (($relative -split '[\\/]') | Where-Object { $_ -in @('.git','node_modules','__pycache__','.vscode','.venv') }) { return }
  if ($_.Name -eq '.env' -or $_.Name -like '*.pyc' -or $_.Extension -eq '.exe') { return }
  $dest = Join-Path $target $relative
  New-Item -ItemType Directory -Force -Path (Split-Path $dest -Parent) | Out-Null
  Copy-Item -LiteralPath $_.FullName -Destination $dest -Force
}
Push-Location $target
try {
  git add -A
  git diff --cached --stat
  git diff --cached --quiet
  if ($LASTEXITCODE -eq 0) { Write-Host 'No changes detected.'; exit 0 }
  git commit -m 'feat: enhanced Come and Find MVP with tests and docs'
  if ($LASTEXITCODE -ne 0) { throw 'Commit failed. Configure your Git identity.' }
  git push -u origin $branch
  if ($LASTEXITCODE -ne 0) { throw 'Push failed. Check your GitHub authentication.' }
  Write-Host "Uploaded: https://github.com/ayan001-git/come-and-find/tree/$branch"
} finally { Pop-Location }
