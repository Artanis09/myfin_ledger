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

	// 1. 거래 유형 판단: 입금(income) vs 출금(transfer) vs 승인(card)
	isDeposit := strings.Contains(text, "입금")
	isWithdrawal := strings.Contains(text, "출금")
	isApproval := strings.Contains(text, "승인")

	if isDeposit {
		result.TxType = "income"
		result.AssetTypeName = "입금"
	} else if isWithdrawal {
		result.TxType = "transfer"
		result.AssetTypeName = "이체"
	} else if isApproval {
		result.TxType = "card"
	}

	// 2. 은행 패턴 (더 넓은 매칭)
	type pattern struct {
		name     string
		patterns []string
	}

	bankPatterns := []pattern{
		{"우리은행", []string{"우리 ", "우리은행", "우리\n"}},
		{"국민은행", []string{"KB국민", "국민은행", "국민 "}},
		{"신한은행", []string{"신한 ", "신한은행"}},
		{"하나은행", []string{"하나 ", "하나은행"}},
		{"농협", []string{"NH ", "농협"}},
		{"기업은행", []string{"IBK ", "기업은행"}},
		{"SC제일은행", []string{"SC ", "SC제일"}},
		{"카카오뱅크", []string{"카카오"}},
		{"토스뱅크", []string{"토스"}},
	}

	// 카드사 패턴 (순서 중요 - 더 구체적인 것 먼저)
	cardPatterns := []pattern{
		{"IBK기업BC", []string{"IBK기업BC", "기업BC", "IBK기업"}},
		{"KB국민카드", []string{"KB국민카드"}},
		{"우리카드", []string{"우리카드"}},
		{"삼성카드", []string{"삼성(", "삼성카드"}},
		{"신한카드", []string{"신한(", "신한카드"}},
		{"현대카드", []string{"현대(", "현대카드"}},
		{"롯데카드", []string{"롯데(", "롯데카드", "롯데9*", "롯데9"}},
		{"하나카드", []string{"하나(", "하나카드"}},
		{"NH농협카드", []string{"NH농협카드", "NH(농협"}},
		{"BC카드", []string{"BC(", "BC카드"}},
	}

	// 승인인 경우 카드사 먼저 감지
	if isApproval {
		for _, card := range cardPatterns {
			found := false
			for _, p := range card.patterns {
				if strings.Contains(text, p) {
					result.CardName = card.name
					result.TxType = "card"
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	// 출금이거나 카드사를 못 찾은 경우 은행 감지
	if isWithdrawal || result.CardName == "" {
		for _, bank := range bankPatterns {
			found := false
			for _, p := range bank.patterns {
				if strings.Contains(text, p) {
					result.CardName = bank.name
					// 출금 키워드가 없어도 은행 메시지면 이체로 간주
					if result.TxType == "unknown" {
						result.TxType = "transfer"
						result.AssetTypeName = "이체"
					}
					found = true
					break
				}
			}
			if found {
				break
			}
		}
	}

	// 3. 금액 추출 (XX,XXX원 or X,XXX,XXX원)
	amountRe := regexp.MustCompile(`([\d,]+)원`)
	if matches := amountRe.FindAllStringSubmatch(text, -1); len(matches) > 0 {
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
		// 은행/카드사 이름 스킵
		skipLine := false
		for _, bank := range bankPatterns {
			for _, p := range bank.patterns {
				if strings.Contains(line, strings.TrimSpace(p)) && len(line) < 10 {
					skipLine = true
					break
				}
			}
		}
		if skipLine {
			continue
		}
		// 가맹점명으로 사용
		if len(line) > 0 && len(line) < 50 {
			result.Description = line
		}
	}

	// 이체의 경우 설명이 없으면 기본값 설정
	if result.TxType == "transfer" && result.Description == "" {
		accountRe := regexp.MustCompile(`([가-힣A-Za-z]+\d+-\d+|[가-힣]+\d+)`)
		if matches := accountRe.FindStringSubmatch(text); len(matches) > 0 {
			result.Description = matches[0] + " 이체"
		} else {
			result.Description = "계좌이체"
		}
	}

	// unknown 타입이고 설명이 없으면 기본값 설정
	if result.TxType == "unknown" && result.Description == "" {
		result.Description = "기타거래"
	}

	// 8. 취소 체크
	if strings.Contains(text, "취소") {
		result.IsCancelled = true
	}

	return result
}
