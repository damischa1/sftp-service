#!/usr/bin/env pwsh
# Complete build and deployment script for SFTP service

param(
    [string]$Region = "eu-north-1",
    [switch]$SkipBuild = $false,
    [switch]$SkipDeploy = $false
)

$ErrorActionPreference = "Stop"

Write-Host "🚀 FUTUR SFTP Service - Complete Build & Deploy" -ForegroundColor Magenta
Write-Host "=" * 50 -ForegroundColor Magenta

if (-not $SkipBuild) {
    Write-Host "📦 Step 1: Building and pushing Docker image..." -ForegroundColor Green
    .\build-and-push.ps1 -Region $Region
    if ($LASTEXITCODE -ne 0) {
        Write-Error "Docker build failed!"
        exit 1
    }
    Write-Host "✅ Docker build completed successfully!" -ForegroundColor Green
    Write-Host ""
}

if (-not $SkipDeploy) {
    Write-Host "☁️  Step 2: Deploying CDK stack..." -ForegroundColor Green
    Push-Location cdk
    try {
        cdk deploy --require-approval never
        if ($LASTEXITCODE -ne 0) {
            Write-Error "CDK deployment failed!"
            exit 1
        }
        Write-Host "✅ CDK deployment completed successfully!" -ForegroundColor Green
    }
    finally {
        Pop-Location
    }
}

Write-Host ""
Write-Host "🎉 FUTUR SFTP Service deployment completed!" -ForegroundColor Magenta
Write-Host "Connect to your SFTP service using the endpoint from CDK output." -ForegroundColor Cyan