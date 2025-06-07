//go:build ignore
// +build ignore

package main

import (
	"bufio"
	"fmt"
	"log"
	"net/smtp"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
	"golang.org/x/term"
)

// ═══════════════════════════════════════════════════════════════════════════════
// 📋 상수 정의
// ═══════════════════════════════════════════════════════════════════════════════

const (
	initialURL = "https://etk.srail.kr/hpg/hra/01/selectScheduleList.do?pageId=TK0101010000"
	maxRetries = 999
)

const (
	// 🚉 기본 페이지 요소들
	dptStationSelector                = "input#dptRsStnCdNm"
	arvStationSelector                = "input#arvRsStnCdNm"
	dateSelector                      = "select#dptDt"
	searchButtonSelector              = "input[value='조회하기']"
	unregisteredReserveButtonSelector = "a.btn_midium.btn_pastel1:has-text('미등록고객 예매')"
	passengerAgreeSelector            = "input#agreeY"
	passengerNameSelector             = "input#custNm"

	// 🔐 로그인 타입 라디오 버튼들
	loginTypeMemberIdSelector = "input#srchDvCd1"
	loginTypeEmailSelector    = "input#srchDvCd2"
	loginTypePhoneSelector    = "input#srchDvCd3"
)

// ═══════════════════════════════════════════════════════════════════════════════
// 🚄 SRT 역 목록 및 사용자 정보 구조체
// ═══════════════════════════════════════════════════════════════════════════════

var srtStations = []string{
	"수서", "동탄", "평택지제", "천안아산", "오송", "대전", "김천구미", "동대구",
	"경주", "울산", "부산", "광명", "서대전", "익산", "정읍", "광주송정", "전주",
	"남원", "곡성", "구례구", "순천", "여천", "여수EXPO", "신경주", "포항",
}

// 👤 승객 정보 구조체
var passengerInfo = struct {
	deptStation         string
	arrivalStation      string
	deptTime            string
	arrivalTime         string
	date                string
	name                string
	phone               string
	password            string
	notificationEmail   string
	notificationEnabled bool
	customerType        string // "unregistered" 또는 "login"
	loginType           string // "member", "email", "phone"
	loginId             string // 로그인 ID (회원번호/이메일/전화번호)
	loginPassword       string // 로그인 비밀번호
}{
	deptStation:         "",
	arrivalStation:      "",
	deptTime:            "",
	arrivalTime:         "",
	date:                "",
	name:                "",
	phone:               "",
	password:            "",
	notificationEmail:   "",
	notificationEnabled: false,
	customerType:        "",
	loginType:           "",
	loginId:             "",
	loginPassword:       "",
}

// 📧 이메일 설정 구조체
var emailConfig = struct {
	smtpHost    string
	smtpPort    string
	senderEmail string
	senderPass  string
}{
	smtpHost:    "",
	smtpPort:    "",
	senderEmail: "",
	senderPass:  "",
}

// ═══════════════════════════════════════════════════════════════════════════════
// 🔧 유틸리티 함수들
// ═══════════════════════════════════════════════════════════════════════════════

// ---------- 유틸리티 함수들 ----------

func must(message string, err error) {
	if err != nil {
		log.Fatalf(message, err)
	}
}

func wait(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

// 🌟 로딩 애니메이션을 표시하는 함수
func showLoadingAnimation(message string, duration int) {
	done := make(chan bool)

	go func() {
		// 더 예쁜 유니코드 스피너 (브라이 패턴)
		spinner := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		i := 0
		for {
			select {
			case <-done:
				return
			default:
				fmt.Printf("\r   %s %s", spinner[i%len(spinner)], message)
				i++
				time.Sleep(100 * time.Millisecond) // 더 빠른 속도
			}
		}
	}()

	time.Sleep(time.Duration(duration) * time.Second)
	done <- true
	fmt.Printf("\r   ✓ %s (완료)\n", message)
}

func safeAction(action func() error, errorMsg string) error {
	if err := action(); err != nil {
		return fmt.Errorf("%s: %w", errorMsg, err)
	}
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// 📋 대화형 메뉴 관련 함수들
// ═══════════════════════════════════════════════════════════════════════════════

func selectFromMenu(title string, items []string) string {
	itemsPerPage := 10
	currentPage := 0
	totalPages := (len(items) + itemsPerPage - 1) / itemsPerPage

	for {
		start := currentPage * itemsPerPage
		end := start + itemsPerPage
		if end > len(items) {
			end = len(items)
		}

		fmt.Print("\033[2J\033[H")
		fmt.Printf("🚄 %s\n", title)
		fmt.Println(strings.Repeat("=", 50))
		fmt.Printf("페이지 %d/%d (총 %d개 역)\n", currentPage+1, totalPages, len(items))
		fmt.Println()

		for i := start; i < end; i++ {
			fmt.Printf("  %d. %s\n", i-start+1, items[i])
		}

		fmt.Println()
		fmt.Println("📋 선택 방법:")
		fmt.Println("  1-10: 번호로 역 선택")
		if currentPage > 0 {
			fmt.Println("  p: 이전 페이지")
		}
		if currentPage < totalPages-1 {
			fmt.Println("  n: 다음 페이지")
		}
		fmt.Println("  q: 프로그램 종료")
		fmt.Print("\n선택하세요: ")

		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(strings.ToLower(input))

		switch input {
		case "q":
			fmt.Println("프로그램을 종료합니다.")
			os.Exit(0)
		case "n":
			if currentPage < totalPages-1 {
				currentPage++
			}
		case "p":
			if currentPage > 0 {
				currentPage--
			}
		default:
			if num, err := strconv.Atoi(input); err == nil {
				if num >= 1 && num <= end-start {
					selectedIndex := start + num - 1
					fmt.Print("\033[2J\033[H")
					return items[selectedIndex]
				}
			}
			fmt.Printf("❌ 잘못된 입력입니다. 1-%d 또는 n/p/q를 입력하세요.\n", end-start)
			fmt.Print("아무 키나 눌러서 계속...")
			reader.ReadString('\n')
		}
	}
}

func selectStation(title string) string {
	return selectFromMenu(title, srtStations)
}

// ═══════════════════════════════════════════════════════════════════════════════
// 🤖 웹 자동화 헬퍼 함수들
// ═══════════════════════════════════════════════════════════════════════════════

func fillInput(page playwright.Page, selector, value, fieldName string) error {
	input := page.Locator(selector)

	if err := safeAction(func() error { return input.Click() }, fieldName+" 클릭 실패"); err != nil {
		return err
	}
	if err := safeAction(func() error { return input.Fill("") }, fieldName+" 입력 비우기 실패"); err != nil {
		return err
	}
	if err := safeAction(func() error { return input.Fill(value) }, fieldName+" 입력 실패"); err != nil {
		return err
	}
	if err := safeAction(func() error { return input.Press("Tab") }, fieldName+" 확정 실패"); err != nil {
		return err
	}

	return nil
}

func selectOption(page playwright.Page, selector, value, fieldName string) error {
	_, err := page.Locator(selector).SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice(value),
	})
	return safeAction(func() error { return err }, fieldName+" 선택 실패")
}

