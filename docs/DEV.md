# Setup

## Go setup

Ensure `go` is installed. Version `1.27.1` is the one used by this project.
Get it here: <https://go.dev/dl/>

To install it, extract the archive in some 'standard' location, then add the `bin/` directory in your `$PATH`.

Also `go` should create a directory in your `$HOME`.
Make sure to add `$HOME/go/bin` to your path if you want to install/use go tools.

You should install `golangci-lint`, check out instruction [here](https://golangci-lint.run/docs/welcome/install/local/)
Then you can run lints like so: `golangci-lint run`

## Frontend

You must have a npm-capable package maneger.

NPM Dependencies:
```
npm i vite \
    svelte \
    @sveltejs/vite-plugin-svelte
```

To build frontend components, go to `frontend/` and execute `npm run build` (or `make frontend` from the root)

## Running the server

```
# Start the database container
docker compose up -d

# Run the server
go run .
```

To make the whole process of `Edit -> Stop Server -> Restart server` streamlined, you should install [fresh](https://github.com/gravityblast/fresh): `go install github.com/gravityblast/fresh@latest`.

Then at the project's root start fresh: `fresh`. It will automatically rebuild and restart the server whenever you edit the source code or localization files.

## Makefile

For convenience, a `Makefile` is provided. To build frontend components and run the server, simply execute `make dev`

# Database

To start the database use docker-compose: `docker compose up -d`

To connect to the database, do this:
```
docker exec -it <container-name> mariadb \
         -u root \
         -p
```
Then enter the password for `root` in `.env` and select the `main` database: `USE main;`

# Server Dependencies

 * [gin](https://github.com/gin-gonic/gin) Web server framework
 * [gorm](https://gorm.io/gorm) Database ORM
 * [gorm-mysql](https://gorm.io/driver/mysql) MySQL driver for Gorm
 * [godotenv](https://github.com/joho/godotenv) Sources .env
 * [bcrypt](https://golang.org/x/crypto/bcrypt) bcrypt implementation
 * [gofluent](https://github.com/hakastein/gofluent) Fluent for Go
 * [testify](https://https://github.com/stretchr/testify) Assertion library

# Static routes

 * `/frontend`: Frontend components
 * `/favicon.ico`: Icon
 * `/static`: Static files

# Tests

If you wish to run tests, you must grant `admin` the rights to create and drop databases: 
```
GRANT ALL CREATE, DROP ON main_test.* TO 'admin'@'%'
FLUSH PRIVILEGES
```

**Running tests:**
To run tests: `go test ./PACKAGE  -v` replace `PACKAGE` with the package to run tests for
