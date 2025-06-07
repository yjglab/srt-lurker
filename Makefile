.PHONY: run build test clean build-all build-windows build-macos build-linux install

# 애플리케이션 이름
APP_NAME=srt-lurker

# 기본 실행
run:
	go run core/main.go

# 현재 플랫폼용 빌드
build:
	go build -o bin/$(APP_NAME) core/main.go

# 모든 플랫폼용 빌드
build-all: build-windows build-macos build-linux
	@echo "✅ 모든 플랫폼용 빌드 완료!"
	@echo "📁 빌드된 파일들:"
	@ls -la bin/

# Windows 64bit 빌드
build-windows:
	@echo "🪟 Windows 64bit 빌드 중..."
	GOOS=windows GOARCH=amd64 go build -o bin/$(APP_NAME)-windows-amd64.exe core/main.go
	@echo "✅ Windows 빌드 완료: bin/$(APP_NAME)-windows-amd64.exe"

# macOS 빌드 (Intel & Apple Silicon)
build-macos:
	@echo "🍎 macOS Intel 64bit 빌드 중..."
	GOOS=darwin GOARCH=amd64 go build -o bin/$(APP_NAME)-macos-amd64 core/main.go
	@echo "🔏 macOS Intel 코드 서명 중..."
	@codesign -s - bin/$(APP_NAME)-macos-amd64 2>/dev/null || echo "⚠️ 코드 서명 실패 (계속 진행)"
	@echo "✅ macOS Intel 빌드 완료: bin/$(APP_NAME)-macos-amd64"
	@echo "🍎 macOS Apple Silicon 빌드 중..."
	GOOS=darwin GOARCH=arm64 go build -o bin/$(APP_NAME)-macos-arm64 core/main.go
	@echo "🔏 macOS Apple Silicon 코드 서명 중..."
	@codesign -s - bin/$(APP_NAME)-macos-arm64 2>/dev/null || echo "⚠️ 코드 서명 실패 (계속 진행)"
	@echo "✅ macOS Apple Silicon 빌드 완료: bin/$(APP_NAME)-macos-arm64"

# Linux 64bit 빌드 (추가)
build-linux:
	@echo "🐧 Linux 64bit 빌드 중..."
	GOOS=linux GOARCH=amd64 go build -o bin/$(APP_NAME)-linux-amd64 core/main.go
	@echo "✅ Linux 빌드 완료: bin/$(APP_NAME)-linux-amd64"

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

# 배포용 패키징
package: build-all
	@echo "📦 배포용 패키징 중..."
	@mkdir -p dist
	@cp bin/$(APP_NAME)-windows-amd64.exe dist/
	@cp bin/$(APP_NAME)-macos-amd64 dist/
	@cp bin/$(APP_NAME)-macos-arm64 dist/
	@cp bin/$(APP_NAME)-linux-amd64 dist/
	@echo "📄 환경 설정 파일 포함 중..."
	@if [ -f .env ]; then \
		echo "✅ 실제 .env 파일을 배포 패키지에 포함"; \
		cp .env dist/; \
	else \
		echo "⚠️ .env 파일이 없어서 기본 .env 파일 생성"; \
		echo "# 공개 여부 설정 (true: 공개, false: 비공개)" > dist/.env; \
		echo "PUBLIC_MODE=true" >> dist/.env; \
		echo "" >> dist/.env; \
		echo "# 비공개 모드일 때 사용할 접근 키 (PUBLIC_MODE=false일 때만 필요)" >> dist/.env; \
		echo "ACCESS_KEY=your_secret_key_here" >> dist/.env; \
		echo "" >> dist/.env; \
		echo "# 이메일 알림 설정 (선택사항)" >> dist/.env; \
		echo "SMTP_HOST=smtp.gmail.com" >> dist/.env; \
		echo "SMTP_PORT=587" >> dist/.env; \
		echo "SENDER_EMAIL=your_email@gmail.com" >> dist/.env; \
		echo "SENDER_PASSWORD=your_app_password" >> dist/.env; \
	fi
	@echo "📄 예시 .env 파일도 생성 (.env.example)..."
	@echo "# 공개 여부 설정 (true: 공개, false: 비공개)" > dist/.env.example
	@echo "PUBLIC_MODE=true" >> dist/.env.example
	@echo "" >> dist/.env.example
	@echo "# 비공개 모드일 때 사용할 접근 키 (PUBLIC_MODE=false일 때만 필요)" >> dist/.env.example
	@echo "ACCESS_KEY=your_secret_key_here" >> dist/.env.example
	@echo "" >> dist/.env.example
	@echo "# 이메일 알림 설정 (선택사항)" >> dist/.env.example
	@echo "SMTP_HOST=smtp.gmail.com" >> dist/.env.example
	@echo "SMTP_PORT=587" >> dist/.env.example
	@echo "SENDER_EMAIL=your_email@gmail.com" >> dist/.env.example
	@echo "SENDER_PASSWORD=your_app_password" >> dist/.env.example
	@echo "✅ 배포용 패키징 완료: dist/ 폴더"

