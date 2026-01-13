package notification

import (
	"fmt"
	"time"
)

// ExpiryNotification 过期通知实体
type ExpiryNotification struct {
	GlobalAddress string    // 外网地址端口
	LocalAddress  string    // 内网地址端口
	Protocol      string    // 协议类型
	Description   string    // 服务描述
	ExpiryDate    time.Time // 到期时间
	NotifyTime    time.Time // 通知时间
}

// DeletionNotification 删除通知实体
type DeletionNotification struct {
	GlobalAddress string    // 外网地址端口
	LocalAddress  string    // 内网地址端口
	Protocol      string    // 协议类型
	Description   string    // 服务描述
	ExpiryDate    time.Time // 到期时间
	DeleteTime    time.Time // 删除时间
}

// FormatMessage 格式化通知消息为Markdown格式
func (n *ExpiryNotification) FormatMessage() string {
	return fmt.Sprintf(`## [通知] 端口映射即将过期

**消息来源：** H3c-MSR2600

**外网地址端口：** %s

**内网地址端口：** %s

**协议类型：** %s

**描述：** %s

**到期时间：** %s

**通知时间：** %s

---

[查看内外网映射关系表](https://alidocs.dingtalk.com/i/nodes/0eMKjyp813EOMaXPH9EkeOZwVxAZB1Gv?utm_scene=team_space)`,
		n.GlobalAddress,
		n.LocalAddress,
		n.Protocol,
		n.Description,
		n.ExpiryDate.Format(time.DateTime),
		n.NotifyTime.Format(time.DateTime),
	)
}

// FormatMessageWithGroup 格式化通知消息为Markdown格式，包含群组信息（已废弃，保持兼容性）
func (n *ExpiryNotification) FormatMessageWithGroup(groupName string) string {
	return n.FormatMessage() // 直接调用不带群组的格式化方法
}

// FormatMessage 格式化删除通知消息为Markdown格式
func (d *DeletionNotification) FormatMessage() string {
	return fmt.Sprintf(`## [通知] 端口映射条目删除

**消息来源：** H3c-MSR2600

**外网地址端口：** %s

**内网地址端口：** %s

**协议类型：** %s

**描述：** %s

**到期时间：** %s

**删除时间：** %s

---

[查看内外网映射关系表](https://alidocs.dingtalk.com/i/nodes/0eMKjyp813EOMaXPH9EkeOZwVxAZB1Gv?utm_scene=team_space)`,
		d.GlobalAddress,
		d.LocalAddress,
		d.Protocol,
		d.Description,
		d.ExpiryDate.Format(time.DateTime),
		d.DeleteTime.Format(time.DateTime),
	)
}