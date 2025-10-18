# Development workflow

These instructions will get you a copy of the project up and running on your local machine for development and testing purposes.

### Prerequisites

* Go 1.25+
* PostgreSQL 18+

## Local development setup
### 1️⃣ Clone

```bash
git clone https://github.com/i-christian/fileShare.git
```

The application has some environment variable. To use the example configuration, make sure to:
#### copy `.env_example` to `.env`
- On the project root
```
  cp .env_example .env
```

Set up postgresql as follows:

#### Log in to PostgreSQL as a Superuser
- ```sudo -u postgres psql```

#### Create the User
- Example: 
  ```
  CREATE USER myuser WITH PASSWORD 'mypass';
  ```

#### Give the user permission to create databases
- ```ALTER USER myuser WITH CREATEDB;```

#### create the database 
- ```
  createdb -U myuser -h localhost fileShare
  ```

#### log into the database
- ```
  psql -Umyuser -hlocalhost fileShare
  ```

#### Ensure that the user `myuser` has sufficient privileges on the database
- ```
  GRANT ALL PRIVILEGES ON DATABASE fileShare TO myuser;
  ```

#### Migrations
The application does apply database migration automatically using embedded migration logic.
  - ```
    cd internal/db/schema/
    ```
Add SQL tables to migration files eg `001_user.sql` && run to create the defined tables: 
  - ```
    goose postgres postgres://myuser:mypass@localhost/fileShare up
    ```

This can be reversed using:
- ```
  goose postgres postgres://myuser:mypass@localhost/fileShare down
  ```

## Secret key hash generation
- Run the following command to generate the `JWT_SECRET`
```
  openssl rand -hex 32
```


## Running the application using MakeFile

Run build make command with tests
```bash
make all
```

Build the application
```bash
make build
```

Run the application
```bash
make run
```

Live reload the application:
```bash
make watch
```

Run the full test suite:
```bash
make test
```

Clean up binary from the last build:
```bash
make clean
```
