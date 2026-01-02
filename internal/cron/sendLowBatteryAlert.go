package cron

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"yxy-go/internal/logic/electricity"
	"yxy-go/internal/svc"
	"yxy-go/internal/types"

	"github.com/ArtisanCloud/PowerWeChat/v3/src/basicService/subscribeMessage/request"
	"github.com/ArtisanCloud/PowerWeChat/v3/src/kernel/power"
	"github.com/zeromicro/go-zero/core/logx"
	"gorm.io/gorm"
)

type SendLowBatteryAlertLogic struct {
	logx.Logger
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewSendLowBatteryAlertLogic(ctx context.Context, svcCtx *svc.ServiceContext) *SendLowBatteryAlertLogic {
	return &SendLowBatteryAlertLogic{
		Logger: logx.WithContext(ctx),
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

type Subscription struct {
	ID        int64  `gorm:"column:id"`
	UserID    int64  `gorm:"column:user_id"`
	OpenID    string `gorm:"column:openid"`
	YxyUID    string `gorm:"column:yxy_uid"`
	Campus    string `gorm:"column:campus"`
	Threshold int64  `gorm:"column:threshold"`
	Count     int64  `gorm:"column:count"`
}

// SendLowBatteryAlertLogic 发送低电量提醒
func (l *SendLowBatteryAlertLogic) SendLowBatteryAlertLogic() {
	stats := struct {
		Total         int // 总条数
		ProcessFailed int // 处理过程中出错的条数（如获取电量失败）
		NeedSend      int // 需要发送的条数（电量低于阈值）
		SendSuccess   int // 发送成功的条数
		SendFailed    int // 发送失败的条数
		NoSend        int // 无需发送的条数（电量高于阈值）
	}{}

	page := 1
	pageSize := 100
	for {
		subscriptions, err := l.querySubscriptionByPage(page, pageSize)
		if err != nil {
			l.Logger.Errorf("Query subscriptions failed: %v", err)
			return
		}
		if len(subscriptions) == 0 {
			break
		}
		stats.Total += len(subscriptions)

		var sendIDs []int64
		for _, subscription := range subscriptions {
			needSend, err := l.processSubscription(subscription)
			if err != nil {
				if errors.Is(err, ErrSendFailed) {
					stats.NeedSend++
					stats.SendFailed++
					l.Logger.Errorf("Send alert to user ID %d (OpenID: %s) failed: %v", subscription.UserID, subscription.OpenID, err)
				} else {
					stats.ProcessFailed++
					l.Logger.Errorf("Process subscription for user ID %d (OpenID: %s) failed: %v", subscription.UserID, subscription.OpenID, err)
				}
				continue
			}
			if needSend {
				sendIDs = append(sendIDs, subscription.ID)
				stats.NeedSend++
				stats.SendSuccess++
			} else {
				stats.NoSend++
			}
		}
		if err := l.decrementSubscriptionCount(sendIDs); err != nil {
			l.Logger.Errorf("Decrement subscription count failed for user IDs: %v, error: %v", sendIDs, err)
		}
		page++
	}
	l.Logger.Infof("Low battery alert statistics: Total=%d, ProcessFailed=%d, NeedSend=%d, SentSuccess=%d, SentFailed=%d, NoSend=%d",
		stats.Total, stats.ProcessFailed, stats.NeedSend, stats.SendSuccess, stats.SendFailed, stats.NoSend)
}

var ErrSendFailed = errors.New("send alert failed")

func (l *SendLowBatteryAlertLogic) processSubscription(subscription Subscription) (bool, error) {
	resp, err := l.getElecSurplus(subscription.YxyUID, subscription.Campus)
	if err != nil {
		return false, fmt.Errorf("get electricity surplus failed: %w", err)
	}
	if resp.Surplus > float64(subscription.Threshold) {
		return false, nil
	}
	if len([]rune(resp.DisplayRoomName)) > 20 {
		resp.DisplayRoomName = strings.Replace(resp.DisplayRoomName, "校区", "", 1)
	}
	mpResp, err := l.svcCtx.MiniProgram.SubscribeMessage.Send(l.ctx, &request.RequestSubscribeMessageSend{
		ToUser:           subscription.OpenID,
		TemplateID:       l.svcCtx.Config.LowBattery.MiniProgram.TemplateID,
		Page:             "/pages/electricity/index",
		MiniProgramState: l.svcCtx.Config.LowBattery.MiniProgram.State,
		Lang:             "zh_CN",
		Data: &power.HashMap{
			"character_string1": power.StringMap{ // 剩余电量
				"value": strconv.FormatFloat(resp.Surplus, 'f', 2, 64),
			},
			"thing2": power.StringMap{ // 寝室地址
				"value": resp.DisplayRoomName,
			},
			"thing3": power.StringMap{ // 备注
				"value": "寝室电量低于 " + strconv.FormatInt(subscription.Threshold, 10) + " 度，请及时充值",
			},
		},
	})
	if err != nil {
		return true, fmt.Errorf("%w: %v", ErrSendFailed, err)
	}
	if mpResp.ErrCode != 0 {
		// errCode: 43101, errMsg: user refuse to accept the msg
		if mpResp.ErrCode == 43101 {
			_ = l.resetSubscriptionCount(subscription.ID)
			return false, nil
		}
		return true, fmt.Errorf("%w: errcode: %d, errmsg: %s", ErrSendFailed, mpResp.ErrCode, mpResp.ErrMsg)
	}
	l.Logger.Infof("Send alert to user ID %d (OpenID: %s) successfully, electricity surplus: %.2f, threshold: %d",
		subscription.UserID, subscription.OpenID, resp.Surplus, subscription.Threshold)
	return true, nil
}

func (l *SendLowBatteryAlertLogic) querySubscriptionByPage(page, pageSize int) ([]Subscription, error) {
	var subscriptions []Subscription
	err := l.svcCtx.DB.Table("low_battery_alert_subscriptions lbas").
		Select("lbas.id, lbas.user_id, lbas.campus, lbas.threshold, lbas.count, u.wechat_open_id as openid, u.yxy_uid as yxy_uid").
		Joins("JOIN users u ON lbas.user_id = u.id").
		Where("lbas.count > 0").
		Limit(pageSize).
		Offset((page - 1) * pageSize).
		Find(&subscriptions).Error
	if err != nil {
		return nil, err
	}
	return subscriptions, nil
}

func (l *SendLowBatteryAlertLogic) decrementSubscriptionCount(ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	err := l.svcCtx.DB.Table("low_battery_alert_subscriptions").
		Where("id IN ?", ids).
		Update("count", gorm.Expr("count - 1")).Error
	return err
}

func (l *SendLowBatteryAlertLogic) resetSubscriptionCount(id int64) error {
	err := l.svcCtx.DB.Table("low_battery_alert_subscriptions").
		Where("id = ?", id).
		Update("count", 0).Error
	return err
}

func (l *SendLowBatteryAlertLogic) getElecSurplus(yxyUID, campus string) (*types.GetElectricitySurplusResp, error) {
	surplusLogic := electricity.NewGetElectricitySurplusLogic(l.ctx, l.svcCtx)
	return surplusLogic.GetElectricitySurplus(&types.GetElectricitySurplusReq{
		Uid:    yxyUID,
		Campus: campus,
	})
}