func clickButton(page playwright.Page, selector, buttonName string) error {
	return safeAction(func() error {
		return page.Locator(selector).Click()
	}, buttonName+" 클릭 실패")
}

func setupDialogHandler(page playwright.Page, acceptDialog bool) {
	page.OnDialog(func(dialog playwright.Dialog) {
		fmt.Printf("   > 대화상자 감지: %s\n", dialog.Message())
		if acceptDialog {
			fmt.Println("   > 자동으로 '확인' 클릭")
			dialog.Accept()
		} else {
			fmt.Println("   > 자동으로 '취소' 클릭")
			dialog.Dismiss()
		}
	})
}

// ═══════════════════════════════════════════════════════════════════════════════
// 💬 사용자 입력 및 인터페이스 함수들
// ═══════════════════════════════════════════════════════════════════════════════

func printHeader(title string) {
	fmt.Println()
	fmt.Println("🚄 " + strings.Repeat("=", 50))
	fmt.Printf("   %s\n", title)
	fmt.Println("   " + strings.Repeat("=", 50))
	fmt.Println()
}

func printSubHeader(title string) {
	fmt.Println()
	fmt.Printf("📋 %s\n", title)
	fmt.Println("   " + strings.Repeat("-", 30))
}

func getUserInput(prompt, defaultValue string, examples ...string) string {
	reader := bufio.NewReader(os.Stdin)

	if defaultValue != "" {
		fmt.Printf("   %s [기본값: %s]: ", prompt, defaultValue)
	} else {
		fmt.Printf("   %s: ", prompt)
	}

	if len(examples) > 0 && examples[0] != "" {
		fmt.Printf("\n   💡 예시: %s\n   입력: ", examples[0])
	}

	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" && defaultValue != "" {
		return defaultValue
	}

	return input
}

// 🔐 비밀번호 입력 전용 함수 (화면에 표시되지 않음)
func getPasswordInput(prompt string) string {
	fmt.Printf("   %s: ", prompt)

	// 터미널을 raw 모드로 설정
	fd := int(os.Stdin.Fd())
	oldState, err := term.MakeRaw(fd)
	if err != nil {
		// raw 모드 설정 실패 시 일반 입력으로 fallback
		fmt.Println("(보안 입력 모드 실패, 일반 입력으로 진행)")
		reader := bufio.NewReader(os.Stdin)
		input, _ := reader.ReadString('\n')
		return strings.TrimSpace(input)
	}
	defer term.Restore(fd, oldState)

	var password []byte
	for {
		char := make([]byte, 1)
		_, err := os.Stdin.Read(char)
		if err != nil {
			break
		}

		// Enter 키 (13 또는 10)
		if char[0] == 13 || char[0] == 10 {
			fmt.Println() // 줄바꿈
			break
		}

		// Backspace 키 (127 또는 8)
		if char[0] == 127 || char[0] == 8 {
			if len(password) > 0 {
				password = password[:len(password)-1]
			}
			continue
		}

		// Ctrl+C (3)
		if char[0] == 3 {
			fmt.Println()
			os.Exit(0)
		}

		// 일반 문자만 추가
		if char[0] >= 32 && char[0] <= 126 {
			password = append(password, char[0])
		}
	}

	return string(password)
}

func getYesNoInput(prompt string, defaultValue bool) bool {
	defaultStr := "N"
	if defaultValue {
		defaultStr = "Y"
	}

	for {
		input := getUserInput(prompt+" (Y/N)", defaultStr)
		input = strings.ToUpper(strings.TrimSpace(input))

		switch input {
		case "Y":
			return true
		case "N":
			return false
		default:
			fmt.Println("   ❌ Y 또는 N으로 입력해주세요.")
			fmt.Println()
		}
	}
}

func getInputWithValidation(prompt, defaultValue string, validator func(string) bool, examples ...string) string {
	for {
		input := getUserInput(prompt, defaultValue, examples...)
		if validator(input) {
			return input
		}
	}
}

// ═══════════════════════════════════════════════════════════════════════════════
// ✅ 입력 검증 함수들
// ═══════════════════════════════════════════════════════════════════════════════

