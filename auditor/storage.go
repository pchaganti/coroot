package auditor

import (
	"fmt"
	"github.com/coroot/coroot/model"
	"github.com/coroot/coroot/timeseries"
	"github.com/coroot/coroot/utils"
	"github.com/dustin/go-humanize"
	"math"
)

func (a *appAuditor) storage() {
	report := model.NewAuditReport(a.w.Ctx, "Storage")

	for _, i := range a.app.Instances {
		for _, v := range i.Volumes {
			fullName := i.Name + ":" + v.MountPoint
			if i.Node != nil {
				if d := i.Node.Disks[v.Device.Value()]; d != nil {
					report.GetOrCreateChartInGroup("I/O latency <selector>, seconds", v.MountPoint).
						AddSeries(i.Name, d.Await)

					report.GetOrCreateChartInGroup("I/O utilization <selector>, %", v.MountPoint).
						AddSeries(i.Name, d.IOUtilizationPercent)

					report.GetOrCreateChartInGroup("IOPS <selector>", fullName).
						Stacked().
						Sorted().
						AddSeries("read", d.ReadOps, "blue").
						AddSeries("write", d.WriteOps, "amber")

					report.GetOrCreateChartInGroup("Bandwidth <selector>, bytes/second", fullName).
						Stacked().
						Sorted().
						AddSeries("read", d.ReadBytes, "blue").
						AddSeries("written", d.WrittenBytes, "amber")

					latencyMs := model.NewTableCell().SetUnit("ms")
					if d.Await != nil {
						latencyMs.SetValue(utils.FormatFloat(d.Await.Last() * 1000))
					}
					ioPercent := model.NewTableCell()
					if d.IOUtilizationPercent != nil {
						if last := d.IOUtilizationPercent.Last(); !math.IsNaN(last) {
							ioPercent.SetValue(fmt.Sprintf("%.0f%%", last))
						}
					}
					space := model.NewTableCell()
					if v.UsedBytes != nil && v.CapacityBytes != nil {
						capacity := v.CapacityBytes.Last()
						usage := v.UsedBytes.Last()
						if usage > 0 && capacity > 0 {
							space.SetValue(fmt.Sprintf(
								"%.0f%% (%s / %s)",
								usage/capacity*100,
								humanize.Bytes(uint64(usage)),
								humanize.Bytes(uint64(capacity))),
							)
						}
					}
					report.GetOrCreateTable("Volume", "Latency", "I/O", "Space", "Device").AddRow(
						model.NewTableCell(fullName),
						latencyMs,
						ioPercent,
						space,
						model.NewTableCell(v.Device.Value()).AddTag(v.Name.Value()),
					)
				}
				report.GetOrCreateChartInGroup("Disk space <selector>, bytes", fullName).
					Stacked().
					AddSeries("used", v.UsedBytes).
					SetThreshold("total", v.CapacityBytes, timeseries.Max)
			}
		}
	}
	a.addReport(report)
}