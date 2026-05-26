# 默认目标：构建客户端和服务端
all: client server

# 构建客户端
client:
	go build -o bin/gonoploy ./cmd/main.go

# 构建服务端
server:
	go build -o bin/gonoploy-server ./cmd/server/

# 构建 WASM 版本客户端
wasm:
	GOOS=js GOARCH=wasm go build -o bin/gonoploy.wasm ./cmd/main.go

# 代码检查
vet:
	go vet ./...

# 运行服务端（开发模式）
run-server:
	go run ./cmd/server/ --port :8080

# 运行客户端
run-client:
	go run ./cmd/main.go

# 清理构建产物
clean:
	rm -rf bin/

# 整理依赖
tidy:
	go mod tidy

.PHONY: all client server wasm vet run-server run-client clean tidy