func validateRequired(value, fieldName string) bool {
	if strings.TrimSpace(value) == "" {
		fmt.Printf("   ❌ %s는 필수 입력 항목입니다.\n\n", fieldName)
		return false
	}
	return true
}

func validatePhone(phone string) bool {
	re := regexp.MustCompile(`^010\d{8}$`)
	if !re.MatchString(phone) {
		fmt.Println("   ❌ 전화번호는 010으로 시작하는 11자리 숫자여야 합니다.")
		fmt.Println("   💡 예시: 01012345678")
		fmt.Println()
		return false
	}
	return true
}

func validateTime(timeStr string) bool {
	if !validateRequired(timeStr, "시간") {
		return false
	}

	// 4자리 숫자인지 확인
	if len(timeStr) != 4 {
		fmt.Println("   ❌ 시간은 4자리 숫자로 입력해주세요.")
		fmt.Println("   💡 예시: 1037 (10시 37분), 0622 (06시 22분)")
		fmt.Println()
		return false
	}

	// 숫자인지 확인
	if _, err := strconv.Atoi(timeStr); err != nil {
		fmt.Println("   ❌ 시간은 숫자만 입력 가능합니다.")
		fmt.Println("   💡 예시: 1037 (10시 37분), 0622 (06시 22분)")
		fmt.Println()
		return false
	}

	// 시간과 분 추출
	hour, _ := strconv.Atoi(timeStr[:2])
	minute, _ := strconv.Atoi(timeStr[2:])

	// 시간 범위 확인 (00~23)
	if hour < 0 || hour > 23 {
		fmt.Println("   ❌ 시간은 00~23 사이여야 합니다.")
		fmt.Println("   💡 예시: 1037 (10시 37분), 0622 (06시 22분)")
		fmt.Println()
		return false
	}

	// 분 범위 확인 (00~59)
	if minute < 0 || minute > 59 {
		fmt.Println("   ❌ 분은 00~59 사이여야 합니다.")
		fmt.Println("   💡 예시: 1037 (10시 37분), 0622 (06시 22분)")
		fmt.Println()
		return false
	}

	return true
}

func validateMonth(monthStr string) bool {
	if !validateRequired(monthStr, "출발 월") {
		return false
	}

	month, err := strconv.Atoi(monthStr)
	if err != nil {
		fmt.Println("   ❌ 월은 숫자로 입력해주세요.")
		fmt.Println()
		return false
	}

	if month < 1 || month > 12 {
		fmt.Println("   ❌ 월은 1~12 사이의 숫자여야 합니다.")
		fmt.Println("   💡 예시: 6")
		fmt.Println()
		return false
	}

	return true
}

func validateDay(dayStr string) bool {
	if !validateRequired(dayStr, "출발 일") {
		return false
	}

	day, err := strconv.Atoi(dayStr)
	if err != nil {
		fmt.Println("   ❌ 일은 숫자로 입력해주세요.")
		fmt.Println()
		return false
	}

	if day < 1 || day > 31 {
		fmt.Println("   ❌ 일은 1~31 사이의 숫자여야 합니다.")
		fmt.Println("   💡 예시: 22")
		fmt.Println()
		return false
	}

	return true
}

func validateDate(dateStr string) bool {
	re := regexp.MustCompile(`^\d{8}$`)
	if !re.MatchString(dateStr) {
		fmt.Println("   ❌ 날짜는 YYYYMMDD 형식으로 입력해주세요.")
		fmt.Println("   💡 예시: 20250622")
		fmt.Println()
		return false
	}

	if _, err := time.Parse("20060102", dateStr); err != nil {
		fmt.Println("   ❌ 유효하지 않은 날짜입니다.")
		fmt.Println()
		return false
	}

	return true
}

func validateEmail(email string) bool {
	if email == "" {
		return true
	}

	re := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !re.MatchString(email) {
		fmt.Println("   ❌ 올바른 이메일 형식이 아닙니다.")
		fmt.Println("   💡 예시: example@gmail.com")
		fmt.Println()
		return false
	}
	return true
}

func validatePassword(password string) bool {
	if !validateRequired(password, "비밀번호") {
		return false
	}
	if len(password) != 5 {
		fmt.Println("   ❌ 비밀번호는 5자리여야 합니다.")
		fmt.Println()
		return false
	}
	if _, err := strconv.Atoi(password); err != nil {
		fmt.Println("   ❌ 비밀번호는 숫자만 입력 가능합니다.")
		fmt.Println()
		return false
	}
	return true
}

// ═══════════════════════════════════════════════════════════════════════════════
// 📝 사용자 입력 수집 메인 함수
// ═══════════════════════════════════════════════════════════════════════════════

