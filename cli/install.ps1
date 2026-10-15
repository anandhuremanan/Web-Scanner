# Installs the latest webscanner command-line tool on Windows.
#
#   irm https://raw.githubusercontent.com/anandhuremanan/website_mapper/main/cli/install.ps1 | iex
#
# Settings (optional, as environment variables):
#   WEBSCANNER_VERSION   a version such as 1.0.0 (default: the latest)
#   INSTALL_DIR          where to put the program
#                        (default: %LOCALAPPDATA%\Programs\webscanner)

$ErrorActionPreference = 'Stop'
$ProgressPreference = 'SilentlyContinue'
$repo = 'anandhuremanan/website_mapper'

function Fail($message) {
    Write-Host "webscanner install: $message" -ForegroundColor Red
    exit 1
}

switch ($env:PROCESSOR_ARCHITECTURE) {
    'AMD64' { $arch = 'amd64' }
    'ARM64' { $arch = 'arm64' }
    default { Fail "unsupported processor $($env:PROCESSOR_ARCHITECTURE)" }
}

if ($env:WEBSCANNER_VERSION) {
    $version = $env:WEBSCANNER_VERSION.TrimStart('v')
} else {
    try {
        $releases = Invoke-RestMethod "https://api.github.com/repos/$repo/releases?per_page=30"
    } catch {
        Fail "could not look up the latest release: $_"
    }
    $release = $releases | Where-Object { $_.tag_name -like 'cli-v*' -and -not $_.draft } | Select-Object -First 1
    if (-not $release) { Fail 'no release of the command-line tool has been published yet' }
    $version = $release.tag_name.Substring(5)
}

$name = "webscanner_${version}_windows_${arch}.zip"
$base = "https://github.com/$repo/releases/download/cli-v$version"
$work = Join-Path ([IO.Path]::GetTempPath()) ("webscanner-" + [guid]::NewGuid())
New-Item -ItemType Directory -Path $work | Out-Null

try {
    Write-Host "Downloading webscanner $version for windows/$arch..."
    Invoke-WebRequest "$base/$name" -OutFile (Join-Path $work $name) -UseBasicParsing
    Invoke-WebRequest "$base/checksums.txt" -OutFile (Join-Path $work 'checksums.txt') -UseBasicParsing

    # Check the download against the published checksum before running anything.
    $line = Get-Content (Join-Path $work 'checksums.txt') | Where-Object { $_ -match " $([regex]::Escape($name))$" } | Select-Object -First 1
    if (-not $line) { Fail "$name is not in checksums.txt" }
    $expected = ($line -split ' ')[0].ToLower()
    $actual = (Get-FileHash (Join-Path $work $name) -Algorithm SHA256).Hash.ToLower()
    if ($expected -ne $actual) { Fail 'the download does not match its checksum; not installing it' }

    Expand-Archive (Join-Path $work $name) -DestinationPath $work -Force

    $dir = $env:INSTALL_DIR
    if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\webscanner' }
    New-Item -ItemType Directory -Force -Path $dir | Out-Null
    Copy-Item (Join-Path $work 'webscanner.exe') (Join-Path $dir 'webscanner.exe') -Force
} finally {
    Remove-Item -Recurse -Force $work -ErrorAction SilentlyContinue
}

# Put the folder on this user's PATH, once.
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if (-not (($userPath -split ';') -contains $dir)) {
    [Environment]::SetEnvironmentVariable('Path', ($userPath.TrimEnd(';') + ";$dir").TrimStart(';'), 'User')
    Write-Host "Added $dir to your PATH; open a new terminal to use it."
}
Write-Host "Installed webscanner $version to $dir\webscanner.exe"
Write-Host 'Try: webscanner -mode passive example.com'
