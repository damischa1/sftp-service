#!/usr/bin/env node
import 'source-map-support/register';
import * as cdk from 'aws-cdk-lib';
import { SftpServiceStack } from '../lib/sftp-service-stack';

const app = new cdk.App();

// Get environment variables or use defaults
const account = process.env.CDK_DEFAULT_ACCOUNT;
const region = process.env.CDK_DEFAULT_REGION || 'eu-west-1';

new SftpServiceStack(app, 'SftpServiceStack', {
  env: { 
    account: account, 
    region: region 
  },
  description: 'SFTP Service with Fargate, S3, and PostgreSQL',
});