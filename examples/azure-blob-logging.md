# Azure Blob Storage Logging for Dev/Test

This example demonstrates how to stream `diagnose-mcp` logs to Azure Blob Storage instead of local files. This is useful when:

- The target environment cannot export log files
- You need centralized log collection for multiple test instances
- You want persistent logs without local disk access

## Prerequisites

1. **Azure Storage Account**: You must have an existing storage account
2. **Azure CLI**: Install from https://docs.microsoft.com/cli/azure/install-azure-cli
3. **Azure Login**: Run `az login` before using the setup scripts

## Quick Setup

### 1. Create Storage Account (if needed)

If you don't have a storage account yet, create one via Azure Portal or CLI:

```bash
# Create resource group (if needed)
az group create --name my-resource-group --location eastus

# Create storage account
az storage account create \
  --name mystorageaccount \
  --resource-group my-resource-group \
  --location eastus \
  --sku Standard_LRS
```

### 2. Run Setup Script

The script will create a container, append blob, and SAS token in your existing storage account:

**Bash:**
```bash
# Run with your storage account name
./examples/setup-azure-blob-logging.sh --account mystorageaccount

# Customize container, blob name, and expiry
./examples/setup-azure-blob-logging.sh --account mystorageaccount \
    --container diagnose-mcp-logs \
    --blob app.log \
    --expiry 1
```

**PowerShell (Windows):**
```powershell
# Run with your storage account name
.\examples\setup-azure-blob-logging.ps1 -StorageAccount mystorageaccount

# Customize container, blob name, and expiry
.\examples\setup-azure-blob-logging.ps1 -StorageAccount mystorageaccount `
    -ContainerName diagnose-mcp-logs `
    -BlobName app.log `
    -SasExpiryDays 1
```

The script will:
- ✅ Verify your storage account exists
- ✅ Create a `diagnose-mcp-logs` container (if needed)
- ✅ Create an append blob (e.g., `dev-test-20251203.log`)
- ✅ Generate a SAS token with write permissions (1 day expiry)
- ✅ Output the full blob URL ready to use

### 3. Use the Generated URL

## Usage

### Basic Usage

```bash
# Stream logs to Azure Blob
diagnose-mcp --log-blob-url "https://mystorageaccount.blob.core.windows.net/logs/dev-test.log?<sas-token>" ./my-mcp-server

# With verbose logging
diagnose-mcp --log-blob-url "https://..." --verbose ./my-mcp-server

# JSON format
diagnose-mcp --log-blob-url "https://..." --json ./my-mcp-server
```

### Using Environment Variables (Recommended for CI/CD)

```bash
# Set blob URL in environment variable
export LOG_BLOB_URL="https://mystorageaccount.blob.core.windows.net/logs/dev-test.log?<sas-token>"

# Reference in script or tool
diagnose-mcp --log-blob-url "$LOG_BLOB_URL" ./my-mcp-server
```

## Configuration Tips

### Buffer Settings

The blob writer batches log writes to reduce API calls:
- **Default flush size**: 64 KB
- **Default flush period**: 5 seconds
- Logs are flushed when buffer is full OR every 5 seconds

For **low-latency dev mode** (immediate writes), you can modify `internal/logger/blobwriter.go`:

```go
config := &BlobWriterConfig{
    FlushOnWrite: true,  // Flush immediately after each log line
}
```

### Multiple Test Instances

Use unique blob names for parallel tests:

```bash
# Instance 1
diagnose-mcp --log-blob-url "https://.../logs/test-instance-1.log?..." ./server

# Instance 2
diagnose-mcp --log-blob-url "https://.../logs/test-instance-2.log?..." ./server
```

### Viewing Logs

Download logs from Azure Portal or CLI:

```bash
# View logs in terminal
az storage blob download \
  --account-name mystorageaccount \
  --container-name logs \
  --name dev-test.log \
  --file - \
  | tail -n 50

# Download to file
az storage blob download \
  --account-name mystorageaccount \
  --container-name logs \
  --name dev-test.log \
  --file local-dev-test.log
```

## Security Best Practices

1. **SAS Token Permissions**: Grant minimal permissions (`acw` = add, create, write)
2. **Expiry Time**: Use short-lived tokens (hours/days for dev, not months)
3. **HTTPS Only**: Always use `--https-only` when generating SAS
4. **Rotation**: Regenerate tokens regularly, especially before they expire
5. **Secrets Management**: Store SAS URLs in:
   - Environment variables (CI/CD)
   - Azure Key Vault (production-like dev)
   - `.env` files (local dev, add to `.gitignore`)

## Cleanup

```bash
# Delete blob only (keeps container and storage account)
az storage blob delete \
  --account-name mystorageaccount \
  --container-name logs \
  --name dev-test.log

# Or delete entire container
az storage container delete \
  --name logs \
  --account-name mystorageaccount
```

## Troubleshooting

### "Failed to create blob client"
- Check that the blob URL is properly URL-encoded
- Verify the SAS token hasn't expired
- Ensure the blob exists and is of type `AppendBlob`

### "Failed to append to blob"
- SAS token may lack write (`w`) permissions
- Blob might not exist or is not an append blob
- Storage account may have network restrictions

### No logs appearing
- Check blob writer is flushing (default: every 5 seconds or 64KB)
- Verify network connectivity to `*.blob.core.windows.net`
- Check stderr for warnings about blob writer initialization

## Cost Considerations

Azure Blob Storage costs for dev/test are minimal:
- **Storage**: ~$0.018/GB/month for LRS (locally redundant)
- **Operations**: ~$0.065 per 10,000 write operations
- **Bandwidth**: Free egress for same-region, small charges for cross-region

**Example**: Streaming 100 MB of logs/day with frequent writes:
- Storage: ~$0.002/month
- Operations: ~$0.01/month
- **Total**: < $0.02/month for moderate dev/test usage

## Alternative: Block Blob (Not Recommended for Logs)

While block blobs are more common, append blobs are better for logs because:
- Append blobs support efficient append-only operations
- No need to download, modify, and re-upload
- Better concurrency for log streaming
- Designed specifically for scenarios like logging

---

**For production logging**, consider Azure Monitor / Log Analytics instead of raw blob storage for better querying, retention policies, and alerting.
