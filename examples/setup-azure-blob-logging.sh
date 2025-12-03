#!/usr/bin/env bash
# setup-azure-blob-logging.sh
# Helper script to create an Azure append blob for diagnose-mcp logging
# NOTE: Requires an existing Azure Storage Account

set -e

# Default values
STORAGE_ACCOUNT=""
CONTAINER_NAME="diagnose-mcp-logs"
BLOB_NAME="dev-test-$(date +%Y%m%d).log"
SAS_EXPIRY_DAYS="1"

# Parse command-line flags
while [[ $# -gt 0 ]]; do
    case $1 in
        -a|--account)
            STORAGE_ACCOUNT="$2"
            shift 2
            ;;
        -c|--container)
            CONTAINER_NAME="$2"
            shift 2
            ;;
        -b|--blob)
            BLOB_NAME="$2"
            shift 2
            ;;
        -e|--expiry)
            SAS_EXPIRY_DAYS="$2"
            shift 2
            ;;
        -h|--help)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  -a, --account NAME      Storage account name (required)"
            echo "  -c, --container NAME    Container name (default: diagnose-mcp-logs)"
            echo "  -b, --blob NAME         Blob name (default: dev-test-YYYYMMDD.log)"
            echo "  -e, --expiry DAYS       SAS token expiry in days (default: 1)"
            echo "  -h, --help              Show this help message"
            echo ""
            echo "Example:"
            echo "  $0 --account mystorageaccount --container diagnose-mcp-logs --blob app.log --expiry 1"
            exit 0
            ;;
        *)
            echo "❌ Unknown option: $1"
            echo "Run '$0 --help' for usage information."
            exit 1
            ;;
    esac
done

echo "🚀 Setting up Azure Blob for diagnose-mcp logging..."
echo ""

# Check if Azure CLI is installed
if ! command -v az &> /dev/null; then
    echo "❌ Azure CLI is not installed. Please install it from:"
    echo "   https://docs.microsoft.com/en-us/cli/azure/install-azure-cli"
    exit 1
fi

# Check if logged in
if ! az account show &> /dev/null; then
    echo "❌ Not logged in to Azure. Please run: az login"
    exit 1
fi

# Validate required parameters
if [ -z "$STORAGE_ACCOUNT" ]; then
    echo "❌ Storage account name is required."
    echo "Run '$0 --help' for usage information."
    exit 1
fi

# Verify storage account exists
echo "🔍 Verifying storage account '$STORAGE_ACCOUNT'..."
if ! az storage account show --name "$STORAGE_ACCOUNT" &> /dev/null; then
    echo "❌ Storage account '$STORAGE_ACCOUNT' not found."
    echo "   Please create it first or specify a different account."
    exit 1
fi

echo "Configuration:"
echo "  Storage Account: $STORAGE_ACCOUNT"
echo "  Container: $CONTAINER_NAME"
echo "  Blob: $BLOB_NAME"
echo "  SAS Token Expiry: $SAS_EXPIRY_DAYS days"
echo ""

# Get account key
ACCOUNT_KEY=$(az storage account keys list \
    --account-name "$STORAGE_ACCOUNT" \
    --query '[0].value' \
    --output tsv)

# Create container (if not exists)
echo "📦 Creating container '$CONTAINER_NAME'..."
az storage container create \
    --name "$CONTAINER_NAME" \
    --account-name "$STORAGE_ACCOUNT" \
    --account-key "$ACCOUNT_KEY" \
    --output none 2>/dev/null || true

# Create append blob with initial header
echo "📝 Creating append blob '$BLOB_NAME'..."
TEMP_FILE=$(mktemp)
# Write initial header to make blob non-empty
printf "# diagnose-mcp log file\n# Created: %s\n\n" "$(date '+%Y-%m-%d %H:%M:%S')" > "$TEMP_FILE"
az storage blob upload \
    --account-name "$STORAGE_ACCOUNT" \
    --account-key "$ACCOUNT_KEY" \
    --container-name "$CONTAINER_NAME" \
    --name "$BLOB_NAME" \
    --type append \
    --file "$TEMP_FILE" \
    --output none 2>/dev/null || echo "  Blob might already exist, continuing..."
rm -f "$TEMP_FILE"

# Generate SAS token
echo "🔑 Generating SAS token (expires in $SAS_EXPIRY_DAYS days)..."
END_DATE=$(date -u -d "$SAS_EXPIRY_DAYS days" '+%Y-%m-%dT%H:%MZ' 2>/dev/null || date -u -v +${SAS_EXPIRY_DAYS}d '+%Y-%m-%dT%H:%MZ')

SAS_TOKEN=$(az storage blob generate-sas \
    --account-name "$STORAGE_ACCOUNT" \
    --account-key "$ACCOUNT_KEY" \
    --container-name "$CONTAINER_NAME" \
    --name "$BLOB_NAME" \
    --permissions acw \
    --expiry "$END_DATE" \
    --https-only \
    --output tsv)

# Build full URL
BLOB_URL="https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER_NAME}/${BLOB_NAME}?${SAS_TOKEN}"

echo ""
echo "✅ Setup complete!"
echo ""
echo "📋 Blob URL (with SAS token):"
echo "$BLOB_URL"
echo ""
echo "💡 Usage:"
echo "  export LOG_BLOB_URL='$BLOB_URL'"
echo "  diagnose-mcp --log-blob-url \"\$LOG_BLOB_URL\" ./my-mcp-server"
echo ""
echo "📥 View logs:"
echo "  az storage blob download --account-name $STORAGE_ACCOUNT --container-name $CONTAINER_NAME --name $BLOB_NAME --file -"
echo ""
echo "🗑️  Cleanup:"
echo "  az storage blob delete --account-name $STORAGE_ACCOUNT --container-name $CONTAINER_NAME --name $BLOB_NAME"
echo ""