func collectUserInput() {
	printHeader("SRT 고속열차 예약 시스템")
	fmt.Println("   🎯 예약에 필요한 정보를 입력해주세요")
	fmt.Println("   ℹ️  각 항목에 대한 예시를 참고하여 정확히 입력해주세요")

	// 🚫 시스템 제한사항 공지
	printSubHeader("⚠️  시스템 제한사항")
	fmt.Println("   🚫 좌석 선택 기능: 현재 제공하지 않음 (자동 배정)")
	fmt.Println("   🚫 인원 수 선택 기능: 현재 제공하지 않음 (1명 기준)")
	fmt.Println("   ℹ️  위 기능들은 추후 업데이트 예정입니다")
	fmt.Println()

	// 👤 고객 유형 선택
	printSubHeader("👤 고객 유형 선택")
	fmt.Println("   1. 미등록 고객 예매 (회원가입 없이 예약)")
	fmt.Println("   2. 로그인 고객 예매 (SRT 회원 로그인)")
	fmt.Println()

	for {
		customerChoice := getUserInput("고객 유형을 선택하세요 (1 또는 2)", "1")
		switch customerChoice {
		case "1":
			passengerInfo.customerType = "unregistered"
			fmt.Println("   ✅ 미등록 고객 예매로 진행합니다.")
			fmt.Println()
			break
		case "2":
			passengerInfo.customerType = "login"
			fmt.Println("   ✅ 로그인 고객 예매로 진행합니다.")
			fmt.Println()
			break
		default:
			fmt.Println("   ❌ 1 또는 2를 입력해주세요.")
			fmt.Println()
			continue
		}
		break
	}

	// 역 정보 선택
	printSubHeader("🚉 역 정보")
	fmt.Println("   출발역을 선택해주세요...")
	time.Sleep(1 * time.Second)
	passengerInfo.deptStation = selectStation("출발역을 선택하세요")

	fmt.Printf("   ✅ 출발역: %s\n", passengerInfo.deptStation)
	fmt.Println("   도착역을 선택해주세요...")
	time.Sleep(1 * time.Second)
	passengerInfo.arrivalStation = selectStation("도착역을 선택하세요")

	fmt.Printf("   ✅ 도착역: %s\n", passengerInfo.arrivalStation)
	fmt.Println()

	// 시간 정보 입력
	printSubHeader("⏰ 시간 정보")

	// 현재 연도 자동 설정
	currentYear := time.Now().Year()
	fmt.Printf("   📅 출발 연도: %d (자동 설정)\n", currentYear)

	// 출발 월 입력
	monthStr := getInputWithValidation(
		"출발 월을 입력하세요 (1~12)",
		"",
		validateMonth,
		"6",
	)

	// 출발 일 입력
	dayStr := getInputWithValidation(
		"출발 일을 입력하세요 (1~31)",
		"",
		validateDay,
		"22",
	)

	// YYYYMMDD 형식으로 조합
	month, _ := strconv.Atoi(monthStr)
	day, _ := strconv.Atoi(dayStr)
	passengerInfo.date = fmt.Sprintf("%04d%02d%02d", currentYear, month, day)

	fmt.Printf("   ✅ 출발날짜: %s (%d년 %d월 %d일)\n", passengerInfo.date, currentYear, month, day)

	// 출발시간 입력 (4자리 숫자로 입력받아 HH:MM 형식으로 변환)
	deptTimeStr := getInputWithValidation(
		"출발시간을 입력하세요 (4자리 숫자)",
		"",
		validateTime,
		"1037",
	)

	// HHMM → HH:MM 형식으로 변환
	deptHour := deptTimeStr[:2]
	deptMinute := deptTimeStr[2:]
	passengerInfo.deptTime = fmt.Sprintf("%s:%s", deptHour, deptMinute)

	fmt.Printf("   ✅ 출발시간: %s\n", passengerInfo.deptTime)

	// 도착시간 입력 (4자리 숫자로 입력받아 HH:MM 형식으로 변환)
	arrivalTimeStr := getInputWithValidation(
		"도착시간을 입력하세요 (4자리 숫자)",
		"",
		validateTime,
		"1207",
	)

	// HHMM → HH:MM 형식으로 변환
	arrivalHour := arrivalTimeStr[:2]
	arrivalMinute := arrivalTimeStr[2:]
	passengerInfo.arrivalTime = fmt.Sprintf("%s:%s", arrivalHour, arrivalMinute)

	fmt.Printf("   ✅ 도착시간: %s\n", passengerInfo.arrivalTime)

	// 예약자 정보 입력 (미등록 고객만)
	if passengerInfo.customerType == "unregistered" {
		printSubHeader("👤 예약자 정보")
		passengerInfo.name = getInputWithValidation(
			"예약자 이름을 입력하세요",
			"",
			func(s string) bool { return validateRequired(s, "예약자 이름") },
			"홍길동",
		)

		passengerInfo.phone = getInputWithValidation(
			"전화번호를 입력하세요 (숫자만)",
			"",
			func(s string) bool { return validateRequired(s, "전화번호") && validatePhone(s) },
			"01012345678",
		)

		// 비밀번호 입력
		printSubHeader("🔐 비밀번호 설정")
		for {
			passengerInfo.password = getPasswordInput("비밀번호를 입력하세요 (5자리 숫자)")
			if validatePassword(passengerInfo.password) {
				break
			}
		}
	} else {
		fmt.Println()
		fmt.Println("   ℹ️ 로그인 고객 예매는 회원 정보를 사용하므로 별도 입력이 불필요해요")

		// 로그인 정보 입력
		printSubHeader("🔐 로그인 정보")
		fmt.Println("   로그인 타입을 선택하세요:")
		fmt.Println("   1. 회원번호로 로그인")
		fmt.Println("   2. 이메일로 로그인")
		fmt.Println("   3. 전화번호로 로그인")
		fmt.Println()

		for {
			loginChoice := getUserInput("로그인 타입을 선택하세요 (1, 2, 3)", "1")
			switch loginChoice {
			case "1":
				passengerInfo.loginType = "member"
				fmt.Println("   ✅ 회원번호 로그인을 선택했습니다.")
				passengerInfo.loginId = getInputWithValidation(
					"회원번호를 입력하세요",
					"",
					func(s string) bool { return validateRequired(s, "회원번호") },
					"1234567890",
				)
				break
			case "2":
				passengerInfo.loginType = "email"
				fmt.Println("   ✅ 이메일 로그인을 선택했습니다.")
				passengerInfo.loginId = getInputWithValidation(
					"이메일을 입력하세요",
					"",
					func(s string) bool { return validateRequired(s, "이메일") && validateEmail(s) },
					"example@gmail.com",
				)
				break
			case "3":
				passengerInfo.loginType = "phone"
				fmt.Println("   ✅ 전화번호 로그인을 선택했습니다.")
				passengerInfo.loginId = getInputWithValidation(
					"전화번호를 입력하세요 (숫자만)",
					"",
					func(s string) bool { return validateRequired(s, "전화번호") && validatePhone(s) },
					"01012345678",
				)
				break
			default:
				fmt.Println("   ❌ 1, 2, 3 중 하나를 입력해주세요.")
				fmt.Println()
				continue
			}
			break
		}

		for {
			passengerInfo.loginPassword = getPasswordInput("로그인 비밀번호를 입력하세요")
			if validateRequired(passengerInfo.loginPassword, "로그인 비밀번호") {
				break
			}
		}
	}

	// 알림 설정
	printSubHeader("📧 알림 설정")
	passengerInfo.notificationEnabled = getYesNoInput("예약 완료 시 이메일 알림을 받으시겠습니까?", false)

	if passengerInfo.notificationEnabled {
		passengerInfo.notificationEmail = getInputWithValidation(
			"알림받을 이메일 주소를 입력하세요",
			"",
			validateEmail,
			"example@gmail.com",
		)
	}

	// 입력 정보 확인
	printSubHeader("✅ 입력 정보 확인")
	fmt.Printf("    고객 유형: %s\n",
		map[string]string{
			"unregistered": "미등록 고객 예매",
			"login":        "로그인 고객 예매",
		}[passengerInfo.customerType])
	fmt.Printf("    출발역: %s (%s)\n", passengerInfo.deptStation, passengerInfo.deptTime)
	fmt.Printf("    도착역: %s (%s)\n", passengerInfo.arrivalStation, passengerInfo.arrivalTime)
	fmt.Printf("    날짜: %s\n", passengerInfo.date)

	if passengerInfo.customerType == "unregistered" {
		fmt.Printf("    예약자: %s\n", passengerInfo.name)
		fmt.Printf("    전화번호: %s\n", passengerInfo.phone)
	} else {
		loginTypeMap := map[string]string{
			"member": "회원번호",
			"email":  "이메일",
			"phone":  "전화번호",
		}
		fmt.Printf("    로그인 타입: %s\n", loginTypeMap[passengerInfo.loginType])
		fmt.Printf("    로그인 ID: %s\n", passengerInfo.loginId)
	}

	if passengerInfo.notificationEnabled {
		fmt.Printf("    알림 이메일: %s\n", passengerInfo.notificationEmail)
	}
	fmt.Println()

	if !getYesNoInput("위 정보가 맞습니까?", true) {
		fmt.Println("   🔄 정보를 다시 입력합니다.")
		collectUserInput()
		return
	}

	fmt.Println("   ✅ 정보 확인 완료! 예약을 시작합니다.")
	fmt.Println()
}

