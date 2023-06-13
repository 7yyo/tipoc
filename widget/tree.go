package widget

import (
	"bufio"
	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"io/fs"
	"net"
	"path/filepath"
	"pictorial/comp"
	"pictorial/log"
	"strings"
)

func NewTree() (*widgets.Tree, error) {
	tree := widgets.NewTree()
	treeNode, err := buildTreeByCatalog()
	if err != nil {
		return nil, err
	}
	if err := appendOther(&treeNode); err != nil {
		return nil, err
	}
	tree.SetNodes(treeNode)
	walkTree(tree)
	if err := appendComponent(tree); err != nil {
		return nil, err
	}
	return tree, nil
}

func buildTreeByCatalog() ([]*widgets.TreeNode, error) {

	f, err := catalogPath.Open(catalog)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var treeNodes []*widgets.TreeNode
	var root *widgets.TreeNode
	var parentNodes []*widgets.TreeNode
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		level := strings.Count(line, ".")
		name := strings.TrimLeft(line, "\t")
		node := widgets.TreeNode{
			Value: newCatalog(name),
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
	return treeNodes, nil
}

var OtherConfig string

func appendOther(treeNodes *[]*widgets.TreeNode) error {
	if OtherConfig != "" {
		othersNode := widgets.TreeNode{
			Value: newCatalog(OtherConfig),
		}
		if err := filepath.Walk(OtherConfig, func(path string, info fs.FileInfo, err error) error {
			if info == nil {
				log.Logger.Warnf("otherConfig %s is nil or not exists, skip", path)
				return nil
			}
			if !info.IsDir() {
				e := NewExample(info.Name(), -1, OtherScript)
				appendExampleNode(&othersNode, e)
			}
			return nil
		}); err != nil {
			return err
		}
		*treeNodes = append(*treeNodes, &othersNode)
	}
	return nil
}

func walkTree(tree *widgets.Tree) {
	tree.Walk(func(node *widgets.TreeNode) bool {
		v := node.Value.String()
		if v == OtherConfig {
			return false
		}
		if len(node.Nodes) != 0 {
			node.Value = newCatalog(v)
		} else {
			idx := getIdxByName(v)
			if _, ok := OTypeMapping[idx]; ok {
				node.Value = newCatalog(v)
			} else {
				if isSafety(v) {
					node.Value = NewExample(v, -1, SafetyScript)
				} else {
					node.Value = NewExample(v, -1, Script)
				}
			}
		}
		return true
	})
}

var oTp OType

func appendComponent(tree *widgets.Tree) error {
	cs, err := comp.New()
	if err != nil {
		return err
	}
	tree.Walk(func(node *widgets.TreeNode) bool {
		switch node.Value.(type) {
		case *Catalog:
			idx := getIdxByName(node.Value.String())
			if _, ok := OTypeMapping[idx]; ok {
				oTp = OTypeMapping[idx]
				switch oTp {
				case Disaster:
					appendLabelNode(node, cs.Map)
				case DataCorrupted:
					appendComponentNode(node, cs.Map, []comp.CType{comp.TiKV, comp.PD})
				default:
					appendComponentNode(node, cs.Map, []comp.CType{comp.TiKV, comp.PD, comp.TiFlash, comp.PD, comp.TiDB})
				}
			}
		}
		return true
	})
	return nil
}

func appendComponentNode(node *widgets.TreeNode, m map[comp.CType][]comp.Component, tp []comp.CType) {
	for k, _ := range m {
		if hitCType(tp, k) {
			cTp := comp.GetCTypeValue(k)
			catalog := newCatalog(cTp)
			cNode := appendCatalogNode(node, catalog)
			for _, c := range m[k] {
				addr := net.JoinHostPort(c.Host, c.Port)
				e := NewExample(addr, k, oTp)
				appendExampleNode(cNode, e)
			}
		}
	}
}

func appendLabelNode(node *widgets.TreeNode, m map[comp.CType][]comp.Component) {
	labels := comp.GetLabelKey(m[comp.TiKV])
	for l, _ := range labels {
		c := newCatalog(l)
		cNode := appendCatalogNode(node, c)
		visited := make(map[string]bool)
		for _, s := range m[comp.TiKV] {
			value := s.Labels[l]
			if visited[value] {
				continue
			}
			visited[value] = true
			e := NewExample(value, comp.TiKV, Disaster)
			appendExampleNode(cNode, e)
		}
	}
}

func hitCType(tp []comp.CType, c comp.CType) bool {
	if tp == nil {
		return true
	}
	for _, t := range tp {
		if t == c {
			return true
		}
	}
	return false
}

func appendExampleNode(treeNode *widgets.TreeNode, e *Example) *widgets.TreeNode {
	node := widgets.TreeNode{
		Value: e,
	}
	treeNode.Nodes = append(treeNode.Nodes, &node)
	return &node
}

func appendCatalogNode(treeNode *widgets.TreeNode, c *Catalog) *widgets.TreeNode {
	node := widgets.TreeNode{
		Value: c,
	}
	treeNode.Nodes = append(treeNode.Nodes, &node)
	return &node
}

func removeTreeNode(tree *widgets.Tree, value string) {
	var newNodes []*widgets.TreeNode
	tree.Walk(func(treeNode *widgets.TreeNode) bool {
		if treeNode.Value.String() != value {
			newNodes = append(newNodes, treeNode)
		}
		return true
	})
	tree.SetNodes(newNodes)
}

func TreeLength(tree *widgets.Tree) int {
	length := 0
	tree.Walk(func(node *widgets.TreeNode) bool {
		length++
		return true
	})
	return length
}

func contains(tree *widgets.Tree, value string) bool {
	arr := treeToArray(tree)
	set := make(map[string]bool)
	for _, v := range arr {
		set[v] = true
	}
	return set[value]
}

func treeToArray(tree *widgets.Tree) []string {
	var array []string
	tree.Walk(func(treeNode *widgets.TreeNode) bool {
		array = append(array, treeNode.Value.String())
		return true
	})
	return array
}

func CleanTree(tree *widgets.Tree) {
	var nodes []*widgets.TreeNode
	tree.SetNodes(nodes)
}

func ScrollTopTree(tree *widgets.Tree) {
	tree.ScrollTop()
	ui.Render(tree)
}
