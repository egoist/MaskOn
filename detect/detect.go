// Package detect finds what's private in a line of text: phone numbers,
// emails, card numbers and the like. It runs on the device, on what OCR
// read, so it has to put up with OCR's habits: spaces in odd places, and
// no certainty about the language.
package detect

import (
	"regexp"
	"slices"
	"strings"
	"unicode"
)

// Kind is what a span of text is.
type Kind uint8

const (
	Phone Kind = iota
	Email
	Link
	Card
	ID
	IP
	Money
	Plate
	Number
	Name
	Address
	Handle
)

// Kinds is every kind, in the order a person would list them.
var Kinds = []Kind{Phone, Email, Card, ID, Name, Address, Money, Plate, IP, Link, Handle, Number}

func (k Kind) String() string {
	return [...]string{"phone", "email", "link", "card", "id", "ip", "money", "plate", "number", "name", "address", "handle"}[k]
}

// Span is a run of a line's runes, from Start up to End.
type Span struct {
	Kind       Kind
	Start, End int
}

type rule struct {
	kind Kind
	re   *regexp.Regexp
	// group is the capture group that's the private part (0: the match).
	group int
	// ok checks a match the pattern alone can't.
	ok func(string) bool
}

// The rules, most certain first: of matches that overlap, the first found
// wins, so a card number isn't also a phone number.
var rules = []rule{
	{kind: Email, re: regexp.MustCompile(`[\p{L}\p{N}._%+-]+@[\p{L}\p{N}-]+(?:\.[\p{L}\p{N}-]+)*\.[A-Za-z]{2,}`)},
	{kind: Link, re: regexp.MustCompile(`(?i)\b(?:https?://|www\.)[^\s，。、；）)】]+`)},
	// A number with a country code is a phone number, whatever its digits
	// would add up to as a card's.
	{kind: Phone, re: regexp.MustCompile(`\+\d{1,3}[ -]?\(?\d{1,4}\)?(?:[ -]?\d{2,4}){2,4}`)},
	// Word boundaries don't work beside Chinese (it's all \w to Go's \b
	// only for ASCII), so the digit rules check their neighbors themselves.
	{kind: Card, re: regexp.MustCompile(`\d(?:[ -]?\d){12,18}`), ok: luhn},
	{kind: ID, re: regexp.MustCompile(`\d{6}(?:19|20)\d{2}(?:0[1-9]|1[0-2])(?:0[1-9]|[12]\d|3[01])\d{3}[\dXx]`), ok: chineseID},
	{kind: ID, re: regexp.MustCompile(`\d{3}-\d{2}-\d{4}`)}, // an SSN
	{kind: IP, re: regexp.MustCompile(`(?:(?:25[0-5]|2[0-4]\d|1?\d?\d)\.){3}(?:25[0-5]|2[0-4]\d|1?\d?\d)`)},
	{kind: IP, re: regexp.MustCompile(`(?i)\b(?:[0-9a-f]{1,4}:){7}[0-9a-f]{1,4}\b`)},
	// Chinese mobiles, then digits in the groups people write phones in.
	{kind: Phone, re: regexp.MustCompile(`1[3-9]\d(?:[ -]?\d){8}`)},
	{kind: Phone, re: regexp.MustCompile(`\(?\d{3}\)?[ -]\d{3}[ -]\d{4}`)},
	{kind: Phone, re: regexp.MustCompile(`0\d{2,3}-\d{7,8}`)},
	{kind: Money, re: regexp.MustCompile(`(?:[$¥￥€£]|USD|CNY|RMB|EUR|GBP)\s?\d[\d,]*(?:\.\d+)?|\d[\d,]*(?:\.\d+)?\s?(?:元|美元|万元|USD|CNY|RMB)`)},
	{kind: Plate, re: regexp.MustCompile(`[京津沪滬渝冀豫云雲辽遼黑湘皖鲁魯新苏蘇浙赣贛鄂桂甘晋晉蒙陕陝吉闽閩贵貴粤粵青藏川宁寧琼瓊][A-Z]·?[A-Z0-9]{5,6}`)},
	{kind: Handle, re: regexp.MustCompile(`@[A-Za-z0-9_.]{2,}`)},
	// A label says what follows, when nothing about the value could.
	{kind: Name, group: 1, re: regexp.MustCompile(`(?i)(?:姓名|收件人|收货人|联系人|持卡人|真实姓名|full name|name|recipient|cardholder)\s*[:：]\s*(.+)`)},
	{kind: Address, group: 1, re: regexp.MustCompile(`(?i)(?:地址|收货地址|住址|配送地址|address|addr|ship to|deliver to)\s*[:：]\s*(.+)`)},
	// What's left of a long number is likely an order, an account or a code.
	{kind: Number, re: regexp.MustCompile(`\d(?:[ -]?\d){7,}`), ok: notDate},
}

