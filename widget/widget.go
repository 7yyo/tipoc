package widget

import (
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"pictorial/log"
)

type Widget struct {
	T *widgets.Tree
	C *widgets.Tree
	O *widgets.List
	P *widgets.Gauge
	L *widgets.List
}

const (
	Tree       = "Tree"
	Chosen     = "Chosen"
	Output     = "Output"
	ProcessBar = "ProcessBar"
	Loader     = "Loader"
)

func CreateTree() *widgets.Tree {
	x, y := ui.TerminalDimensions()
	t, err := NewTree()
	if err != nil {
		panic(err)
	}
	t.TextStyle = ui.NewStyle(ui.ColorClear)
	t.Title = Tree
	t.TitleStyle = ui.NewStyle(ui.ColorClear)
	t.SetRect(0, 0, x/3, y/2)
	t.Block.BorderStyle = ui.NewStyle(ui.ColorClear)
	t.SelectedRowStyle = ui.Style{
		Fg:       ui.ColorGreen,
		Bg:       ui.ColorClear,
		Modifier: ui.ModifierBold,
	}
	return t
}

func NewChosen() *widgets.Tree {
	x, y := ui.TerminalDimensions()
	c := widgets.NewTree()
	c.TextStyle = ui.NewStyle(ui.ColorClear)
	nodes := make([]*widgets.TreeNode, 0)
	c.SetNodes(nodes)
	c.TitleStyle = ui.NewStyle(ui.ColorClear)
	c.SetRect(x/3, 0, 2*x/3, y/2)
	c.Block.BorderStyle = ui.NewStyle(ui.ColorClear)
	c.SelectedRowStyle = ui.Style{
		Fg:       ui.ColorGreen,
		Bg:       ui.ColorClear,
		Modifier: ui.ModifierBold,
	}
	return c
}

func NewOutput() *widgets.List {
	x, y := ui.TerminalDimensions()
	o := widgets.NewList()
	o.Title = Output
	o.WrapText = false
	o.TitleStyle = ui.NewStyle(ui.ColorClear)
	o.SetRect(0, int(float64(y)*0.93), 2*x/3, int((float64(y)/10)*5))
	o.Block.BorderStyle = ui.NewStyle(ui.ColorClear)
	o.SelectedRowStyle = ui.Style{
		Fg:       ui.ColorGreen,
		Bg:       ui.ColorClear,
		Modifier: ui.ModifierBold,
	}
	o.TextStyle = ui.Style{
		Fg: ui.ColorClear,
		Bg: ui.ColorClear,
	}
	return o
}

func NewLoader() *widgets.List {
	x, y := ui.TerminalDimensions()
	l := widgets.NewList()
	l.Title = Loader
	l.WrapText = false
	l.TitleStyle = ui.NewStyle(ui.ColorClear)
	l.SetRect(2*x/3, 0, x, y-1)
	l.Block.BorderStyle = ui.NewStyle(ui.ColorClear)
	l.SelectedRowStyle = ui.Style{
		Fg:       ui.ColorGreen,
		Bg:       ui.ColorClear,
		Modifier: ui.ModifierBold,
	}
	l.TextStyle = ui.Style{
		Fg: ui.ColorClear,
		Bg: ui.ColorClear,
	}
	return l
}

func NewProcessBar() *widgets.Gauge {
	x, y := ui.TerminalDimensions()
	p := widgets.NewGauge()
	p.Title = ProcessBar
	p.Percent = 0
	p.SetRect(0, int(float64(y)*0.94), 2*x/3, y-1)
	p.BarColor = ui.ColorGreen
	p.Block.BorderStyle = ui.NewStyle(ui.ColorClear)
	p.TitleStyle = ui.NewStyle(ui.ColorClear)
	return p
}

func (w *Widget) ScrollRight() {
	switch w.T.SelectedNode().Value.(type) {
	case *Example:
		e := ChangeToExample(w.T.SelectedNode())
		if len(w.T.SelectedNode().Nodes) == 0 {
			conflictOrDuplicate := false
			w.C.Walk(func(node *widgets.TreeNode) bool {
				targetNode := ChangeToExample(node)
				if e.isConflict(targetNode.OType) {
					conflictOrDuplicate = true
					log.Logger.Warnf("conflict catalog: %s - %s", GetOTypeValue(e.OType), GetOTypeValue(node.Value.(*Example).OType))
					return false
				}
				if contains(w.C, e.Value) {
					conflictOrDuplicate = true
					log.Logger.Warnf("duplicate: [%s] %s ", GetOTypeValue(e.OType), e.Value)
					return false
				}
				return true
			})
			if conflictOrDuplicate {
				return
			}
			newNode := widgets.TreeNode{
				Value: NewExample(e.Value, e.CType, e.OType),
			}
			var newChosen []*widgets.TreeNode
			w.C.Walk(func(treeNode *widgets.TreeNode) bool {
				newChosen = append(newChosen, treeNode)
				return true
			})
			newChosen = append(newChosen, &newNode)
			w.C.SetNodes(newChosen)
			w.C.ScrollBottom()
			w.C.Title = GetOTypeValue(e.OType)
		}
	case *Catalog:
		w.T.Expand()
	default:
		log.Logger.Warnf("unkown node type: %s", w.T.SelectedNode().Value)
	}
}

func (w *Widget) ScrollBackSpace() {
	if w.C.SelectedNode() != nil {
		value := w.C.SelectedNode().Value.String()
		removeTreeNode(w.C, value)
		if TreeLength(w.C) == 0 {
			w.C.Title = ""
		}
		w.C.ScrollUp()
	}
}

func (w *Widget) WalkTreeScript() (map[string][]string, error) {
	examples := make(map[string][]string)
	w.T.Walk(func(node *widgets.TreeNode) bool {
		switch example := node.Value.(type) {
		case *Example:
			switch example.OType {
			case Script, SafetyScript, OtherScript:
				value, err := example.getScriptValue()
				if err != nil {
					log.Logger.Error(err)
					return false
				}
				examples[example.Value] = value
			}
		}
		return true
	})
	return examples, nil
}

func (w *Widget) RefreshProcessBar(idx int) {
	x := (float64(idx) / float64(TreeLength(w.C))) * 100
	w.P.Percent = int(x) % 101
	w.C.ScrollDown()
	ui.Render(w.P, w.C)
}
