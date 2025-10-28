# Rajoitettu SFTP Palvelu

Go-pohjainen SFTP-palvelin rajoitetuilla käyttöoikeuksilla PostgreSQL-autentikoinnilla ja S3-tallennuksella.

## Käyttäjätietokanta

Palvelu käyttää **olemassa olevaa PostgreSQL-tietokantaa** käyttäjätietojen lukemiseen. 

### Palvelun rooli:
- ✅ **Lukee** käyttäjätiedot olemassa olevasta tietokannasta
- ✅ **Autentikoi** käyttäjiä SFTP-yhteyksiä varten  
- ❌ **Ei hallitse** käyttäjiä (tehdään muualla)

## Käyttöoikeudet

Käyttäjillä on seuraavat **rajoitetut oikeudet**:

### ✅ Sallitut toiminnot:
1. **Juurihakemiston listaus** (`/`) - Näyttää vain `in` ja `Hinnat` kansiot
2. **Siirtyminen kansioihin**:
   - `/in/` - sisään tulo kansio (**vain kirjoitus**, max 100KB, PostgreSQL)
   - `/Hinnat/` - hintatiedostojen kansio (luku/kirjoitus, S3)
3. **Tiedostojen kirjoittaminen**:
   - `/in/` → PostgreSQL tietokantaan (max 100KB)
   - `/Hinnat/` → S3 bucketiin
4. **Tiedostojen lukeminen** vain `/Hinnat/` kansiosta (S3)
5. **Kansioiden listaus** `/in/` (PostgreSQL) ja `/Hinnat/` (S3) sisällä
6. **Uusien alikansioiden luominen** vain `/Hinnat/` sisälle

### ❌ Kielletyt toiminnot:
- **Ei poisto-oikeuksia** (tiedostot tai kansiot)
- **Ei uudelleennimeämisoikeuksia**
- **Ei pääsyä muihin kansioihin** kuin `/in/` ja `/Hinnat/`
- **Ei kirjoitusoikeuksia juurihakemistoon** (`/`)
- **Ei kansioiden poisto-oikeuksia**

## Arkkitehtuuri

```
SFTP Client → SFTP Server (Rajoitetut oikeudet) → PostgreSQL (Auth + /in/ Storage) + S3 (/Hinnat/ Storage)
```

### Kaksoistallennusjärjestelmä

1. **/in/ kansio** → PostgreSQL tietokanta
   - Max tiedostokoko: 100KB
   - Vain kirjoitusoikeudet
   - Tallennetaan `incoming_files` tauluun

2. **/Hinnat/ kansio** → S3 bucket
   - Normaali tiedostotallennus
   - Luku- ja kirjoitusoikeudet
   - Käyttäjäkohtaiset prefiksit

## S3 Tallennusrakenne (/Hinnat/)

```
S3 Bucket/
├── käyttäjä1/
│   └── Hinnat/
│       ├── hintalista.csv
│       └── tarjoukset/
│           └── tarjous1.pdf
└── käyttäjä2/
    └── Hinnat/
        └── tiedosto2.txt
```

## PostgreSQL Tallennusrakenne (/in/)

```sql
-- incoming_files taulu
CREATE TABLE incoming_files (
    id SERIAL PRIMARY KEY,
    username VARCHAR(255) NOT NULL,
    filename VARCHAR(255) NOT NULL,
    content TEXT NOT NULL,
    uploaded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
```

## Käyttöönoton Pikaohje

### 1. Asenna riippuvuudet
```bash
go mod tidy
```

### 2. Konfiguroi ympäristö
```bash
cp .env.example .env
# Muokkaa .env tiedostoa AWS-tunnuksillasi
```

### 3. Käynnistä palvelut
```bash
docker-compose up -d
```

### 4. Olemassa oleva tietokanta

Palvelu odottaa että PostgreSQL-tietokannassa on jo `users` taulu käyttäjätiedoilla:

```sql
-- Tietokannassa tulee olla users taulu tässä muodossa:
-- users (id, username, password_hash, is_active, created_at, updated_at)
```

### 5. Testaa yhteys
```bash
sftp -P 2222 käyttäjänimi@localhost
```

## SFTP-komentojen käyttö

