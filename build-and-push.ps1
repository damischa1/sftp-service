#!/usr/bin/env pwsh
# Build and push SFTP service Docker image to ECR

param(
    [string]$Region = "eu-north-1",
    [string]$AccountId = "854586313951",
    [string]$ImageName = "futur-sftp-service",
    [string]$Tag = "latest"
)

$ErrorActionPreference = "Stop"

Write-Host "Building and pushing SFTP service Docker image..." -ForegroundColor Green

# Set variables
$RepositoryUri = "$AccountId.dkr.ecr.$Region.amazonaws.com/$ImageName"
$FullImageName = "${RepositoryUri}:${Tag}"

Write-Host "Repository URI: $RepositoryUri" -ForegroundColor Yellow
Write-Host "Full image name: $FullImageName" -ForegroundColor Yellow

# Login to ECR
Write-Host "Logging in to Amazon ECR..." -ForegroundColor Blue
aws ecr get-login-password --region $Region | docker login --username AWS --password-stdin $RepositoryUri

# Create ECR repository if it doesn't exist
Write-Host "Creating ECR repository if it doesn't exist..." -ForegroundColor Blue
try {
    aws ecr describe-repositories --repository-names $ImageName --region $Region | Out-Null
    Write-Host "Repository $ImageName already exists" -ForegroundColor Green
} catch {
    Write-Host "Creating repository $ImageName..." -ForegroundColor Yellow
    aws ecr create-repository --repository-name $ImageName --region $Region
}

# Build Docker image
Write-Host "Building Docker image..." -ForegroundColor Blue
docker build -t $ImageName .

# Tag image for ECR
Write-Host "Tagging image for ECR..." -ForegroundColor Blue
docker tag $ImageName $FullImageName

# Push image to ECR
Write-Host "Pushing image to ECR..." -ForegroundColor Blue
docker push $FullImageName

Write-Host "Successfully built and pushed image: $FullImageName" -ForegroundColor Green
Write-Host "You can now deploy using: cdk deploy" -ForegroundColor Cyan