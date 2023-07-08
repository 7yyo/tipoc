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
	"pictorial/operator"
	"strings"
)

var OtherConfig string

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

	catalog, err := readCatalog()
	if err != nil {
		return nil, err
	}
	defer catalog.Close()

	var treeNodes []*widgets.TreeNode
	var root *widgets.TreeNode
	var parentNodes []*widgets.TreeNode

	scanner := bufio.NewScanner(catalog)
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

func appendOther(treeNodes *[]*widgets.TreeNode) error {
	if OtherConfig != "" {
		othersNode := widgets.TreeNode{
			Value: newCatalog(OtherConfig),
		}
		if err := filepath.Walk(OtherConfig, func(path string, info fs.FileInfo, err error) error {
			if info == nil {
				log.Logger.Warnf("otherConfig %s is empty or not exists, skip", path)
				return nil
			}
			if !info.IsDir() {
				e := NewExample(info.Name(), -1, operator.OtherScript)
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
			idx := getIdxByValue(v)
			if IsCompMapping(idx) {
				node.Value = newCatalog(v)
			} else {
				switch {
				case isSafety(v):
					node.Value = NewExample(v, comp.NoBody, operator.SafetyScript)
				case isLoadDataTPCC(v):
					node.Value = NewExample(v, comp.NoBody, operator.LoadDataTPCC)
				case isLoadDataImportInto(v):
					node.Value = NewExample(v, comp.NoBody, operator.LoadDataImportInto)
				case isLoadData(v):
					node.Value = NewExample(v, comp.NoBody, operator.LoadData)
				case isSelectIntoOutFile(v):
					node.Value = NewExample(v, comp.NoBody, operator.LoadDataSelectIntoOutFile)
				case isDataSeparation(v):
					node.Value = NewExample(v, comp.NoBody, operator.DataSeparation)
				case isDataDistribution(v):
					node.Value = NewExample(v, comp.NoBody, operator.DataDistribution)
				case isOnlineDDLAddIndex(v):
					node.Value = NewExample(v, comp.NoBody, operator.OnlineDDLAddIndex)
				case isInstallSysBench(v):
					node.Value = NewExample(v, comp.NoBody, operator.InstallSysBench)
				default:
					node.Value = NewExample(v, comp.NoBody, operator.Script)
				}
			}
		}
		return true
	})
}

var oTp operator.OType

func appendComponent(tree *widgets.Tree) error {
	cs, err := comp.New()
	if err != nil {
		return err
	}
	tree.Walk(func(node *widgets.TreeNode) bool {
		switch node.Value.(type) {
		case *Catalog:
			idx := getIdxByValue(node.Value.String())
			if IsCompMapping(idx) {
				oTp = OTypeCompMapping[idx]
				switch oTp {
				case operator.Disaster:
					appendLabelNode(node, cs.Map)
				case operator.DataCorrupted, operator.DiskFull:
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
	var addr string
	for cType, _ := range m {
		if hitCType(tp, cType) {
			cTp := comp.GetCTypeValue(cType)
			catalog := newCatalog(cTp)
			cNode := appendCatalogNode(node, catalog)
			for _, c := range m[cType] {
				addr = net.JoinHostPort(c.Host, c.Port)
				e := NewExample(addr, cType, oTp)
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
			e := NewExample(value, comp.TiKV, operator.Disaster)
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
