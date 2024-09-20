package handles

import (
	"github.com/alist-org/alist/v3/drivers/thunder"
	"github.com/alist-org/alist/v3/drivers/thunder_share"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/server/common"
	"github.com/gin-gonic/gin"
)

type ShareFileReq struct {
	Id       string `json:"id" required:"true"`
	ShareId  string `json:"share_id" required:"true"`
	SharePwd string `json:"share_pwd"`
}

func GetShareFile(c *gin.Context) {

	var req ShareFileReq
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
	t := thunder_share.Common{
		req.ShareId,
		req.SharePwd,
		"",
		thunderDriver,
	}
	files, err := t.GetFiles(req.Id)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}
	common.SuccessResp(c, files)
}

type ShareLinkReq struct {
	Id            string `json:"id" required:"true"`
	ShareId       string `json:"share_id" required:"true"`
	SharePwd      string `json:"share_pwd"`
	PassCodeToken string `json:"pass_code_token"`
	TempPathId    string `json:"temp_path_id" required:"true"`
	Hash          string `json:"hash" required:"true"`
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
	t := thunder_share.Common{
		req.ShareId,
		req.SharePwd,
		req.PassCodeToken,
		thunderDriver,
	}
	files, err := t.GetLink(req.TempPathId, req.Id, req.Hash)
	if err != nil {
		common.ErrorStrResp(c, err.Error(), 400)
		return
	}
	common.SuccessResp(c, files)
}