// ═══════════════════════════════════════════════════════════════════════════════
// 🚀 자동화 단계별 처리 함수들
// ═══════════════════════════════════════════════════════════════════════════════

func step1SetStations(page playwright.Page) error {
	fmt.Println("🚉 1단계: 출발역/도착역 설정")

	fmt.Printf("   > 출발역: %s\n", passengerInfo.deptStation)
	if err := fillInput(page, dptStationSelector, passengerInfo.deptStation, "출발역"); err != nil {
		return err
	}

	fmt.Printf("   > 도착역: %s\n", passengerInfo.arrivalStation)
	if err := fillInput(page, arvStationSelector, passengerInfo.arrivalStation, "도착역"); err != nil {
		return err
	}

	fmt.Println("   ✓ 출발역/도착역 설정 완료")
	return nil
}

func step2SetDate(page playwright.Page) error {
	fmt.Println("📅 2단계: 출발 날짜 설정")
	if err := selectOption(page, dateSelector, passengerInfo.date, "날짜"); err != nil {
		return err
	}
	fmt.Println("   ✓ 출발 날짜 설정 완료")
	return nil
}

func step3SearchTrains(page playwright.Page) error {
	fmt.Println("🔍 3단계: 열차 조회")
	if err := clickButton(page, searchButtonSelector, "조회 버튼"); err != nil {
		return err
	}

	// 로딩 애니메이션과 함께 대기
	showLoadingAnimation("열차 정보를 조회하는 중이에요", 3)

	return nil
}

func step4CheckAvailability(page playwright.Page) error {
	fmt.Println("📋 4단계: 예약 가능 열차 확인")

	netfunnelLocator := page.Locator("div#NetFunnel_Skin_Top")
	if count, _ := netfunnelLocator.Count(); count > 0 {
		fmt.Println("   ⏳ 대기열에 진입했어요")

		// 대기열 진입 애니메이션
		done := make(chan bool)
		go func() {
			// 더 예쁜 대기열 스피너 (회전하는 점들)
			spinner := []string{"🔄", "🔃", "🔄", "🔃"}
			i := 0
			for {
				select {
				case <-done:
					return
				default:
					fmt.Printf("\r   %s 대기열에서 순서를 기다리는 중이에요 (최대 1분)", spinner[i%len(spinner)])
					i++
					time.Sleep(250 * time.Millisecond) // 더 빠른 속도 (500ms → 250ms)
				}
			}
		}()

		netfunnelLocator.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateHidden,
			Timeout: playwright.Float(1000 * 60),
		})

		done <- true
		fmt.Printf("\r   ✅ 대기열 통과 완료!                                    \n")
	}

	// 열차 정보 확인 중 애니메이션
	showLoadingAnimation("예약 가능한 열차를 확인하는 중이에요", 1)

	trs, err := page.Locator("tbody > tr").All()
	if err != nil {
		return err
	}

	for _, tr := range trs {
		tds, err := tr.Locator("td").All()
		if err != nil {
			return err
		}

		if len(tds) < 5 {
			continue
		}

		dept, err := tds[3].Locator("em").TextContent()
		if err != nil {
			continue
		}
		arrival, err := tds[4].Locator("em").TextContent()
		if err != nil {
			continue
		}

		fmt.Println(dept, arrival)

		if strings.Contains(dept, passengerInfo.deptTime) && strings.Contains(arrival, passengerInfo.arrivalTime) {
			fmt.Println("   ✓ 예약 가능한 열차 발견")
			return nil
		}
	}

	return fmt.Errorf("예약 가능한 열차를 찾을 수 없습니다")
}

