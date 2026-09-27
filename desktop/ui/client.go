package ui

import (
	"context"
	"dotenvless/desktop/privacy"
	"dotenvless/internal/application"
	"dotenvless/internal/vault"
	"dotenvless/internal/workspace"
	"errors"
	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"io"
	"strings"
)

type Client struct {
	Service                                                         *application.Service
	Window                                                          fyne.Window
	Root                                                            fyne.CanvasObject
	Key                                                             *widget.Entry
	Value                                                           *secretEntry
	SaveButton                                                      *widget.Button
	Message                                                         *widget.Label
	view                                                            application.View
	selected, scope, sharedID                                       string
	guard                                                           *privacy.Session
	search                                                          *widget.Entry
	visible                                                         []application.ProjectInfo
	projects, sharedList                                            *widget.List
	table                                                           *widget.Table
	scopeSelect, reference                                          *widget.Select
	sharedLabels                                                    map[string]string
	title, origin, revealed, sharedUses, sharedRevealed, statusText *widget.Label
	sharedName                                                      *widget.Entry
	sharedValue                                                     *secretEntry
	migration                                                       *widget.Button
	tabs                                                            *container.AppTabs
	pending                                                         *application.ImportPreview
	scanCancel                                                      context.CancelFunc
	scanGeneration                                                  uint64
	updating, closed                                                bool
}