// halfWidth turns the full-width marks OCR's Chinese model likes to read,
// whatever was written, into the ones the rules know: rune for rune, so the
// spans' places hold in the original.
var halfWidth = strings.NewReplacer(
	"：", ":", "（", "(", "）", ")", "，", ",", "．", ".", "＋", "+", "－", "-", "—", "-", "–", "-", "＠", "@", "＄", "$",
	"０", "0", "１", "1", "２", "2", "３", "3", "４", "4", "５", "5", "６", "6", "７", "7", "８", "8", "９", "9",
)

// Line finds the private spans of a line, in the order they come.
func Line(text string) []Span {
	text = halfWidth.Replace(text)
	var found []Span
	runeAt := runeOffsets(text)
	for _, r := range rules {
		for _, m := range r.re.FindAllStringSubmatchIndex(text, -1) {
			a, b := m[2*r.group], m[2*r.group+1]
			if a < 0 {
				continue
			}
			s := text[a:b]
			if r.ok != nil && !r.ok(s) {
				continue
			}
			if digitsEdge(s) && !standsAlone(text, a, b) {
				continue
			}
			span := Span{r.kind, runeAt(a), runeAt(b)}
			if !slices.ContainsFunc(found, func(o Span) bool { return o.Start < span.End && span.Start < o.End }) {
				found = append(found, span)
			}
		}
	}
	slices.SortFunc(found, func(a, b Span) int { return a.Start - b.Start })
	return found
}

// runeOffsets turns byte offsets in s into rune offsets.
func runeOffsets(s string) func(int) int {
	return func(b int) int { return len([]rune(s[:b])) }
}

func digitsEdge(s string) bool {
	return s != "" && unicode.IsDigit(rune(s[0])) && unicode.IsDigit(rune(s[len(s)-1]))
}

// standsAlone says the digits at s[a:b] aren't part of a longer number.
func standsAlone(s string, a, b int) bool {
	return (a == 0 || !unicode.IsDigit(rune(s[a-1]))) && (b == len(s) || !unicode.IsDigit(rune(s[b])))
}

func digits(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsDigit(r) {
			return r
		}
		return -1
	}, s)
}

var date = regexp.MustCompile(`^\d{4}[ -]\d{1,2}[ -]\d{1,2}$`)

func notDate(s string) bool { return !date.MatchString(s) }

// luhn is the check digit of card numbers.
func luhn(s string) bool {
	d := digits(s)
	if len(d) < 13 || len(d) > 19 || strings.Count(d, d[:1]) == len(d) {
		return false
	}
	sum := 0
	for i := range len(d) {
		n := int(d[len(d)-1-i] - '0')
		if i%2 == 1 {
			if n *= 2; n > 9 {
				n -= 9
			}
		}
		sum += n
	}
	return sum%10 == 0
}

// chineseID checks a mainland ID card number's last character.
func chineseID(s string) bool {
	if len(s) != 18 {
		return false
	}
	weights := [...]int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	sum := 0
	for i, w := range weights {
		sum += int(s[i]-'0') * w
	}
	return "10X98765432"[sum%11] == strings.ToUpper(s[17:])[0]
}
