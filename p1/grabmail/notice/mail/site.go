package mail

import (
	"fmt"
)

func siteName(siteID int) string {
	switch siteID {
	case 1:
		return "智联"
	case 2:
		return "51job"
	case 3:
		return "猎聘"
	case 4:
		return "大街"
	case 7:
		return "中国人才热线"
	case 9:
		return "中华英才网"
	case 11:
		return "拉勾"
	case 12:
		return "58同城"
	case 13:
		return "赶集"
	case 20:
		return "领英"
	case 21:
		return "中国金融人才网"
	case 22:
		return "新安人才网"
	case 23:
		return "成都人才网"
	case 24:
		return "聘宝"
	case 25:
		return "找萝卜"
	case 26:
		return "人才啊"
	case 27:
		return "看准网"
	case 28:
		return "桔子网"
	case 30:
		return "猎上网"
	case 31:
		return "卓聘"
	case 32:
		return "昆山人才"
	case 33:
		return "Boss直聘"
	case 34:
		return "建筑英才网"
	case 35:
		return "简历咖"
	case 36:
		return "妙招网"
	case 37:
		return "纷简历"
	case 38:
		return "脉脉"
	case 39:
		return "维基百科"
	case 40:
		return "新浪微博"
	case 41:
		return "职友圈"
	case 42:
		return "天眼查"
	case 43:
		return "猎萝卜"
	case 44:
		return "卓博"
	case 45:
		return "实习僧"
	case 46:
		return "汇博"
	case 47:
		return "温州人力资源网"
	case 48:
		return "服装人才网"
	case 49:
		return "牧通人才网"
	case 51:
		return "汽车人才网"
	case 52:
		return "中国医疗人才网"
	case 53:
		return "北极星"
	case 10003:
		return "丁香园"
	case 10001:
		return "全国统一社会信用代码信息核查系统"
	case 10002:
		return "知乎"
	default:
		return fmt.Sprintf("Unknown SiteID %d", siteID)
	}
}
