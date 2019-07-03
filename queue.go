package queue

import(
	"runtime"
	"sync/atomic"
	"errors"
)

// 错误变量命名
var (
	ErrDisposed = errors.New(`队列: 队列资源被释放`)
	ErrTimeout = errors.New(`队列: 超时错误!`)
	ErrEmptyQueue = errors.New(`队列: 空队列错误!`)
)

type Queue interface{
	Put(interface{}) (bool, error)
	Get()(interface{}, error)
	Offer(interface{}) (bool, error)
	Length() uint64
	IsEmpty() bool
	IsFull() bool
	Dispose()
	IsDisposed() bool
}

// 队列数据存储结构
type cache struct{
	pos		uint64
	value	interface{}
}

/*
默认队列是一个环形队列:
容量取为2的N次幂
这样计算索引时可以使用 pos & mask;

padding的作用是解决CPU cache line问题：
字段在内存中是连续存放的, pusPos和getPos使用了使用了16字节的内存(主流的CPU Cache的Cache Line大小都是64字节),
它们很可能被存放在同一个cache line中,修改其中的任何一个字段都会导致其它字段缓存被淘汰，因此读取操作将会变慢
给结构体的字段直接增加padding.每一个padding都跟一个CPU cache line一样大，这样就能确保每个在一个cache line中
*/
type DefaultQueue struct{
	_padding0      [8]uint64
	putPos		uint64 //生产者put所在位置，入队标识
	_padding1      [8]uint64
	getPos		uint64 //消费者get所在位置，出队标识
	_padding2      [8]uint64
	capacity	uint64 //队列容量
	_padding3      [8]uint64
	mask 		uint64 // 一个队列长度得掩码，可以用来和pos做且操作
	_padding4      [8]uint64
	disposed	uint64 //dispose用于对资源进行释放操作
	_padding5      [8]uint64
	data		[]*cache //缓存数据
}

// 插入数据
func (q *DefaultQueue)Put(item interface{}) (bool, error){
	return q.put(item, false)
}

func (q *DefaultQueue)Offer(item interface{}) (bool, error){
	return q.put(item, true)
}

func (q *DefaultQueue)put(item interface{}, offer bool) (bool, error){
	var c *cache
	putPos := atomic.LoadUint64(&q.putPos)
	i := 0 //重复失败次数计数器

	BREAKLABEL: //break 标签，用于跳出循环不再执行for循环里的代码
	for {
		if atomic.LoadUint64(&q.disposed) == 1{
			return false, ErrDisposed
		}

		// putPos和掩码做且运算, 可以得到掩码或者putPos值中的最小者
		c = q.data[putPos&q.mask]

		// 队列粒度编号
		seq := atomic.LoadUint64(&c.pos)

		// dif表示 颗粒度编号和插入队列的位置的差值
		// 如果相等则插入数据
		// 如果颗粒编号比队列编号大，则说明改位置已被插入，需要重新取putPos
		switch dif := seq - putPos;{
		case dif == 0:
			// 原子粒度的比较交换，比较旧值和地址中被操作值，如果相等, 被操作值赋予新值
			// q.putPos + 1
			if atomic.CompareAndSwapUint64(&q.putPos, putPos, putPos+1){
				break BREAKLABEL
			}
		case dif < 0:
			panic(`Ring buffer in a compromised state during a put operation.`)
		default:
			// 如果seq大了，putPos就重新获取数据
			putPos = atomic.LoadUint64(&q.putPos)
		}

		if offer{
			return false, nil
		}

		// 如果重复10000次失败，切换到其他协程运行
		if i == 10000 {
			runtime.Gosched() // 用于让出CPU时间片,切换协程
			i = 0
		} else {
			// runtime.Gosched()
			i++
		}
	}

	// 将item存入指定cache中
	c.value = item
	atomic.StoreUint64(&c.pos, putPos+1) //将cache pos +1，表示该位置已经存入一个值
	return true, nil
}

// 获取数据
func (q *DefaultQueue)Get() (interface{}, error) {
	var c *cache
	getPos := atomic.LoadUint64(&q.getPos)
	i := 0 //重复失败次数计数器 

	BREAKLABEL: //break 标签，用于跳出循环不再执行for循环里的代码
	for {
		if atomic.LoadUint64(&q.disposed) == 1{
			return false, ErrDisposed
		}

		// getPos和掩码做且运算
		c = q.data[getPos&q.mask]

		seq := atomic.LoadUint64(&c.pos)

		// put方法会让cache.pos+1，也就是seq = getPos + 1，则可以执行get
		switch dif := seq - (getPos + 1);{
		case dif == 0:
			if atomic.CompareAndSwapUint64(&q.getPos, getPos, getPos+1){
				break BREAKLABEL
			}
		case dif < 0:
			panic(`Ring buffer in compromised state during a get operation.`)
		default:
			// 如果seq大了，getPos就重新获取数据
			getPos = atomic.LoadUint64(&q.getPos)
		}

		// 如果重复10000次失败，切换到其他协程运行
		if i == 10000 {
			runtime.Gosched() // 用于让出CPU时间片,切换协程
			i = 0
		} else {
			i++
		}
	}
	result := c.value //cache里存的值
	c.value = nil
	// q.mask+1 是一个ring循环,正好一圈
	// 也就是获取值后，ring pos 增大一圈
	atomic.StoreUint64(&c.pos, getPos+q.mask+1)
	return result, nil
}

// 队列长度
func (q *DefaultQueue) Length() uint64 {
	return atomic.LoadUint64(&q.putPos) - atomic.LoadUint64(&q.getPos)
}

func (q *DefaultQueue)IsEmpty() bool{
	return q.Length() == 0
}

func (q *DefaultQueue)IsFull() bool{
	return q.Length() == q.capacity
}

// 释放队列资源
func (q *DefaultQueue) Dispose() {
	atomic.CompareAndSwapUint64(&q.disposed, 0, 1)
}

// 是否释放资源
func (q *DefaultQueue) IsDisposed() bool {
	return atomic.LoadUint64(&q.disposed) == 1
}

// 创建一个DefaultQueue结构体
func NewDefaultQueue(size uint64) *DefaultQueue{
	size = roundUp(size)
	data := make([]*cache, size)
	for i := uint64(0); i < size; i++ {
		data[i] = &cache{pos: i}
	}
	mask := size - 1 // 构建一个队列长度的所有值为1 的掩码
	return &DefaultQueue{
		capacity: size,
		mask: mask,
		data: data,
	}
}

// 保证生成2整除得数字
func roundUp(v uint64) uint64 {
	v--
	v |= v >> 1
	v |= v >> 2
	v |= v >> 4
	v |= v >> 8
	v |= v >> 16
	v |= v >> 32
	v++
	return v
}