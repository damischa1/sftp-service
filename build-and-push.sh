#!/bin/bash
# Build and push SFTP service Docker image to ECR

set -e

# Default values
REGION="${AWS_DEFAULT_REGION:-eu-north-1}"
ACCOUNT_ID="${AWS_ACCOUNT_ID:-854586313951}"
IMAGE_NAME="futur-sftp-service"
TAG="latest"

# Parse command line arguments
while [[ $# -gt 0 ]]; do
  case $1 in
    --region)
      REGION="$2"
      shift 2
      ;;
    --account-id)
      ACCOUNT_ID="$2"
      shift 2
      ;;
    --image-name)
      IMAGE_NAME="$2"
      shift 2
      ;;
    --tag)
      TAG="$2"
      shift 2
      ;;
    *)
      echo "Unknown option $1"
      exit 1
      ;;
  esac
done

echo "🚀 Building and pushing SFTP service Docker image..."

# Set variables
REPOSITORY_URI="${ACCOUNT_ID}.dkr.ecr.${REGION}.amazonaws.com/${IMAGE_NAME}"
FULL_IMAGE_NAME="${REPOSITORY_URI}:${TAG}"

echo "📍 Repository URI: $REPOSITORY_URI"
echo "🏷️  Full image name: $FULL_IMAGE_NAME"

# Login to ECR
echo "🔐 Logging in to Amazon ECR..."
aws ecr get-login-password --region $REGION | docker login --username AWS --password-stdin $REPOSITORY_URI

# Create ECR repository if it doesn't exist
echo "📦 Creating ECR repository if it doesn't exist..."
if aws ecr describe-repositories --repository-names $IMAGE_NAME --region $REGION >/dev/null 2>&1; then
    echo "✅ Repository $IMAGE_NAME already exists"
else
    echo "🆕 Creating repository $IMAGE_NAME..."
    aws ecr create-repository --repository-name $IMAGE_NAME --region $REGION
fi

# Build Docker image
echo "🔨 Building Docker image..."
docker build -t $IMAGE_NAME .

# Tag image for ECR
echo "🏷️  Tagging image for ECR..."
docker tag $IMAGE_NAME $FULL_IMAGE_NAME

# Push image to ECR
echo "📤 Pushing image to ECR..."
docker push $FULL_IMAGE_NAME

echo "✅ Successfully built and pushed image: $FULL_IMAGE_NAME"
echo "🚀 You can now deploy using: cdk deploy"