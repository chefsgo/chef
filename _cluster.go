package chef

import (
	"fmt"
	"sync"

	"github.com/hashicorp/memberlist"
)

//待处理，集群需要驱动化
//并且集群是一个特殊化的模块，一开始就要连接的

type (
	share   = map[string][]byte
	cluster struct {
		mutex      sync.RWMutex
		core       *kernel
		shares     share
		client     *memberlist.Memberlist
		broadcasts *memberlist.TransmitLimitedQueue
	}
	broadcast struct {
		msg    []byte
		notify chan<- struct{}
	}
)

func (c *cluster) connect() error {
	// name := core.config.role

	// meta := Map{
	// 	"name":    core.config.name,
	// 	"role":    core.config.role,
	// 	"version": core.config.version,
	// }

	// //解析元数据

	// config := memberlist.DefaultLANConfig()
	// config.Name = name
	// config.BindAddr = host
	// config.BindPort = ppp
	// config.Events = c
	// config.Delegate = c
	// config.LogOutput = io.Discard
	// if key != "" {
	// 	config.SecretKey = []byte(key)
	// }

	// client, err := memberlist.Create(config)
	// if err != nil {
	// 	return err
	// }

	// if joins != nil && len(joins) > 0 {
	// 	_, err = client.Join(joins)
	// 	if err != nil {
	// 		return err
	// 	}
	// }

	// c.client = client
	// c.broadcasts = &memberlist.TransmitLimitedQueue{
	// 	NumNodes: func() int {
	// 		return client.NumMembers()
	// 	},
	// 	RetransmitMult: 3,
	// }

	return nil
}

// Invalidates
func (b *broadcast) Invalidates(other memberlist.Broadcast) bool {
	return false
}

// Message
func (b *broadcast) Message() []byte {
	return b.msg
}

// Finished
func (b *broadcast) Finished() {
	if b.notify != nil {
		close(b.notify)
	}
}

func (c *cluster) NotifyJoin(node *memberlist.Node) {
	fmt.Println("NotifyJoin", node.Address())
}
func (c *cluster) NotifyLeave(node *memberlist.Node) {
	fmt.Println("NotifyLeave", node.Address())
}
func (c *cluster) NotifyUpdate(node *memberlist.Node) {
	fmt.Println("NotifyUpdate", node.Address())
}

func (c *cluster) NodeMeta(limit int) []byte {
	fmt.Println("NodeMeta", limit)
	return []byte{}
}

func (c *cluster) NotifyMsg(b []byte) {
	fmt.Println("NotifyMsg", len(b))

	if len(b) == 0 {
		return
	}

	var shares share
	err := JSONUnmarshal(b, &shares)
	if err != nil {
		return
	}

	c.mutex.Lock()
	for k, v := range shares {
		if v == nil {
			delete(c.shares, k)
		} else {
			c.shares[k] = v
		}
	}
	c.mutex.Unlock()
}

func (c *cluster) GetBroadcasts(overhead, limit int) [][]byte {
	return c.broadcasts.GetBroadcasts(overhead, limit)
}
func (c *cluster) LocalState(join bool) []byte {
	c.mutex.RLock()
	shares := c.shares
	c.mutex.RUnlock()

	bytes, err := JSONMarshal(shares)
	if err != nil {
		return []byte{}
	}

	return bytes
}
func (c *cluster) MergeRemoteState(buf []byte, join bool) {
	fmt.Println("MergeRemoteState", len(buf), join)
	if false == join || nil == buf || 0 == len(buf) {
		return
	}

	var shares share
	if err := JSONMarshal(buf, &shares); err != nil {
		return
	}
	c.mutex.Lock()
	for k, v := range shares {
		c.shares[k] = v
	}
	c.mutex.Unlock()
}
