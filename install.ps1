# soulink CLI 一键安装脚本（Windows，从 GitHub Releases 下载）。
#   irm https://www.soulink.cam/install.ps1 | iex
# 先审计再执行：
#   irm https://www.soulink.cam/install.ps1 -OutFile install.ps1
#   notepad install.ps1   # 审查后执行：.\install.ps1
# 环境变量：
#   SOULINK_INSTALL_DIR    安装目录（默认 $env:LOCALAPPDATA\Programs\soulink，无需管理员）
#   SOULINK_VERSION        指定版本（默认经 GitHub API 解析 latest）
#   SOULINK_GITHUB_MIRROR  显式指定镜像域名（如 gh-proxy.com，不带 https:// 前缀）
# 下载源顺序：SOULINK_GITHUB_MIRROR > github.com 直连 > 内置镜像（gh-proxy.com、ghproxy.net）。
# 国内直连超时属正常，脚本自动回落镜像；无论走哪个源都做 sha256 强校验。
# 需要 PowerShell 5.1+ 或 PowerShell 7+。

$ErrorActionPreference = 'Stop'

function Err([string]$Message) {
    [Console]::Error.WriteLine("错误：$Message")
    exit 1
}

function Get-SoulinkArch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    if ([string]::IsNullOrEmpty($arch)) {
        Err '无法识别架构（PROCESSOR_ARCHITECTURE 为空）'
    }
    switch -Regex ($arch) {
        '^(AMD64|X64|x86_64)$' { return 'amd64' }
        '^(ARM64|aarch64)$' { return 'arm64' }
        default { Err "不支持的架构：$arch（仅支持 amd64 / arm64）" }
    }
}

function Resolve-SoulinkVersion([string]$Repo) {
    $version = $env:SOULINK_VERSION
    if (-not [string]::IsNullOrWhiteSpace($version)) {
        return $version.Trim()
    }
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest" -UseBasicParsing
    } catch {
        Err "查询最新版本失败；可设 SOULINK_VERSION=vX.Y.Z 跳过 GitHub API"
    }
    if (-not $release.tag_name) {
        Err '解析最新版本号失败'
    }
    return [string]$release.tag_name
}

function Get-ChecksumFromFile([string]$ChecksumFile, [string]$AssetName) {
    foreach ($line in Get-Content -LiteralPath $ChecksumFile) {
        if ([string]::IsNullOrWhiteSpace($line)) { continue }
        $parts = $line.Trim() -split '\s+', 2
        if ($parts.Count -eq 2 -and $parts[1] -eq $AssetName) {
            return $parts[0].ToLowerInvariant()
        }
    }
    return $null
}

function Show-PathNotice([string]$BinName, [string]$InstallDir) {
    $pathEntries = $env:PATH -split ';' | ForEach-Object { $_.TrimEnd('\') }
    $normalized = $InstallDir.TrimEnd('\')
    foreach ($entry in $pathEntries) {
        if ($entry -and [string]::Equals($entry, $normalized, [System.StringComparison]::OrdinalIgnoreCase)) {
            return
        }
    }
    Write-Host "提示：$InstallDir 不在 PATH 中。可执行以下命令加入用户 PATH（新开终端生效）："
    Write-Host "  [Environment]::SetEnvironmentVariable('PATH', `$env:PATH + ';$InstallDir', 'User')"
}

# Windows PowerShell 5.1 默认不启用 TLS 1.2，补上。
try {
    [Net.ServicePointManager]::SecurityProtocol = `
        [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12
} catch {
    # 运行时已协商现代 TLS 则忽略。
}

$Repo = 'soulink9/soulink'
$Bin = 'soulink.exe'
$InstallDir = if (-not [string]::IsNullOrWhiteSpace($env:SOULINK_INSTALL_DIR)) {
    $env:SOULINK_INSTALL_DIR.Trim()
} else {
    Join-Path $env:LOCALAPPDATA 'Programs\soulink'
}

$arch = Get-SoulinkArch
$Version = Resolve-SoulinkVersion $Repo
$pkg = "soulink_${Version}_windows_${arch}.zip"

$Mirror = if (-not [string]::IsNullOrWhiteSpace($env:SOULINK_GITHUB_MIRROR)) {
    $env:SOULINK_GITHUB_MIRROR.Trim() -replace '^https?://' -replace '/$'
} else {
    $null
}
if ($Mirror -and $Mirror -match '\s') {
    Err "SOULINK_GITHUB_MIRROR 含有空格：'$Mirror'"
}
# 候选下载源：显式镜像 > 直连 > 内置镜像
$BuiltinMirrors = @('gh-proxy.com', 'ghproxy.net')
$bases = @()
if ($Mirror) { $bases += "https://$Mirror/github.com/$Repo/releases/download/$Version" }
$bases += "https://github.com/$Repo/releases/download/$Version"
foreach ($m in $BuiltinMirrors) { $bases += "https://$m/github.com/$Repo/releases/download/$Version" }

$tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("soulink-install-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Force -Path $tmp | Out-Null

try {
    $pkgPath = Join-Path $tmp $pkg
    $sumPath = Join-Path $tmp 'checksums.txt'

    Write-Host "下载 soulink $Version (windows/$arch)..."
    $base = $null
    foreach ($candidate in $bases) {
        try {
            Invoke-WebRequest -Uri "$candidate/$pkg" -OutFile $pkgPath -UseBasicParsing -TimeoutSec 1800
            $base = $candidate
            break
        } catch {
            [Console]::Error.WriteLine("提示：$candidate 不可达，尝试下一个源")
        }
    }
    if (-not $base) {
        Err '所有下载源均失败；可设 SOULINK_GITHUB_MIRROR=<镜像域名> 后重试'
    }
    if (-not $base.StartsWith('https://github.com/')) {
        [Console]::Error.WriteLine("提示：经由第三方镜像 `"$base`" 下载（完整性由 sha256 校验）")
    }

    try {
        Invoke-WebRequest -Uri "$base/checksums.txt" -OutFile $sumPath -UseBasicParsing -TimeoutSec 15
    } catch {
        Err "checksums.txt 下载失败（$base）"
    }

    $want = Get-ChecksumFromFile $sumPath $pkg
    if ([string]::IsNullOrEmpty($want)) {
        Err "checksums.txt 中找不到 $pkg"
    }
    $got = (Get-FileHash -LiteralPath $pkgPath -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($got -ne $want) {
        Err "校验和不匹配（got $got, want $want）"
    }

    Expand-Archive -LiteralPath $pkgPath -DestinationPath $tmp -Force
    $binary = Join-Path $tmp "soulink-windows-$arch.exe"
    if (-not (Test-Path -LiteralPath $binary)) {
        Err "压缩包内找不到 soulink-windows-$arch.exe"
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    Copy-Item -LiteralPath $binary -Destination (Join-Path $InstallDir $Bin) -Force

    Write-Host "已安装 soulink $Version -> $InstallDir\$Bin"
    Show-PathNotice $Bin $InstallDir
    Write-Host ""
    Write-Host "直接使用（匿名、免注册）："
    Write-Host "  soulink 3000"
    Write-Host ""
    Write-Host "固定域名：到 https://hub.soulink.cam 注册后执行"
    Write-Host "  soulink config set token <token>"
} finally {
    Remove-Item -LiteralPath $tmp -Recurse -Force -ErrorAction SilentlyContinue
}
