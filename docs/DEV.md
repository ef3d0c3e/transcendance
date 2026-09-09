# Setup

## Go setup

Ensure `go` is installed. Version `1.27.1` is the one used by this project.
Get it here: <https://go.dev/dl/>

To install it, extract the archive in some 'standard' location, then add the `bin/` directory in your `$PATH`.

## Frontend

You must have a npm-capable package maneger.

NPM Dependencies:
```
npm i vite \
    @sveltejs/vite-plugin-svelte
```

To build frontend components, go to `frontend/` and execute `npm build`

## Running the server

```
# Start the database container
docker compose -d

# Run the server
go run .
```

## Makefile

For convenience, a `Makefile` is provided. To build frontend components and run the server, simply execute `make dev`

# Database

To connect to the database, do this:
```
docker exec -it <container-name> mariadb \
         -u root \
         -p
```
Then enter the password for `root` in `.env` and select the `main` database: `USE main;`

# Dependencies

 * [gin](https://github.com/gin-gonic/gin) Web server framework
 * [gorm](https://gorm.io/gorm) Database ORM
 * [gorm-mysql](https://gorm.io/driver/mysql) MySQL driver for Gorm
 * [godotenv](https://github.com/joho/godotenv) Sources .env
 * [bcrypt](https://golang.org/x/crypto/bcrypt) bcrypt implementation

