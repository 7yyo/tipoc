package server

import (
	"bufio"
	"embed"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"io/fs"
	"io/ioutil"
	"net"
	"path/filepath"
	"pictorial/log"
	"strings"
)

//go:embed "*"
var svr embed.FS

const itemCatalog = "item.catalog"

const (
	up      = "<Up>"
	down    = "<Down>"
	left    = "<Left>"
	right   = "<Right>"
	enter   = "<Enter>"
	ctrlC   = "<C-c>"
	sUp     = ","
	sDown   = "."
	sDelete = "<Backspace>"
)

type widgetTp int

const (
	tree widgetTp = iota
	selected
	lg
	processBar
	ldr
)

type widget struct {
	tree       *widgets.Tree
	selected   *widgets.Tree
	lg         *widgets.List
	processBar *widgets.Gauge
	loader     *widgets.List
}

func newTree(t string, x1 int, y1 int, x2 int, y2 int) *widgets.Tree {
	tree := widgets.NewTree()
	tree.TextStyle = ui.NewStyle(ui.ColorClear)
	tree.Title = t
	if t == "selected" {
		nodes := make([]*widgets.TreeNode, 0)
		tree.SetNodes(nodes)
	}
	tree.TitleStyle = ui.NewStyle(ui.ColorClear)
	tree.SetRect(x1, y1, x2, y2)
	tree.Block.BorderStyle = ui.NewStyle(ui.ColorClear)
	tree.SelectedRowStyle = ui.Style{
		Fg:       ui.ColorGreen,
		Bg:       ui.ColorClear,
		Modifier: ui.ModifierBold,
	}
	return tree
}

func newList(rows []string, t string, x1 int, y1 int, x2 int, y2 int) *widgets.List {
	list := widgets.NewList()
	list.Title = t
	list.WrapText = false
	list.Rows = rows
	list.TitleStyle = ui.NewStyle(ui.ColorClear)
	list.SetRect(x1, y1, x2, y2)
	list.Block.BorderStyle = ui.NewStyle(ui.ColorClear)
	list.SelectedRowStyle = ui.Style{
		Fg:       ui.ColorGreen,
		Bg:       ui.ColorClear,
		Modifier: ui.ModifierBold,
	}
	list.TextStyle = ui.Style{
		Fg: ui.ColorClear,
		Bg: ui.ColorClear,
	}
	return list
}

func newGauge(t string, x1 int, y1 int, x2 int, y2 int) *widgets.Gauge {
	gauge := widgets.NewGauge()
	gauge.Title = t
	gauge.Percent = 0
	gauge.SetRect(x1, y1, x2, y2)
	gauge.BarColor = ui.ColorGreen
	gauge.Block.BorderStyle = ui.NewStyle(ui.ColorClear)
	gauge.TitleStyle = ui.NewStyle(ui.ColorClear)
	return gauge
}

var oType operatorTp
var others string

func (w *widget) buildTreeByCatalog() error {

	var treeNodes []*widgets.TreeNode
	var root *widgets.TreeNode
	var parentNodes []*widgets.TreeNode

	f, err := svr.Open(itemCatalog)
	if err != nil {
		return err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		level := strings.Count(line, ".")
		name := strings.TrimLeft(line, "\t")
		node := widgets.TreeNode{
			Value: option{
				value: name,
			},
		}
		if level == 0 {
			if root != nil {
				treeNodes = append(treeNodes, root)
			}
			root = &node
			parentNodes = []*widgets.TreeNode{root}
		} else {
			for len(parentNodes) > level {
				parentNodes = parentNodes[:len(parentNodes)-1]
			}
			parent := parentNodes[len(parentNodes)-1]
			parent.Nodes = append(parent.Nodes, &node)
			if level > len(parentNodes)-1 {
				parentNodes = append(parentNodes, parent.Nodes[len(parent.Nodes)-1])
			} else {
				parentNodes[level] = parent.Nodes[len(parent.Nodes)-1]
			}
		}
	}
	if root != nil {
		treeNodes = append(treeNodes, root)
	}
	appendOthers(&treeNodes)
	w.tree.SetNodes(treeNodes)
	w.walkTreeNode()

	components, err := getComponents()
	if err != nil {
		return err
	}
	labelKey := getLabelKey(components[tikv])

	w.tree.Walk(func(treeNode *widgets.TreeNode) bool {
		value := treeNode.Value.String()
		idx := getIdxByName(value)
		if _, ok := operatorMapping[idx]; ok {
			oType = operatorMapping[idx]
			switch oType {
			case disaster:
				if len(labelKey) != 0 {
					for labelName, _ := range labelKey {
						appendNode(
							treeNode, labelName, true, -1, disaster)
					}
				}
			default:
				appendComponent(components, treeNode, oType)
			}
		}

		cType := getComponentType(value)
		for _, n := range components[cType] {
			addr := net.JoinHostPort(n.host, n.port)
			addr = n.isPdLeader(addr)
			appendNode(treeNode, addr, false, cType, oType)
		}
		if labelKey[value] {
			visited := make(map[string]bool)
			for _, s := range components[tikv] {
				value := s.labels[value]
				if visited[value] {
					continue
				}
				visited[value] = true
				appendNode(treeNode, value, false, tikv, oType)
			}
		}
		return true
	})

	return nil
}

