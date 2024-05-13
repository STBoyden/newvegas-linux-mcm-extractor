if (!(Get-Command just -ErrorAction SilentlyContinue))
{
    Write-Output "just.exe not found in path. Please install."
    exit 1
}

just fmt