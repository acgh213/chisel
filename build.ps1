# chisel build script (Windows PowerShell)
# Produces a self-contained directory under dist/

$ErrorActionPreference = "Stop"

Write-Host "building chisel..." -ForegroundColor Cyan

# 1. Build the Go binary.
go build -ldflags "-s -w" -o chisel.exe .
if (-not $?) { throw "go build failed" }

# 2. Create dist directory.
$dist = "dist\chisel"
Remove-Item -Recurse -Force $dist -ErrorAction SilentlyContinue
New-Item -ItemType Directory -Force $dist | Out-Null

# 3. Copy binary and Python backend.
Copy-Item chisel.exe $dist\
Copy-Item chisel.py $dist\

# 4. Copy docs.
Copy-Item README.md $dist\
Copy-Item CHANGELOG.md $dist\

Write-Host "packaged in $dist\" -ForegroundColor Green
Write-Host "  chisel.exe"
Write-Host "  chisel.py"
Write-Host "  README.md"
Write-Host "  CHANGELOG.md"
Write-Host ""
Write-Host "to run:  .\$dist\chisel.exe new my-novel"
Write-Host "         .$dist\chisel.exe my-novel"
Write-Host ""
Write-Host "notes:" -ForegroundColor Yellow
Write-Host "  - the TUI runs standalone (no dependencies)"
Write-Host "  - LLM features need Python 3 + 'pip install openai'"
