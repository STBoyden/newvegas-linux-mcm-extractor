ext := if os() == "windows" { ".exe" } else { "" }

[private]
prebuild:
    mkdir -p out
    go mod tidy

build: prebuild
    go build -o out/app{{ ext }}

run: build
    ./out/app{{ ext }}