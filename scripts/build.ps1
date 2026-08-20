param(
    [string]$Version = "0.5.0"
)

$ErrorActionPreference = "Stop"
$projectRoot = Split-Path -Parent $PSScriptRoot
$portableGo = Join-Path $projectRoot ".tools\go\bin\go.exe"

if (Test-Path -LiteralPath $portableGo) {
    $go = $portableGo
} else {
    $goCommand = Get-Command go -ErrorAction SilentlyContinue
    if (-not $goCommand) {
        throw "Go não encontrado. Instale Go 1.24+ ou coloque a toolchain em .tools\go."
    }
    $go = $goCommand.Source
}

$dist = Join-Path $projectRoot "dist"
$payloadDirectory = Join-Path $projectRoot "installer\payload"
$launcherPayload = Join-Path $payloadDirectory "zyr.exe"
$gitComponentPayload = Join-Path $payloadDirectory "zyr-git-commit.exe"
$setup = Join-Path $dist "ZyrGitCommit-Setup.exe"
New-Item -ItemType Directory -Force -Path $dist | Out-Null
New-Item -ItemType Directory -Force -Path $payloadDirectory | Out-Null

# dist/ contém somente artefatos gerados. Remover executáveis anteriores evita
# que builds de versões diferentes deixem mais de um instalador na entrega.
Get-ChildItem -LiteralPath $dist -File -Filter "*.exe" -ErrorAction SilentlyContinue |
    ForEach-Object { [System.IO.File]::Delete($_.FullName) }

$previousCgoEnabled = $env:CGO_ENABLED
Push-Location $projectRoot
try {
    $env:CGO_ENABLED = "0"
    & $go test ./...
    if ($LASTEXITCODE -ne 0) { throw "Os testes falharam." }

    & $go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $launcherPayload ./cmd/zyr
    if ($LASTEXITCODE -ne 0) { throw "Falha ao compilar o launcher zyr.exe." }

    & $go build -trimpath -ldflags "-s -w -X main.version=$Version" -o $gitComponentPayload ./cmd/zyr-git-component
    if ($LASTEXITCODE -ne 0) { throw "Falha ao compilar o componente Git Commit." }

    & $go build -tags installerbuild -trimpath -ldflags "-s -w -X main.version=$Version" -o $setup ./installer
    if ($LASTEXITCODE -ne 0) { throw "Falha ao compilar o instalador." }

    $artifacts = @((Get-Item -LiteralPath $setup))
    $lines = foreach ($artifact in $artifacts) {
        $hash = (Get-FileHash -Algorithm SHA256 -LiteralPath $artifact.FullName).Hash.ToLowerInvariant()
        "$hash  $($artifact.Name)"
    }
    Set-Content -LiteralPath (Join-Path $dist "SHA256SUMS.txt") -Value $lines -Encoding utf8
    Write-Host "Build concluído em $dist"
} finally {
    if (Test-Path -LiteralPath $launcherPayload) {
        [System.IO.File]::Delete($launcherPayload)
    }
    if (Test-Path -LiteralPath $gitComponentPayload) {
        [System.IO.File]::Delete($gitComponentPayload)
    }
    if ($null -eq $previousCgoEnabled) {
        Remove-Item Env:CGO_ENABLED -ErrorAction SilentlyContinue
    } else {
        $env:CGO_ENABLED = $previousCgoEnabled
    }
    Pop-Location
}
