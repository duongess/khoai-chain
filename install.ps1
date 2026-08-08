param (
    [string]$Version = "latest"
)

$Repo = "duongess/khoai-chain"

if ($Version -eq "latest" -or [string]::IsNullOrEmpty($Version)) {
    $ReleaseInfo = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
    $Version = $ReleaseInfo.tag_name
}

$FileName = "khoai-src-$Version.zip"
$DownloadUrl = "https://github.com/$Repo/releases/download/$Version/$FileName"

Invoke-WebRequest -Uri $DownloadUrl -OutFile $FileName