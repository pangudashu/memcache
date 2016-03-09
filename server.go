package memcache

import (
	"crypto/md5"
	"hash/crc32"
	"math"
	"strconv"
	"time"
)

const (
	VIRTUAL_NODE_PER = 160
)

type Server struct {
	Address  string
	Weight   int
	MaxConn  int
	InitConn int
	IdleTime time.Duration
	isActive bool
	pool     *ConnectionPool
	nodeList []uint32
}

type Nodes struct {
	serverNodeMap map[uint32]*Server //hask_key => Server
	nodeList      []uint32
	nodeCnt       int
}

func createServerNode(servers []*Server) *Nodes { /*{{{*/
	nodes := &Nodes{}

	total_weight := 0
	for _, v := range servers {
		if v.Weight > 0 {
			total_weight += v.Weight
		} else {
			v.Weight = 1
			total_weight++
		}
	}

	total_node := (VIRTUAL_NODE_PER / 4) * len(servers)

	nodes.serverNodeMap = make(map[uint32]*Server, (VIRTUAL_NODE_PER+4)*len(servers))
	nodes.nodeList = make([]uint32, (VIRTUAL_NODE_PER+4)*len(servers))

	cnt := 0
	for _, s := range servers {
		//create connection pool
		if s.pool == nil {
			s.pool = open(s.Address, s.MaxConn, s.InitConn, s.IdleTime)
		}

		//计算实际分配的虚拟节点数
		node_cnt := int(math.Ceil(float64(total_node) * (float64(s.Weight) / float64(total_weight))))

		s.nodeList = make([]uint32, node_cnt*4)

		for i := 0; i < node_cnt; i++ {
			node_position := createKetamaHash(s.Address, i)
			for j, node := range node_position {
				nodes.serverNodeMap[node] = s
				nodes.nodeList[cnt] = node

				s.nodeList[i*4+j] = node
				cnt++
			}
		}
	}
	quickSort(nodes.nodeList, 0, len(nodes.nodeList)-1)

	bad_node := 0
	for _, v := range nodes.nodeList {
		if v == 0 {
			bad_node++
		} else {
			break
		}
	}
	if bad_node > 0 {
		nodes.nodeList = nodes.nodeList[bad_node:]
	}

	nodes.nodeCnt = len(nodes.nodeList)

	return nodes
} /*}}}*/

//每个address节对应4个虚拟node
func createKetamaHash(key string, i int) []uint32 { /*{{{*/
	addr := key + "#" + strconv.Itoa(i)

	code_byte := md5.Sum([]byte(addr))

	hashs := make([]uint32, 4)
	for n := 0; n < 4; n++ {
		hashs[n] = (uint32(code_byte[3+n*4]&0xFF) << 24) | (uint32(code_byte[2+n*4]&0xFF) << 16) | (uint32(code_byte[1+n*4]&0xFF) << 8) | uint32(code_byte[0+n*4]&0xFF)
	}
	return hashs
} /*}}}*/

func quickSort(array []uint32, low, high int) { /*{{{*/
	if low >= high {
		return
	}

	key := array[low]
	tmpLow := low
	tmpHigh := high
	for {
		for array[tmpHigh] > key {
			tmpHigh--
		}
		for array[tmpLow] <= key && tmpLow < tmpHigh {
			tmpLow++
		}

		if tmpLow >= tmpHigh {
			break
		}
		array[tmpLow], array[tmpHigh] = array[tmpHigh], array[tmpLow]
	}
	//将key放到中间
	array[tmpLow], array[low] = array[low], array[tmpLow]

	quickSort(array, low, tmpLow-1)
	quickSort(array, tmpLow+1, high)
} /*}}}*/

var hash_crc32_table = crc32.MakeTable(0xFFFFFFFF)

func (nodes *Nodes) getServerByKey(key string) *Server { /*{{{*/
	hash_key := crc32.Checksum([]byte(key), hash_crc32_table)

	node := nodes.getNodeByHash(hash_key)

	if node == 0 {
		return nil
	} else {
		return nodes.serverNodeMap[node]
	}
} /*}}}*/

//折半查找
func (nodes *Nodes) getNodeByHash(hash_key uint32) (node uint32) { /*{{{*/
	if nodes.nodeCnt < 1 {
		return 0
	}
	//大于最后一个节点分配给第一个节点
	if hash_key > nodes.nodeList[nodes.nodeCnt-1] {
		return nodes.nodeList[0]
	}

	left := 0
	right := nodes.nodeCnt
	index := 0
	for {
		mid_index := (right + left) / 2
		if hash_key == nodes.nodeList[mid_index] {
			index = mid_index
			break
		}
		if hash_key > nodes.nodeList[mid_index] {
			left = mid_index
		} else {
			right = mid_index
		}

		if right-left == 1 {
			index = right
			break
		}
	}
	return nodes.nodeList[index]
} /*}}}*/