### Sallitut komennot:
```bash
# Listaa juurihakemisto (näyttää vain 'in' ja 'Hinnat')
ls

# Siirry kansioihin
cd in
cd Hinnat

# Listaa kansion sisältö
ls

# Lataa tiedosto
put local_file.txt

# Luo alikansio
mkdir uusi_kansio

# Lataa tiedosto palvelimelta
get remote_file.txt
```

### Kielletyt komennot (palauttavat virheen):
```bash
# Poista tiedosto - EI SALLITTU
rm tiedosto.txt

# Uudelleennimeä - EI SALLITTU  
rename vanha.txt uusi.txt

# Poista kansio - EI SALLITTU
rmdir kansio

# Kirjoita juurihakemistoon - EI SALLITTU
put tiedosto.txt /

# Siirry muihin kansioihin - EI SALLITTU
cd /tmp
cd /home
```

## Turvallisuusominaisuudet

- **SSH host key** automaattisesti generoitu ja tallennettu
- **bcrypt salasanahashit** tietokannassa
- **Käyttäjäeristys** - jokainen käyttäjä näkee vain omat tiedostonsa
- **Polkujen validointi** - estää pääsyn kiellettyihin hakemistoihin
- **Operaatioiden rajoittaminen** - vain lukeminen, kirjoittaminen ja listaus
- **TLS salaus** kaikille SFTP-yhteyksille

## Lokitiedot ja Seuranta

Palvelu kirjaa kaikki:
- Autentikoinnin yritykset
- Tiedosto-operaatiot (onnistuneet ja evätyt)
- Yhteyksien avaukset ja sulkemiset
- Käyttöoikeusvirheet

```bash
# Katso palvelun lokeja
docker logs sftp-service

# Katso tietokannan lokeja  
docker logs sftp-postgres
```

## Käyttöoikeuksien Testaaminen

```bash
# Yhdistä SFTP:llä
sftp -P 2222 testuser@localhost

# Testaa sallittuja operaatioita
ls                    # Tulostaa: in  Hinnat
cd in                 # Onnistuu
put test.txt          # Onnistuu
ls                    # Näyttää tiedostot
cd /Hinnat            # Onnistuu
mkdir uusi_kansio     # Onnistuu

# Testaa kiellettyjä operaatioita
rm test.txt           # Virhe: access denied
cd /tmp               # Virhe: access denied
put test.txt /        # Virhe: write not allowed
```

## Vianetsintä

### Yleiset ongelmat:

1. **Yhteys evätty**: Tarkista portti 2222 ja palomuuri
2. **Autentikointi epäonnistui**: Varmista käyttäjätunnus ja salasana
3. **Tiedoston lataus epäonnistui**: Tarkista AWS-tunnukset ja S3-bucket
4. **"Access denied" virheet**: Normaali käyttäytyminen rajoitetuille poluille

### Lokien tarkistus:
```bash
# Palvelun lokeja
docker logs -f sftp-service

# Tietokantaa
docker logs -f sftp-postgres
```

## ☁️ AWS Deployment (CDK)

Projekti sisältää valmiin AWS CDK -konfiguraation tuotantokäyttöönottoa varten.

### AWS-infrastruktuuri:
- **ECS Fargate** - Containerien ajamiseen
- **PostgreSQL RDS** - Käyttäjätiedot ja saapuvat tilaukset
- **S3 Bucket** - Hinnastotiedostot
- **Network Load Balancer** - SFTP-liikenteen jakamiseen
- **VPC** - Verkko-infrastruktuuri

### Pikaopas AWS-käyttöönottoon:

```bash
# Siirry CDK-kansioon
cd cdk

# Käynnistä automaattinen käyttöönotto (Windows)
.\deploy.ps1 -Region "eu-west-1"

# Käynnistä automaattinen käyttöönotto (Linux/MacOS)  
chmod +x deploy.sh
./deploy.sh
```

Katso täydelliset ohjeet: [`cdk/README.md`](cdk/README.md)

### AWS-kustannukset (arvio):
- **~$78/kuukausi** peruskäytössä (EU-West-1)
- Sisältää: Fargate, RDS, Load Balancer, NAT Gateway
- Ei sisällä: S3-tallennusta ja tiedonsiirtoa
