ext := if os() == "windows" { ".exe" } else { "" }

[private]
prebuild:
    mkdir -p out
    go mod tidy

fmt:
    go run mvdan.cc/gofumpt@latest -w main.go
    @echo "Finished formatting code."

build: prebuild
    go build -o out/app{{ ext }}

run: build
    ./out/app{{ ext }}