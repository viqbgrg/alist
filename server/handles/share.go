package handles

import (
	"github.com/alist-org/alist/v3/drivers/thunder"
	"github.com/alist-org/alist/v3/drivers/thunder_share"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
)

type ShareLinkReq struct {
	Id       string `json:"id" required:"true"`
	ShareId  string `json:"share_id" required:"true"`
	SharePwd string `json:"share_pwd"`
}

func GetShareLink(c *gin.Context) {

	var req ShareLinkReq
	if err := c.ShouldBind(&req); err != nil {
		common.ErrorResp(c, err, 400)
		return
	}
	storages := op.GetAllStorages()
	var thunderDriver *thunder.ThunderExpert
	for _, storage := range storages {
		thunderDriver1, ok := storage.(*thunder.ThunderExpert)
		if ok {
			thunderDriver = thunderDriver1
			break
		}
	}
	if thunderDriver == nil {
		common.ErrorStrResp(c, "unsupported storage driver for offline download, only Pikpak is supported", 400)
		return
	}
	files, err := thunder_share.GetFiles(req.Id, req.ShareId, req.SharePwd, "", *thunderDriver)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}
	common.SuccessResp(c, files)
}
