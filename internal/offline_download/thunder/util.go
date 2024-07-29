package thunder

import (
	"context"
	"github.com/alist-org/alist/v3/drivers/thunder"
	"time"

	"github.com/Xhofe/go-cache"
	"github.com/alist-org/alist/v3/internal/op"
	"github.com/alist-org/alist/v3/pkg/singleflight"
)

var taskCache = cache.NewMemCache(cache.WithShards[[]thunder.OfflineTask](16))
var taskG singleflight.Group[[]thunder.OfflineTask]

func (p *Thunder) GetTasks(thunderDriver *thunder.ThunderExpert) ([]thunder.OfflineTask, error) {
	key := op.Key(thunderDriver, "/drive/v1/task")
	if !p.refreshTaskCache {
		if tasks, ok := taskCache.Get(key); ok {
			return tasks, nil
		}
	}
	p.refreshTaskCache = false
	tasks, err, _ := taskG.Do(key, func() ([]thunder.OfflineTask, error) {
		ctx := context.Background()
		phase := []string{}
		tasks, err := thunderDriver.OfflineList(ctx, "", phase)
		if err != nil {
			return nil, err
		}
		// 添加缓存 10s
		if len(tasks) > 0 {
			taskCache.Set(key, tasks, cache.WithEx[[]thunder.OfflineTask](time.Second*10))
		} else {
			taskCache.Del(key)
		}
		return tasks, nil
	})
	if err != nil {
		return nil, err
	}
	return tasks, nil
}
