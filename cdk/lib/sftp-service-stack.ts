import * as cdk from 'aws-cdk-lib';
import * as ecs from 'aws-cdk-lib/aws-ecs';
import * as ec2 from 'aws-cdk-lib/aws-ec2';
import * as elbv2 from 'aws-cdk-lib/aws-elasticloadbalancingv2';

import * as logs from 'aws-cdk-lib/aws-logs';
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

      ],
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

    // Container Definition
    const container = taskDefinition.addContainer('SftpContainer', {
      image: ecs.ContainerImage.fromRegistry('your-account-id.dkr.ecr.region.amazonaws.com/sftp-service:latest'),
      logging: ecs.LogDrivers.awsLogs({
        streamPrefix: 'sftp-service',
        logGroup,
      }),
      environment: {
        AUTH_API_URL: 'https://your-auth-api.com',
        PRICELIST_API_URL: 'https://your-pricelist-api.com',
        PRICELIST_API_KEY: 'your-pricelist-api-key-here',
        ORDERS_API_URL: 'https://your-orders-api.com',
        ORDERS_API_KEY: 'your-orders-api-key-here',
      },
    });

    // Add port mapping for SFTP
    container.addPortMappings({
      containerPort: 22,
      protocol: ecs.Protocol.TCP,
    });

    // Security Group for SFTP service
    const sftpSecurityGroup = new ec2.SecurityGroup(this, 'SftpSecurityGroup', {
      vpc,
      description: 'Security group for SFTP service',
      allowAllOutbound: true,
    });

    // Allow SFTP traffic (port 22)
    sftpSecurityGroup.addIngressRule(
      ec2.Peer.anyIpv4(),
      ec2.Port.tcp(22),
      'SFTP access'
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
      port: 22,
      protocol: elbv2.Protocol.TCP,
      vpc,
      targetType: elbv2.TargetType.IP,
      healthCheck: {
        protocol: elbv2.Protocol.TCP,
        port: '22',
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


  }
}