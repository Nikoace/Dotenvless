package ui

import (
	"context"
	"dotenvless/internal/application"

	"fmt"
	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/widget"
	"strings"
)

func (c *Client) addProject() {
	revision := c.view.Revision
	d := dialog.NewFolderOpen(func(uri fyne.ListableURI, e error) {
		if e != nil {
			c.Message.SetText("无法打开目录。")
			return
		}
		if uri == nil || c.closed {
			return
		}
		p, e := c.Service.Register(uri.Path(), revision)
		if e != nil {
			c.fail(e)
			return
		}
		c.SelectProject(p.ID)
	}, c.Window)
	d.SetConfirmText("添加工程")
	d.SetDismissText("取消")
	d.Show()
}
func (c *Client) discoverProjects() {
	d := dialog.NewFolderOpen(func(uri fyne.ListableURI, e error) {
		if e != nil {
			c.Message.SetText("无法打开目录。")
			return
		}
		if uri == nil || c.closed {
			return
		}
		c.startDiscovery(uri.Path())
	}, c.Window)
	d.SetConfirmText("扫描此目录")
	d.SetDismissText("取消")
	d.Show()
}
func (c *Client) startDiscovery(root string) {
	if c.scanCancel != nil {
		c.scanCancel()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c.scanCancel = cancel
	c.scanGeneration++
	generation := c.scanGeneration
	revision := c.view.Revision
	progress := dialog.NewCustom("发现 Git 工程", "取消", container.NewVBox(widget.NewLabel("正在扫描目录；跳过依赖、构建目录和目录链接。"), widget.NewProgressBarInfinite()), c.Window)
	progress.SetOnClosed(cancel)
	progress.Show()
	go func() {
		candidates, e := application.Discover(ctx, root)
		fyne.Do(func() {
			if c.closed || generation != c.scanGeneration {
				return
			}
			cancelled := ctx.Err() != nil
			progress.Hide()
			c.scanCancel = nil
			if cancelled {
				return
			}
			if e != nil {
				c.fail(e)
				return
			}
			if len(candidates) == 0 {
				c.Message.SetText("未发现 Git 工程；普通目录可通过「添加目录」登记。")
				return
			}
			options := make([]string, len(candidates))
			for i, p := range candidates {
				options[i] = p.Root
			}
			choices := widget.NewCheckGroup(options, nil)
			content := container.NewVScroll(choices)
			content.SetMinSize(fyne.NewSize(600, 350))
			confirm := dialog.NewCustomConfirm("选择要登记的工程", "添加所选", "取消", content, func(ok bool) {
				if !ok || c.closed {
					return
				}
				added := 0
				for _, path := range choices.Selected {
					if _, e := c.Service.Register(path, revision); e != nil {
						c.fail(e)
						c.Message.SetText(fmt.Sprintf("已添加 %d 个工程。%s", added, c.Message.Text))
						return
					}
					added++
					revision++
				}
				_ = c.Refresh()
				c.Message.SetText(fmt.Sprintf("已添加 %d 个工程。", added))
			}, c.Window)
			confirm.Show()
		})
	}()
}
func (c *Client) removeProject() {
	p, ok := c.currentProject()
	if !ok {
		return
	}
	revision := c.view.Revision
	c.confirm("移除工程记录", "删除该工程在 Vault 中的全部配置。工程文件和共享条目会保留。\n"+p.Name+"\n"+p.Root, func() {
		e := c.Service.RemoveProject(p.ID, revision)
		if e == nil {
			c.selected = ""
			c.scope = "local"
		}
		c.afterWrite(e)
	})
}
func (c *Client) environmentDialog(copyCurrent bool) {
	if _, ok := c.currentProject(); !ok {
		c.Message.SetText("请先选择工程。")
		return
	}
	id, source, revision := c.selected, "", c.view.Revision
	title := "新建环境"
	if copyCurrent {
		if c.scope == "base" {
			c.Message.SetText("基础配置由所有环境继承；请先选择要复制的环境。")
			return
		}
		source = c.scope
		title = "复制环境"
	}
	name := widget.NewEntry()
	name.SetPlaceHolder("例如 dev、test、staging")
	d := dialog.NewCustomConfirm(title, "创建", "取消", container.NewVBox(widget.NewLabel("使用小写字母开头，可包含数字、下划线和短横线，最多 32 字符。"), name), func(ok bool) {
		if !ok {
			return
		}
		e := c.Service.CreateEnvironment(id, name.Text, source, revision)
		if e != nil {
			c.fail(e)
			return
		}
		c.selected = id
		c.SelectScope(name.Text)
	}, c.Window)
	d.Resize(fyne.NewSize(550, 180))
	d.Show()
}
func (c *Client) renameEnvironment() {
	if c.scope == "base" || c.scope == "local" {
		c.Message.SetText("基础配置与 local 保留，不能重命名。")
		return
	}
	id, old, revision := c.selected, c.scope, c.view.Revision
	name := widget.NewEntry()
	name.SetText(old)
	d := dialog.NewCustomConfirm("重命名环境", "重命名", "取消", name, func(ok bool) {
		if !ok {
			return
		}
		e := c.Service.RenameEnvironment(id, old, name.Text, revision)
		if e != nil {
			c.fail(e)
			return
		}
		c.selected = id
		c.SelectScope(name.Text)
	}, c.Window)
	d.Show()
}
func (c *Client) deleteEnvironment() {
	if c.scope == "base" || c.scope == "local" {
		c.Message.SetText("基础配置与 local 保留，不能删除。")
		return
	}
	id, scope, revision := c.selected, c.scope, c.view.Revision
	c.confirm("删除环境", "删除 "+scope+" 的所有覆盖、引用和停用标记。基础配置会保留。", func() {
		e := c.Service.DeleteEnvironment(id, scope, revision)
		if e == nil {
			c.scope = "local"
		}
		c.afterWrite(e)
	})
}
func (c *Client) importFile() {
	if _, ok := c.currentProject(); !ok {
		c.Message.SetText("请先选择工程与配置层。")
		return
	}
	id, scope, revision := c.selected, c.scope, c.view.Revision
	c.cancelImport()
	d := dialog.NewFileOpen(func(r fyne.URIReadCloser, e error) {
		if e != nil {
			c.Message.SetText("无法读取所选文件。")
			return
		}
		if r == nil {
			return
		}
		defer r.Close()
		if c.closed || c.selected != id || c.scope != scope || c.view.Revision != revision {
			return
		}
		preview, e := c.PrepareImport(r)
		if e != nil {
			c.fail(e)
			return
		}
		text := "目标：" + scope + "\n键名：\n" + strings.Join(preview.Keys, "\n")
		if len(preview.Conflicts) > 0 {
			text += "\n\n冲突键名（包括继承）：\n" + strings.Join(preview.Conflicts, "\n")
		}
		label := widget.NewLabel(text)
		label.Wrapping = fyne.TextWrapWord
		overwrite := widget.NewCheck("明确覆盖冲突项", nil)
		content := container.NewBorder(widget.NewLabel("仅预览键名。确认后整批写入；原文件保留。"), overwrite, nil, nil, container.NewVScroll(label))
		confirm := dialog.NewCustomConfirm("导入预览", "导入", "取消", content, func(ok bool) {
			if !ok {
				c.cancelImport()
				return
			}
			if e := c.ApplyImport(overwrite.Checked); e != nil {
				c.cancelImport()
				c.fail(e)
			}
		}, c.Window)
		confirm.Resize(fyne.NewSize(560, 450))
		confirm.Show()
	}, c.Window)
	d.SetConfirmText("预览")
	d.SetDismissText("取消")
	d.Show()
}
func (c *Client) example() {
	p, ok := c.currentProject()
	if !ok {
		return
	}
	if p.Root == "" || !p.Available {
		c.Message.SetText("请先关联可用的工程目录。")
		return
	}
	if e := c.Service.Example(p.ID, c.scope, p.Root); e != nil {
		c.fail(e)
		return
	}
	c.Message.SetText("已创建 .env.example，仅包含所选配置的有效键名。已有文件不会覆盖。")
}
func (c *Client) migrate() {
	c.confirm("升级 Vault", "先保留原始密文备份，再将旧变量迁入各工程的 local。\n升级后请使用 v0.2 CLI。旧记录通过原目录匹配，不会猜测归属。", func() {
		_, e := c.Service.Store.Migrate()
		if e != nil {
			c.fail(e)
			return
		}
		_ = c.Refresh()
		c.Message.SetText("升级完成。原始密文备份位于 Vault 同目录，文件名后缀为 .v1.bak。")
	})
}
