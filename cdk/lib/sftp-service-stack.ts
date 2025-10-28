import * as cdk from 'aws-cdk-lib';
import * as ecs from 'aws-cdk-lib/aws-ecs';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as elbv2 from 'aws-cdk-lib/aws-elasticloadbalancingv2';
import * as s3 from 'aws-cdk-lib/aws-s3';
import * as rds from 'aws-cdk-lib/aws-rds';
import * as logs from 'aws-cdk-lib/aws-logs';
import * as iam from 'aws-cdk-lib/aws-iam';
import * as secretsmanager from 'aws-cdk-lib/aws-secretsmanager';
import { Construct } from 'constructs';

export class SftpServiceStack extends cdk.Stack {
  constructor(scope: Construct, id: string, props?: cdk.StackProps) {
    super(scope, id, props);

    // VPC for the infrastructure
    const vpc = new ec2.Vpc(this, 'SftpServiceVpc', {
      maxAzs: 2,
      natGateways: 1,
      subnetConfiguration: [
        {
          cidrMask: 24,
          name: 'PublicSubnet',
          subnetType: ec2.SubnetType.PUBLIC,
        },
        {
          cidrMask: 24,
          name: 'PrivateSubnet',
          subnetType: ec2.SubnetType.PRIVATE_WITH_EGRESS,
        },
        {
          cidrMask: 24,
          name: 'DatabaseSubnet',
          subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
        },
      ],
    });

    // S3 Bucket for pricelist files (Hinnat directory)
    const pricelistBucket = new s3.Bucket(this, 'PricelistBucket', {
      bucketName: `sftp-pricelist-${cdk.Stack.of(this).account}-${cdk.Stack.of(this).region}`,
      versioned: false,
      publicReadAccess: false,
      blockPublicAccess: s3.BlockPublicAccess.BLOCK_ALL,
      encryption: s3.BucketEncryption.S3_MANAGED,
      lifecycleRules: [
        {
          id: 'DeleteOldVersions',
          enabled: true,
          expiration: cdk.Duration.days(90), // Keep files for 90 days
        },
      ],
      removalPolicy: cdk.RemovalPolicy.RETAIN, // Keep bucket when stack is deleted
    });

    // Database subnet group
    const dbSubnetGroup = new rds.SubnetGroup(this, 'DatabaseSubnetGroup', {
      vpc,
      description: 'Subnet group for SFTP service database',
      vpcSubnets: {
        subnetType: ec2.SubnetType.PRIVATE_ISOLATED,
      },
    });

    // Database credentials secret
    const dbCredentials = new secretsmanager.Secret(this, 'DatabaseCredentials', {
      description: 'SFTP Service Database Credentials',
      generateSecretString: {
        secretStringTemplate: JSON.stringify({ username: 'sftpuser' }),
        generateStringKey: 'password',
        excludeCharacters: ' %+~`#$&*()|[]{}:;<>?!\\/\'"',
        includeSpace: false,
        passwordLength: 32,
      },
    });

    // PostgreSQL database for user authentication and incoming orders
    const database = new rds.DatabaseInstance(this, 'SftpDatabase', {
      engine: rds.DatabaseInstanceEngine.postgres({
        version: rds.PostgresEngineVersion.VER_15,
      }),
      instanceType: ec2.InstanceType.of(ec2.InstanceClass.T3, ec2.InstanceSize.MICRO),
      credentials: rds.Credentials.fromSecret(dbCredentials),
      vpc,
      subnetGroup: dbSubnetGroup,
      databaseName: 'sftpdb',
      allocatedStorage: 20,
      maxAllocatedStorage: 100,
      deleteAutomatedBackups: false,
      backupRetention: cdk.Duration.days(7),
      deletionProtection: true,
      removalPolicy: cdk.RemovalPolicy.RETAIN,
    });

    // ECS Cluster for Fargate
    const cluster = new ecs.Cluster(this, 'SftpCluster', {
      vpc,
      clusterName: 'sftp-service-cluster',
    });

    // CloudWatch Log Group
    const logGroup = new logs.LogGroup(this, 'SftpLogGroup', {
      logGroupName: '/ecs/sftp-service',
      retention: logs.RetentionDays.ONE_MONTH,
      removalPolicy: cdk.RemovalPolicy.DESTROY,
    });

    // Task Definition
    const taskDefinition = new ecs.FargateTaskDefinition(this, 'SftpTaskDefinition', {
      memoryLimitMiB: 1024,
      cpu: 512,
    });

    // Grant S3 permissions to task role
    pricelistBucket.grantRead(taskDefinition.taskRole);
    
    // Grant database secret access to task role
    dbCredentials.grantRead(taskDefinition.taskRole);

    // Container Definition
    const container = taskDefinition.addContainer('SftpContainer', {
      image: ecs.ContainerImage.fromRegistry('your-account-id.dkr.ecr.region.amazonaws.com/sftp-service:latest'),
      logging: ecs.LogDrivers.awsLogs({
        streamPrefix: 'sftp-service',
        logGroup,
      }),
      environment: {
        AWS_REGION: cdk.Stack.of(this).region,
        S3_BUCKET_NAME: pricelistBucket.bucketName,
      },
      secrets: {
        DATABASE_URL: ecs.Secret.fromSecretsManager(dbCredentials, 'connectionString'),
        AWS_ACCESS_KEY_ID: ecs.Secret.fromSecretsManager(dbCredentials, 'accessKeyId'),
        AWS_SECRET_KEY: ecs.Secret.fromSecretsManager(dbCredentials, 'secretAccessKey'),
      },
    });

    // Add port mapping for SFTP
    container.addPortMappings({
      containerPort: 2222,
      protocol: ecs.Protocol.TCP,
    });

    // Security Group for SFTP service
    const sftpSecurityGroup = new ec2.SecurityGroup(this, 'SftpSecurityGroup', {
      vpc,
      description: 'Security group for SFTP service',
      allowAllOutbound: true,
    });

    // Allow SFTP traffic (port 2222)
    sftpSecurityGroup.addIngressRule(
      ec2.Peer.anyIpv4(),
      ec2.Port.tcp(2222),
      'SFTP access'
    );

    // Allow database access from SFTP service
    database.connections.allowFrom(
      sftpSecurityGroup,
      ec2.Port.tcp(5432),
      'Allow database access from SFTP service'
    );

    // Fargate Service
    const service = new ecs.FargateService(this, 'SftpService', {
      cluster,
      taskDefinition,
      desiredCount: 1,
      assignPublicIp: true,
      vpcSubnets: {
        subnetType: ec2.SubnetType.PUBLIC, // SFTP needs public IP for client access
      },
      securityGroups: [sftpSecurityGroup],
      platformVersion: ecs.FargatePlatformVersion.LATEST,
    });

    // Network Load Balancer for SFTP (TCP traffic)
    const nlb = new elbv2.NetworkLoadBalancer(this, 'SftpLoadBalancer', {
      vpc,
      internetFacing: true,
      loadBalancerName: 'sftp-service-nlb',
    });

    // Target Group for SFTP service
    const targetGroup = new elbv2.NetworkTargetGroup(this, 'SftpTargetGroup', {
      port: 2222,
      protocol: elbv2.Protocol.TCP,
      vpc,
      targetType: elbv2.TargetType.IP,
      healthCheck: {
        protocol: elbv2.Protocol.TCP,
        port: '2222',
      },
    });

    // Add Fargate service to target group
    service.attachToNetworkTargetGroup(targetGroup);

    // Listener for NLB
    nlb.addListener('SftpListener', {
      port: 22, // Standard SFTP port
      protocol: elbv2.Protocol.TCP,
      defaultTargetGroups: [targetGroup],
    });

    // Outputs
    new cdk.CfnOutput(this, 'SftpEndpoint', {
      value: nlb.loadBalancerDnsName,
      description: 'SFTP server endpoint (connect on port 22)',
    });

    new cdk.CfnOutput(this, 'PricelistBucketName', {
      value: pricelistBucket.bucketName,
      description: 'S3 bucket name for pricelist files',
    });

    new cdk.CfnOutput(this, 'DatabaseEndpoint', {
      value: database.instanceEndpoint.hostname,
      description: 'PostgreSQL database endpoint',
    });

    new cdk.CfnOutput(this, 'DatabaseSecretArn', {
      value: dbCredentials.secretArn,
      description: 'Database credentials secret ARN',
    });
  }
}