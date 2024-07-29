package thunder_share

import (
	"github.com/alist-org/alist/v3/internal/driver"
	"github.com/alist-org/alist/v3/internal/op"
)

type Addition struct {
	driver.RootID
	ThunderPath string `json:"thunder_path" required:"true"`
	ShareId     string `json:"share_id" required:"true"`
	SharePwd    string `json:"share_pwd"`
	//优先使用视频链接代替下载链接
	UseVideoUrl bool `json:"use_video_url"`
}

var config = driver.Config{
	Name:        "ThunderShare",
	LocalSort:   true,
	NoUpload:    true,
	DefaultRoot: "",
}

func init() {
	op.RegisterDriver(func() driver.Driver {
		return &ThunderShare{}
	})
}