func step5ClickReserve(page playwright.Page) error {
	fmt.Println("🎯 5단계: 예약 시도")

	trs, err := page.Locator("tbody > tr").All()
	if err != nil {
		return err
	}

	for _, tr := range trs {
		tds, err := tr.Locator("td").All()
		if err != nil {
			continue
		}

		if len(tds) < 7 {
			continue
		}

		dept, err := tds[3].Locator("em").TextContent()
		if err != nil {
			continue
		}
		arrival, err := tds[4].Locator("em").TextContent()
		if err != nil {
			continue
		}

		if strings.Contains(dept, passengerInfo.deptTime) && strings.Contains(arrival, passengerInfo.arrivalTime) {
			fullText, err := tds[6].Locator("span:has-text('매진')").Count()
			if err != nil {
				continue
			}
			if fullText > 0 {
				return fmt.Errorf("매진된 열차입니다 - 예매를 다시 시도합니다")
			}

			reserveButton := tds[6].Locator("a > span:has-text('예약하기')")
			if err := reserveButton.Click(); err != nil {
				continue
			}
			fmt.Println("   ✓ 예약하기 버튼 클릭 완료")
			return nil
		}
	}

	return fmt.Errorf("예약하기 버튼을 찾을 수 없습니다")
}

func step6ChooseReservationType(page playwright.Page) error {
	showLoadingAnimation("예매 페이지로 이동하는 중이에요", 1)
	fmt.Println("🛂 6단계: 예매 경로 선택")

	setupDialogHandler(page, true)

	// 미등록 고객인 경우 미등록고객 예매 버튼 클릭
	if passengerInfo.customerType == "unregistered" {
		fmt.Println("   > 미등록 고객 예매 선택")
		if err := clickButton(page, unregisteredReserveButtonSelector, "미등록고객 예매 버튼"); err != nil {
			return err
		}
		fmt.Println("   ✓ 미등록고객 예매 버튼 클릭 및 대화상자 처리 완료")
	}

	// 로그인 고객인 경우 스킵

	return nil
}

func step7LoginProcess(page playwright.Page) error {
	if passengerInfo.customerType == "unregistered" {
		// 미등록 고객: 예약자 정보 입력 화면 확인
		currentURL := page.URL()
		if !strings.Contains(currentURL, "selectReservationForm") {
			return fmt.Errorf("예약 페이지로 이동하지 못했습니다 (현재 URL: %s)", currentURL)
		}
		fmt.Println("   ✓ 예약자 정보 입력 화면으로 이동 완료")
		return nil
	} else {
		// 로그인 고객: 실제 로그인 처리
		return step7ProcessLogin(page)
	}
}

func step7ProcessLogin(page playwright.Page) error {
	fmt.Println("▶ 7단계: 로그인 처리")

	// 로그인 타입에 따른 라디오 버튼 선택 및 입력 필드 selector 생성
	var loginTypeSelector string
	var loginIdSelector string
	var loginPasswordSelector string
	var loginSubmitSelector string

	switch passengerInfo.loginType {
	case "member":
		loginTypeSelector = loginTypeMemberIdSelector
		loginIdSelector = "input#srchDvNm01"
		loginPasswordSelector = "input#hmpgPwdCphd01"
		loginSubmitSelector = "div.srchDvCd1 input.loginSubmit"
		fmt.Println("   > 회원번호 로그인 선택")
	case "email":
		loginTypeSelector = loginTypeEmailSelector
		loginIdSelector = "input#srchDvNm02"
		loginPasswordSelector = "input#hmpgPwdCphd02"
		loginSubmitSelector = "div.srchDvCd2 input.loginSubmit"
		fmt.Println("   > 이메일 로그인 선택")
	case "phone":
		loginTypeSelector = loginTypePhoneSelector
		loginIdSelector = "input#srchDvNm03"
		loginPasswordSelector = "input#hmpgPwdCphd03"
		loginSubmitSelector = "div.srchDvCd3 input.loginSubmit"
		fmt.Println("   > 전화번호 로그인 선택")
	default:
		return fmt.Errorf("알 수 없는 로그인 타입: %s", passengerInfo.loginType)
	}

	// 로그인 타입 라디오 버튼 클릭
	if err := clickButton(page, loginTypeSelector, "로그인 타입"); err != nil {
		return fmt.Errorf("로그인 타입 선택 실패: %w", err)
	}

	showLoadingAnimation("로그인 폼을 준비하는 중이에요", 1)

	// 로그인 ID 입력
	fmt.Printf("   > 로그인 ID 입력: %s\n", passengerInfo.loginId)
	if err := fillInput(page, loginIdSelector, passengerInfo.loginId, "로그인 ID"); err != nil {
		return fmt.Errorf("로그인 ID 입력 실패: %w", err)
	}

	// 로그인 비밀번호 입력
	fmt.Printf("   > 로그인 비밀번호 입력: %s\n", strings.Repeat("*", len(passengerInfo.loginPassword)))
	if err := fillInput(page, loginPasswordSelector, passengerInfo.loginPassword, "로그인 비밀번호"); err != nil {
		return fmt.Errorf("로그인 비밀번호 입력 실패: %w", err)
	}

	// 로그인 버튼 클릭
	fmt.Println("   > 로그인 버튼 클릭")
	if err := clickButton(page, loginSubmitSelector, "로그인 제출"); err != nil {
		return fmt.Errorf("로그인 버튼 클릭 실패: %w", err)
	}

	showLoadingAnimation("로그인 처리 중이에요", 3)

	// 로그인 성공 확인 (URL이나 특정 요소로 확인 가능)
	currentURL := page.URL()
	if strings.Contains(currentURL, "login") {
		return fmt.Errorf("로그인에 실패했습니다. 아이디나 비밀번호를 확인해주세요")
	}

	// '나중에 변경하기' 링크가 있으면 클릭
	laterChangeLink := page.Locator("a:has-text('나중에 변경하기')")
	if count, _ := laterChangeLink.Count(); count > 0 {
		fmt.Println("   > '나중에 변경하기' 링크 발견, 클릭합니다...")
		if err := laterChangeLink.Click(); err != nil {
			fmt.Printf("   ⚠️ '나중에 변경하기' 링크 클릭 실패 (계속 진행): %v\n", err)
		} else {
			fmt.Println("   ✓ '나중에 변경하기' 링크 클릭 완료")
			showLoadingAnimation("페이지 이동을 기다리는 중이에요", 2)
		}
	}

	fmt.Println("   ✓ 로그인 완료")

	// 로그인 고객은 여기서 예약이 완료됨
	fmt.Println("🎉 로그인 고객 예약이 완료되었습니다!")

	return nil
}

