//go:build ignore
// +build ignore

// ===================================================================
// 패키지 선언 및 임포트 구역
// ===================================================================
package main

import (
	"fmt"     // 포맷된 문자열 출력
	"log"     // 로깅 기능
	"reflect" // 타입 비교용 리플렉션
	"strings" // 문자열 조작용
	"time"    // 시간 조작용

	"github.com/playwright-community/playwright-go" // Playwright Go 바인딩
)

// ===================================================================
// 헬퍼 함수 정의 구역
// ===================================================================

// must: 에러 체크 헬퍼 함수
func must(message string, err error) {
	if err != nil {
		log.Fatalf(message, err)
	}
}

// wait: 대기 함수 (초 단위)
func wait(seconds int) {
	time.Sleep(time.Duration(seconds) * time.Second)
}

// eq: 값 비교 헬퍼 함수
// 값 불일치 시 패닉 발생
// 테스트 어설션용
func eq(expected, actual interface{}) {
	if !reflect.DeepEqual(expected, actual) {
		panic(fmt.Sprintf("%v does not equal %v", actual, expected))
	}
}

// safeAction: 안전한 액션 실행 헬퍼
func safeAction(action func() error, errorMsg string) error {
	if err := action(); err != nil {
		return fmt.Errorf("%s: %w", errorMsg, err)
	}
	return nil
}

// fillInput: 입력 필드 채우기 헬퍼 (클릭 → 입력 → 탭)
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

// selectOption: 셀렉트 옵션 선택 헬퍼
func selectOption(page playwright.Page, selector, value, fieldName string) error {
	_, err := page.Locator(selector).SelectOption(playwright.SelectOptionValues{
		Values: playwright.StringSlice(value),
	})
	return safeAction(func() error { return err }, fieldName+" 선택 실패")
}

// clickButton: 버튼 클릭 헬퍼
func clickButton(page playwright.Page, selector, buttonName string) error {
	button := page.Locator(selector)
	if err := safeAction(func() error { return button.Click() }, buttonName+" 클릭 실패"); err != nil {
		return err
	}
	return nil
}

// checkElementExists: 요소 존재 확인 헬퍼
func checkElementExists(page playwright.Page, selector, elementName string) error {
	count, err := page.Locator(selector).Count()
	if err != nil {
		return fmt.Errorf("%s 확인 실패: %w", elementName, err)
	}
	if count == 0 {
		return fmt.Errorf("%s가 존재하지 않습니다", elementName)
	}
	return nil
}

// setupDialogHandler: 대화상자 처리 헬퍼
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

// ===================================================================
// 상수 정의 구역
// ===================================================================
const (
	initialURL = "https://etk.srail.kr/hpg/hra/01/selectScheduleList.do?pageId=TK0101010000"
	maxRetries = 10 // 최대 재시도 횟수
)

var passengerInfo = struct {
	deptStation     string
	arrivalStation  string
	deptTime        string
	arrivalTime     string
	date            string
	name            string
	phone           string
	password        string
	passwordConfirm string
}{
	deptStation:     "동탄",
	arrivalStation:  "전주",
	deptTime:        "19:26",
	arrivalTime:     "20:51",
	date:            "20250622",
	name:            "홍길동",
	phone:           "01012345678",
	password:        "123456",
	passwordConfirm: "123456",
}

// 필드 선택자 상수
const (
	dptStationSelector                = "input#dptRsStnCdNm"
	arvStationSelector                = "input#arvRsStnCdNm"
	dateSelector                      = "select#dptDt"
	searchButtonSelector              = "input[value='조회하기']"
	unregisteredReserveButtonSelector = "a.btn_midium.btn_pastel1:has-text('미등록고객 예매')"

	// 예약자 정보 입력 폼 선택자
	passengerAgreeSelector = "input#agreeY"
	passengerNameSelector  = "input#custNm"
)

// ===================================================================
// 단계별 처리 함수들
// ===================================================================

// step1SetStations: 1단계 - 출발역/도착역 설정
func step1SetStations(page playwright.Page) error {
	fmt.Println("▶ 1단계: 출발역/도착역 설정")

	fmt.Println("   > 출발역: ", passengerInfo.deptStation)
	if err := fillInput(page, dptStationSelector, passengerInfo.deptStation, "출발역"); err != nil {
		return err
	}

	fmt.Println("   > 도착역: ", passengerInfo.arrivalStation)
	if err := fillInput(page, arvStationSelector, passengerInfo.arrivalStation, "도착역"); err != nil {
		return err
	}

	fmt.Println("   ✓ 출발역/도착역 설정 완료")
	return nil
}

