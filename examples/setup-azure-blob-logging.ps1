# setup-azure-blob-logging.ps1
# Helper script to create an Azure append blob for diagnose-mcp logging
# NOTE: Requires an existing Azure Storage Account

<#
.SYNOPSIS
    Creates an Azure append blob for diagnose-mcp logging.

.DESCRIPTION
    This script creates a container and append blob in an existing Azure Storage Account,
    then generates a SAS token for diagnose-mcp to write logs.

.PARAMETER StorageAccount
    The name of the existing Azure Storage Account (required).

.PARAMETER ContainerName
    The name of the container to create or use (default: "diagnose-mcp-logs").

.PARAMETER BlobName
    The name of the append blob to create (default: "dev-test-YYYYMMDD.log").

.PARAMETER SasExpiryDays
    Number of days until the SAS token expires (default: 1).

.EXAMPLE
    .\setup-azure-blob-logging.ps1 -StorageAccount mystorageaccount
    .\setup-azure-blob-logging.ps1 -StorageAccount mystorageaccount -ContainerName diagnose-mcp-logs -BlobName app.log -SasExpiryDays 1
#>

param(
    [Parameter(Mandatory=$true, HelpMessage="Name of the existing Azure Storage Account")]
    [string]$StorageAccount,

    [Parameter(Mandatory=$false)]
    [string]$ContainerName = "diagnose-mcp-logs",

    [Parameter(Mandatory=$false)]
    [string]$BlobName = "dev-test-$(Get-Date -Format 'yyyyMMdd').log",

    [Parameter(Mandatory=$false)]
    [int]$SasExpiryDays = 1
)

Write-Host "🚀 Setting up Azure Blob for diagnose-mcp logging..." -ForegroundColor Cyan
Write-Host ""

# Check if Azure CLI is installed
if (-not (Get-Command az -ErrorAction SilentlyContinue)) {
    Write-Host "❌ Azure CLI is not installed. Please install it from:" -ForegroundColor Red
    Write-Host "   https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
}

# Check if logged in
try {
    az account show 2>&1 | Out-Null
} catch {
    Write-Host "❌ Not logged in to Azure. Please run: az login" -ForegroundColor Red
    exit 1
}

# Verify storage account exists
Write-Host "🔍 Verifying storage account '$StorageAccount'..." -ForegroundColor Yellow
$accountExists = az storage account show --name $StorageAccount 2>$null
if (-not $accountExists) {
    Write-Host "❌ Storage account '$StorageAccount' not found." -ForegroundColor Red
    Write-Host "   Please create it first or specify a different account."
    exit 1
}

Write-Host "Configuration:"
Write-Host "  Storage Account: $StorageAccount"
Write-Host "  Container: $ContainerName"
Write-Host "  Blob: $BlobName"
Write-Host "  SAS Token Expiry: $SasExpiryDays days"
Write-Host ""

# Get account key
$AccountKey = az storage account keys list `
    --account-name $StorageAccount `
    --query '[0].value' `
    --output tsv

# Create container (if not exists)
Write-Host "📦 Creating container '$ContainerName'..." -ForegroundColor Yellow
az storage container create `
    --name $ContainerName `
    --account-name $StorageAccount `
    --account-key $AccountKey `
    --output none 2>$null

# Create append blob with initial header
Write-Host "📝 Creating append blob '$BlobName'..." -ForegroundColor Yellow
$tempFile = [System.IO.Path]::GetTempFileName()
try {
    # Write initial header to make blob non-empty
    $header = "# diagnose-mcp log file`n# Created: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')` n`n"
    [System.IO.File]::WriteAllText($tempFile, $header)
    
    az storage blob upload `
        --account-name $StorageAccount `
        --account-key $AccountKey `
        --container-name $ContainerName `
        --name $BlobName `
        --type append `
        --file $tempFile `
        --output none 2>&1 | Out-Null
    
    if ($LASTEXITCODE -ne 0) {
        Write-Host "  Blob might already exist, continuing..." -ForegroundColor Yellow
    }
} finally {
    Remove-Item $tempFile -ErrorAction SilentlyContinue
}

# Generate SAS token
Write-Host "🔑 Generating SAS token (expires in $SasExpiryDays days)..." -ForegroundColor Yellow
$EndDate = (Get-Date).AddDays($SasExpiryDays).ToString("yyyy-MM-ddTHH:mmZ")

$SasToken = az storage blob generate-sas `
    --account-name $StorageAccount `
    --account-key $AccountKey `
    --container-name $ContainerName `
    --name $BlobName `
    --permissions acw `
    --expiry $EndDate `
    --https-only `
    --output tsv

# Build full URL - use string concatenation to avoid interpolation issues
$BlobUrl = "https://$StorageAccount.blob.core.windows.net/$ContainerName/$BlobName" + "?" + $SasToken

Write-Host ""
Write-Host "✅ Setup complete!" -ForegroundColor Green
Write-Host ""
Write-Host "📋 Blob URL (with SAS token):" -ForegroundColor Cyan
Write-Host $BlobUrl -ForegroundColor White
Write-Host ""
Write-Host "💡 Usage:" -ForegroundColor Cyan
Write-Host "  `$env:LOG_BLOB_URL = '$BlobUrl'" -ForegroundColor White
Write-Host "  diagnose-mcp --log-blob-url `"`$env:LOG_BLOB_URL`" ./my-mcp-server" -ForegroundColor White
Write-Host ""
Write-Host "📥 View logs:" -ForegroundColor Cyan
Write-Host "  az storage blob download --account-name $StorageAccount --container-name $ContainerName --name $BlobName --file -" -ForegroundColor White
Write-Host ""
Write-Host "🗑️  Cleanup:" -ForegroundColor Cyan
Write-Host "  az storage blob delete --account-name $StorageAccount --container-name $ContainerName --name $BlobName" -ForegroundColor White
Write-Host ""
