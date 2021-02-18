package gee

import "strings"

type node struct {
	// 待匹配路由，例如 /p/:lang，只有在最后一层存储
	// 搜索时到达最后一层并且pattern不为空则匹配成功
	pattern string
	// 路由中的一部分，例如 :lang
	part string
	// 子节点，例如 [doc, tutorial, intro]
	children []*node
	// 是否精确匹配，part 含有 : 或 * 时为true
	isWild bool
}

//第一个匹配成功的节点，用于insert
func (n *node) matchChild(part string) *node {
	//遍历node节点的子节点
	for _, child := range n.children {
		//如果孩子节点的part == 搜索的part 或 该子节点是模糊匹配节点
		if child.part == part || child.isWild {
			//返回第一个查找到的子节点
			return child
		}
	}
	return nil
}

//所有匹配成功的节点，用于search
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	//遍历node节点的子节点
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

//pattern 插入的URL
//parts []string 为pattern按照/进行分割得到的切片
//height 当前插入高度
func (n *node) insert(pattern string, parts []string, height int) {
	//到达插入高度，为pattern赋值
	//只有当前node的pattern不为空则匹配成功
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	part := parts[height]
	//获取第一个匹配成功的节点
	child := n.matchChild(part)
	//该节点下没有匹配成功的子节点
	if child == nil {
		//插入新的节点，不为pattern赋值
		child = &node{part: part, isWild: part[0] == ':' || part[0] == '*'}
		n.children = append(n.children, child) //父节点的children插入新建的子节点
	}
	//递归调用
	child.insert(pattern, parts, height+1)
}

func (n *node) search(parts []string, height int) *node {
	//1、匹配到了第len(parts)层节点 2、*通配符
	if len(parts) == height || strings.HasPrefix(n.part, "*") {
		//如果pattern为空，匹配失败（对应情况1失败）
		if n.pattern == "" {
			return nil
		}
		//否则返回第n层节点（情况1：匹配成功，情况2：通配）
		return n
	}

	part := parts[height]
	//获取匹配成功的节点列表
	children := n.matchChildren(part)
	//遍历匹配成功的节点列表
	for _, child := range children {
		//递归调用
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}
	return nil
}
