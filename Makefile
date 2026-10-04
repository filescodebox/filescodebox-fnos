.PHONY: build test vet run fpack docker clean

# FilesCodeBox 飞牛(fnOS)应用适配层
# 独立构建钉 go.mod 正式版本;在本仓目录直接跑会拾取上层 go.work 联编本地 core,
# 两者皆可——见 README「本地开发」。

build:            ## 编译 fnos-adapter 二进制
	go build -o bin/fnos-adapter ./cmd/fnos-adapter

test:             ## 单元测试
	go test ./...

vet:
	go vet ./...

run:              ## 降级模式运行(无飞牛凭证,业务全功能)
	go run ./cmd/fnos-adapter

fpack:            ## 飞牛应用包(需 fnpack 二进制, developer.fnnas.com/docs/cli/fnpack/)
	cd fnos && fnpack build
	@echo "✓ 产物 fnos/filescodebox.fpk"

docker:           ## 构建容器镜像(上下文=本仓库,依赖走 module proxy)
	docker build -t fnos:latest .

clean:
	rm -rf bin/ fnos/*.fpk