func step8FillPassengerInfoUnregistered(page playwright.Page) error {
	fmt.Println("▶ 8단계: 예약자 정보 입력 (미등록 고객)")

	if err := clickButton(page, passengerAgreeSelector, "개인정보수집 동의 체크박스"); err != nil {
		return err
	}

	if err := fillInput(page, passengerNameSelector, passengerInfo.name, "예약자 이름"); err != nil {
		return err
	}

	inputValues := []struct {
		value string
		desc  string
	}{
		{passengerInfo.phone[:3], "전화번호 앞자리"},
		{passengerInfo.phone[3:7], "전화번호 중간자리"},
		{passengerInfo.phone[7:], "전화번호 뒷자리"},
		{passengerInfo.password, "비밀번호"},
		{passengerInfo.password, "비밀번호 확인"},
	}

	for _, input := range inputValues {
		if err := page.Keyboard().Type(input.value); err != nil {
			return fmt.Errorf("%s 입력 실패: %w", input.desc, err)
		}
		fmt.Printf("   ✓ %s 입력 완료\n", input.desc)

		if err := page.Keyboard().Press("Tab"); err != nil {
			return fmt.Errorf("%s 입력 후 Tab 이동 실패: %w", input.desc, err)
		}
	}

	fmt.Println("   ✓ 예약자 정보 입력 완료")
	// 예약 확정 (Tab + Enter)
	fmt.Println("   > 예약 확정 버튼으로 이동 및 클릭")

	if err := page.Keyboard().Press("Enter"); err != nil {
		return fmt.Errorf("예약 확정 Enter 키 입력 실패: %w", err)
	}

	showLoadingAnimation("예약을 처리하는 중이에요", 3)

	fmt.Println("🎉 미등록 고객 예약이 완료되었어요!")

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// 🔄 예약 시도 메인 함수
// ═══════════════════════════════════════════════════════════════════════════════

func attemptReservation(page playwright.Page, attempt int) error {
	fmt.Printf("\n↻ 시도 %d/%d 시작...\n", attempt, maxRetries)
	fmt.Println(strings.Repeat("=", 50))

	if attempt > 1 {
		fmt.Println("⟳ 페이지 새로고침...")
		if _, err := page.Reload(); err != nil {
			return fmt.Errorf("페이지 새로고침 실패: %w", err)
		}
		showLoadingAnimation("페이지를 새로고침하고 있어요", 3)
	}

	steps := []func(playwright.Page) error{
		step1SetStations,
		step2SetDate,
		step3SearchTrains,
		step4CheckAvailability,
		step5ClickReserve,
		step6ChooseReservationType,
		step7LoginProcess,
	}

	// 미등록 고객만 step8 (예약자 정보 입력) 필요
	if passengerInfo.customerType == "unregistered" {
		steps = append(steps, step8FillPassengerInfoUnregistered)
	}

	for i, step := range steps {
		if i == 4 {
			showLoadingAnimation("예약 시도를 준비하는 중이에요", 3)
		}
		if err := step(page); err != nil {
			return err
		}
	}

	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// 📧 이메일 알림 관련 함수들
// ═══════════════════════════════════════════════════════════════════════════════

func sendNotificationEmail(success bool, message string) error {
	if !passengerInfo.notificationEnabled {
		fmt.Println("   ℹ️ 이메일 발송이 비활성화되어 있어요")
		return nil
	}

	var subject, body string
	if success {
		customerTypeText := "미등록고객"
		if passengerInfo.customerType == "login" {
			customerTypeText = "로그인고객"
		}

		subject = fmt.Sprintf("🚄 SRT %s 예약 성공 알림", customerTypeText)

		reserverName := passengerInfo.name
		if passengerInfo.customerType == "login" {
			reserverName = "회원정보 사용"
		}

		body = fmt.Sprintf(`SRT 예약이 성공적으로 완료되었습니다!

📍 예약 정보:
- 고객유형: %s
- 출발역: %s (%s)
- 도착역: %s (%s)
- 날짜: %s
- 예약자: %s

💡 10분 안에 결제를 완료해주세요!

%s`,
			customerTypeText,
			passengerInfo.deptStation, passengerInfo.deptTime,
			passengerInfo.arrivalStation, passengerInfo.arrivalTime,
			passengerInfo.date,
			reserverName,
			message)
	} else {
		subject = "⚠️ SRT 미등록고객 예약 실패 알림"
		body = fmt.Sprintf(`SRT 예약에 실패했습니다.

📍 시도한 예약 정보:
- 출발역: %s (%s)
- 도착역: %s (%s)
- 날짜: %s

❌ 오류: %s

다시 시도하거나 수동으로 예약해주세요.`,
			passengerInfo.deptStation, passengerInfo.deptTime,
			passengerInfo.arrivalStation, passengerInfo.arrivalTime,
			passengerInfo.date,
			message)
	}

	msg := []byte("To: " + passengerInfo.notificationEmail + "\r\n" +
		"Subject: " + subject + "\r\n" +
		"MIME-Version: 1.0\r\n" +
		"Content-Type: text/plain; charset=UTF-8\r\n" +
		"\r\n" +
		body + "\r\n")

	auth := smtp.PlainAuth("", emailConfig.senderEmail, emailConfig.senderPass, emailConfig.smtpHost)
	err := smtp.SendMail(emailConfig.smtpHost+":"+emailConfig.smtpPort, auth,
		emailConfig.senderEmail, []string{passengerInfo.notificationEmail}, msg)

	if err != nil {
		return fmt.Errorf("이메일 발송 실패: %w", err)
	}

	fmt.Println("   ✅ 예약 알림 이메일이 발송되었습니다")
	return nil
}

// ═══════════════════════════════════════════════════════════════════════════════
// ⚙️ 설정 로드 함수
// ═══════════════════════════════════════════════════════════════════════════════

func loadConfig() {
	if err := godotenv.Load(); err != nil {
		fmt.Println("⚠️ .env 파일을 찾을 수 없습니다. 기본값을 사용합니다.")
		return
	}

	if host := os.Getenv("SMTP_HOST"); host != "" {
		emailConfig.smtpHost = host
	}
	if port := os.Getenv("SMTP_PORT"); port != "" {
		emailConfig.smtpPort = port
	}
	if email := os.Getenv("SENDER_EMAIL"); email != "" {
		emailConfig.senderEmail = email
	}
	if pass := os.Getenv("SENDER_PASSWORD"); pass != "" {
		emailConfig.senderPass = pass
	}

	fmt.Println("✅ 환경변수에서 보안 데이터 설정을 로드했습니다")
}

// ═══════════════════════════════════════════════════════════════════════════════
// 🎯 메인 함수
// ═══════════════════════════════════════════════════════════════════════════════

func main() {
	loadConfig()

	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\n⚠️ 치명적 오류 발생!")
			fmt.Printf("오류 내용: %v\n", r)
		}
	}()

	collectUserInput()

	fmt.Println("▶ SRT 예약 자동화 시작...")
	fmt.Printf("최대 %d회까지 재시도합니다.\n", maxRetries)
	fmt.Println(strings.Repeat("=", 60))

	fmt.Println("▶ 브라우저 초기화")
	pw, err := playwright.Run()
	must("Playwright 실행 실패: %w", err)

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
	})
	must("브라우저 실행 실패: %w", err)

	context, err := browser.NewContext()
	must("브라우저 컨텍스트 생성 실패: %w", err)

	page, err := context.NewPage()
	must("페이지 생성 실패: %w", err)

	_, err = page.Goto(initialURL)
	must("페이지 이동 실패: %w", err)

	fmt.Println("   ✓ 브라우저 초기화 완료")
	showLoadingAnimation("시스템을 준비하는 중이에요", 1)

	var lastError error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := attemptReservation(page, attempt)
		if err == nil {
			fmt.Printf("\n✨ 성공! %d번째 시도에서 예약에 성공했습니다!\n", attempt)
			fmt.Println("ℹ️ 지금 결제를 진행하세요. 10분 후 브라우저가 자동으로 종료됩니다.")

			if err := sendNotificationEmail(true, ""); err != nil {
				fmt.Printf("이메일 발송 실패: %v\n", err)
			}

			// 성공 시 카운트다운 표시
			fmt.Println()
			for i := 600; i > 0; i-- {
				minutes := i / 60
				seconds := i % 60
				fmt.Printf("\r   ⏰ 자동 종료까지: %02d:%02d (결제를 완료해주세요)", minutes, seconds)
				time.Sleep(1 * time.Second)
			}
			fmt.Println()
			break
		}

		lastError = err
		fmt.Printf("✗ 시도 %d 실패: %v\n", attempt, err)

		if attempt < maxRetries {
			waitTime := 3
			fmt.Printf("⏸️ %d초 후 재시도합니다...\n", waitTime)
			showLoadingAnimation("다음 시도를 준비하는 중이에요", waitTime)
		}
	}

	if lastError != nil {
		fmt.Printf("\n⚠️ %d회 모든 시도가 실패했습니다!\n", maxRetries)
		fmt.Printf("마지막 오류: %v\n", lastError)
		fmt.Println("↻ 프로그램을 다시 실행해보거나 수동으로 예약을 시도하세요.")
		wait(5)

		if err := sendNotificationEmail(false, lastError.Error()); err != nil {
			fmt.Printf("이메일 발송 실패: %v\n", err)
		}
	}

	browser.Close()
	pw.Stop()
	fmt.Println("   ✓ 리소스 정리 완료")
}
