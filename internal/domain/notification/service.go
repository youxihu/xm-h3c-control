package notification

// Service 通知服务接口
type Service interface {
	// SendNotification 发送过期通知
	SendNotification(notification *ExpiryNotification) error
	// SendDeletionNotification 发送删除通知
	SendDeletionNotification(notification *DeletionNotification) error
}