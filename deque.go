package deque

// minCapacity is the smallest capacity that deque may have.
// Must be power of 2 for bitwise modulus: x % n == x & (n - 1).
// 最小容量；要求是2的幂次
const minCapacity = 16

// Deque represents a single instance of the deque data structure.
type Deque struct {
	buf    []interface{} // 缓冲区
	head   int           // 头指针 指向的是头部的位置
	tail   int           // 尾指针 指向的是尾部的下一个位置 下次pushBack应该放置的位置
	count  int           // 计数
	minCap int           // 最小容量
}

// Len returns the number of elements currently stored in the queue.
// 获取其中的元素个数
func (q *Deque) Len() int {
	return q.count
}

// PushBack appends an element to the back of the queue.  Implements FIFO when
// elements are removed with PopFront(), and LIFO when elements are removed
// with PopBack().
// 向末尾添加一个元素
func (q *Deque) PushBack(elem interface{}) {
	q.growIfFull()
	// 放到最后的位置上
	q.buf[q.tail] = elem
	// Calculate new tail position.
	// 计算尾部节点的位置
	q.tail = q.next(q.tail)
	// 元素个数减1
	q.count++
}

// PushFront prepends an element to the front of the queue.
// 向头部添加一个元素
func (q *Deque) PushFront(elem interface{}) {
	q.growIfFull()

	// Calculate new head position.
	// 计算头部的位置
	q.head = q.prev(q.head)
	q.buf[q.head] = elem
	// 元素个数加1
	q.count++
}

// PopFront removes and returns the element from the front of the queue.
// Implements FIFO when used with PushBack().  If the queue is empty, the call
// panics.
// 从头部节点中获取一个元素
func (q *Deque) PopFront() interface{} {
	if q.count <= 0 {
		// 这里panic好吗？所以用的时候一定要结合着长度字段来一起使用是吗
		panic("deque: PopFront() called on empty queue")
	}
	// 获取头部元素，然后将头部位置置为nil
	ret := q.buf[q.head]
	q.buf[q.head] = nil
	// Calculate new head position.
	// 将头部节点的位置向后挪
	q.head = q.next(q.head)
	// 数量--
	q.count--
	// 必要的时候进行缩容
	q.shrinkIfExcess()
	return ret
}

// PopBack removes and returns the element from the back of the queue.
// Implements LIFO when used with PushBack().  If the queue is empty, the call
// panics.
func (q *Deque) PopBack() interface{} {
	if q.count <= 0 {
		// 这里同上
		panic("deque: PopBack() called on empty queue")
	}
	// 向后一步获取尾部节点的位置
	// Calculate new tail position
	q.tail = q.prev(q.tail)
	// 获取尾部节点，同时将位置上的数据置为nil
	// Remove value at tail.
	ret := q.buf[q.tail]
	q.buf[q.tail] = nil
	// 数量--
	q.count--
	// 缩容
	q.shrinkIfExcess()
	return ret
}

// Front returns the element at the front of the queue.  This is the element
// that would be returned by PopFront().  This call panics if the queue is
// empty.
// 获取头结点
func (q *Deque) Front() interface{} {
	if q.count <= 0 {
		panic("deque: Front() called when empty")
	}
	// 获取头部节点的数据
	return q.buf[q.head]
}

// Back returns the element at the back of the queue.  This is the element
// that would be returned by PopBack().  This call panics if the queue is
// empty.
// 获取尾节点
func (q *Deque) Back() interface{} {
	if q.count <= 0 {
		panic("deque: Back() called when empty")
	}
	// 获取尾部节点的数据
	return q.buf[q.prev(q.tail)]
}

// At returns the element at index i in the queue without removing the element
// from the queue.  This method accepts only non-negative index values.  At(0)
// refers to the first element and is the same as Front().  At(Len()-1) refers
// to the last element and is the same as Back().  If the index is invalid, the
// call panics.
//
// The purpose of At is to allow Deque to serve as a more general purpose
// circular buffer, where items are only added to and removed from the ends of
// the deque, but may be read from any place within the deque.  Consider the
// case of a fixed-size circular log buffer: A new entry is pushed onto one end
// and when full the oldest is popped from the other end.  All the log entries
// in the buffer must be readable without altering the buffer contents.
func (q *Deque) At(i int) interface{} {
	// 获取第i个元素
	if i < 0 || i >= q.count {
		panic("deque: At() called with index out of range")
	}
	// bitwise modulus
	// 根据头节点的位置和buf的长度，计算位置
	return q.buf[(q.head+i)&(len(q.buf)-1)]
}

// Clear removes all elements from the queue, but retains the current capacity.
// This is useful when repeatedly reusing the queue at high frequency to avoid
// GC during reuse.  The queue will not be resized smaller as long as items are
// only added.  Only when items are removed is the queue subject to getting
// resized smaller.
func (q *Deque) Clear() {
	// bitwise modulus
	// 依次将buffer内的所有数据都置为nil
	modBits := len(q.buf) - 1
	for h := q.head; h != q.tail; h = (h + 1) & modBits {
		q.buf[h] = nil
	}
	// 各种位置都进行复位
	q.head = 0
	q.tail = 0
	q.count = 0
}