func appendOthers(treeNodes *[]*widgets.TreeNode) {
	if others != "" {
		dir, err := ioutil.ReadDir(others)
		if err != nil {
			log.Logger.Error(err)
			return
		}
		if len(dir) != 0 {
			othersNode := widgets.TreeNode{
				Value: newOption(others, true, -1, -1),
			}
			*treeNodes = append(*treeNodes, &othersNode)
			err := filepath.Walk(others, func(path string, info fs.FileInfo, err error) error {
				if !info.IsDir() {
					appendNode(
						&othersNode, info.Name(), false, -1, otherSql)
				}
				return nil
			})
			if err != nil {
				log.Logger.Error(err)
				return
			}
		} else {
			log.Logger.Warnf("%s has no script, skip.", others)
		}
	}
}

func (w *widget) walkTreeNode() {
	w.tree.Walk(func(node *widgets.TreeNode) bool {
		v := node.Value.String()
		// other had process in appendOthers, so jump loop
		if v == others {
			return false
		}
		if len(node.Nodes) != 0 {
			node.Value = newOption(v, true, -1, -1)
		} else {
			idx := getIdxByName(v)
			if o, ok := operatorMapping[idx]; ok {
				node.Value = newOption(v, true, -1, o)
			} else {
				if isSafety(v) {
					node.Value = newOption(v, false, -1, safetySql)
				} else {
					node.Value = newOption(v, false, -1, sql)
				}
			}
		}
		return true
	})
	log.Logger.Debug("walk tree node complete.")
}

func (w *widget) walkTreeScript() (map[string]script, error) {
	scripts := make(map[string]script)
	w.tree.Walk(func(node *widgets.TreeNode) bool {
		if node.Value.(option).isCatalog || node.Value.(option).componentTp != -1 {
			return true
		}
		script, err := getScript(node.Value.(option))
		if err != nil {
			log.Logger.Warn(err.Error())
		}
		scripts[node.Value.String()] = *script
		return true
	})
	return scripts, nil
}

func (w *widget) right() {
	selected := w.tree.SelectedNode().Value
	if len(w.tree.SelectedNode().Nodes) == 0 && !w.contains(selected.String()) {
		if w.conflict(selected.(option).operatorTp) {
			log.Logger.Warnf("conflicting items")
			return
		}
		s := selected.(option)
		node := widgets.TreeNode{
			Value: newOption(s.value, s.isCatalog, s.componentTp, s.operatorTp),
		}
		var newSelectedNode []*widgets.TreeNode
		w.selected.Walk(func(treeNode *widgets.TreeNode) bool {
			newSelectedNode = append(newSelectedNode, treeNode)
			return true
		})
		newSelectedNode = append(newSelectedNode, &node)
		w.selected.SetNodes(newSelectedNode)
		w.selected.ScrollBottom()
		w.selected.Title = getOperatorTp(selected.(option).operatorTp)
	} else {
		w.tree.Expand()
	}
}

func (w *widget) removeSelected() {
	if w.selected.SelectedNode() != nil {
		name := w.selected.SelectedNode().Value.String()
		w.removeTreeNode(name)
		w.clearSelectedTitle()
		w.selected.ScrollUp()
	}
}

func (w *widget) clearSelectedTitle() {
	if w.treeLength(selected) == 0 {
		w.selected.Title = ""
	}
}

func (w *widget) contains(s string) bool {
	arr := w.selected2Arr()
	set := make(map[string]bool)
	for _, v := range arr {
		set[v] = true
	}
	return set[s]
}

func (w *widget) removeTreeNode(name string) {
	var newSelectedNode []*widgets.TreeNode
	w.selected.Walk(func(treeNode *widgets.TreeNode) bool {
		if treeNode.Value.String() != name {
			newSelectedNode = append(newSelectedNode, treeNode)
		}
		return true
	})
	w.selected.SetNodes(newSelectedNode)
}

func (w *widget) cleanSelected() {
	var emptyTreeNode []*widgets.TreeNode
	w.selected.SetNodes(emptyTreeNode)
}

func (w *widget) selected2Arr() []string {
	var arr []string
	w.selected.Walk(func(treeNode *widgets.TreeNode) bool {
		arr = append(arr, treeNode.Value.String())
		return true
	})
	return arr
}

// When selecting, there must be no operator-level conflicts
func (w *widget) conflict(o operatorTp) bool {
	if w.selected.SelectedNode() == nil {
		return false
	}
	var oType operatorTp
	w.selected.Walk(func(node *widgets.TreeNode) bool {
		oType = node.Value.(option).operatorTp
		return false
	})
	return oType != o
}

func (w *widget) treeLength(tp widgetTp) int {
	var t *widgets.Tree
	switch tp {
	case tree:
		t = w.tree
	case selected:
		t = w.selected
	}
	length := 0
	t.Walk(func(treeNode *widgets.TreeNode) bool {
		length++
		return true
	})
	return length
}

func (w *widget) refresh(idx int) {
	x := (float64(idx) / float64(w.treeLength(selected))) * 100
	w.processBar.Percent = int(x) % 101
	w.selected.ScrollDown()
	ui.Render(w.processBar, w.selected)
}

func appendComponent(cs map[componentTp][]component, treeNode *widgets.TreeNode, oType operatorTp) {
	for cType, _ := range cs {
		appendNode(treeNode, getComponentTp(cType), true, -1, oType)
	}
}

func appendNode(treeNode *widgets.TreeNode, value string, is bool, c componentTp, o operatorTp) {
	treeNode.Nodes = append(treeNode.Nodes, &widgets.TreeNode{
		Value: newOption(
			value, is, c, o,
		),
	})
}

func newOption(v string, is bool, c componentTp, o operatorTp) option {
	return option{
		value:       v,
		isCatalog:   is,
		componentTp: c,
		operatorTp:  o,
	}
}

func getIdxByName(name string) string {
	return strings.Split(name, " ")[0]
}
