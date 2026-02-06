package srv

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"srv.exe.dev/db/dbgen"
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
	TxType            string `json:"tx_type"`
	AssetTypeName     string `json:"asset_type"`
	AssetTypeID       int64  `json:"asset_type_id"`
}

func (s *Server) HandleParseSMS(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Text string `json:"text"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}

	result := s.parseSMSWithDB(r.Context(), req.Text)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"success":     true,
		"transaction": result,
	})
}

// parseSMSWithDB parses SMS with DB context for card matching
func (s *Server) parseSMSWithDB(ctx context.Context, text string) ParsedSMS {
	result := ParsedSMS{TxType: "unknown"}

	if strings.TrimSpace(text) == "" {
		return result
	}

	queries := dbgen.New(s.DB)

	// 1. DB에서 매핑 룰 조회
	rules, _ := queries.GetSMSMappingRules(ctx)
	
	// 2. 거래 유형 판단 (우선순위 기반)
	type matchedRule struct {
		ruleType string
		priority int64
	}
	var matches []matchedRule

	for _, rule := range rules {
		if strings.Contains(text, rule.Keyword) {
			matches = append(matches, matchedRule{
				ruleType: rule.RuleType,
				priority: rule.Priority,
			})
		}
	}

	// 우선순위 정렬
	sort.Slice(matches, func(i, j int) bool {
		return matches[i].priority > matches[j].priority
	})

	// 유형 결정 및 자산 자동 매칭
	if len(matches) > 0 {
		switch matches[0].ruleType {
		case "income":
			result.TxType = "income"
			result.AssetTypeName = "입금"
			result.AssetTypeID = -1 // 프론트엔드 가상 ID: 입금
		case "expense":
			result.TxType = "expense"
			result.AssetTypeName = "이체"
			// DB에서 '이체' 자산 ID 조회
			var assetID int64
			err := s.DB.QueryRowContext(ctx, "SELECT id FROM asset_types WHERE name = '이체' AND type = 'expense' LIMIT 1").Scan(&assetID)
			if err == nil {
				result.AssetTypeID = assetID
			}
		case "card":
			result.TxType = "card"
		}
	}

	// 3. 카드 내역인 경우 카드명 매칭
	if result.TxType == "card" || result.TxType == "unknown" {
		result = s.matchCardFromDB(ctx, text, result)
	}

	// 4. 금액 추출
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

	// 5. 날짜 추출
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

	// 6. 시간 추출
	timeRe := regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	if matches := timeRe.FindStringSubmatch(text); len(matches) > 2 {
		result.Time = matches[1] + ":" + matches[2]
	}

	// 7. 할부 체크
	if strings.Contains(text, "일시불") {
		result.IsInstallment = false
	} else {
		installmentRe := regexp.MustCompile(`(\d+)개월`)
		if matches := installmentRe.FindStringSubmatch(text); len(matches) > 1 {
			result.IsInstallment = true
			result.InstallmentMonths, _ = strconv.ParseInt(matches[1], 10, 64)
		}
	}

	// 8. 가맹점명 추출
	result.Description = extractDescription(text, dateRe)

	// 9. 취소 체크
	if strings.Contains(text, "취소") {
		result.IsCancelled = true
	}

	// 10. 기본값 설정
	if result.TxType == "unknown" {
		result.TxType = "expense"
		result.AssetTypeName = "기타"
	}
	if result.Description == "" {
		result.Description = "기타거래"
	}

	return result
}

// matchCardFromDB matches card name from SMS text against DB cards
func (s *Server) matchCardFromDB(ctx context.Context, text string, result ParsedSMS) ParsedSMS {
	queries := dbgen.New(s.DB)
	cards, err := queries.GetAllCardsWithKeywords(ctx)
	if err != nil {
		return result
	}

	// 카드사 키워드 패턴
	cardKeywords := []struct {
		pattern string
		name    string
	}{
		{"IBK기업BC", "IBK기업BC"},
		{"IBK기업", "IBK기업"},
		{"기업BC", "기업BC"},
		{"KB국민", "KB국민카드"},
		{"국민카드", "KB국민카드"},
		{"우리BC", "우리카드"},
		{"우리카드", "우리카드"},
		{"우리(", "우리카드"},
		{"삼성카드", "삼성카드"},
		{"삼성(", "삼성카드"},
		{"신한카드", "신한카드"},
		{"신한(", "신한카드"},
		{"현대카드", "현대카드"},
		{"현대(", "현대카드"},
		{"롯데카드", "롯데카드"},
		{"롯데(", "롯데카드"},
		{"하나카드", "하나카드"},
		{"하나(", "하나카드"},
		{"NH농협", "NH농협카드"},
		{"농협카드", "NH농협카드"},
		{"BC카드", "BC카드"},
		{"BC(", "BC카드"},
	}

	// SMS에서 카드사 키워드 찾기
	var detectedCardName string
	for _, kw := range cardKeywords {
		if strings.Contains(text, kw.pattern) {
			detectedCardName = kw.name
			slog.Info("matched card keyword", "pattern", kw.pattern, "name", kw.name)
			break
		}
	}

	if detectedCardName == "" {
		slog.Info("no card keyword matched")
		return result
	}

	// DB 카드와 매칭 (앞글자 우선순위)
	type cardMatch struct {
		card  dbgen.GetAllCardsWithKeywordsRow
		score int
	}
	var cardMatches []cardMatch

	for _, card := range cards {
		score := 0
		cardNameLower := strings.ToLower(card.Name)
		detectedLower := strings.ToLower(detectedCardName)

		// 완전 일치
		if cardNameLower == detectedLower {
			score = 100
		} else if strings.Contains(cardNameLower, detectedLower) || strings.Contains(detectedLower, cardNameLower) {
			// 부분 일치
			score = 50
		} else {
			// 앞글자 매칭
			for i := 0; i < len(cardNameLower) && i < len(detectedLower); i++ {
				if cardNameLower[i] == detectedLower[i] {
					score++
				} else {
					break
				}
			}
		}

		if score > 0 {
			cardMatches = append(cardMatches, cardMatch{card: card, score: score})
		}
	}

	// 점수순 정렬
	sort.Slice(cardMatches, func(i, j int) bool {
		return cardMatches[i].score > cardMatches[j].score
	})

	if len(cardMatches) > 0 {
		best := cardMatches[0]
		result.CardName = best.card.Name
		result.TxType = "card"
		result.AssetTypeName = best.card.Name
		if best.card.AssetTypeID != nil {
			result.AssetTypeID = *best.card.AssetTypeID
		}
	} else {
		result.CardName = detectedCardName
		result.TxType = "card"
		result.AssetTypeName = detectedCardName
	}

	return result
}

// extractDescription extracts merchant name from SMS
func extractDescription(text string, dateRe *regexp.Regexp) string {
	// 한 줄 SMS에서 가맹점명 추출 시도
	// 예: "[KB국민카드] 쿠팡(쿠페이) 30,100원 02/06 10:43 일시불"
	// 예: "KB국민 쿠팡(쿠페이) 30,100원 02/06 10:43"
	if !strings.Contains(text, "\n") {
		// 카드명 다음, 금액 전의 문자열을 추출
		// 먼저 금액 위치 찾기
		amountRe := regexp.MustCompile(`([\d,]+)원`)
		amountMatch := amountRe.FindStringIndex(text)
		if amountMatch != nil {
			// 금액 앞부분에서 가맹점명 추출
			preText := text[:amountMatch[0]]
			// 대괄호 내용 제거 ([카드명])
			bracketRe := regexp.MustCompile(`\[[^\]]*\]`)
			preText = bracketRe.ReplaceAllString(preText, "")
			// 카드사 키워드 제거
			cardKeywords := []string{"KB국민", "삼성", "우리", "하나", "BC", "롯데", "신한", "IBK기업", "NH", "농협", "현대", "씨티"}
			for _, kw := range cardKeywords {
				preText = strings.ReplaceAll(preText, kw, "")
			}
			preText = strings.TrimSpace(preText)
			if len(preText) > 0 && len(preText) < 50 {
				return preText
			}
		}
	}

	// 여러 줄 SMS 처리
	lines := strings.Split(text, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" ||
			strings.Contains(line, "원") ||
			strings.Contains(line, "승인") ||
			strings.Contains(line, "출금") ||
			strings.Contains(line, "입금") ||
			strings.Contains(line, "Web발신") ||
			strings.Contains(line, "누적") ||
			strings.Contains(line, "잔액") ||
			strings.HasPrefix(line, "[") ||
			strings.HasPrefix(line, "*") ||
			strings.Contains(line, "님") {
			continue
		}
		if dateRe.MatchString(line) || strings.Contains(line, ":") {
			continue
		}
		if len(line) > 0 && len(line) < 50 {
			return line
		}
	}
	return ""
}

// Legacy function for backward compatibility
func parseSMS(text string) ParsedSMS {
	result := ParsedSMS{TxType: "unknown"}
	if strings.TrimSpace(text) == "" {
		return result
	}

	// 간단한 키워드 기반 판단
	if strings.Contains(text, "입금") {
		result.TxType = "income"
		result.AssetTypeName = "입금"
	} else if strings.Contains(text, "출금") {
		result.TxType = "expense"
		result.AssetTypeName = "이체"
	} else if strings.Contains(text, "승인") || strings.Contains(text, "누적") {
		result.TxType = "card"
	}

	// 금액
	amountRe := regexp.MustCompile(`([\d,]+)원`)
	if matches := amountRe.FindAllStringSubmatch(text, -1); len(matches) > 0 {
		for _, match := range matches {
			if len(match) > 1 {
				amountStr := strings.ReplaceAll(match[1], ",", "")
				amount, _ := strconv.ParseInt(amountStr, 10, 64)
				if amount > 0 {
					result.Amount = amount
					break
				}
			}
		}
	}

	// 날짜
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

	// 시간
	timeRe := regexp.MustCompile(`(\d{1,2}):(\d{2})`)
	if matches := timeRe.FindStringSubmatch(text); len(matches) > 2 {
		result.Time = matches[1] + ":" + matches[2]
	}

	// 가맹점명
	result.Description = extractDescription(text, dateRe)
	if result.Description == "" {
		result.Description = "기타거래"
	}

	// 취소
	if strings.Contains(text, "취소") {
		result.IsCancelled = true
	}

	return result
}
