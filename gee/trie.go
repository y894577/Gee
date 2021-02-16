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
	for _, child := range n.children {
		if child.part == part || child.isWild {
			return child
		}
	}
	return nil
}

//所有匹配成功的节点，用于search
func (n *node) matchChildren(part string) []*node {
	nodes := make([]*node, 0)
	for _, child := range n.children {
		if child.part == part || child.isWild {
			nodes = append(nodes, child)
		}
	}
	return nodes
}

//parts []string 匹配规则
func (n *node) insert(pattern string, parts []string, height int) {
	//到达插入高度，为pattern赋值
	if len(parts) == height {
		n.pattern = pattern
		return
	}

	part := parts[height]
	//获取第一个匹配成功的节点
	child := n.matchChild(part)
	//该节点不匹配，没有匹配到当前part的节点
	if child == nil {
		//插入新的节点，没有为pattern赋值
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

	for _, child := range children {
		result := child.search(parts, height+1)
		if result != nil {
			return result
		}
	}
	return nil
}

func (r *router) getRoute(method string, path string) (*node, map[string]string) {
	searchParts := parsePattern(path)
	params := make(map[string]string)
	root, ok := r.roots[method]

	if !ok {
		return nil, nil
	}

	n := root.search(searchParts, 0) //查找是否匹配成功

	if n != nil { //n不为空，匹配成功
		parts := parsePattern(n.pattern)
		for index, part := range parts {
			if part[0] == ':' {
				params[part[1:]] = searchParts[index]
			}
			if part[0] == '*' && len(part) > 1 {
				params[part[1:]] = strings.Join(searchParts[index:], "/")
				break
			}
		}
		return n, params
	}

	return nil, nil
}
