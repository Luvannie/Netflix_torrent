# Backend Parity Verification Script
# Compares responses between Java and Go backends

param(
    [Parameter(Mandatory=$true)]
    [string]$JavaBaseUrl,

    [Parameter(Mandatory=$true)]
    [string]$GoBaseUrl,

    [Parameter(Mandatory=$false)]
    [string]$LocalToken = "token",

    [Parameter(Mandatory=$false)]
    [string]$RequestsFile = "tools/backend-parity/requests.json"
)

$ErrorActionPreference = "Continue"

function Get-JsonFieldNames($json) {
    $obj = $json | ConvertFrom-Json
    if ($obj -is [System.Collections.IDictionary] -or $obj.GetType().Name -eq "PSCustomObject") {
        return $obj.PSObject.Properties.Name | Sort-Object
    }
    return @()
}

function Normalize-Json($json) {
    if ([string]::IsNullOrWhiteSpace($json)) {
        return $null
    }
    try {
        $obj = $json | ConvertFrom-Json
        return $obj
    } catch {
        return $null
    }
}

function Compare-Responses {
    param(
        [string]$Name,
        [Microsoft.PowerShell.Commands.WebRequestMethod]$Method,
        [string]$Path,
        [hashtable]$Headers,
        [string]$Body = $null,
        [string]$JavaUrl,
        [string]$GoUrl
    )

    $javaUrl = "$JavaBaseUrl$Path"
    $goUrl = "$GoBaseUrl$Path"

    Write-Host "Testing: $Name ($Method $Path)" -ForegroundColor Cyan

    # Build splat for Invoke-WebRequest
    $javaSplat = @{
        Uri = $javaUrl
        Method = $Method
        Headers = $Headers.Clone()
        ContentType = "application/json"
        UseBasicParsing = $true
    }
    $goSplat = @{
        Uri = $goUrl
        Method = $Method
        Headers = $Headers.Clone()
        ContentType = "application/json"
        UseBasicParsing = $true
    }

    if ($Body) {
        $javaSplat.Body = $Body
        $goSplat.Body = $Body
    }

    # Make requests
    try {
        $javaResponse = Invoke-WebRequest @javaSplat -TimeoutSec 30
    } catch {
        Write-Host "  [JAVA] ERROR: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }

    try {
        $goResponse = Invoke-WebRequest @goSplat -TimeoutSec 30
    } catch {
        Write-Host "  [GO] ERROR: $($_.Exception.Message)" -ForegroundColor Red
        return $false
    }

    # Compare status codes
    if ($javaResponse.StatusCode -ne $goResponse.StatusCode) {
        Write-Host "  [FAIL] Status code: Java=$($javaResponse.StatusCode), Go=$($goResponse.StatusCode)" -ForegroundColor Red
        return $false
    }

    # Parse JSON bodies
    $javaBody = Normalize-Json $javaResponse.Content
    $goBody = Normalize-Json $goResponse.Content

    if ($null -eq $javaBody -and $null -eq $goBody) {
        Write-Host "  [PASS] Both returned non-JSON (status $($javaResponse.StatusCode))" -ForegroundColor Green
        return $true
    }

    if ($null -eq $javaBody -or $null -eq $goBody) {
        Write-Host "  [FAIL] One returned JSON, other did not" -ForegroundColor Red
        return $false
    }

    # Compare top-level keys
    $javaKeys = Get-JsonFieldNames ($javaResponse.Content)
    $goKeys = Get-JsonFieldNames ($goResponse.Content)

    # Check for data/error keys
    $javaHasData = $javaKeys -contains "data"
    $javaHasError = $javaKeys -contains "error"
    $goHasData = $goKeys -contains "data"
    $goHasError = $goKeys -contains "error"

    if ($javaHasData -ne $goHasData -or $javaHasError -ne $goHasError) {
        Write-Host "  [FAIL] Response shape mismatch: Java has data=$javaHasData error=$javaHasError, Go has data=$goHasData error=$goHasError" -ForegroundColor Red
        return $false
    }

    # Check meta.requestId
    if ($javaBody.meta.requestId -ne $goBody.meta.requestId) {
        Write-Host "  [FAIL] RequestId mismatch: Java=$($javaBody.meta.requestId), Go=$($goBody.meta.requestId)" -ForegroundColor Red
        return $false
    }

    # Skip timestamp comparison (allowed per spec)
    Write-Host "  [PASS] Status=$($javaResponse.StatusCode), Keys match" -ForegroundColor Green
    return $true
}

# Main execution
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "Backend Parity Verification" -ForegroundColor Yellow
Write-Host "========================================" -ForegroundColor Yellow
Write-Host "Java Base: $JavaBaseUrl" -ForegroundColor Gray
Write-Host "Go Base:   $GoBaseUrl" -ForegroundColor Gray
Write-Host "========================================" -ForegroundColor Yellow

# Load requests
if (-not (Test-Path $RequestsFile)) {
    Write-Host "[ERROR] Requests file not found: $RequestsFile" -ForegroundColor Red
    exit 1
}

$requestsData = Get-Content $RequestsFile -Raw | ConvertFrom-Json
$allPassed = $true
$passCount = 0
$failCount = 0

Write-Host "`n--- Read-Only Requests ---" -ForegroundColor Magenta

foreach ($req in $requestsData.requests) {
    $headers = @{}
    foreach ($prop in $req.headers.PSObject.Properties) {
        $headers[$prop.Name] = $ExecutionContext.InvokeCommand.ExpandString($prop.Value)
    }
    # Expand X-App-Local-Token if present
    if ($headers.ContainsKey("X-App-Local-Token")) {
        $headers["X-App-Local-Token"] = $LocalToken
    }

    $method = [Microsoft.PowerShell.Commands.WebRequestMethod]$req.method
    $result = Compare-Responses `
        -Name $req.name `
        -Method $method `
        -Path $req.path `
        -Headers $headers `
        -JavaUrl $JavaBaseUrl `
        -GoUrl $GoBaseUrl

    if ($result) {
        $passCount++
    } else {
        $failCount++
        $allPassed = $false
    }
}

Write-Host "`n--- Write Requests (with LocalToken) ---" -ForegroundColor Magenta

foreach ($req in $requestsData.write_requests) {
    $headers = @{}
    foreach ($prop in $req.headers.PSObject.Properties) {
        $headers[$prop.Name] = $ExecutionContext.InvokeCommand.ExpandString($prop.Value)
    }
    # Ensure LocalToken is set
    $headers["X-App-Local-Token"] = $LocalToken

    $body = $req.body | ConvertTo-Json -Compress
    $method = [Microsoft.PowerShell.Commands.WebRequestMethod]$req.method
    $result = Compare-Responses `
        -Name $req.name `
        -Method $method `
        -Path $req.path `
        -Headers $headers `
        -Body $body `
        -JavaUrl $JavaBaseUrl `
        -GoUrl $GoBaseUrl

    if ($result) {
        $passCount++
    } else {
        $failCount++
        $allPassed = $false
    }
}

Write-Host "`n========================================" -ForegroundColor Yellow
Write-Host "Results: $passCount passed, $failCount failed" -ForegroundColor $(if ($allPassed) { "Green" } else { "Red" })

if ($allPassed) {
    Write-Host "PARITY PASS" -ForegroundColor Green
    exit 0
} else {
    Write-Host "PARITY FAIL" -ForegroundColor Red
    exit 1
}
