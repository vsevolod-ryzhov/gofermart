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
- go run cmd/gophermart/main.go -d="host=localhost user=postgres_user password=postgres_password dbname=postgres_db sslmode=disable"
- go build -o cmd/gophermart/gophermart cmd/gophermart/*.go
- migrate -database "postgresql://postgres_user:postgres_password@localhost:5432/postgres_db?sslmode=disable" -path ./migrations up

## Endpoints examples
-  curl -X POST -H "Content-Type: application/json" -d '{"login":"admin", "password":"pwd"}' 127.0.0.1:8080/api/user/register -v --compressed