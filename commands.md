## Tests
- go test ./...
- go build -o cmd/gophermart/gophermart cmd/gophermart/*.go &&
  ./gophermarttest-darwin-arm64 \
  -test.v -test.run=^TestGophermart$ \
  -gophermart-binary-path=cmd/gophermart/gophermart \
  -gophermart-host=localhost \
  -gophermart-port=8080 \
  -gophermart-database-uri="postgresql://postgres_user:postgres_password@localhost:5432/postgres_db?sslmode=disable" \
  -accrual-binary-path=cmd/accrual/accrual_darwin_arm64 \
  -accrual-host=localhost \
  -accrual-port=8081 \
  -accrual-database-uri="postgresql://postgres_user:postgres_password@localhost:5432/postgres_db?sslmode=disable"


## Misc
- go test ./... -coverprofile cover.out
- go tool cover -html=cover.out
- docker-compose up -d
- docker-compose down
- go run cmd/gophermart/main.go -r="localhost:8081" -d="host=localhost user=postgres_user password=postgres_password dbname=postgres_db sslmode=disable"
- go build -o cmd/gophermart/gophermart cmd/gophermart/*.go
- migrate create -ext sql -dir ./migrations -seq <create_tableName_table>
- migrate -database "postgresql://postgres_user:postgres_password@localhost:5432/postgres_db?sslmode=disable" -path ./migrations up

## Endpoints examples
- curl -X POST -H "Content-Type: application/json" -r="localhost:8081" -d '{"login":"admin", "password":"pwd1"}' 127.0.0.1:8080/api/user/register -v --compressed
- curl -X POST -H "Content-Type: application/json" -r="localhost:8081" -d '{"login":"admin", "password":"pwd1"}' 127.0.0.1:8080/api/user/login -v --compressed
- curl -H "Content-Type: application/json" -r="localhost:8081" -H "Cookie: auth_token=INSERT_TOKEN_HERE" 127.0.0.1:8080/api/user/orders -v --compressed
- curl -H "Content-Type: application/json" -r="localhost:8081" -H "Cookie: auth_token=auth_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMTEsImV4cCI6MTc2NjE2MTAwOSwiaWF0IjoxNzY2MDc0NjA5fQ.kWyksEJyCNs9dphABrK90TH7eUhroklmEnUNi-aV8dc" 127.0.0.1:8080/api/user/orders -v --compressed
- curl -X POST -d '{"order":"660624454287", "sum":"367.40"}' -H "Content-Type: application/json" -r="localhost:8081" -H "Cookie: auth_token=auth_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMDcsImV4cCI6MTc2NjE1OTg4MSwiaWF0IjoxNzY2MDczNDgxfQ.Up7DYAm0YHVA-ABRGWTiQXFAEaqke-_ch_Rwe1EhTHE" 127.0.0.1:8080/api/user/balance/withdraw -v --compressed
- curl -X POST -d '{"order":"660624454287", "sum":"367.40"}' -H "Content-Type: application/json" -r="localhost:8081" -H "Cookie: auth_token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJ1c2VyX2lkIjoxMDcsImV4cCI6MTc2NjE1OTg4MSwiaWF0IjoxNzY2MDczNDgxfQ.Up7DYAm0YHVA-ABRGWTiQXFAEaqke-_ch_Rwe1EhTHE" 127.0.0.1:8080/api/user/balance/withdraw -v --compressed