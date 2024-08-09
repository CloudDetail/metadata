package cache

import (
	"sync"

	"github.com/CloudDetail/metadata/model/resource"
)

var _ resource.ResHandler = &NodeList{}

type NodeList struct {
	*resource.Resources

	UIDMap  sync.Map
	IP2Node sync.Map
}

func NewNodeList(_ resource.ResType, resList []*resource.Resource) resource.ResHandler {
	nl := &NodeList{
		Resources: &resource.Resources{
			ResType: resource.NodeType,
			ResList: resList,
		},
	}

	if resList == nil {
		nl.Resources.ResList = []*resource.Resource{}
		return nl
	}

	// 重建查询表
	for _, res := range resList {
		node := Node{
			Resource: res,
		}
		nl.IP2Node.Store(node.NodeIP(), &node)
		nl.UIDMap.Store(node.ResUID, &node)
	}

	return nl
}

func (nl *NodeList) Reset(resList []*resource.Resource) {
	newIP2NodeMap := sync.Map{}
	newNodeUIDMap := sync.Map{}
	// 重建查询表
	for _, res := range resList {
		node := Node{
			Resource: res,
		}
		newIP2NodeMap.Store(node.NodeIP(), &node)
		newNodeUIDMap.Store(node.ResUID, &node)
	}
	nl.IP2Node = newIP2NodeMap
	nl.UIDMap = newNodeUIDMap
}

func (nl *NodeList) GetNodeByIP(nodeIP string) *Node {
	val, find := nl.IP2Node.Load(nodeIP)
	if find {
		return val.(*Node)
	}
	return nil
}

func (nl *NodeList) AddResource(res *resource.Resource) {
	node := Node{
		Resource: res,
	}
	nl.IP2Node.Store(node.NodeIP(), &node)
	nl.UIDMap.Store(node.ResUID, &node)
	nl.Resources.AddResource(res)
}

func (nl *NodeList) UpdateResource(res *resource.Resource) {
	node := Node{
		Resource: res,
	}

	oldNode, find := nl.UIDMap.Load(res.ResUID)
	if find {
		oldNode, ok := oldNode.(*Node)
		if ok && oldNode.NodeIP() != node.NodeIP() {
			nl.IP2Node.Delete(oldNode.NodeIP())
		}
	}

	nl.IP2Node.Store(node.NodeIP(), &node)
	nl.UIDMap.Store(node.ResUID, &node)
	nl.Resources.UpdateResource(res)
}

func (nl *NodeList) DeleteResource(res *resource.Resource) {
	node := Node{
		Resource: res,
	}
	nl.IP2Node.Delete(node.NodeIP())
	nl.UIDMap.Delete(node.ResUID)
	nl.Resources.DeleteResource(res)
}

type Node struct {
	*resource.Resource
}

func (node *Node) NodeIP() string {
	val, find := node.StringAttr[resource.NodeInternalIP]
	if find {
		return val
	}
	return node.StringAttr[resource.NodeExternalIP]
}

func (node *Node) InternalIP() string {
	return node.StringAttr[resource.NodeInternalIP]
}

func (node *Node) ExternalIP() string {
	return node.StringAttr[resource.NodeExternalIP]
}

func (node *Node) NodeHostName() string {
	return node.StringAttr[resource.NodeHostName]
}
