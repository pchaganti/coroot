package cache

import (
	"context"
	"fmt"
	"github.com/coroot/coroot/cache/chunk"
	"github.com/coroot/coroot/constructor"
	"github.com/coroot/coroot/db"
	"github.com/coroot/coroot/model"
	"github.com/coroot/coroot/prom"
	"github.com/coroot/coroot/timeseries"
)

func (c *Cache) GetCacheClient(projectId db.ProjectId) *Client {
	return &Client{
		cache:     c,
		projectId: projectId,
	}
}

type Client struct {
	cache     *Cache
	projectId db.ProjectId
}

func (c *Client) QueryRange(ctx context.Context, query string, from, to timeseries.Time, step timeseries.Duration) ([]model.MetricValues, error) {
	c.cache.lock.RLock()
	defer c.cache.lock.RUnlock()
	projData := c.cache.byProject[c.projectId]
	if projData == nil {
		return nil, fmt.Errorf("unknown project: %s", c.projectId)
	}
	if step == prom.StepUndefined {
		step = projData.refreshInterval
	}
	from = from.Truncate(step)
	to = to.Truncate(step)
	queryHash := hash(query)
	qData := projData.queries[queryHash]
	if qData == nil {
		return nil, fmt.Errorf("%w: %s", constructor.ErrUnknownQuery, query)
	}
	start := from
	end := to
	res := map[uint64]model.MetricValues{}
	resPoints := int(to.Sub(from)/step + 1)
	for _, ch := range qData.chunksOnDisk {
		if ch.From > end || ch.From.Add(timeseries.Duration(ch.PointsCount-1)*ch.Step) < start {
			continue
		}
		err := chunk.Read(ch.Path, from, resPoints, step, res)
		if err != nil {
			return nil, err
		}
	}
	r := make([]model.MetricValues, 0, len(res))
	for _, mv := range res {
		r = append(r, mv)
	}
	return r, nil
}

func (c *Client) GetTo() (timeseries.Time, error) {
	to, err := c.cache.getMinUpdateTime(c.projectId)
	if err != nil {
		return 0, err
	}

	if to.IsZero() {
		return 0, nil
	}

	c.cache.lock.RLock()
	defer c.cache.lock.RUnlock()
	projData := c.cache.byProject[c.projectId]
	if projData == nil {
		return 0, fmt.Errorf("unknown project: %s", c.projectId)
	}
	step := projData.refreshInterval

	return to.Add(-step), nil
}

func (c *Client) GetStatus() (*Status, error) {
	return c.cache.getStatus(c.projectId)
}