func NewClient(s *application.Service, w fyne.Window, clip privacy.Clipboard) *Client {
	c := &Client{Service: s, Window: w, scope: "local", sharedLabels: map[string]string{}}
	c.Message = widget.NewLabel("就绪")
	c.Message.Wrapping = fyne.TextWrapWord
	c.guard = privacy.New(clip, func(f func()) { fyne.Do(f) })
	c.guard.OnError = func(error) { c.Message.SetText("剪贴板操作未完成，请重试或手动清除。") }
	c.Key = widget.NewEntry()
	c.Key.SetPlaceHolder("变量名，例如 DATABASE_URL")
	c.Value = newSecretEntry(c.guard)
	c.Value.SetPlaceHolder("输入新值；留空保存为空字符串")
	c.sharedName = widget.NewEntry()
	c.sharedName.SetPlaceHolder("共享条目名称")
	c.sharedValue = newSecretEntry(c.guard)
	c.sharedValue.SetPlaceHolder("输入新值；留空保存为空字符串")
	c.title = widget.NewLabelWithStyle("选择工程", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
	c.title.Wrapping = fyne.TextWrapWord
	c.origin = widget.NewLabel("选择变量查看来源，或直接新增变量")
	c.origin.Wrapping = fyne.TextWrapWord
	c.revealed = widget.NewLabel("••••••")
	c.revealed.Wrapping = fyne.TextWrapBreak
	c.sharedRevealed = widget.NewLabel("••••••")
	c.sharedRevealed.Wrapping = fyne.TextWrapBreak
	c.sharedUses = widget.NewLabel("共享条目仅通过显式引用使用。")
	c.sharedUses.Wrapping = fyne.TextWrapWord
	c.statusText = widget.NewLabel("选择工程后检查状态。")
	c.statusText.Wrapping = fyne.TextWrapWord
	c.scopeSelect = widget.NewSelect([]string{"基础配置", "local"}, func(v string) {
		if c.updating {
			return
		}
		if v == "基础配置" {
			v = "base"
		}
		c.SelectScope(v)
	})
	c.reference = widget.NewSelect(nil, nil)
	c.reference.PlaceHolder = "选择共享条目"
	c.SaveButton = widget.NewButtonWithIcon("保存独立值", theme.DocumentSaveIcon(), c.saveValue)
	c.SaveButton.Importance = widget.HighImportance
	c.build()
	w.SetContent(c.Root)
	_ = c.Refresh()
	return c
}
func (c *Client) build() {
	c.search = widget.NewEntry()
	c.search.SetPlaceHolder("搜索工程名称或路径")
	c.search.OnChanged = func(string) { c.filterProjects() }
	c.projects = widget.NewList(func() int { return len(c.visible) }, func() fyne.CanvasObject {
		name := widget.NewLabelWithStyle("", fyne.TextAlignLeading, fyne.TextStyle{Bold: true})
		name.Truncation = fyne.TextTruncateEllipsis
		path := widget.NewLabel("")
		path.Truncation = fyne.TextTruncateEllipsis
		return container.NewVBox(name, path)
	}, func(i widget.ListItemID, o fyne.CanvasObject) {
		if i >= len(c.visible) {
			return
		}
		p := c.visible[i]
		box := o.(*fyne.Container)
		label := p.Name
		if label == "" {
			label = "待关联 · " + shortID(p.ID)
		}
		state := "空配置"
		if p.Configured {
			state = "已配置"
		}
		label += fmt.Sprintf(" · %d 环境 · %s", len(p.Environments), state)
		box.Objects[0].(*widget.Label).SetText(label)
		path := p.Root
		if path == "" {
			path = "添加原目录以关联已有配置"
		} else if !p.Available {
			path = "目录不可用 · " + path
		}
		box.Objects[1].(*widget.Label).SetText(path)
	})
	c.projects.OnSelected = func(i widget.ListItemID) {
		if !c.updating && i < len(c.visible) {
			c.SelectProject(c.visible[i].ID)
		}
	}
	sidebar := container.NewBorder(container.NewVBox(
		widget.NewLabelWithStyle("工程", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), c.search,
		container.NewGridWithColumns(2, widget.NewButton("添加目录", c.addProject), widget.NewButton("批量发现", c.discoverProjects))),
		widget.NewButton("移除工程记录", c.removeProject), nil, nil, c.projects)
	c.table = widget.NewTable(func() (int, int) { return len(c.view.Entries) + 1, 3 }, func() fyne.CanvasObject {
		l := widget.NewLabel("")
		l.Truncation = fyne.TextTruncateEllipsis
		return l
	}, func(cell widget.TableCellID, o fyne.CanvasObject) {
		label := o.(*widget.Label)
		if cell.Row == 0 {
			label.SetText([]string{"变量名", "来源", "状态"}[cell.Col])
			return
		}
		if cell.Row > len(c.view.Entries) {
			return
		}
		e := c.view.Entries[cell.Row-1]
		values := []string{e.Key, sourceLabel(e), kindLabel(e.Kind)}
		label.SetText(values[cell.Col])
	})
	c.table.SetColumnWidth(0, 210)
	c.table.SetColumnWidth(1, 185)
	c.table.SetColumnWidth(2, 90)
	c.table.OnSelected = func(cell widget.TableCellID) {
		if cell.Row < 1 || cell.Row > len(c.view.Entries) {
			return
		}
		e := c.view.Entries[cell.Row-1]
		c.HideSecrets()
		c.Value.SetText("")
		c.Key.SetText(e.Key)
		c.origin.SetText("来源：" + sourceLabel(e) + "\n状态：" + kindLabel(e.Kind) + "\n保存独立值只影响当前配置层。")
	}
	editor := container.NewVBox(
		widget.NewLabelWithStyle("变量编辑", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), c.origin,
		widget.NewLabel("变量名"), c.Key, widget.NewLabel("替换为独立值"), c.Value, c.SaveButton,
		c.reference, widget.NewButton("使用共享引用", c.bindShared),
		container.NewGridWithColumns(2, widget.NewButton("删除本层", c.unset), widget.NewButton("在此环境停用", c.disable)),
		widget.NewSeparator(), widget.NewLabel("已保存值 · 默认遮蔽"), c.revealed,
		container.NewGridWithColumns(2, widget.NewButton("查看 15 秒", c.revealValue), widget.NewButton("复制值", c.copyValue)),
	)
	editorScroll := container.NewVScroll(editor)
	editorScroll.SetMinSize(fyne.NewSize(285, 350))
	split := container.NewHSplit(c.table, editorScroll)
	split.Offset = .64
	controls := container.NewVBox(c.title,
		container.NewBorder(nil, nil, widget.NewLabel("配置层"), nil, c.scopeSelect),
		container.NewGridWithColumns(4, widget.NewButton("新建环境", func() { c.environmentDialog(false) }), widget.NewButton("复制环境", func() { c.environmentDialog(true) }), widget.NewButton("重命名环境", c.renameEnvironment), widget.NewButton("删除环境", c.deleteEnvironment)),
		container.NewGridWithColumns(3, widget.NewButton("新增变量", func() {
			c.HideSecrets()
			c.Key.SetText("")
			c.Value.SetText("")
			c.origin.SetText("新增变量 · 留空值将保存为空字符串")
		}), widget.NewButton("导入 dotenv", c.importFile), widget.NewButton("生成 .env.example", c.example)),
	)
	projectPage := container.NewBorder(controls, nil, nil, nil, split)
	c.sharedList = widget.NewList(func() int { return len(c.view.Shared) }, func() fyne.CanvasObject { return widget.NewLabel("") },
		func(i widget.ListItemID, o fyne.CanvasObject) {
			if i < len(c.view.Shared) {
				s := c.view.Shared[i]
				o.(*widget.Label).SetText(fmt.Sprintf("%s · %d 处引用", s.Name, len(s.Uses)))
			}
		})
	c.sharedList.OnSelected = func(i widget.ListItemID) {
		if i >= len(c.view.Shared) {
			return
		}
		s := c.view.Shared[i]
		c.HideSecrets()
		c.sharedID = s.ID
		c.sharedName.SetText(s.Name)
		c.sharedValue.SetText("")
		c.sharedUses.SetText(usageText(s.Uses))
	}
	sharedEditor := container.NewVBox(
		widget.NewLabelWithStyle("共享条目", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), c.sharedName, c.sharedValue,
		widget.NewButton("保存共享值", c.saveShared), widget.NewButton("只修改名称", c.renameShared),
		widget.NewSeparator(), c.sharedUses, c.sharedRevealed,
		container.NewGridWithColumns(2, widget.NewButton("查看 15 秒", c.revealShared), widget.NewButton("复制值", c.copyShared)),
		widget.NewButton("删除共享条目", c.deleteShared),
	)
	sharedSplit := container.NewHSplit(c.sharedList, container.NewVScroll(sharedEditor))
	sharedSplit.Offset = .45
	sharedPage := container.NewBorder(widget.NewButton("新增共享条目", func() {
		c.HideSecrets()
		c.sharedID = ""
		c.sharedName.SetText("")
		c.sharedValue.SetText("")
		c.sharedUses.SetText("保存后，在工程中显式引用该条目。")
	}), nil, nil, nil, sharedSplit)
	statusPage := container.NewBorder(container.NewGridWithColumns(2, widget.NewButton("检查所选配置与工作区", c.checkStatus), widget.NewButton("复制启动命令", c.copyCommand)), nil, nil, nil, container.NewVScroll(c.statusText))
	c.tabs = container.NewAppTabs(container.NewTabItem("工程", projectPage), container.NewTabItem("共享库", sharedPage), container.NewTabItem("状态与接入", statusPage))
	c.tabs.OnSelected = func(*container.TabItem) { c.HideSecrets(); c.Value.SetText(""); c.sharedValue.SetText("") }
	main := container.NewHSplit(sidebar, c.tabs)
	main.Offset = .23
	c.migration = widget.NewButton("升级旧 Vault", c.migrate)
	header := container.NewBorder(nil, nil, widget.NewLabelWithStyle("Dotenvless  ·  本地 ENV 管理中心", fyne.TextAlignLeading, fyne.TextStyle{Bold: true}), container.NewHBox(c.migration, widget.NewButtonWithIcon("刷新", theme.ViewRefreshIcon(), func() { _ = c.Refresh() })))
	c.Root = container.NewBorder(header, c.Message, nil, nil, main)
}
func shortID(id string) string {
	if len(id) > 10 {
		return id[:10]
	}
	return id
}
func kindLabel(kind string) string {
	switch kind {
	case "shared":
		return "共享引用"
	case "disabled":
		return "已停用"
	default:
		return "独立值"
	}
}
func sourceLabel(e application.Entry) string {
	source := e.Source
	if source == "base" {
		source = "基础配置"
	}
	if e.Inherited {
		source += "（继承）"
	}
	if e.Kind == "shared" {
		name := e.SharedName
		if name == "" {
			name = "引用失效"
		}
		source += " / " + name
	}
	return source
}
func usageText(uses []application.Usage) string {
	if len(uses) == 0 {
		return "当前没有引用。共享值不会自动注入工程。"
	}
	var lines []string
	lines = append(lines, "保存的引用（基础引用也可能被环境覆盖）：")
	for _, u := range uses {
		name := u.ProjectName
		if name == "" {
			name = shortID(u.ProjectID)
		}
		scope := u.Scope
		if scope == "base" {
			scope = "基础配置"
		}
		lines = append(lines, name+" / "+scope+" / "+u.Key)
	}
	lines = append(lines, "更新在下一次启动时生效，运行中的进程保持原配置。")
	return strings.Join(lines, "\n")
}
func (c *Client) filterProjects() {
	if c.projects == nil {
		return
	}
	q := strings.ToLower(strings.TrimSpace(c.search.Text))
	c.visible = nil
	for _, p := range c.view.Projects {
		if q == "" || strings.Contains(strings.ToLower(p.Name+" "+p.Root), q) {
			c.visible = append(c.visible, p)
		}
	}
	c.projects.Refresh()
}
func (c *Client) Refresh() error {
	c.HideSecrets()
	c.cancelImport()
	c.Key.SetText("")
	c.Value.SetText("")
	c.sharedValue.SetText("")
	view, e := c.Service.View(c.selected, c.scope)
	if e != nil {
		c.fail(e)
		return e
	}
	c.view = view
	c.updating = true
	defer func() { c.updating = false }()
	c.migration.Hide()
	if view.Version == 1 {
		c.migration.Show()
	}
	c.filterProjects()
	c.table.Refresh()
	c.sharedList.Refresh()
	if c.sharedID != "" {
		found := false
		for _, item := range view.Shared {
			if item.ID == c.sharedID {
				found = true
				c.sharedName.SetText(item.Name)
				c.sharedUses.SetText(usageText(item.Uses))
				break
			}
		}
		if !found {
			c.sharedID = ""
			c.sharedName.SetText("")
			c.sharedUses.SetText("共享条目已移除。")
		}
	}
	c.sharedLabels = map[string]string{}
	options := []string{}
	for _, item := range view.Shared {
		label := item.Name + " [" + shortID(item.ID) + "]"
		options = append(options, label)
		c.sharedLabels[label] = item.ID
	}
	c.reference.Options = options
	c.reference.ClearSelected()
	c.reference.Refresh()
	p, found := c.currentProject()
	if found {
		c.title.SetText(p.Name + "\n" + p.Root)
		scopes := append([]string{"基础配置"}, p.Environments...)
		c.scopeSelect.Options = scopes
		choice := c.scope
		if choice == "base" {
			choice = "基础配置"
		}
		c.scopeSelect.SetSelected(choice)
		c.scopeSelect.Refresh()
		c.statusText.SetText("工程：" + p.Name + "\n目录：" + p.Root + "\n配置层：" + choice + "\n\n进入工程目录后执行：\n" + c.command() + "\n\n变量值只注入目标子进程；界面选择不改变 CLI 默认 local 环境。")
	} else {
		c.title.SetText("选择工程")
		c.scopeSelect.ClearSelected()
		c.statusText.SetText("选择工程后检查状态。")
	}
	c.origin.SetText("选择变量查看来源，或直接新增变量")
	if view.Version == 1 {
		c.Message.SetText("检测到旧 Vault：可以查看；修改前请显式升级，并同时使用新版 CLI。")
	} else if view.SelectionError != "" {
		c.fail(errors.New(view.SelectionError))
	} else {
		c.Message.SetText(fmt.Sprintf("%d 个工程 · %d 个共享条目 · 配置版本 %d", len(view.Projects), len(view.Shared), view.Revision))
	}
	return nil
}
func (c *Client) currentProject() (application.ProjectInfo, bool) {
	for _, p := range c.view.Projects {
		if p.ID == c.selected {
			return p, true
		}
	}
	return application.ProjectInfo{}, false
}
func (c *Client) SelectProject(id string)  { c.selected = id; c.scope = "local"; _ = c.Refresh() }
func (c *Client) SelectScope(scope string) { c.scope = scope; _ = c.Refresh() }
func (c *Client) HideSecrets() {
	c.guard.Hide()
	if c.Value != nil {
		c.Value.Password = true
		c.Value.Refresh()
	}
	if c.sharedValue != nil {
		c.sharedValue.Password = true
		c.sharedValue.Refresh()
	}
}
func (c *Client) Close() {
	if c.closed {
		return
	}
	c.closed = true
	c.scanGeneration++
	if c.scanCancel != nil {
		c.scanCancel()
	}
	c.cancelImport()
	c.Value.SetText("")
	c.sharedValue.SetText("")
	c.guard.Close()
}
func (c *Client) fail(e error) {
	if e == nil {
		return
	}
	switch {
	case errors.Is(e, vault.ErrConflict):
		c.Message.SetText("配置已被其他操作修改，请刷新后重新输入。")
	case errors.Is(e, vault.ErrMigration):
		c.Message.SetText("请先升级旧 Vault，再修改配置。")
	case errors.Is(e, vault.ErrNotInitialized):
		c.Message.SetText("请先添加或选择已登记工程。")
	default:
		c.Message.SetText("操作未完成：" + e.Error())
	}
}
func (c *Client) afterWrite(e error) {
	if e != nil {
		c.fail(e)
		return
	}
	_ = c.Refresh()
}
func (c *Client) saveValue() {
	value := []byte(c.Value.Text)
	defer clear(value)
	c.Value.SetText("")
	c.afterWrite(c.Service.Set(c.selected, c.scope, c.Key.Text, value, c.view.Revision))
}
func (c *Client) bindShared() {
	id := c.sharedLabels[c.reference.Selected]
	if id == "" {
		c.Message.SetText("请先选择共享条目。")
		return
	}
	c.afterWrite(c.Service.BindShared(c.selected, c.scope, c.Key.Text, id, c.view.Revision))
}
func (c *Client) confirm(title, message string, action func()) {
	d := dialog.NewCustomConfirm(title, "确认", "取消", widget.NewLabel(message), func(ok bool) {
		if ok {
			action()
		}
	}, c.Window)
	d.Show()
}
func (c *Client) unset() {
	key, e := vault.NormalizeKey(c.Key.Text)
	if e != nil {
		c.fail(e)
		return
	}
	id, scope, rev := c.selected, c.scope, c.view.Revision
	c.confirm("删除本层绑定", "删除 "+key+" 的本层绑定；存在基础配置时会恢复继承。", func() { c.afterWrite(c.Service.Unset(id, scope, key, rev)) })
}
func (c *Client) disable() {
	key, e := vault.NormalizeKey(c.Key.Text)
	if e != nil {
		c.fail(e)
		return
	}
	id, scope, rev := c.selected, c.scope, c.view.Revision
	c.confirm("停用环境变量", "该环境不会注入 "+key+"，并排除父终端的同名变量。", func() { c.afterWrite(c.Service.Disable(id, scope, key, rev)) })
}
func (c *Client) revealValue() {
	value, e := c.Service.Reveal(c.selected, c.scope, c.Key.Text)
	if e != nil {
		c.fail(e)
		return
	}
	c.guard.Reveal(value, func(text string) {
		if text == "" {
			text = "••••••"
		}
		c.revealed.SetText(text)
	})
}
func (c *Client) copyValue() {
	value, e := c.Service.Reveal(c.selected, c.scope, c.Key.Text)
	if e == nil {
		e = c.guard.Copy(value)
	}
	if e != nil {
		c.fail(e)
	} else {
		c.Message.SetText("已复制；30 秒后仅清除此份剪贴板内容。")
	}
}
func (c *Client) revealShared() {
	value, e := c.Service.RevealShared(c.sharedID)
	if e != nil {
		c.fail(e)
		return
	}
	c.guard.Reveal(value, func(text string) {
		if text == "" {
			text = "••••••"
		}
		c.sharedRevealed.SetText(text)
	})
}
func (c *Client) copyShared() {
	value, e := c.Service.RevealShared(c.sharedID)
	if e == nil {
		e = c.guard.Copy(value)
	}
	if e != nil {
		c.fail(e)
	} else {
		c.Message.SetText("已复制；30 秒后仅清除此份剪贴板内容。")
	}
}
func (c *Client) saveShared() {
	id, name, revision := c.sharedID, c.sharedName.Text, c.view.Revision
	value := []byte(c.sharedValue.Text)
	c.sharedValue.SetText("")
	save := func() {
		defer clear(value)
		next, e := c.Service.SaveShared(id, name, value, revision)
		if e == nil {
			c.sharedID = next
		}
		c.afterWrite(e)
	}
	if id == "" {
		save()
		return
	}
	uses := ""
	for _, item := range c.view.Shared {
		if item.ID == id {
			uses = usageText(item.Uses)
		}
	}
	content := widget.NewLabel(uses)
	content.Wrapping = fyne.TextWrapWord
	d := dialog.NewCustomConfirm("更新共享值", "更新全部引用", "取消", content, func(ok bool) {
		if ok {
			save()
		} else {
			clear(value)
		}
	}, c.Window)
	d.Resize(fyne.NewSize(520, 350))
	d.Show()
}
func (c *Client) renameShared() {
	c.afterWrite(c.Service.RenameShared(c.sharedID, c.sharedName.Text, c.view.Revision))
}
func (c *Client) deleteShared() {
	id, rev := c.sharedID, c.view.Revision
	c.confirm("删除共享条目", "仅无引用的条目可以删除。", func() {
		e := c.Service.DeleteShared(id, rev)
		if e == nil {
			c.sharedID = ""
			c.sharedName.SetText("")
		}
		c.afterWrite(e)
	})
}
func (c *Client) PrepareImport(r io.Reader) (*application.ImportPreview, error) {
	c.cancelImport()
	p, e := c.Service.PreviewImport(c.selected, c.scope, r)
	if e != nil {
		return nil, e
	}
	if p.Revision != c.view.Revision {
		p.Close()
		return nil, vault.ErrConflict
	}
	c.pending = p
	return p, nil
}
func (c *Client) ApplyImport(overwrite bool) error {
	if c.pending == nil {
		return errors.New("import preview was cancelled")
	}
	e := c.Service.CommitImport(c.pending, overwrite)
	if e == nil {
		c.pending = nil
		_ = c.Refresh()
	}
	return e
}
func (c *Client) cancelImport() {
	if c.pending != nil {
		c.pending.Close()
		c.pending = nil
	}
}
func (c *Client) command() string {
	scope := c.scope
	if scope == "base" {
		scope = "local"
	}
	return "dvl --env " + scope + " run -- <COMMAND>"
}
func (c *Client) copyCommand() {
	if _, ok := c.currentProject(); !ok {
		return
	}
	c.Window.Clipboard().SetContent(c.command())
	c.Message.SetText("已复制不含变量值的启动命令，请替换 <COMMAND>。")
}
func (c *Client) checkStatus() {
	p, ok := c.currentProject()
	if !ok {
		return
	}
	if p.Root == "" || !p.Available {
		c.Message.SetText("工程目录不可用；旧记录可通过添加原目录关联。")
		return
	}
	e := c.Service.Verify(p.ID, c.scope)
	if e != nil {
		c.fail(e)
		return
	}
	report, e := workspace.Check(p.Root)
	if e != nil {
		c.fail(errors.New("工作区检查未完成"))
		return
	}
	lines := []string{"DPAPI：所选配置解密验证通过", "检查范围：" + workspace.Scope, "目录：" + p.Root}
	for _, file := range report.Files {
		lines = append(lines, file+" · Git: "+report.Git[file])
	}
	if len(report.Files) == 0 {
		lines = append(lines, "在检查范围内未发现环境文件；未读取文件内容。")
	}
	lines = append(lines, "\n进入工程目录后运行：", c.command())
	c.statusText.SetText(strings.Join(lines, "\n"))
}