// Rotate rotates the deque n steps front-to-back.  If n is negative, rotates
// back-to-front.  Having Deque provide Rotate() avoids resizing that could
// happen if implementing rotation using only Pop and Push methods.
func (q *Deque) Rotate(n int) {
	// 如果数量小于等于1的时候，不需要进行rotate；因为正反一致
	if q.count <= 1 {
		return
	}
	// Rotating a multiple of q.count is same as no rotation.
	// 先根据count取余；因为结果一致；没有必要rotate那么多步
	n %= q.count
	if n == 0 {
		return
	}
	//  取模用的
	modBits := len(q.buf) - 1
	// If no empty space in buffer, only move head and tail indexes.
	// 如果头尾节点处于同一个位置 这个时候有可能是啥也没有；也有可能是满了；
	// 这两种情况都属于直接挪就可以了；因为已经构成一个循环链表了，头在哪里都可以
	if q.head == q.tail {
		// Calculate new head and tail using bitwise modulus.
		// 重新计算头部尾部的位置
		q.head = (q.head + n) & modBits
		q.tail = (q.tail + n) & modBits
		return
	}
	// 说明是往前rotate
	if n < 0 {
		// Rotate back to front.
		for ; n < 0; n++ {
			// Calculate new head and tail using bitwise modulus.
			// 先往前挪一位
			q.head = (q.head - 1) & modBits
			q.tail = (q.tail - 1) & modBits
			// Put tail value at head and remove value at tail.
			// 而后把尾部的值放到头部
			q.buf[q.head] = q.buf[q.tail]
			q.buf[q.tail] = nil
		}
		return
	}
	// 说明是往后rotate
	// Rotate front to back.
	for ; n > 0; n-- {
		// Put head value at tail and remove value at head.
		// 把头部的值放到尾部；同时删除头部的值
		q.buf[q.tail] = q.buf[q.head]
		q.buf[q.head] = nil
		// Calculate new head and tail using bitwise modulus.
		// 重新计算头部尾部的位置
		q.head = (q.head + 1) & modBits
		q.tail = (q.tail + 1) & modBits
	}
}

// SetMinCapacity sets a minimum capacity of 2^minCapacityExp.  If the value of
// the minimum capacity is less than or equal to the minimum allowed, then
// capacity is set to the minimum allowed.  This may be called at anytime to
// set a new minimum capacity.
//
// Setting a larger minimum capacity may be used to prevent resizing when the
// number of stored items changes frequently across a wide range.
// 设置minCap；输入的参数居然是一个minCapacityExp
func (q *Deque) SetMinCapacity(minCapacityExp uint) {
	if 1<<minCapacityExp > minCapacity {
		q.minCap = 1 << minCapacityExp
	} else {
		// 最小的容量是minCap
		q.minCap = minCapacity
	}
}

// prev returns the previous buffer position wrapping around buffer.
// 计算前一个节点的位置；这个是个循环数组
func (q *Deque) prev(i int) int {
	return (i - 1) & (len(q.buf) - 1) // bitwise modulus
}

// next returns the next buffer position wrapping around buffer.
// 计算下一个节点的位置；这个是个循环数组
func (q *Deque) next(i int) int {
	return (i + 1) & (len(q.buf) - 1) // bitwise modulus
}

// growIfFull resizes up if the buffer is full.
// 当buffer满了的时候进行扩容
func (q *Deque) growIfFull() {
	if len(q.buf) == 0 {
		// 最小的容量是16
		if q.minCap == 0 {
			q.minCap = minCapacity
		}
		// 直接申请了16个这么大的容量
		q.buf = make([]interface{}, q.minCap)
		return
	}
	// 如果count和len相等，说明当前情况下已经满了；则进行resize 扩容
	if q.count == len(q.buf) {
		q.resize()
	}
}

// shrinkIfExcess resize down if the buffer 1/4 full.
// 当容量小于1/4的时候进行缩容
func (q *Deque) shrinkIfExcess() {
	// 当前缓冲区的容量超过最小容量；同时count扩大4倍才和buf的容量相等，则进行缩容
	if len(q.buf) > q.minCap && (q.count<<2) == len(q.buf) {
		q.resize()
	}
}

// resize resizes the deque to fit exactly twice its current contents.  This is
// used to grow the queue when it is full, and also to shrink it when it is
// only a quarter full.
// 在这里进行扩容和缩容
func (q *Deque) resize() {
	// 在当前的基础之上扩2倍
	newBuf := make([]interface{}, q.count<<1)
	// 尾巴在后面，头在前面；没有行成折叠
	if q.tail > q.head {
		// 将老buf中的数据copy到新buf中
		copy(newBuf, q.buf[q.head:q.tail])
	} else {
		// 如果形成了折叠
		// 从头部到buf的最末端；这是第一段；第二段是从0到tail这一段
		n := copy(newBuf, q.buf[q.head:])
		// copy第二段数据
		copy(newBuf[n:], q.buf[:q.tail])
	}
	// 重新
	q.head = 0
	q.tail = q.count
	// 把新的buf copy到老的上面去
	q.buf = newBuf
}
