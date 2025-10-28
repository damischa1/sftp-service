# SFTP Service - AWS CDK Deployment

Tämä CDK-stack luo kaikki tarvittavat AWS-resurssit SFTP-palvelun ajamiseen AWS Fargate-palvelussa.

## 📋 Mitä CDK luo

### 🏗️ Infrastruktuuri
- **VPC** kolmella subnet-tyypillä (Public, Private, Database)
- **NAT Gateway** ulospäin meneville yhteyksille
- **Security Groups** verkkoliikenteen rajoittamiseen

### 💾 Tallennusratkaisut
- **S3 Bucket** hinnastotiedostoille (`/Hinnat` hakemisto)
- **PostgreSQL RDS** käyttäjäautentikoinnille ja saapuville tilauksille
- **CloudWatch Logs** loggaukselle

### 🚀 Palvelu
- **ECS Fargate Cluster** containerien ajamiseen
- **Fargate Service** SFTP-palvelimen ajamiseen
- **Network Load Balancer** TCP-liikenteen kuorman jakamiseen
- **Target Group** Fargate-palvelulle

### 🔐 Tietoturva
- **Secrets Manager** tietokannan salasanoille
- **IAM Roles** minimaalisilla oikeuksilla
- **VPC Security Groups** verkkoliikenteen rajoittamiseen

## 🚀 Käyttöönotto

### Edellytykset
1. **AWS CLI** asennettu ja konfiguroitu
2. **Docker** asennettu palvelimen rakentamiseen
3. **Node.js** ja **npm** CDK:ta varten
4. **AWS CDK v2** asennettu globaalisti: `npm install -g aws-cdk`

### Automaattinen käyttöönotto

#### Windows (PowerShell):
```powershell
cd cdk
.\deploy.ps1 -Region "eu-west-1"
```

#### Linux/MacOS (Bash):
```bash
cd cdk
chmod +x deploy.sh
./deploy.sh
```

### Manuaalinen käyttöönotto

1. **Asenna riippuvuudet:**
   ```bash
   cd cdk
   npm install
   ```

2. **Rakenna ja työnnä Docker-image ECR:ään:**
   ```bash
   # Hae AWS account ID
   AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
   AWS_REGION="eu-west-1"
   
   # Luo ECR repository
   aws ecr create-repository --repository-name sftp-service --region $AWS_REGION
   
   # Kirjaudu ECR:ään
   aws ecr get-login-password --region $AWS_REGION | docker login --username AWS --password-stdin $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com
   
   # Rakenna ja työnnä image
   cd ..
   docker build -t sftp-service:latest .
   docker tag sftp-service:latest $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/sftp-service:latest
   docker push $AWS_ACCOUNT_ID.dkr.ecr.$AWS_REGION.amazonaws.com/sftp-service:latest
   ```

3. **Päivitä CDK-stacki oikealla image URI:lla:**
   ```typescript
   // lib/sftp-service-stack.ts tiedostossa, päivitä:
   image: ecs.ContainerImage.fromRegistry('YOUR-ACCOUNT-ID.dkr.ecr.YOUR-REGION.amazonaws.com/sftp-service:latest'),
   ```

4. **Bootstrap CDK-ympäristö:**
   ```bash
   cd cdk
   cdk bootstrap
   ```

5. **Deploy stack:**
   ```bash
   cdk deploy SftpServiceStack
   ```

## 📊 Stack Outputs

Käyttöönoton jälkeen saat seuraavat tärkeät tiedot:

- **SftpEndpoint**: Load balancer DNS-nimi SFTP-yhteyksille
- **PricelistBucketName**: S3 bucket hinnastotiedostoille  
- **DatabaseEndpoint**: PostgreSQL tietokannan endpoint
- **DatabaseSecretArn**: Tietokannan salasanojen ARN

## 🔧 Konfigurointi käyttöönoton jälkeen

### 1. Tietokannan alustus
```sql
-- Yhdistä tietokantaan ja luo taulut
psql -h YOUR-DB-ENDPOINT -U sftpuser -d sftpdb

-- Luo taulut (katso ../schema.sql)
\i schema.sql

-- Lisää käyttäjiä
INSERT INTO users (username, password_hash) VALUES 
('testuser', '$2a$10$...');  -- bcrypt hash
```

### 2. S3 Bucket sisällön lisäys
```bash
# Lataa hinnastotiedosto S3:een
aws s3 cp salhydro_kaikki.zip s3://YOUR-BUCKET-NAME/testuser/Hinnat/salhydro_kaikki.zip
```

### 3. SFTP-yhteyden testaus
```bash
# Testaa yhteyttä
sftp -P 22 testuser@YOUR-LOAD-BALANCER-DNS
```

## 🛠️ Kehitys ja muutokset

### Stack päivitys:
```bash
cd cdk
cdk diff        # Näytä muutokset
cdk deploy      # Käytä muutokset
```

### Stack poisto:
```bash
cd cdk
cdk destroy SftpServiceStack
```

⚠️ **Huomio**: RDS ja S3 bucket säilyvät poiston jälkeen turvallisuussyistä.

## 💰 Kustannukset

Arvioitu kuukausikustannus (eu-west-1):
- **Fargate** (1 task, 0.5 vCPU, 1GB RAM): ~$15/kk
- **RDS t3.micro**: ~$15/kk  
- **Network Load Balancer**: ~$16/kk
- **NAT Gateway**: ~$32/kk + data transfer
- **S3 Storage**: muuttuva data määrän mukaan

**Yhteensä: ~$78/kk** + data transfer costs

## 🔍 Vianetsintä

### Container ei käynnisty:
1. Tarkista CloudWatch-logit: `/ecs/sftp-service`
2. Varmista ECR image URI on oikea
3. Tarkista environment-muuttujat

### SFTP-yhteys ei toimi:
1. Tarkista Security Group säännöt
2. Varmista Load Balancer on käynnissä
3. Testaa Target Group health check

### Tietokantayhteys ei toimi:
1. Tarkista Secrets Manager konfiguraatio
2. Varmista VPC reitit tietokantaan
3. Tarkista Security Group säännöt