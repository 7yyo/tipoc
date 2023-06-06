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

const catalog = "item.catalog"

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

const (
	sql = iota
	other
	kill
	crash
	dataCorrupted
	recoverSystemd
	disaster
	scaleIn
)

func getOperatorTp(i int) string {
	switch i {
	case sql:
		return "SQL"
	case kill:
		return "KILL"
	case crash:
		return "CRASH"
	case dataCorrupted:
		return "DATA_CORRUPTED"
	case recoverSystemd:
		return "RECOVER_SYSTEMD"
	case disaster:
		return "DISASTER"
	case scaleIn:
		return "SCALE_IN"
	default:
		return ""
	}
}

type item struct {
	value       string
	operator    int
	componentTp string
}

func (i item) String() string {
	return i.value
}

const (
	tree = iota
	selected
	lg
	pBar
	ldr
)

type widget struct {
	tree       *widgets.Tree
	selected   *widgets.Tree
	lg         *widgets.List
	processBar *widgets.Gauge
	loader     *widgets.List
	show       *widgets.List
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

var pa int
var others string

func (w *widget) setTreeNodes() error {

	treeNodes, err := w.buildTreeByCatalog()
	if err != nil {
		return err
	}
	if err := w.appendOthers(&treeNodes); err != nil {
		return err
	}
	w.tree.SetNodes(treeNodes)

	nodes, err := getComponents()
	if err != nil {
		return err
	}
	labelKey := getLabelKey(nodes)
	w.tree.Walk(func(treeNode *widgets.TreeNode) bool {
		name := treeNode.Value.String()
		if isOperator(name) {
			pa = whichOperator(name)
			switch pa {
			case disaster:
				if len(labelKey) != 0 {
					for key, _ := range labelKey {
						appendTreeNode(treeNode, key, "", pa)
					}
				}
			default:
				appendNodes(nodes, treeNode, "", pa)
			}
		}
		for _, n := range nodes[name] {
			addr := net.JoinHostPort(n.host, n.port)
			addr = n.isPdLeader(addr)
			appendTreeNode(treeNode, addr, name, pa)
		}
		if labelKey[name] {
			visited := make(map[string]bool)
			for _, s := range nodes["tikv"] {
				value := s.labels[name]
				if visited[value] {
					continue
				}
				visited[value] = true
				appendTreeNode(treeNode, value, "", pa)
			}
		}
		return true
	})
	return nil
}

func (w *widget) buildTreeByCatalog() ([]*widgets.TreeNode, error) {
	var treeNodes []*widgets.TreeNode
	var root *widgets.TreeNode
	var parentNodes []*widgets.TreeNode

	f, err := svr.Open(catalog)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		level := strings.Count(line, ".")
		name := strings.TrimLeft(line, "\t")
		node := &widgets.TreeNode{
			Value: item{
				value:    name,
				operator: sql,
			},
		}
		if level == 0 {
			if root != nil {
				treeNodes = append(treeNodes, root)
			}
			root = node
			parentNodes = []*widgets.TreeNode{root}
		} else {
			for len(parentNodes) > level {
				parentNodes = parentNodes[:len(parentNodes)-1]
			}
			parent := parentNodes[len(parentNodes)-1]
			parent.Nodes = append(parent.Nodes, node)
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
	return treeNodes, nil
}

func (w *widget) appendOthers(treeNodes *[]*widgets.TreeNode) error {
	if others != "" {
		dir, err := ioutil.ReadDir(others)
		if err != nil {
			return err
		}
		if len(dir) != 0 {
			othersNode := widgets.TreeNode{
				Value: item{
					value: others,
				},
			}
			*treeNodes = append(*treeNodes, &othersNode)
			err := filepath.Walk(others, func(path string, info fs.FileInfo, err error) error {
				if !info.IsDir() {
					appendTreeNode(&othersNode, strings.TrimSuffix(info.Name(), suffix), "", other)
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			log.Logger.Warnf("%s has no script, skip.", others)
		}
	}
	return nil
}

func (w *widget) walkTreeScript() (map[string]script, error) {
	ss := make(map[string]script)
	var err error
	var sc []string
	w.tree.Walk(func(node *widgets.TreeNode) bool {
		value := node.Value.String()
		operator := node.Value.(item).operator
		if node.Nodes == nil && (operator == sql || operator == other) {
			sc, err = getScript(value, operator)
			if err != nil {
				return true
			}
			ss[value] = script{
				sql: sc, name: value, tp: sql,
			}
		}
		return true
	})
	return ss, nil
}

func (w *widget) right() {
	selected := w.tree.SelectedNode().Value
	if len(w.tree.SelectedNode().Nodes) == 0 && !w.contains(selected.String()) {
		if w.conflict(selected.(item).operator) {
			log.Logger.Warnf("conflicting items")
			return
		}
		node := widgets.TreeNode{
			Value: item{
				value:       selected.(item).value,
				operator:    selected.(item).operator,
				componentTp: selected.(item).componentTp,
			},
		}
		var newSelectedNode []*widgets.TreeNode
		w.selected.Walk(func(treeNode *widgets.TreeNode) bool {
			newSelectedNode = append(newSelectedNode, treeNode)
			return true
		})
		newSelectedNode = append(newSelectedNode, &node)
		w.selected.SetNodes(newSelectedNode)
		w.selected.Expand()
		w.selected.ScrollBottom()
		w.selected.Title = getOperatorTp(selected.(item).operator)
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

func (w *widget) selected2Arr() []string {
	var arr []string
	w.selected.Walk(func(treeNode *widgets.TreeNode) bool {
		arr = append(arr, treeNode.Value.String())
		return true
	})
	return arr
}

func (w *widget) conflict(action int) bool {
	if w.selected.SelectedNode() == nil {
		return false
	}
	var a int
	w.selected.Walk(func(node *widgets.TreeNode) bool {
		a = node.Value.(item).operator
		return false
	})
	return a != action
}

func (w *widget) treeLength(tp int) int {
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

func appendNodes(nodes map[string][]component, treeNode *widgets.TreeNode, tp string, action int) {
	for name := range nodes {
		appendTreeNode(treeNode, name, tp, action)
	}
}

func appendTreeNode(treeNode *widgets.TreeNode, name, tp string, o int) {
	treeNode.Nodes = append(treeNode.Nodes, &widgets.TreeNode{
		Value: item{
			value:       name,
			componentTp: tp,
			operator:    o,
		},
	})
}
