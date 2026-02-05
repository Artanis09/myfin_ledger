package srv

import (
	"encoding/json"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type ParsedSMS struct {
	CardName          string `json:"card_name"`
	Amount            int64  `json:"amount"`
	Description       string `json:"description"`
	Date              string `json:"date"`
	Time              string `json:"time"`
	IsInstallment     bool   `json:"is_installment"`
	InstallmentMonths int64  `json:"installment_months"`
	IsCancelled       bool   `json:"is_cancelled"`
	TxType            string `json:"tx_type"`    // "card" or "transfer" or "unknown"
	AssetTypeName     string `json:"asset_type"` // 자산 유형명
}

func (s *Server) HandleParseSMS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	result := parseSMS(req.Text)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func parseSMS(text string) ParsedSMS {
	result := ParsedSMS{
		TxType: "unknown",
	}

	// 빈 텍스트 처리
	if strings.TrimSpace(text) == "" {
		return result
	}

	// 1. 거래 유형 판단: 출금 vs 승인
	isWithdrawal := strings.Contains(text, "출금")
	isApproval := strings.Contains(text, "승인")

	if isWithdrawal {
		result.TxType = "transfer"
		result.AssetTypeName = "이체"
	} else if isApproval {
		result.TxType = "card"
	}

	// 2. 카드사/은행 감지
	bankPatterns := map[string][]string{
		// 은행 (출금용)
		"우리":   {"우리 ", "우리은행"},
		"국민":   {"KB국민", "국민은행"},
		"신한":   {"신한 ", "신한은행"},
		"하나":   {"하나 ", "하나은행"},
		"농협":   {"NH ", "농협"},
		"기업":   {"IBK", "기업은행"},
		"SC제일": {"SC ", "SC제일"},
		"카카오": {"카카오"},
		"토스":   {"토스"},
	}

	cardPatterns := map[string][]string{
		"KB국민카드": {"KB국민카드", "KB국민"},
		"우리카드":   {"우리카드"},
		"삼성카드":   {"삼성(", "삼성카드"},
		"신한카드":   {"신한(", "신한카드"},
		"현대카드":   {"현대(", "현대카드"},
		"롯데카드":   {"롯데(", "롯데카드", "롯데9*", "롯데9"},
		"하나카드":   {"하나(", "하나카드"},
		"NH농협카드": {"NH(", "NH농협"},
		"BC카드":    {"BC(", "BC카드"},
		"IBK기업BC": {"IBK기업BC", "기업BC"},
	}

	// 출금인 경우 은행명 감지
	if isWithdrawal {
		for bankName, patterns := range bankPatterns {
			for _, p := range patterns {
				if strings.Contains(text, p) {
					result.CardName = bankName + "은행"
					result.AssetTypeName = "이체"
					break
				}
			}
			if result.CardName != "" {
				break
			}
		}
	}

	// 승인인 경우 카드사 감지
	if isApproval || result.CardName == "" {
		for cardName, patterns := range cardPatterns {
			for _, p := range patterns {
				if strings.Contains(text, p) {
					result.CardName = cardName
					if result.TxType == "unknown" {
						result.TxType = "card"
					}
					break
				}
			}
			if result.CardName != "" {
				break
			}
		}
	}

	// 3. 금액 추출 (XX,XXX원 or X,XXX,XXX원)
	amountRe := regexp.MustCompile(`([\d,]+)원`)
	if matches := amountRe.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		// 첫 번째로 찾은 금액 사용 (출금/승인 금액)
		for _, match := range matches {
			if len(match) > 1 {
				amountStr := strings.ReplaceAll(match[1], ",", "")
				amount, err := strconv.ParseInt(amountStr, 10, 64)
				if err == nil && amount > 0 {
					result.Amount = amount
					break
				}
			}
		}
	}

	// 4. 날짜 추출 (MM/DD)
	dateRe := regexp.MustCompile(`(\d{1,2})/(\d{1,2})`)
	if matches := dateRe.FindStringSubmatch(text); len(matches) > 2 {
		month, _ := strconv.Atoi(matches[1])
		day, _ := strconv.Atoi(matches[2])
		year := time.Now().Year()
		// 현재 월보다 크면 작년으로 판단
		if month > int(time.Now().Month()) {
			year--
		}
		result.Date = time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.Local).Format("2006-01-02")
	}

	// 5. 시간 추출 (HH:MM)
	timeRe := regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	if matches := timeRe.FindStringSubmatch(text); len(matches) > 2 {
		result.Time = matches[1] + ":" + matches[2]
	}

	// 6. 할부 체크
	if strings.Contains(text, "일시불") {
		result.IsInstallment = false
	} else {
		installmentRe := regexp.MustCompile(`(\d+)개월`)
		if matches := installmentRe.FindStringSubmatch(text); len(matches) > 1 {
			result.IsInstallment = true
			result.InstallmentMonths, _ = strconv.ParseInt(matches[1], 10, 64)
		}
	}

	// 7. 가맹점명/설명 추출
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// 스킵할 패턴들
		if line == "" ||
			strings.Contains(line, "원") ||
			strings.Contains(line, "승인") ||
			strings.Contains(line, "출금") ||
			strings.Contains(line, "Web발신") ||
			strings.Contains(line, "누적") ||
			strings.Contains(line, "잔액") ||
			strings.Contains(line, "총누적") ||
			strings.HasPrefix(line, "[") ||
			strings.HasPrefix(line, "*") ||
			strings.Contains(line, "님") {
			continue
		}
		// 날짜/시간 패턴 스킵
		if dateRe.MatchString(line) || strings.Contains(line, ":") {
			continue
		}
		// 가맹점명으로 사용
		if len(line) > 0 && len(line) < 50 {
			result.Description = line
		}
	}

	// 출금의 경우 설명이 없으면 "계좌이체"로 설정
	if result.TxType == "transfer" && result.Description == "" {
		// 이체 대상 계좌 정보 추출 시도 (현대02-028 같은 형식)
		accountRe := regexp.MustCompile(`([가-힣A-Za-z]+\d+-\d+|[가-힣]+\d+)`)
		if matches := accountRe.FindStringSubmatch(text); len(matches) > 0 {
			result.Description = matches[0] + " 이체"
		} else {
			result.Description = "계좌이체"
		}
	}

	// 8. 취소 체크
	if strings.Contains(text, "취소") {
		result.IsCancelled = true
	}

	return result
}
