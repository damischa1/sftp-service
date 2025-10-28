#!/bin/bash

# SFTP Service AWS Deployment Script
# This script builds and pushes the Docker image to ECR, then deploys the CDK stack

set -e

# Configuration
AWS_REGION=${AWS_REGION:-"eu-west-1"}
ECR_REPOSITORY_NAME="sftp-service"
CDK_STACK_NAME="SftpServiceStack"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}🚀 Starting SFTP Service deployment to AWS...${NC}"

# Get AWS account ID
AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
if [ -z "$AWS_ACCOUNT_ID" ]; then
    echo -e "${RED}❌ Failed to get AWS account ID. Please check your AWS credentials.${NC}"
    exit 1
fi

echo -e "${GREEN}✅ AWS Account ID: $AWS_ACCOUNT_ID${NC}"
echo -e "${GREEN}✅ AWS Region: $AWS_REGION${NC}"

# ECR repository URI
ECR_URI="$AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/$ECR_REPOSITORY_NAME"

# Step 1: Create ECR repository if it doesn't exist
echo -e "${YELLOW}📦 Creating ECR repository if needed...${NC}"
aws ecr describe-repositories --repository-names $ECR_REPOSITORY_NAME --region $AWS_REGION 2>/dev/null || {
    echo -e "${YELLOW}📦 Creating new ECR repository: $ECR_REPOSITORY_NAME${NC}"
    aws ecr create-repository \
        --repository-name $ECR_REPOSITORY_NAME \
        --region $AWS_REGION \
        --image-scanning-configuration scanOnPush=true
}

# Step 2: Login to ECR
echo -e "${YELLOW}🔐 Logging in to ECR...${NC}"
aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $ECR_URI

# Step 3: Build Docker image
echo -e "${YELLOW}🔨 Building Docker image...${NC}"
cd ..  # Go back to project root
docker build -t $ECR_REPOSITORY_NAME:latest .

# Step 4: Tag and push image to ECR
echo -e "${YELLOW}📤 Tagging and pushing image to ECR...${NC}"
docker tag $ECR_REPOSITORY_NAME:latest $ECR_URI:latest
docker push $ECR_URI:latest

# Get the image digest
IMAGE_DIGEST=$(aws ecr describe-images \
    --repository-name $ECR_REPOSITORY_NAME \
    --region $AWS_REGION \
    --query 'imageDetails[0].imageDigest' \
    --output text)

echo -e "${GREEN}✅ Image pushed successfully!${NC}"
echo -e "${GREEN}   Repository: $ECR_URI${NC}"
echo -e "${GREEN}   Digest: $IMAGE_DIGEST${NC}"

# Step 5: Deploy CDK stack
echo -e "${YELLOW}☁️  Deploying CDK stack...${NC}"
cd cdk

# Install dependencies if needed
if [ ! -d "node_modules" ]; then
    echo -e "${YELLOW}📥 Installing CDK dependencies...${NC}"
    npm install
fi

# Update the stack with the correct ECR image URI
echo -e "${YELLOW}🔄 Updating stack with ECR image URI...${NC}"
sed -i.bak "s|your-account-id.dkr.ecr.region.amazonaws.com/sftp-service:latest|$ECR_URI:latest|g" lib/sftp-service-stack.ts

# Bootstrap CDK if needed
echo -e "${YELLOW}🏗️  Bootstrapping CDK environment...${NC}"
npx cdk bootstrap aws://$AWS_ACCOUNT_ID/$AWS_REGION

# Deploy the stack
echo -e "${YELLOW}🚀 Deploying CDK stack...${NC}"
npx cdk deploy $CDK_STACK_NAME --require-approval never

# Restore the template file
mv lib/sftp-service-stack.ts.bak lib/sftp-service-stack.ts

echo -e "${GREEN}🎉 Deployment completed successfully!${NC}"
echo -e "${BLUE}📋 Next steps:${NC}"
echo -e "   1. Update your database with user accounts"
echo -e "   2. Upload pricelist files to the S3 bucket"
echo -e "   3. Test SFTP connection using the load balancer endpoint"
echo -e ""
echo -e "${BLUE}📊 To view stack outputs:${NC}"
echo -e "   aws cloudformation describe-stacks --stack-name $CDK_STACK_NAME --region $AWS_REGION --query 'Stacks[0].Outputs'"