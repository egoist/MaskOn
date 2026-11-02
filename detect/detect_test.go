package detect

import "testing"

func spanTexts(line string) map[Kind][]string {
	rs := []rune(halfWidth.Replace(line))
	out := map[Kind][]string{}
	for _, s := range Line(line) {
		out[s.Kind] = append(out[s.Kind], string(rs[s.Start:s.End]))
	}
	return out
}

func TestLine(t *testing.T) {
	for _, c := range []struct {
		line string
		kind Kind
		want string
	}{
		{"电话：13812345678", Phone, "13812345678"},
		{"Tel +86 138 1234 5678 ok", Phone, "+86 138 1234 5678"},
		{"Call (415) 555-0132 now", Phone, "(415) 555-0132"},
		{"Phone：（415） 555-0132", Phone, "(415) 555-0132"},
		{"车牌粵B12345", Plate, "粵B12345"},
		{"Email： a@b.co", Email, "a@b.co"},
		{"+1 415 555 0132", Phone, "+1 415 555 0132"},
		{"座机 010-88886666", Phone, "010-88886666"},
		{"mail me at ada.l+x@example.co.uk.", Email, "ada.l+x@example.co.uk"},
		{"4242 4242 4242 4242", Card, "4242 4242 4242 4242"},
		{"卡号6222021234567890128", Card, "6222021234567890128"},
		{"身份证 11010519491231002X", ID, "11010519491231002X"},
		{"SSN 078-05-1120", ID, "078-05-1120"},
		{"from 192.168.1.10:80", IP, "192.168.1.10"},
		{"总计 ¥1,299.00", Money, "¥1,299.00"},
		{"Total: $42.50", Money, "$42.50"},
		{"合计 128.5 元", Money, "128.5 元"},
		{"车牌 粤B12345 已驶入", Plate, "粤B12345"},
		{"https://example.com/a?b=c next", Link, "https://example.com/a?b=c"},
		{"follow @some_one here", Handle, "@some_one"},
		{"姓名：张三", Name, "张三"},
		{"Ship to: 1901 Oak St", Address, "1901 Oak St"},
		{"订单号 2026110112345", Number, "2026110112345"},
	} {
		got := spanTexts(c.line)[c.kind]
		if len(got) != 1 || got[0] != c.want {
			t.Errorf("%q: %s = %q, want %q (all: %v)", c.line, c.kind, got, c.want, Line(c.line))
		}
	}
}

func TestLineLeavesAlone(t *testing.T) {
	for _, line := range []string{
		"", "Hello, world", "10:07", "2026-11-01", "第 3 页", "Version 1.2.3", "iPhone 17 Pro",
		"1234 5678 9012 3456x", // fails the card check, but is a number: see below
	} {
		for _, s := range Line(line) {
			if line == "1234 5678 9012 3456x" && s.Kind == Number {
				continue
			}
			t.Errorf("%q: found %v", line, s)
		}
	}
}

func TestRuneOffsets(t *testing.T) {
	spans := Line("电话13812345678，谢谢")
	if len(spans) != 1 || spans[0] != (Span{Phone, 2, 13}) {
		t.Fatalf("%v", spans)
	}
}

func TestFullWidthKeepsPlaces(t *testing.T) {
	line := "电话（415） 555-0132 好"
	spans := Line(line)
	if len(spans) != 1 || spans[0].Start != 2 || spans[0].End != 16 {
		t.Fatalf("%v", spans)
	}
	if got := string([]rune(line)[spans[0].Start:spans[0].End]); got != "（415） 555-0132" {
		t.Fatal(got)
	}
}

func TestChecks(t *testing.T) {
	if luhn("4242 4242 4242 4241") {
		t.Error("luhn passed a bad check digit")
	}
	if luhn("0000 0000 0000 0000") {
		t.Error("luhn passed all the same digit")
	}
	if chineseID("110105194912310021") {
		t.Error("chineseID passed a bad check character")
	}
}

func TestOverlapFirstRuleWins(t *testing.T) {
	spans := Line("4242424242424242")
	if len(spans) != 1 || spans[0].Kind != Card {
		t.Fatalf("%v", spans)
	}
}
