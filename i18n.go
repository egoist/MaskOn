package main

import (
	"fmt"
	"strings"

	"github.com/egoist/maskon/detect"
)

var chinese bool

func setLanguage(locale string) { chinese = strings.HasPrefix(strings.ToLower(locale), "zh") }

// t is the text in the app's language: the English, looked up in Chinese.
func t(en string) string {
	if chinese {
		if z, ok := zh[en]; ok {
			return z
		}
	}
	return en
}

// tf formats t(en).
func tf(en string, args ...any) string { return fmt.Sprintf(t(en), args...) }

var zh = map[string]string{
	// Onboarding
	"Auto Mask": "自动打码",
	"Maskon finds QR codes, barcodes, phone numbers and more, and covers them in one tap.": "Maskon 会检测二维码、条形码、电话号码等敏感内容，一键全部盖住。",
	"Mask everything":       "一键打码",
	"Nice, just like that!": "很好，就是这样！",
	"Tap to Mask":           "轻点打码",
	"Spot something it missed? Tap it, and it's covered.":       "自动检测漏掉的内容，\n轻点一下就能自己盖住。",
	"Try tapping the name":                                      "试试轻点姓名",
	"Mask a Single Word":                                        "打码单个词",
	"Press and hold text to pick exactly which words to cover.": "长按一段文字，精确挑选要盖住的词。",
	"Press and hold the address":                                "长按这行地址",
	"Tap a word to cover it":                                    "轻点想要盖住的词",
	"Mask Any Area":                                             "框选打码",
	"Drag across anything, and a box covers it.":                "在任何地方拖动手指，\n拖出的方框就能盖住它。",
	"Drag across the card":                                      "在卡片上拖一下",
	"Continue":                                                  "继续",
	"Next":                                                      "下一步",
	"Get Started":                                               "开始使用",
	"Ship to":                                                   "收货地址",
	// Home
	"Screenshots":                 "截图",
	"All Photos":                  "全部照片",
	"Add":                         "添加",
	"Pick a photo":                "选择照片",
	"Photo":                       "照片",
	"How it works":                "使用说明",
	"Allow access to your photos": "允许访问你的照片",
	"Maskon shows your latest screenshots here. Nothing ever leaves your device.": "Maskon 会在这里显示你最新的截图，所有内容都不会离开你的设备。",
	"Allow":                               "允许访问",
	"Open Settings":                       "打开设置",
	"Choose More Photos":                  "选择更多照片",
	"Nothing here yet":                    "这里还没有内容",
	"Take a screenshot, or pick a photo.": "截一张图，或者选择一张照片。",
	// Editor
	"Finding what's private…":  "正在检测敏感内容…",
	"Couldn't read this image": "无法读取这张图片",
	"Mask All (%d)":            "全部打码（%d）",
	"Save":                     "保存",
	"Saved to Photos":          "已保存到照片",
	"Couldn't save: %s":        "保存失败：%s",
	"Share":                    "分享",
	"Undo":                     "撤销",
	"Redo":                     "重做",
	"Remove all":               "全部移除",
	"Style":                    "样式",
	"Green":                    "绿色",
	"Black":                    "黑色",
	"White":                    "白色",
	"Mosaic":                   "马赛克",
	"Masked. Tap to remove.":   "已打码，轻点移除",
	"Tap to mask":              "轻点打码",
	"Done":                     "完成",
	// Kinds
	"Faces":     "人脸",
	"Codes":     "二维码",
	"Phones":    "电话",
	"Emails":    "邮箱",
	"Links":     "链接",
	"Cards":     "银行卡",
	"IDs":       "证件号",
	"IPs":       "IP 地址",
	"Amounts":   "金额",
	"Plates":    "车牌",
	"Numbers":   "数字",
	"Names":     "姓名",
	"Addresses": "地址",
	"Handles":   "账号",
}

// kindTitle is a chip's title for a kind of text.
func kindTitle(k detect.Kind) string {
	switch k {
	case detect.Phone:
		return t("Phones")
	case detect.Email:
		return t("Emails")
	case detect.Link:
		return t("Links")
	case detect.Card:
		return t("Cards")
	case detect.ID:
		return t("IDs")
	case detect.IP:
		return t("IPs")
	case detect.Money:
		return t("Amounts")
	case detect.Plate:
		return t("Plates")
	case detect.Name:
		return t("Names")
	case detect.Address:
		return t("Addresses")
	case detect.Handle:
		return t("Handles")
	}
	return t("Numbers")
}
