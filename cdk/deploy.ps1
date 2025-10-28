# SFTP Service AWS Deployment Script (PowerShell)
# This script builds and pushes the Docker image to ECR, then deploys the CDK stack

param(
    [string]$Region = "eu-west-1",
    [string]$RepositoryName = "sftp-service",
    [string]$StackName = "SftpServiceStack"
)

# Error handling
$ErrorActionPreference = "Stop"

Write-Host "🚀 Starting SFTP Service deployment to AWS..." -ForegroundColor Blue

# Get AWS account ID
try {
    $AwsAccountId = (aws sts get-caller-identity --query Account --output text)
    if (-not $AwsAccountId) {
        throw "Failed to get AWS account ID"
    }
} catch {
    Write-Host "❌ Failed to get AWS account ID. Please check your AWS credentials." -ForegroundColor Red
    exit 1
}

Write-Host "✅ AWS Account ID: $AwsAccountId" -ForegroundColor Green
Write-Host "✅ AWS Region: $Region" -ForegroundColor Green

# ECR repository URI
$EcrUri = "$AwsAccountId.dkr.ecr.$Region.amazonaws.com/$RepositoryName"

# Step 1: Create ECR repository if it doesn't exist
Write-Host "📦 Creating ECR repository if needed..." -ForegroundColor Yellow
try {
    aws ecr describe-repositories --repository-names $RepositoryName --region $Region 2>$null
} catch {
    Write-Host "📦 Creating new ECR repository: $RepositoryName" -ForegroundColor Yellow
    aws ecr create-repository `
        --repository-name $RepositoryName `
        --region $Region `
        --image-scanning-configuration scanOnPush=true
}

# Step 2: Login to ECR
Write-Host "🔐 Logging in to ECR..." -ForegroundColor Yellow
$LoginCommand = aws ecr get-login-password --region $Region
$LoginCommand | docker login --username AWS --password-stdin $EcrUri

# Step 3: Build Docker image
Write-Host "🔨 Building Docker image..." -ForegroundColor Yellow
Set-Location ..  # Go back to project root
docker build -t "$($RepositoryName):latest" .

# Step 4: Tag and push image to ECR
Write-Host "📤 Tagging and pushing image to ECR..." -ForegroundColor Yellow
docker tag "$($RepositoryName):latest" "$EcrUri:latest"
docker push "$EcrUri:latest"

# Get the image digest
$ImageDigest = aws ecr describe-images `
    --repository-name $RepositoryName `
    --region $Region `
    --query 'imageDetails[0].imageDigest' `
    --output text

Write-Host "✅ Image pushed successfully!" -ForegroundColor Green
Write-Host "   Repository: $EcrUri" -ForegroundColor Green
Write-Host "   Digest: $ImageDigest" -ForegroundColor Green

# Step 5: Deploy CDK stack
Write-Host "☁️  Deploying CDK stack..." -ForegroundColor Yellow
Set-Location cdk

# Install dependencies if needed
if (-not (Test-Path "node_modules")) {
    Write-Host "📥 Installing CDK dependencies..." -ForegroundColor Yellow
    npm install
}

# Update the stack with the correct ECR image URI
Write-Host "🔄 Updating stack with ECR image URI..." -ForegroundColor Yellow
$StackContent = Get-Content "lib/sftp-service-stack.ts" -Raw
$UpdatedContent = $StackContent -replace "your-account-id\.dkr\.ecr\.region\.amazonaws\.com/sftp-service:latest", "$EcrUri:latest"
Set-Content "lib/sftp-service-stack.ts" -Value $UpdatedContent

# Bootstrap CDK if needed
Write-Host "🏗️  Bootstrapping CDK environment..." -ForegroundColor Yellow
npx cdk bootstrap "aws://$AwsAccountId/$Region"

# Deploy the stack
Write-Host "🚀 Deploying CDK stack..." -ForegroundColor Yellow
npx cdk deploy $StackName --require-approval never

# Restore the template file
$OriginalContent = $StackContent
Set-Content "lib/sftp-service-stack.ts" -Value $OriginalContent

Write-Host "🎉 Deployment completed successfully!" -ForegroundColor Green
Write-Host "📋 Next steps:" -ForegroundColor Blue
Write-Host "   1. Update your database with user accounts"
Write-Host "   2. Upload pricelist files to the S3 bucket"
Write-Host "   3. Test SFTP connection using the load balancer endpoint"
Write-Host ""
Write-Host "📊 To view stack outputs:" -ForegroundColor Blue
Write-Host "   aws cloudformation describe-stacks --stack-name $StackName --region $Region --query 'Stacks[0].Outputs'"