# 배포용 설명서 생성
release-notes:
	@echo "📝 배포용 설명서 생성 중..."
	@echo "# SRT Lurker 배포 파일" > dist/README.md
	@echo "" >> dist/README.md
	@echo "## 플랫폼별 실행 파일" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "- **Windows**: \`$(APP_NAME)-windows-amd64.exe\`" >> dist/README.md
	@echo "- **macOS (Intel)**: \`$(APP_NAME)-macos-amd64\`" >> dist/README.md
	@echo "- **macOS (Apple Silicon)**: \`$(APP_NAME)-macos-arm64\`" >> dist/README.md
	@echo "- **Linux**: \`$(APP_NAME)-linux-amd64\`" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "## 사용 방법" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "1. 해당 플랫폼의 실행 파일을 다운로드하세요" >> dist/README.md
	@echo "2. **중요**: \`.env.example\` 파일을 \`.env\`로 이름을 변경하고 설정을 수정하세요" >> dist/README.md
	@echo "3. 실행 파일과 \`.env\` 파일을 같은 폴더에 두세요" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "### Windows 사용자" >> dist/README.md
	@echo "- \`.exe\` 파일을 더블클릭하여 실행" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "### macOS 사용자" >> dist/README.md
	@echo "- 터미널에서 실행하세요: \`./파일명\`" >> dist/README.md
	@echo "- **Intel Mac**: \`srt-lurker-macos-amd64\` 사용" >> dist/README.md
	@echo "- **Apple Silicon Mac (M1/M2/M3)**: \`srt-lurker-macos-arm64\` 사용" >> dist/README.md
	@echo "- 보안 경고가 나타나면 '시스템 환경설정 > 보안 및 개인 정보 보호'에서 허용해주세요" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "### Linux 사용자" >> dist/README.md
	@echo "- 실행 권한 부여: \`chmod +x srt-lurker-linux-amd64\`" >> dist/README.md
	@echo "- 실행: \`./srt-lurker-linux-amd64\`" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "## 환경 설정 (.env 파일)" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "**반드시 \`.env.example\`을 \`.env\`로 이름 변경 후 사용하세요!**" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "\`\`\`env" >> dist/README.md
	@echo "# 공개 여부 설정 (true: 공개, false: 비공개)" >> dist/README.md
	@echo "PUBLIC_MODE=false" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "# 비공개 모드일 때 사용할 접근 키" >> dist/README.md
	@echo "ACCESS_KEY=your_secret_key_here" >> dist/README.md
	@echo "" >> dist/README.md
	@echo "# 이메일 알림 설정 (선택사항)" >> dist/README.md
	@echo "SMTP_HOST=smtp.gmail.com" >> dist/README.md
	@echo "SMTP_PORT=587" >> dist/README.md
	@echo "SENDER_EMAIL=your_email@gmail.com" >> dist/README.md
	@echo "SENDER_PASSWORD=your_app_password" >> dist/README.md
	@echo "\`\`\`" >> dist/README.md
	@echo "✅ 설명서 생성 완료: dist/README.md"

# 전체 릴리즈 준비
release: clean package release-notes
	@echo "🚀 릴리즈 준비 완료!"
	@echo "📁 dist/ 폴더의 파일들을 배포하세요"
