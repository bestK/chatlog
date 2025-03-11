package infobar

import (
	"fmt"

	"github.com/sjzar/chatlog/internal/ui/style"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const (
	Title = "infobar"
)

// InfoBarViewHeight info bar height.
const (
	InfoBarViewHeight = 6
	accountRow        = 0
	pidRow            = 1
	statusRow         = 2
	dataUsageRow      = 3
	workUsageRow      = 4
	httpServerRow     = 5

	// 列索引
	labelCol1 = 0 // 第一列标签
	valueCol1 = 1 // 第一列值
	labelCol2 = 2 // 第二列标签
	valueCol2 = 3 // 第二列值
	totalCols = 4
)

// InfoBar implements the info bar primitive.
type InfoBar struct {
	*tview.Box
	title string
	table *tview.Table
}

// NewInfoBar returns info bar view.
func New() *InfoBar {
	table := tview.NewTable()
	headerColor := style.InfoBarItemFgColor

	// Account 和 Version 行
	table.SetCell(
		accountRow,
		labelCol1,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "Account:")),
	)
	table.SetCell(accountRow, valueCol1, tview.NewTableCell(""))

	table.SetCell(
		accountRow,
		labelCol2,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "Version:")),
	)
	table.SetCell(accountRow, valueCol2, tview.NewTableCell(""))

	// PID 和 ExePath 行
	table.SetCell(
		pidRow,
		labelCol1,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "PID:")),
	)
	table.SetCell(pidRow, valueCol1, tview.NewTableCell(""))

	table.SetCell(
		pidRow,
		labelCol2,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "ExePath:")),
	)
	table.SetCell(pidRow, valueCol2, tview.NewTableCell(""))

	// Status 和 Key 行
	table.SetCell(
		statusRow,
		labelCol1,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "Status:")),
	)
	table.SetCell(statusRow, valueCol1, tview.NewTableCell(""))

	table.SetCell(
		statusRow,
		labelCol2,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "Data Key:")),
	)
	table.SetCell(statusRow, valueCol2, tview.NewTableCell(""))

	// Data Usage 和 Data Dir 行
	table.SetCell(
		dataUsageRow,
		labelCol1,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "Data Usage:")),
	)
	table.SetCell(dataUsageRow, valueCol1, tview.NewTableCell(""))

	table.SetCell(
		dataUsageRow,
		labelCol2,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "Data Dir:")),
	)
	table.SetCell(dataUsageRow, valueCol2, tview.NewTableCell(""))

	// Work Usage 和 Work Dir 行
	table.SetCell(
		workUsageRow,
		labelCol1,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "Work Usage:")),
	)
	table.SetCell(workUsageRow, valueCol1, tview.NewTableCell(""))

	table.SetCell(
		workUsageRow,
		labelCol2,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "Work Dir:")),
	)
	table.SetCell(workUsageRow, valueCol2, tview.NewTableCell(""))

	// HTTP Server 行
	table.SetCell(
		httpServerRow,
		labelCol1,
		tview.NewTableCell(fmt.Sprintf(" [%s::]%s", headerColor, "HTTP Server:")),
	)
	table.SetCell(httpServerRow, valueCol1, tview.NewTableCell(""))

	// infobar
	infoBar := &InfoBar{
		Box:   tview.NewBox(),
		title: Title,
		table: table,
	}

	return infoBar
}

func (info *InfoBar) UpdateAccount(account string) {
	info.table.GetCell(accountRow, valueCol1).SetText(account)
}

func (info *InfoBar) UpdateBasicInfo(pid int, version string, exePath string) {
	info.table.GetCell(pidRow, valueCol1).SetText(fmt.Sprintf("%d", pid))
	info.table.GetCell(pidRow, valueCol2).SetText(exePath)
	info.table.GetCell(accountRow, valueCol2).SetText(version)
}

func (info *InfoBar) UpdateStatus(status string) {
	info.table.GetCell(statusRow, valueCol1).SetText(status)
}

func (info *InfoBar) UpdateDataKey(key string) {
	info.table.GetCell(statusRow, valueCol2).SetText(key)
}

func (info *InfoBar) UpdateDataUsageDir(dataUsage string, dataDir string) {
	info.table.GetCell(dataUsageRow, valueCol1).SetText(dataUsage)
	info.table.GetCell(dataUsageRow, valueCol2).SetText(dataDir)
}

func (info *InfoBar) UpdateWorkUsageDir(workUsage string, workDir string) {
	info.table.GetCell(workUsageRow, valueCol1).SetText(workUsage)
	info.table.GetCell(workUsageRow, valueCol2).SetText(workDir)
}

// UpdateHTTPServer updates HTTP Server value.
func (info *InfoBar) UpdateHTTPServer(server string) {
	info.table.GetCell(httpServerRow, valueCol1).SetText(server)
}

// Draw draws this primitive onto the screen.
func (info *InfoBar) Draw(screen tcell.Screen) {
	info.Box.DrawForSubclass(screen, info)
	info.Box.SetBorder(false)

	x, y, width, height := info.GetInnerRect()

	info.table.SetRect(x, y, width, height)
	info.table.SetBorder(false)
	info.table.Draw(screen)
}