// step2SetDate: 2단계 - 출발 날짜 설정
func step2SetDate(page playwright.Page) error {
	fmt.Println("▶ 2단계: 출발 날짜 설정")
	if err := selectOption(page, dateSelector, passengerInfo.date, "날짜"); err != nil {
		return err
	}
	fmt.Println("   ✓ 출발 날짜 설정 완료")
	return nil
}

// step3SearchTrains: 3단계 - 열차 조회
func step3SearchTrains(page playwright.Page) error {
	fmt.Println("▶ 3단계: 열차 조회")
	if err := clickButton(page, searchButtonSelector, "조회 버튼"); err != nil {
		return err
	}
	wait(3) // 조회 결과 로딩 대기
	fmt.Println("   ✓ 조회 완료")
	return nil
}

// step4CheckAvailability: 4단계 - 예약 가능한 열차 확인
func step4CheckAvailability(page playwright.Page) error {
	fmt.Println("▶ 4단계: 예약 가능 열차 확인")
	// 만약 div#NetFunnel_Skin_Top 가 나온다면 진입 대기중이므로 없어질 때까지 기다림.
	netfunnelLocator := page.Locator("div#NetFunnel_Skin_Top")
	if count, _ := netfunnelLocator.Count(); count > 0 {
		fmt.Println("   ⏳ 진입 대기 중...")
		netfunnelLocator.WaitFor(playwright.LocatorWaitForOptions{
			State:   playwright.WaitForSelectorStateHidden,
			Timeout: playwright.Float(1000 * 60), // 최대 1분 대기
		})
		fmt.Println("   ✓ 진입 중...")
	}

	wait(1)

	// 모든 tr을 확인합니다. 각 tr에서 4번째 td가 10:37을 텍스트로 가지고 5번째 td가 12:07을 텍스트로 가진다면 예약 가능한 열차로 확인.
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

		// TextContent() 메서드의 에러 처리
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

// step5ClickReserve: 5단계 - 예약 시도
func step5ClickReserve(page playwright.Page) error {
	fmt.Println("▶ 5단계: 예약 시도")
	// 19:26 -> 20:51 열차의 예약하기 버튼 클릭
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

		// TextContent() 메서드의 에러 처리
		dept, err := tds[3].Locator("em").TextContent()
		if err != nil {
			continue
		}
		arrival, err := tds[4].Locator("em").TextContent()
		if err != nil {
			continue
		}

		if strings.Contains(dept, passengerInfo.deptTime) && strings.Contains(arrival, passengerInfo.arrivalTime) {
			// 매진 텍스트를 가진 요소가 있으면 예약 불가능하므로 에러 반환
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

// step6GoToUnregistered: 6-1단계 - 미등록고객 예매 페이지로 이동
func step6GoToUnregistered(page playwright.Page) error {
	wait(1)
	fmt.Println("▶ 6-1단계: 미등록고객 예매 페이지로 이동")

	// confirm 대화상자 핸들러 설정 (자동으로 "확인" 클릭)
	setupDialogHandler(page, true)

	// 미등록고객 예매라는 텍스트를 가지며 btn_midium btn_pastel1라는 클래스를 가진 a 태그 클릭
	if err := clickButton(page, unregisteredReserveButtonSelector, "미등록고객 예매 버튼"); err != nil {
		return err
	}

	fmt.Println("   ✓ 미등록고객 예매 버튼 클릭 및 대화상자 처리 완료")
	return nil
}

// step7VerifyReservationPage: 7단계 - 예약자 정보 입력 화면으로 이동
func step7VerifyReservationPage(page playwright.Page) error {
	currentURL := page.URL()
	if !strings.Contains(currentURL, "selectReservationForm") {
		return fmt.Errorf("예약 페이지로 이동하지 못했습니다 (현재 URL: %s)", currentURL)
	}

	fmt.Println("   ✓ 예약자 정보 입력 화면으로 이동 완료")
	return nil
}

// step8FillPassengerInfo: 8단계 - 예약자 정보폼에 정보 입력
func step8FillPassengerInfo(page playwright.Page) error {
	fmt.Println("▶ 8단계: 예약자 정보폼에 정보 입력")

	// 동의 체크박스 클릭
	if err := clickButton(page, passengerAgreeSelector, "개인정보수집 동의 체크박스"); err != nil {
		return err
	}

	// 이름 입력
	if err := fillInput(page, passengerNameSelector, passengerInfo.name, "예약자 이름"); err != nil {
		return err
	}

	// Tab으로 이동하며 순차적으로 입력
	inputValues := []struct {
		value string
		desc  string
	}{
		{passengerInfo.phone[:3], "전화번호 앞자리"},
		{passengerInfo.phone[3:7], "전화번호 중간자리"},
		{passengerInfo.phone[7:], "전화번호 뒷자리"},
		{passengerInfo.password, "비밀번호"},
		{passengerInfo.passwordConfirm, "비밀번호 확인"},
	}

	for _, input := range inputValues {
		// 현재 포커스된 요소에 입력
		if err := page.Keyboard().Type(input.value); err != nil {
			return fmt.Errorf("%s 입력 실패: %w", input.desc, err)
		}
		fmt.Printf("   ✓ %s 입력 완료\n", input.desc)

		// Tab으로 다음 필드로 이동
		if err := page.Keyboard().Press("Tab"); err != nil {
			return fmt.Errorf("%s 입력 후 Tab 이동 실패: %w", input.desc, err)
		}
	}

	fmt.Println("   ✓ 예약자 정보폼에 정보 입력 완료")
	return nil
}

// step9SubmitForm: 9단계 - 예약자 정보폼 제출 확인
func step9SubmitForm(page playwright.Page) error {
	fmt.Println("▶ 9단계: 예약자 정보폼 제출 확인")

	// 예약자 정보폼 제출 버튼 클릭
	if err := page.Keyboard().Press("Enter"); err != nil {
		return fmt.Errorf("확인 버튼 클릭 실패: %w", err)
	}
	setupDialogHandler(page, true)

	fmt.Println("   ✓ 예약자 정보폼 제출 완료")
	return nil
}

// ===================================================================
// 예약 시도 함수
// ===================================================================
func attemptReservation(page playwright.Page, attempt int) error {
	fmt.Printf("\n↻ 시도 %d/%d 시작...\n", attempt, maxRetries)
	fmt.Println("=" + strings.Repeat("=", 50))

	// 페이지 새로고침으로 초기화 (2번째 시도부터)
	if attempt > 1 {
		fmt.Println("⟳ 페이지 새로고침...")
		if _, err := page.Reload(); err != nil {
			return fmt.Errorf("페이지 새로고침 실패: %w", err)
		}
		wait(3)
	}

	// 단계별 실행
	if err := step1SetStations(page); err != nil {
		return err
	}
	if err := step2SetDate(page); err != nil {
		return err
	}
	if err := step3SearchTrains(page); err != nil {
		return err
	}
	if err := step4CheckAvailability(page); err != nil {
		return err
	}
	wait(3)
	if err := step5ClickReserve(page); err != nil {
		return err
	}
	if err := step6GoToUnregistered(page); err != nil {
		return err
	}
	if err := step7VerifyReservationPage(page); err != nil {
		return err
	}
	if err := step8FillPassengerInfo(page); err != nil {
		return err
	}
	// if err := step9SubmitForm(page); err != nil {
	// 	return err
	// }

	fmt.Println("   ✨ 예약 페이지로 이동 성공!")
	return nil
}

// ===================================================================
// 메인 함수 - SRT 예약 자동화 (재시도 로직 포함)
// ===================================================================
func main() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Println("\n⚠️ 치명적 오류 발생!")
			fmt.Printf("오류 내용: %v\n", r)
		}
	}()

	fmt.Println("▶ SRT 예약 자동화 시작...")
	fmt.Printf("최대 %d회까지 재시도합니다.\n", maxRetries)
	fmt.Println("=" + strings.Repeat("=", 60))

	// 브라우저 초기화
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
	wait(1)

	// 재시도 로직
	var lastError error
	for attempt := 1; attempt <= maxRetries; attempt++ {
		err := attemptReservation(page, attempt)
		if err == nil {
			fmt.Printf("\n✨ 성공! %d번째 시도에서 예약에 성공했습니다!\n", attempt)
			fmt.Println("ℹ️ 10분 안에 결제를 진행하세요. 이후 브라우저가 자동으로 종료됩니다.")

			// 성공 시 10분 대기 후 종료
			wait(600)
			break
		}

		lastError = err
		fmt.Printf("✗ 시도 %d 실패: %v\n", attempt, err)

		if attempt < maxRetries {
			waitTime := 3
			fmt.Printf("⏸️ %d초 후 재시도합니다...\n", waitTime)
			wait(waitTime)
		}
	}

	// 모든 시도 실패 시
	if lastError != nil {
		fmt.Printf("\n⚠️ %d회 모든 시도가 실패했습니다!\n", maxRetries)
		fmt.Printf("마지막 오류: %v\n", lastError)
		fmt.Println("↻ 프로그램을 다시 실행해보거나 수동으로 예약을 시도하세요.")
		wait(5)
	}

	// 정리 작업
	browser.Close()
	pw.Stop()
	fmt.Println("   ✓ 리소스 정리 완료")
}
