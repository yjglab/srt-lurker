# 🚄 SRT Lurker

SRT 고속열차 예약 자동화 시스템입니다. 원하는 시간대의 열차 예약을 자동으로 시도하고, 성공 시 이메일 알림을 발송합니다.

## 📋 주요 기능

- 🎯 **자동 예약**: 원하는 시간대 열차 예약 자동 시도
- 🔐 **접근 제어**: 공개/비공개 모드 지원
- 👤 **다중 예약 타입**: 미등록 고객 / 로그인 고객 예약 지원
- 📧 **이메일 알림**: 예약 성공/실패 시 자동 알림
- 🖥️ **크로스 플랫폼**: Windows, macOS, Linux 지원
- 🔄 **자동 재시도**: 최대 999회 재시도
- 🌐 **브라우저 자동화**: Playwright 기반 실제 웹 브라우저 제어

## 🛠️ 개발 환경 구성

### 1. 사전 요구사항

- **Go 1.19+**: [Go 설치 가이드](https://golang.org/doc/install)
- **Git**: 소스코드 관리
- **Make**: 빌드 자동화 (macOS/Linux 기본 설치, Windows는 [여기서 설치](https://gnuwin32.sourceforge.net/packages/make.htm))

### 2. 저장소 클론

```bash
git clone https://github.com/yjglab/srt-lurker.git
cd srt-lurker
```

### 3. 종속성 설치

```bash
# Go 모듈 다운로드
make install

# 또는 직접 실행
go mod download
```

### 4. Playwright 설치

```bash
# Playwright 브라우저 설치 (처음 한 번만)
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
```

### 5. 환경변수 설정

```bash
# .env 파일 생성 (프로젝트 루트에)
cp .env.example .env

# .env 파일 편집
nano .env
```

**.env 파일 설정 예시:**

```env
# 공개 여부 설정 (true: 공개, false: 비공개)
PUBLIC_MODE=true

# 비공개 모드일 때 사용할 접근 키
ACCESS_KEY=your_secret_key_here

# 이메일 알림 설정 (선택사항)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SENDER_EMAIL=your_email@gmail.com
SENDER_PASSWORD=your_app_password
```

### 6. 개발 실행

```bash
# 개발 모드로 실행
make run

# 또는 직접 실행
go run core/main.go
```

## 📦 빌드 및 배포

### 사용 가능한 빌드 명령어

```bash
# 현재 플랫폼용 빌드
make build

# 모든 플랫폼용 빌드
make build-all

# 개별 플랫폼 빌드
make build-windows    # Windows 64bit
make build-macos      # macOS Intel + Apple Silicon
make build-linux      # Linux 64bit

# 전체 배포 패키지 생성
make release
```

### 배포 절차 (상세)

#### 1단계: 환경 설정 확인

```bash
# .env 파일 설정 확인
cat .env

# 필요시 설정 수정
nano .env
```

#### 2단계: 전체 배포 빌드

```bash
# 모든 플랫폼용 빌드 + 배포 패키지 생성
make release
```

이 명령은 다음을 수행합니다:

- 기존 빌드 파일 정리 (`make clean`)
- 모든 플랫폼용 실행 파일 빌드 (`make build-all`)
- macOS 파일에 자동 코드 서명
- 배포 폴더(`dist/`) 생성 및 파일 복사
- 실제 `.env` 파일 포함
- `.env.example` 파일 생성
- 상세한 사용법 가이드(`README.md`) 생성

#### 3단계: 배포 파일 확인

```bash
# 배포 파일 확인
ls -la dist/

# 다음 파일들이 생성됨:
# - srt-lurker-windows-amd64.exe  (Windows용)
# - srt-lurker-macos-amd64        (macOS Intel용)
# - srt-lurker-macos-arm64        (macOS Apple Silicon용)
# - srt-lurker-linux-amd64        (Linux용)
# - .env                          (실제 환경 설정)
# - .env.example                  (예시 환경 설정)
# - README.md                     (사용법 가이드)
```

#### 4단계: 배포 패키지 압축

```bash
# 배포용 압축 파일 생성
cd dist
zip -r ../srt-lurker-release-$(date +%Y%m%d).zip .
cd ..

# 압축 파일 확인
ls -la srt-lurker-release-*.zip
```

#### 5단계: GitHub Release 생성 (선택사항)

```bash
# Git 태그 생성
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# GitHub에서 Release 페이지로 이동하여 압축 파일 업로드
```

## 🎯 사용법

### 일반 사용자용

1. **배포 파일 다운로드**

   - GitHub Release에서 최신 버전 다운로드
   - 압축 해제

2. **환경 설정**

   ```bash
   # .env.example을 .env로 복사
   cp .env.example .env

   # 설정 파일 편집 (원하는 모드 설정)
   nano .env
   ```

3. **실행**
   - **Windows**: `srt-lurker-windows-amd64.exe` 더블클릭
   - **macOS**: 터미널에서 `./srt-lurker-macos-arm64` (M1/M2/M3) 또는 `./srt-lurker-macos-amd64` (Intel)
   - **Linux**: `chmod +x srt-lurker-linux-amd64 && ./srt-lurker-linux-amd64`

### 개발자용

```bash
# 개발 모드 실행
make run

# 테스트 실행
make test

# 빌드 파일 정리
make clean
```

## ⚙️ 고급 설정

### 접근 제어 설정

**공개 모드** (`PUBLIC_MODE=true`):

- 누구나 바로 사용 가능
- 접근 키 불필요

**비공개 모드** (`PUBLIC_MODE=false`):

- 접근 키 입력 필요
- 최대 3회 시도 가능
- `ACCESS_KEY` 설정 필수

### 이메일 알림 설정

**Gmail 사용 시**:

1. Gmail 2단계 인증 활성화
2. 앱 비밀번호 생성
3. `.env`에 앱 비밀번호 입력

```env
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SENDER_EMAIL=your_email@gmail.com
SENDER_PASSWORD=your_app_password  # 앱 비밀번호
```

## 🔧 문제 해결

### 일반적인 문제

**1. Playwright 실행 오류**

```bash
# Playwright 브라우저 재설치
go run github.com/playwright-community/playwright-go/cmd/playwright@latest install
```

**2. macOS 보안 경고**

```bash
# 실행 권한 부여
chmod +x srt-lurker-macos-arm64

# quarantine 속성 제거 (필요시)
xattr -d com.apple.quarantine srt-lurker-macos-arm64
```

**3. Windows 실행 문제**

- 터미널(CMD/PowerShell)에서 실행
- Windows Defender 예외 설정 추가

**4. .env 파일 로딩 실패**

- 실행 파일과 `.env` 파일이 같은 폴더에 있는지 확인
- 파일 권한 확인 (`chmod 644 .env`)

### 디버깅

```bash
# 상세 로그와 함께 실행
go run core/main.go -v

# 개발 모드에서 브라우저 표시
# (headless: false가 기본 설정됨)
```

## 🤝 기여하기

1. Fork the project
2. Create your feature branch (`git checkout -b feature/AmazingFeature`)
3. Commit your changes (`git commit -m 'Add some AmazingFeature'`)
4. Push to the branch (`git push origin feature/AmazingFeature`)
5. Open a Pull Request

## 📄 라이선스

이 프로젝트는 MIT 라이선스 하에 배포됩니다. 자세한 내용은 `LICENSE` 파일을 참조하세요.

## ⚠️ 주의사항

- 이 도구는 교육 및 개인 사용 목적으로만 제작되었습니다
- 웹사이트의 서비스 약관을 준수해서 사용하세요
- 과도한 요청으로 인한 서비스 차단에 주의하세요
- 개인정보 보호에 각별히 주의하세요

## 📞 문의

**제작자**: jameskyeong ([@yjglab](https://github.com/yjglab))

- **GitHub Issues**: [Issues 페이지](https://github.com/yjglab/srt-lurker/issues)
- **Email**: yjgdesign@gmail.com

</div>
