.PHONY: run build test clean

# 기본 실행
run:
	go run core/main.go

# 빌드
build:
	go build -o bin/main core/main.go

# 테스트 실행
test:
	go test ./...

# 바이너리 및 캐시 파일 정리
clean:
	rm -rf bin
	go clean

# 의존성 다운로드
install:
	go mod download
